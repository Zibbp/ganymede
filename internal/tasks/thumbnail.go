package tasks

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/exec"
)

type GenerateStaticThumbnailArgs struct {
	VideoId string `json:"video_id"`
}

func (GenerateStaticThumbnailArgs) Kind() string { return TaskGenerateStaticThumbnails }

func (args GenerateStaticThumbnailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1,
	}
}

func (w GenerateStaticThumbnailArgs) Timeout(job *river.Job[GenerateStaticThumbnailArgs]) time.Duration {
	return 1 * time.Minute
}

type GenerateStaticThubmnailWorker struct {
	river.WorkerDefaults[GenerateStaticThumbnailArgs]
}

func (w GenerateStaticThubmnailWorker) Work(ctx context.Context, job *river.Job[GenerateStaticThumbnailArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	videoUUID, err := uuid.Parse(job.Args.VideoId)
	if err != nil {
		return err
	}

	video, err := store.Client.Vod.Get(ctx, videoUUID)
	if err != nil {
		return err
	}

	// get random time
	time := rand.Intn(video.Duration)

	// generate full-res thumbnail
	err = exec.GenerateStaticThumbnail(ctx, video.VideoPath, time, video.ThumbnailPath, "")
	if err != nil {
		return err
	}

	// generate webp thumbnail
	err = exec.GenerateStaticThumbnail(ctx, video.VideoPath, time, video.WebThumbnailPath, "640x360")
	if err != nil {
		return err
	}

	return nil
}

type GenerateSpriteThumbnailArgs struct {
	VideoId string `json:"video_id"`
}

func (GenerateSpriteThumbnailArgs) Kind() string { return TaskGenerateSpriteThumbnails }

func (args GenerateSpriteThumbnailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1,
		Queue:       QueueGenerateThumbnailSprites,
	}
}

func (w GenerateSpriteThumbnailArgs) Timeout(job *river.Job[GenerateSpriteThumbnailArgs]) time.Duration {
	return 1 * time.Hour
}

type GenerateSpriteThumbnailWorker struct {
	river.WorkerDefaults[GenerateSpriteThumbnailArgs]
}

func (w GenerateSpriteThumbnailWorker) Work(ctx context.Context, job *river.Job[GenerateSpriteThumbnailArgs]) error {
	logger := log.With().Str("task", job.Kind).Str("job_id", fmt.Sprintf("%d", job.ID)).Logger()
	// Get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// Start task heartbeat
	go startHeartBeatForTask(ctx, HeartBeatInput{
		TaskId: job.ID,
		conn:   store.ConnPool,
	})

	videoUUID, err := uuid.Parse(job.Args.VideoId)
	if err != nil {
		return err
	}

	video, err := store.Client.Vod.Get(ctx, videoUUID)
	if err != nil {
		return err
	}
	videoChannel := video.QueryChannel()
	channel, err := videoChannel.Only(ctx)
	if err != nil {
		return err
	}

	logger.Info().Str("video_id", video.ID.String()).Msg("generating sprite thumbnails for video")

	env := config.GetEnvConfig()

	// Create temp directory
	tmpThumbnailsDirectory, err := os.MkdirTemp(env.TempDir, video.ID.String())
	if err != nil {
		return err
	}

	rootVideoPath := fmt.Sprintf("%s/%s/%s", env.VideosDir, channel.Name, video.FolderName)
	spritesDirectory := fmt.Sprintf("%s/sprites", rootVideoPath)

	err = os.MkdirAll(spritesDirectory, os.ModePerm)
	if err != nil {
		return err
	}

	thumbnailWidth := 220
	thumbnailHeight := 124
	spriteTilesX := 5
	spriteTilesY := 10
	thumbnailInterval := 60 // default to thumbnails every 60 seconds

	switch {
	case video.Duration < 60:
		thumbnailInterval = 1 // thumbnails every 1 second for <60 seconds
	case video.Duration < 300:
		thumbnailInterval = 2 // thumbnails every 2 seconds for <5 minutes
	case video.Duration < 900:
		thumbnailInterval = 10 // thumbnails every 10 seconds for <15 minutes
	case video.Duration < 1800:
		thumbnailInterval = 30 // thumbnails every 30 seconds for <30 minutes
	}

	logger.Debug().Str("video_id", video.ID.String()).Str("thumbnails", tmpThumbnailsDirectory).Str("sprites", spritesDirectory).Msg("sprite thumbnail paths")

	// Create thumbnails
	generateThumbnailsConfig := exec.GenerateThumbnailsInput{
		Video:        video.VideoPath,
		ThumbnailDir: tmpThumbnailsDirectory,
		Interval:     thumbnailInterval,
		Width:        thumbnailWidth,
		Height:       thumbnailHeight,
	}
	err = exec.GenerateThumbnails(generateThumbnailsConfig)
	if err != nil {
		return fmt.Errorf("error generating thumbnails: %v", err)
	}

	// Create sprites with thumbnails
	createSpritesConfig := exec.CreateSpritesInput{
		SpriteDir:    spritesDirectory,
		ThumbnailDir: tmpThumbnailsDirectory,
		Width:        thumbnailWidth,
		Height:       thumbnailHeight,
		TilesX:       spriteTilesX,
		TilesY:       spriteTilesY,
	}
	spritePaths, err := exec.CreateSprites(createSpritesConfig)
	if err != nil {
		return fmt.Errorf("error generating sprites: %v", err)
	}

	// Enable sprite thumbnails for video
	_, err = video.Update().SetSpriteThumbnailsEnabled(true).SetSpriteThumbnailsImages(spritePaths).SetSpriteThumbnailsInterval(thumbnailInterval).SetSpriteThumbnailsRows(spriteTilesY).SetSpriteThumbnailsColumns(spriteTilesX).SetSpriteThumbnailsWidth(thumbnailWidth).SetSpriteThumbnailsHeight(thumbnailHeight).Save(ctx)
	if err != nil {
		return fmt.Errorf("error updating video: %v", err)
	}

	// Delete temporary thumbnail directory
	err = os.RemoveAll(tmpThumbnailsDirectory)
	if err != nil {
		return err
	}

	logger.Info().Str("video_id", video.ID.String()).Msg("generated video thumbnail sprites")

	return nil
}
