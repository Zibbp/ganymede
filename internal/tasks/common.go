package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/utils"
)

// ////////////////////
// Create Directory //
// ///////////////////
type CreateDirectoryArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (CreateDirectoryArgs) Kind() string { return string(utils.TaskCreateFolder) }

func (w CreateDirectoryArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Queue:       "default",
		Tags:        []string{"archive"},
	}
}

func (w CreateDirectoryArgs) Timeout(job *river.Job[CreateDirectoryArgs]) time.Duration {
	return 1 * time.Minute
}

type CreateDirectoryWorker struct {
	river.WorkerDefaults[CreateDirectoryArgs]
}

func (w CreateDirectoryWorker) Work(ctx context.Context, job *river.Job[CreateDirectoryArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskCreateFolder,
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

	// create directory
	// uses the videos directory from the the environment config
	c := config.GetEnvConfig()
	path := fmt.Sprintf("%s/%s/%s", c.VideosDir, dbItems.Channel.Name, dbItems.Video.FolderName)
	err = utils.CreateDirectory(path)
	if err != nil {
		return err
	}

	// set queue status to completed
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskCreateFolder,
	})
	if err != nil {
		return err
	}

	// continue with next job
	if job.Args.Continue {
		client := river.ClientFromContext[pgx.Tx](ctx)
		_, err := client.Insert(ctx, &SaveVideoInfoArgs{
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

// //////////////////
// Save Video Info //
// //////////////////
type SaveVideoInfoArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (SaveVideoInfoArgs) Kind() string { return string(utils.TaskSaveInfo) }

func (args SaveVideoInfoArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Queue:       "default",
		Tags:        []string{"archive"},
	}
}

func (w SaveVideoInfoArgs) Timeout(job *river.Job[SaveVideoInfoArgs]) time.Duration {
	return 1 * time.Minute
}

type SaveVideoInfoWorker struct {
	river.WorkerDefaults[SaveVideoInfoArgs]
}

func (w SaveVideoInfoWorker) Work(ctx context.Context, job *river.Job[SaveVideoInfoArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskSaveInfo,
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

	platformService, err := PlatformFromContext(ctx)
	if err != nil {
		return err
	}

	var info interface{}

	if dbItems.Queue.LiveArchive {
		info, err = platformService.GetLiveStream(ctx, dbItems.Channel.Name)
		if err != nil {
			return err
		}
	} else if dbItems.Video.Type == utils.Clip {
		info, err = platformService.GetClip(ctx, dbItems.Video.ExtID)
		if err != nil {
			return err
		}
	} else {
		videoInfo, err := platformService.GetVideo(ctx, dbItems.Video.ExtID, true, true)
		if err != nil {
			return err
		}

		// add chapters to database
		chapterService := chapter.NewService(store)
		for _, chapter := range videoInfo.Chapters {
			_, err = chapterService.CreateChapter(chapter, dbItems.Video.ID)
			if err != nil {
				return err
			}
		}

		// add muted segments to database
		for _, segment := range videoInfo.MutedSegments {
			// parse twitch duration
			segmentEnd := segment.Offset + segment.Duration
			if segmentEnd > int(videoInfo.Duration.Seconds()) {
				segmentEnd = int(videoInfo.Duration.Seconds())
			}
			// insert into database
			_, err := store.Client.MutedSegment.Create().SetStart(segment.Offset).SetEnd(segmentEnd).SetVod(&dbItems.Video).Save(ctx)
			if err != nil {
				return err
			}
		}

		info = videoInfo
	}

	// write info to file
	err = utils.WriteJsonFile(info, fmt.Sprintf("%s/%s/%s/%s-info.json", config.GetEnvConfig().VideosDir, dbItems.Channel.Name, dbItems.Video.FolderName, dbItems.Video.FileName))
	if err != nil {
		return err
	}

	// set queue status to completed
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskSaveInfo,
	})
	if err != nil {
		return err
	}

	// continue with next job
	if job.Args.Continue {
		client := river.ClientFromContext[pgx.Tx](ctx)
		_, err := client.Insert(ctx, &DownloadThumbnailArgs{
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

// //////////////////////
// Download Thumbnails //
// //////////////////////
type DownloadThumbnailArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (DownloadThumbnailArgs) Kind() string { return string(utils.TaskDownloadThumbnail) }

func (args DownloadThumbnailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Queue:       "default",
		Tags:        []string{"archive"},
	}
}

func (w DownloadThumbnailArgs) Timeout(job *river.Job[DownloadThumbnailArgs]) time.Duration {
	return 1 * time.Minute
}

type DownloadTumbnailsWorker struct {
	river.WorkerDefaults[DownloadThumbnailArgs]
}

func (w DownloadTumbnailsWorker) Work(ctx context.Context, job *river.Job[DownloadThumbnailArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadThumbnail,
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

	platformService, err := PlatformFromContext(ctx)
	if err != nil {
		return err
	}

	var thumbnailUrl string
	var fullResThumbnailUrl string
	var webResThumbnailUrl string

	// create the correct URL for each thumbnail type
	if dbItems.Queue.LiveArchive {
		info, err := platformService.GetLiveStream(ctx, dbItems.Channel.Name)
		if err != nil {
			return err
		}
		thumbnailUrl = info.ThumbnailURL
		fullResThumbnailUrl = replaceThumbnailPlaceholders(thumbnailUrl, "1920", "1080", dbItems.Queue.LiveArchive)
		webResThumbnailUrl = replaceThumbnailPlaceholders(thumbnailUrl, "640", "360", dbItems.Queue.LiveArchive)

	} else if dbItems.Video.Type == utils.Clip {
		info, err := platformService.GetClip(ctx, dbItems.Video.ExtID)
		if err != nil {
			return err
		}
		thumbnailUrl = info.ThumbnailURL
		fullResThumbnailUrl = thumbnailUrl
		webResThumbnailUrl = thumbnailUrl
	} else {
		info, err := platformService.GetVideo(ctx, dbItems.Video.ExtID, false, false)
		if err != nil {
			return err
		}
		thumbnailUrl = info.ThumbnailURL
		fullResThumbnailUrl = replaceThumbnailPlaceholders(thumbnailUrl, "1920", "1080", dbItems.Queue.LiveArchive)
		webResThumbnailUrl = replaceThumbnailPlaceholders(thumbnailUrl, "640", "360", dbItems.Queue.LiveArchive)
	}

	err = utils.DownloadAndSaveFile(fullResThumbnailUrl, dbItems.Video.ThumbnailPath)
	if err != nil {
		return err
	}
	err = utils.DownloadAndSaveFile(webResThumbnailUrl, dbItems.Video.WebThumbnailPath)
	if err != nil {
		return err
	}

	// set queue status to completed
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadThumbnail,
	})
	if err != nil {
		return err
	}

	// continue with next jobs
	if job.Args.Continue {
		client := river.ClientFromContext[pgx.Tx](ctx)
		if dbItems.Queue.LiveArchive {
			_, err := client.Insert(ctx, &DownloadLiveVideoArgs{
				Continue: true,
				Input:    job.Args.Input,
			}, nil)
			if err != nil {
				return err
			}

			// Check if channel has a live edge
			// If so queue a job to update the live stream metadata if enabled
			hasLive, err := store.Client.Channel.Query().Where(entChannel.ID(dbItems.Channel.ID)).QueryLive().Exist(ctx)
			if err != nil {
				return err
			}

			if hasLive {
				// Queue live stream metadata update job for in the future if the channel has it enabled
				watchedChannel, err := dbItems.Channel.QueryLive().First(ctx)
				if err != nil {
					return err
				}
				if watchedChannel.UpdateMetadataMinutes > 0 {
					_, err = client.Insert(ctx, &UpdateLiveStreamMetadataArgs{
						Continue: false,
						Input:    job.Args.Input,
					}, &river.InsertOpts{
						ScheduledAt: time.Now().Add(time.Duration(watchedChannel.UpdateMetadataMinutes) * time.Minute),
					})
					if err != nil {
						return err
					}
				}
			}

		} else {
			_, err = client.Insert(ctx, &DownloadVideoArgs{
				Continue: true,
				Input:    job.Args.Input,
			}, nil)
			if err != nil {
				return err
			}

			// download chat if needed
			if dbItems.Queue.ArchiveChat {
				_, err = client.Insert(ctx, &DownloadChatArgs{
					Continue: true,
					Input:    job.Args.Input,
				}, nil)
				if err != nil {
					return err
				}
			}
		}
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}

	return nil
}

// //////////////////////////////
// Update Live Stream Metadata //
// //////////////////////////////
//
// Update live stream metadata task that is run X minutes after a live stream archive is started.
//
// This is used to update the video metadata with the correct title and non-blank thumbnail.
type UpdateLiveStreamMetadataArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (UpdateLiveStreamMetadataArgs) Kind() string { return string(utils.TaskUpdateLiveStreamMetadata) }

func (args UpdateLiveStreamMetadataArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Tags:        []string{archive_tag, allow_fail_tag},
	}
}

func (w UpdateLiveStreamMetadataArgs) Timeout(job *river.Job[UpdateLiveStreamMetadataArgs]) time.Duration {
	return 1 * time.Minute
}

type UpdateLiveStreamMetadataWorker struct {
	river.WorkerDefaults[UpdateLiveStreamMetadataArgs]
}

func (w UpdateLiveStreamMetadataWorker) Work(ctx context.Context, job *river.Job[UpdateLiveStreamMetadataArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	platformService, err := PlatformFromContext(ctx)
	if err != nil {
		return err
	}

	client := river.ClientFromContext[pgx.Tx](ctx)

	// Queue thumbnail download job
	_, err = client.Insert(ctx, &DownloadThumbnailArgs{Continue: false, Input: job.Args.Input}, nil)
	if err != nil {
		return err
	}

	// Queue save info job
	_, err = client.Insert(ctx, &SaveVideoInfoArgs{Continue: false, Input: job.Args.Input}, nil)
	if err != nil {
		return err
	}

	// Manually update the live stream metadata
	info, err := platformService.GetLiveStream(ctx, dbItems.Channel.Name)
	if err != nil {
		return err
	}

	// Update the video metadata with the live stream info
	_, err = dbItems.Video.Update().SetTitle(info.Title).Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

// UpdateStreamVideoId is scheduled to run after a livestream archive finishes. It will attempt to update the external ID of the stream video (vod).
//
// Has two use modes:
// - Supply a Queue ID to update the video ID of the video related to the queue
// - Do not supply a Queue ID (set to uuid.Nil) to update the video IDs of all videos
type UpdateStreamVideoIdArgs struct {
	Input ArchiveVideoInput `json:"input"`
}

func (UpdateStreamVideoIdArgs) Kind() string { return TaskUpdateStreamVideoId }

func (args UpdateStreamVideoIdArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 2,
		Queue:       "default",
		Tags:        []string{"archive"},
	}
}

func (w UpdateStreamVideoIdArgs) Timeout(job *river.Job[UpdateStreamVideoIdArgs]) time.Duration {
	return 10 * time.Minute
}

type UpdateStreamVideoIdWorker struct {
	river.WorkerDefaults[UpdateStreamVideoIdArgs]
}

func (w UpdateStreamVideoIdWorker) Work(ctx context.Context, job *river.Job[UpdateStreamVideoIdArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	logger.Info().Msg("starting task")

	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// start task heartbeat
	go startHeartBeatForTask(ctx, HeartBeatInput{
		TaskId: job.ID,
		conn:   store.ConnPool,
	})

	platformService, err := PlatformFromContext(ctx)
	if err != nil {
		return err
	}

	var channels []*ent.Channel
	var videos []*ent.Vod

	// check if queue id is set and only one video needs to be updated
	if job.Args.Input.QueueId != uuid.Nil {
		dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
		if err != nil {
			return err
		}
		channels = []*ent.Channel{&dbItems.Channel}
		videos = []*ent.Vod{&dbItems.Video}
	}

	if len(channels) == 0 {
		channels, err = store.Client.Channel.Query().All(ctx)
		if err != nil {
			return err
		}
	}

	// loop over each channel and get all channel videos
	// this is necessary because the 'streamid' is not an id we can query from APIs
	for _, channel := range channels {
		logger.Info().Str("channel", channel.Name).Msg("fetching channel videos")

		// only get videos if no queue id is set
		if len(videos) == 0 {
			videos, err = store.Client.Vod.Query().Where(vod.HasChannelWith(entChannel.ID(channel.ID))).All(ctx)
			if err != nil {
				return err
			}
		}

		// get all channel videos from platform
		platformVideos, err := platformService.GetVideos(ctx, channel.ExtID, platform.VideoTypeArchive, false, false)
		if err != nil {
			return err
		}

		logger.Info().Str("channel", channel.Name).Msgf("found %d videos in platform", len(platformVideos))

		for _, video := range videos {
			if video.Type != utils.Live {
				continue
			}
			if video.ExtID == "" {
				continue
			}

			// attempt to find video in list of platform videos
			for _, platformVideo := range platformVideos {
				if platformVideo.StreamID == video.ExtStreamID {
					logger.Info().Str("channel", channel.Name).Str("video_id", video.ID.String()).Msg("found video in platform")
					_, err := store.Client.Vod.UpdateOneID(video.ID).SetExtID(platformVideo.ID).Save(ctx)
					if err != nil {
						return err
					}
					// TODO: kick off job to save chapters and muted segments?
					break
				}
			}

		}

	}

	logger.Info().Msg("task completed")

	return nil
}
