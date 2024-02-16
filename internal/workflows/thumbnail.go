package workflows

import (
	"time"

	"github.com/zibbp/ganymede/internal/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func GenerateThumbnailsForVideo(ctx workflow.Context, videoId string) (string, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           "generate-thumbnails",
		HeartbeatTimeout:    90 * time.Second,
		StartToCloseTimeout: 168 * time.Hour,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Minute,
			BackoffCoefficient: 2,
			MaximumAttempts:    2,
			MaximumInterval:    15 * time.Minute,
		},
	})
	var output string
	err := workflow.ExecuteActivity(ctx, activities.GenerateThumbnailsForVideo, videoId).Get(ctx, &output)
	if err != nil {
		return "", err
	}

	return output, nil
}
