package tasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entChapter "github.com/zibbp/ganymede/ent/chapter"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/nfo"
)

// GenerateNFOFilesArgs generates a sidecar for one video when VideoID is set,
// or backfills every completed archive when it is nil.
type GenerateNFOFilesArgs struct {
	VideoID *uuid.UUID `json:"video_id,omitempty" river:"unique"`
}

func (GenerateNFOFilesArgs) Kind() string { return TaskGenerateNFOFiles }

func (GenerateNFOFilesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		UniqueOpts:  archiveUniqueOpts(),
	}
}

func (w *GenerateNFOFilesWorker) Timeout(job *river.Job[GenerateNFOFilesArgs]) time.Duration {
	return 10 * time.Minute
}

type GenerateNFOFilesWorker struct {
	river.WorkerDefaults[GenerateNFOFilesArgs]
}

func (w GenerateNFOFilesWorker) Work(ctx context.Context, job *river.Job[GenerateNFOFilesArgs]) error {
	logger := log.With().Str("task", job.Kind).Int64("job_id", job.ID).Logger()
	logger.Info().Msg("starting task")

	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	if job.Args.VideoID != nil {
		video, err := store.Client.Vod.Query().
			Where(entVod.ID(*job.Args.VideoID), entVod.Processing(false)).
			WithChannel().
			WithChapters(func(query *ent.ChapterQuery) {
				query.Order(entChapter.ByStart())
			}).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				logger.Warn().Str("video_id", job.Args.VideoID.String()).Msg("completed video not found; skipping NFO generation")
				return nil
			}
			return fmt.Errorf("fetch video %s for NFO generation: %w", job.Args.VideoID, err)
		}
		if err := ensureVideoNFO(logger, video); err != nil {
			return err
		}
		logger.Info().Msg("task completed")
		return nil
	}

	const batchSize = 100
	offset := 0
	var errs []error
	for {
		videos, err := store.Client.Vod.Query().
			Where(entVod.Processing(false)).
			Order(entVod.ByID()).
			Limit(batchSize).
			Offset(offset).
			WithChannel().
			WithChapters(func(query *ent.ChapterQuery) {
				query.Order(entChapter.ByStart())
			}).
			All(ctx)
		if err != nil {
			return fmt.Errorf("fetch videos for NFO generation: %w", err)
		}
		if len(videos) == 0 {
			break
		}

		for _, video := range videos {
			if err := ensureVideoNFO(logger, video); err != nil {
				logger.Error().Err(err).Str("video_id", video.ID.String()).Msg("failed to ensure video NFO")
				errs = append(errs, err)
			}
		}
		offset += len(videos)
	}

	if len(errs) > 0 {
		return fmt.Errorf("one or more NFO files could not be generated: %w", errors.Join(errs...))
	}

	logger.Info().Msg("task completed")
	return nil
}

func ensureVideoNFO(logger zerolog.Logger, video *ent.Vod) error {
	if video.VideoPath == "" {
		logger.Warn().Str("video_id", video.ID.String()).Msg("video has no media path; skipping NFO generation")
		return nil
	}

	sidecarPath, err := nfo.SidecarPath(video.VideoPath)
	if err != nil {
		logger.Warn().Err(err).Str("video_id", video.ID.String()).Str("video_path", video.VideoPath).Msg("unsupported media path; skipping NFO generation")
		return nil
	}

	info, err := os.Stat(video.VideoPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn().Str("video_id", video.ID.String()).Str("video_path", video.VideoPath).Msg("media file does not exist; skipping NFO generation")
			return nil
		}
		return fmt.Errorf("stat media file %s: %w", video.VideoPath, err)
	}
	if !info.Mode().IsRegular() {
		logger.Warn().Str("video_id", video.ID.String()).Str("video_path", video.VideoPath).Msg("media path is not a regular file; skipping NFO generation")
		return nil
	}

	channel, err := video.Edges.ChannelOrErr()
	if err != nil {
		return fmt.Errorf("load channel for video %s: %w", video.ID, err)
	}
	studio := strings.TrimSpace(channel.DisplayName)
	if studio == "" {
		studio = channel.Name
	}

	genres := make([]string, 0, len(video.Edges.Chapters))
	seenGenres := make(map[string]struct{}, len(video.Edges.Chapters))
	for _, chapter := range video.Edges.Chapters {
		genre := strings.TrimSpace(chapter.Title)
		if genre == "" {
			continue
		}
		key := strings.ToLower(genre)
		if _, exists := seenGenres[key]; exists {
			continue
		}
		seenGenres[key] = struct{}{}
		genres = append(genres, genre)
	}

	created, err := nfo.CreateMovieIfMissing(sidecarPath, nfo.MovieMetadata{
		Title:      video.Title,
		Premiered:  video.StreamedAt,
		Studio:     studio,
		Genres:     genres,
		Platform:   string(video.Platform),
		ExternalID: video.ExtID,
	})
	if err != nil {
		return fmt.Errorf("create NFO for video %s: %w", video.ID, err)
	}
	if created {
		logger.Info().Str("video_id", video.ID.String()).Str("nfo_path", sidecarPath).Msg("created video NFO")
	} else {
		logger.Debug().Str("video_id", video.ID.String()).Str("nfo_path", sidecarPath).Msg("video NFO already exists; preserving it")
	}
	return nil
}
