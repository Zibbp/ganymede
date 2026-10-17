package tasks

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/platform"
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

	// Set chat start time; it is the chat/video sync fallback anchor.
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

	// Convert chat truncated to the finalized video duration.
	videoDurationSec, err := resolveLiveChatVideoDurationSec(ctx, store, dbItems)
	if err != nil {
		return err
	}

	videoStartTime, err := resolveLiveChatVideoStartTime(ctx, store, platform, dbItems, videos, videoDurationSec)
	if err != nil {
		return err
	}

	err = utils.ConvertTwitchLiveChatToTDLChat(dbItems.Video.TmpLiveChatDownloadPath, dbItems.Video.TmpLiveChatConvertPath, dbItems.Channel.Name, dbItems.Video.ID.String(), dbItems.Video.ExtID, channelIdInt, videoStartTime, string(previousVideoID), videoDurationSec)
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

// resolveLiveChatVideoDurationSec returns the finalized duration for chat
// truncation. Zero means proceed without a cutoff; trimming never fails.
func resolveLiveChatVideoDurationSec(ctx context.Context, store *database.Database, dbItems *GetDatabaseItemsResponse) (int, error) {
	queueID := dbItems.Queue.ID.String()
	if dbItems.Video.Duration > 1 {
		return dbItems.Video.Duration, nil
	}

	if path := recoverableLiveVideoInputPath(&dbItems.Video); utils.FileExists(path) {
		if probeDuration, err := exec.GetVideoDuration(ctx, path); err == nil {
			return probeDuration, nil
		} else if ctx.Err() != nil {
			return 0, ctx.Err()
		}
	}

	// Temp may already be moved; probe finalized media before waiting.
	finalCandidates := []string{}
	if dbItems.Video.VideoHlsPath != "" {
		finalCandidates = append(finalCandidates,
			filepath.Join(dbItems.Video.VideoHlsPath, liveCaptureID(&dbItems.Video)+"-video.m3u8"))
	} else if dbItems.Video.VideoPath != "" {
		finalCandidates = append(finalCandidates, dbItems.Video.VideoPath)
	}
	for _, path := range finalCandidates {
		if !utils.FileExists(path) {
			continue
		}
		if probeDuration, err := exec.GetVideoDuration(ctx, path); err == nil {
			return probeDuration, nil
		} else if ctx.Err() != nil {
			return 0, ctx.Err()
		}
	}

	// Wait briefly for concurrent post-process to publish duration.
	pollDeadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(pollDeadline) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		v, err := store.Client.Vod.Get(ctx, dbItems.Video.ID)
		if err == nil && v.Duration > 1 {
			return v.Duration, nil
		} else if err != nil && ctx.Err() != nil {
			return 0, ctx.Err()
		}
	}

	log.Warn().Str("queue_id", queueID).Msg("could not resolve video duration for chat truncation; proceeding without cutoff")
	return 0, nil
}

// pickStreamVod returns the archive VOD recorded for the archived broadcast,
// matched by its stream ID; nil when the platform has not published it yet.
func pickStreamVod(videos []platform.VideoInfo, extStreamID string) *platform.VideoInfo {
	for i := range videos {
		if videos[i].StreamID == extStreamID {
			return &videos[i]
		}
	}
	return nil
}

// resolveLiveChatVideoStartTime returns the wall-clock time that video second
// 0 of a live archive corresponds to: platform VOD stream end minus captured
// duration (the capture drains buffered content until the stream ends), so
// chat offsets align with the captured media. Fallbacks and sanity-bound
// failures anchor to the capture start (queue chat_start); persistence
// failures abort the conversion so River can retry.
func resolveLiveChatVideoStartTime(ctx context.Context, store *database.Database, platformService platform.Platform, dbItems *GetDatabaseItemsResponse, platformVideos []platform.VideoInfo, videoDurationSec int) (time.Time, error) {
	logger := log.With().
		Str("queue_id", dbItems.Queue.ID.String()).
		Str("video_id", dbItems.Video.ID.String()).
		Str("channel", dbItems.Channel.Name).
		Logger()

	reference := dbItems.Queue.ChatStart
	if reference.IsZero() {
		reference = dbItems.Video.CreatedAt
	}
	if reference.IsZero() || videoDurationSec <= 0 {
		return persistLiveChatVideoStartTime(ctx, store, dbItems, reference)
	}

	platformVod := pickStreamVod(platformVideos, dbItems.Video.ExtStreamID)
	// The archive VOD appears on the platform several minutes after the
	// broadcast ends (UpdateStreamVideoId is scheduled 10 min later), hence
	// the poll budget.
	if platformVod == nil {
		const (
			pollInterval = 30 * time.Second
			pollBudget   = 10 * time.Minute
		)
		deadline := time.Now().Add(pollBudget)
		warnedFetchError := false
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				logger.Debug().Msg("giving up resolving live archive video start time; context cancelled")
				return reference, nil
			case <-time.After(pollInterval):
			}

			videos, err := platformService.GetVideos(ctx, dbItems.Channel.ExtID, platform.VideoTypeArchive, false, false)
			if err != nil {
				if ctx.Err() != nil {
					logger.Debug().Msg("giving up resolving live archive video start time; context cancelled")
					return reference, nil
				}
				// Transient API failures retry on the next poll tick.
				if !warnedFetchError {
					logger.Warn().Err(err).Msg("failed to fetch channel videos while resolving live archive video start time; retrying until deadline")
					warnedFetchError = true
				} else {
					logger.Debug().Err(err).Msg("failed to fetch channel videos while resolving live archive video start time; retrying until deadline")
				}
				continue
			}
			if platformVod = pickStreamVod(videos, dbItems.Video.ExtStreamID); platformVod != nil {
				break
			}
		}
	}

	if platformVod == nil {
		logger.Debug().Msg("platform VOD not found for live archive; anchoring chat to capture start time")
		return persistLiveChatVideoStartTime(ctx, store, dbItems, reference)
	}

	// Twitch archive VOD created_at is the broadcast start time; the stream
	// end is that start plus the VOD duration.
	streamEnd := platformVod.CreatedAt
	if platformVod.Duration > 0 {
		streamEnd = platformVod.CreatedAt.Add(platformVod.Duration)
	}

	anchor, ok := utils.ComputeLiveChatAnchor(reference, streamEnd, videoDurationSec)
	if !ok {
		logger.Warn().
			Time("capture_start", reference).
			Time("stream_end", streamEnd).
			Int("video_duration_sec", videoDurationSec).
			Msg("stream end does not bound the captured video; anchoring chat to capture start time")
		return persistLiveChatVideoStartTime(ctx, store, dbItems, reference)
	}

	logger.Info().
		Time("video_start_at", anchor).
		Dur("capture_latency", reference.Sub(anchor)).
		Msg("resolved live archive video start time from platform VOD")
	return persistLiveChatVideoStartTime(ctx, store, dbItems, anchor)
}

// persistLiveChatVideoStartTime records the chat/video anchor on the video so
// the frontend absolute-time display matches the chat sync.
func persistLiveChatVideoStartTime(ctx context.Context, store *database.Database, dbItems *GetDatabaseItemsResponse, videoStartTime time.Time) (time.Time, error) {
	if videoStartTime.IsZero() {
		return videoStartTime, nil
	}

	if _, err := store.Client.Vod.UpdateOneID(dbItems.Video.ID).SetVideoStartAt(videoStartTime).Save(ctx); err != nil {
		return videoStartTime, fmt.Errorf("persist video start time for video %s: %w", dbItems.Video.ID, err)
	}
	return videoStartTime, nil
}
