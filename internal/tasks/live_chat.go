package tasks

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
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
	}
}

func (w DownloadLiveChatArgs) Timeout(job *river.Job[DownloadLiveChatArgs]) time.Duration {
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
	client := river.ClientFromContext[pgx.Tx](ctx)

	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadChat,
	})
	if err != nil {
		return err
	}

	// start task heartbeat
	go startHeartBeatForTask(ctx, HeartBeatInput{
		TaskId: job.ID,
		conn:   store.ConnPool,
	})

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
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// create new context to finish the task
			ctx = context.Background()
		} else {
			return err
		}
	}

	// set queue status to completed
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadChat,
	})
	if err != nil {
		return err
	}

	// continue with next job
	if job.Args.Continue {
		_, err := client.Insert(ctx, &ConvertLiveChatArgs{
			Continue: true,
			Input:    job.Args.Input,
		}, nil)
		if err != nil {
			return err
		}
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
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
	}
}

func (w ConvertLiveChatArgs) Timeout(job *river.Job[ConvertLiveChatArgs]) time.Duration {
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

	// start task heartbeat
	go startHeartBeatForTask(ctx, HeartBeatInput{
		TaskId: job.ID,
		conn:   store.ConnPool,
	})

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
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

		return nil
	}

	// get channel
	platform, err := PlatformFromContext(ctx)
	if err != nil {
		return err
	}
	channel, err := platform.GetChannel(ctx, dbItems.Channel.Name)
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

	// set queue status to completed
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskConvertChat,
	})
	if err != nil {
		return err
	}

	// continue with next job
	if job.Args.Continue {
		client := river.ClientFromContext[pgx.Tx](ctx)
		// render chat if needed
		if dbItems.Queue.TaskChatRender != utils.Success {
			_, err := client.Insert(ctx, &RenderChatArgs{
				Continue: true,
				Input:    job.Args.Input,
			}, nil)
			if err != nil {
				return err
			}
			// else move chat as rendering is not needed
		} else {
			_, err := client.Insert(ctx, &MoveChatArgs{
				Continue: true,
				Input:    job.Args.Input,
			}, nil)
			if err != nil {
				return err
			}
		}
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}

	return nil
}
