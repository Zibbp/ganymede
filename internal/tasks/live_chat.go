package tasks

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/utils"
)

// //////////////////////
// Download Chat (VOD) //
// //////////////////////
type DownloadLiveChatArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (DownloadLiveChatArgs) Kind() string { return string(utils.TaskDownloadLiveChat) }

func (args DownloadLiveChatArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1,
		Tags:        []string{"archive"},
		UniqueOpts:  archiveUniqueOpts(),
	}
}

func (w *DownloadLiveChatWorker) Timeout(job *river.Job[DownloadLiveChatArgs]) time.Duration {
	return 49 * time.Hour
}

type DownloadLiveChatWorker struct {
	river.WorkerDefaults[DownloadLiveChatArgs]
}

func (w DownloadLiveChatWorker) Work(ctx context.Context, job *river.Job[DownloadLiveChatArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}
	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadChat,
	})
	if err != nil {
		return err
	}

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// Set chat start time
	if !dbItems.Queue.ChatStart.IsZero() {
		log.Debug().Str("task_id", fmt.Sprintf("%d", job.ID)).Msg("chat start time already set, skipping")
	} else {
		chatStartTime := time.Now()
		_, err = dbItems.Queue.Update().SetChatStart(chatStartTime).Save(ctx)
		if err != nil {
			return err
		}
	}

	// download chat
	log.Info().Str("task_id", fmt.Sprintf("%d", job.ID)).Msgf("starting live chat download for %s", dbItems.Channel.Name)
	err = exec.SaveTwitchLiveChatToFile(ctx, dbItems.Channel.Name, dbItems.Video.TmpLiveChatDownloadPath)
	remotelyCancelled := false
	var cancellationErr error
	if err != nil {
		if errors.Is(err, context.Canceled) {
			if !errors.Is(context.Cause(ctx), rivertype.ErrJobCancelledRemotely) {
				return err
			}
			remotelyCancelled = true
			cancellationErr = err
			finalizeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), liveArchiveFinalizationTimeout)
			defer cancel()
			ctx = finalizeCtx
		} else {
			return err
		}
	}

	next := []transactionalJob{}
	if job.Args.Continue {
		next = append(next, transactionalJob{Args: &ConvertLiveChatArgs{Continue: true, Input: nextArchiveInput(job.Args.Input)}})
	}
	err = setQueueStatusAndEnqueue(ctx, store, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadChat,
	}, next...)
	if err != nil {
		return err
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}
	if remotelyCancelled {
		return cancellationErr
	}

	return nil
}

// ////////////////////
// Convert Live Chat //
// ///////////////////
type ConvertLiveChatArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (ConvertLiveChatArgs) Kind() string { return string(utils.TaskConvertChat) }

func (args ConvertLiveChatArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Tags:        []string{"archive"},
		UniqueOpts:  archiveUniqueOpts(),
	}
}

func (w *ConvertLiveChatWorker) Timeout(job *river.Job[ConvertLiveChatArgs]) time.Duration {
	return 49 * time.Hour
}

type ConvertLiveChatWorker struct {
	river.WorkerDefaults[ConvertLiveChatArgs]
}

func (w ConvertLiveChatWorker) Work(ctx context.Context, job *river.Job[ConvertLiveChatArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskConvertChat,
	})
	if err != nil {
		return err
	}

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// Ensure any crash-left pending live chat messages are merged into the
	// primary JSON file before conversion/post-processing reads it.
	if err := exec.RecoverTwitchLiveChatPendingFile(dbItems.Video.TmpLiveChatDownloadPath); err != nil {
		return err
	}

	// check that the chat file exists
	if !utils.FileExists(dbItems.Video.TmpLiveChatDownloadPath) {
		log.Info().Str("task_id", fmt.Sprintf("%d", job.ID)).Msg("chat file does not exist; setting chat status to complete")

		// set queue status to completed
		_, err := dbItems.Queue.Update().SetTaskChatConvert(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).Save(ctx)
		if err != nil {
			return err
		}

		// set video chat to empty
		_, err = dbItems.Video.Update().SetChatPath("").SetChatVideoPath("").Save(ctx)
		if err != nil {
			return err
		}

		return checkIfTasksAreDone(ctx, store.Client, job.Args.Input)
	}

	// get channel
	platform, err := PlatformFromContext(ctx)
	if err != nil {
		return err
	}
	channel, err := platform.GetChannel(ctx, &dbItems.Channel.Name, nil)
	if err != nil {
		return err
	}
	channelIdInt, err := strconv.Atoi(channel.ID)
	if err != nil {
		return err
	}

	// need the ID of a previous video for channel emotes and badges
	videos, err := platform.GetVideos(ctx, channel.ID, "archive", false, false)
	if err != nil {
		return err
	}

	// TODO: repalce with something else?
	// attempt to find video of current livestream
	var previousVideoID string
	for _, video := range videos {
		if video.ID == dbItems.Video.ExtID {
			previousVideoID = video.ID
			// update the video item in the database
			_, err = dbItems.Video.Update().SetExtID(video.ID).Save(ctx)
			if err != nil {
				return err
			}
			break
		}
	}

	// if no previous video, use the first video
	if previousVideoID == "" && len(videos) > 0 {
		previousVideoID = videos[0].ID
		// if no videos at all, use a random id
	} else if previousVideoID == "" {
		previousVideoID = "132195945"
	}

	// convert chat
	err = utils.ConvertTwitchLiveChatToTDLChat(dbItems.Video.TmpLiveChatDownloadPath, dbItems.Video.TmpLiveChatConvertPath, dbItems.Channel.Name, dbItems.Video.ID.String(), dbItems.Video.ExtID, channelIdInt, dbItems.Queue.ChatStart, string(previousVideoID))
	if err != nil {
		return err
	}

	// run TwitchDownloader "chatupdate" to embed emotes and badges
	err = exec.UpdateTwitchChat(ctx, dbItems.Video)
	if err != nil {
		return err
	}
	if err := utils.EnrichTwitchChatMetadataFromLiveChat(dbItems.Video.TmpLiveChatDownloadPath, dbItems.Video.TmpChatDownloadPath); err != nil {
		return err
	}

	next := []transactionalJob{}
	if job.Args.Continue {
		if dbItems.Queue.TaskChatRender != utils.Success {
			next = append(next, transactionalJob{Args: &RenderChatArgs{Continue: true, Input: nextArchiveInput(job.Args.Input)}})
		} else {
			next = append(next, transactionalJob{Args: &MoveChatArgs{Continue: true, Input: nextArchiveInput(job.Args.Input)}})
		}
	}
	err = setQueueStatusAndEnqueue(ctx, store, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskConvertChat,
	}, next...)
	if err != nil {
		return err
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}

	return nil
}
