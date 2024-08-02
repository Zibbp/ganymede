package tasks

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/zibbp/ganymede/internal/exec"
)

type GenerateStaticThumbnailArgs struct {
	VideoId string `json:"video_id"`
}

func (GenerateStaticThumbnailArgs) Kind() string { return "generate_static_thumbnail" }

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
