package tasks

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/utils"
)

func TestArchiveMiddlewareAddsQueueMetadataAndPreservesExistingMetadata(t *testing.T) {
	t.Parallel()
	queueID := uuid.New()
	encodedArgs, err := json.Marshal(CreateDirectoryArgs{
		Continue: true,
		Input: ArchiveVideoInput{
			QueueId:            queueID,
			RecoveryGeneration: 3,
		},
	})
	require.NoError(t, err)

	params := &rivertype.JobInsertParams{
		EncodedArgs: encodedArgs,
		Metadata:    []byte(`{"trace_id":"abc"}`),
		Tags:        []string{archive_tag},
	}
	middleware := NewArchiveMiddleware()
	_, err = middleware.InsertMany(context.Background(), []*rivertype.JobInsertParams{params}, func(context.Context) ([]*rivertype.JobInsertResult, error) {
		return nil, nil
	})
	require.NoError(t, err)

	var metadata struct {
		TraceID  string             `json:"trace_id"`
		Ganymede archiveJobMetadata `json:"ganymede"`
	}
	require.NoError(t, json.Unmarshal(params.Metadata, &metadata))
	require.Equal(t, "abc", metadata.TraceID)
	require.Equal(t, queueID, metadata.Ganymede.QueueID)
	require.Equal(t, 3, metadata.Ganymede.RecoveryGeneration)
}

func TestNewRecoveredArchiveArgsIncrementsGenerationAndPreservesJobSettings(t *testing.T) {
	t.Parallel()
	queueID := uuid.New()
	encoded, err := json.Marshal(DownloadVideoArgs{Continue: true, Input: ArchiveVideoInput{QueueId: queueID}})
	require.NoError(t, err)

	recovered, err := newRecoveredArchiveArgs(&rivertype.JobRow{
		EncodedArgs: encoded,
		Kind:        (DownloadVideoArgs{}).Kind(),
		Priority:    2,
		Queue:       QueueVideoDownload,
		Tags:        []string{archive_tag},
	}, 2, 3)
	require.NoError(t, err)

	var decoded RiverJobArgs
	require.NoError(t, json.Unmarshal(recovered.raw, &decoded))
	require.Equal(t, queueID, decoded.Input.QueueId)
	require.Equal(t, 2, decoded.Input.RecoveryGeneration)
	require.Equal(t, 3, recovered.opts.MaxAttempts)
	require.Equal(t, QueueVideoDownload, recovered.opts.Queue)
	require.True(t, recovered.opts.UniqueOpts.ByArgs)
}

func TestShouldFinalizeArchiveError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
		job  *rivertype.JobRow
		err  error
		want bool
	}{
		{
			name: "transient retry",
			ctx:  context.Background(),
			job:  &rivertype.JobRow{Kind: string(utils.TaskDownloadVideo), Attempt: 2, MaxAttempts: 5},
			err:  context.DeadlineExceeded,
		},
		{
			name: "final attempt",
			ctx:  context.Background(),
			job:  &rivertype.JobRow{Kind: string(utils.TaskDownloadVideo), Attempt: 5, MaxAttempts: 5},
			err:  context.DeadlineExceeded,
			want: true,
		},
		{
			name: "non-live remote cancellation",
			ctx:  context.Background(),
			job:  &rivertype.JobRow{Kind: string(utils.TaskDownloadVideo), Attempt: 1, MaxAttempts: 5},
			err:  rivertype.ErrJobCancelledRemotely,
			want: true,
		},
		{
			name: "live remote cancellation finalized by worker",
			ctx:  context.Background(),
			job:  &rivertype.JobRow{Kind: string(utils.TaskDownloadLiveVideo), Attempt: 1, MaxAttempts: 1},
			err:  rivertype.ErrJobCancelledRemotely,
		},
	}

	shutdownCtx, cancel := context.WithCancel(context.Background())
	cancel()
	tests = append(tests, struct {
		name string
		ctx  context.Context
		job  *rivertype.JobRow
		err  error
		want bool
	}{
		name: "worker shutdown",
		ctx:  shutdownCtx,
		job:  &rivertype.JobRow{Kind: string(utils.TaskDownloadVideo), Attempt: 5, MaxAttempts: 5},
		err:  context.Canceled,
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, shouldFinalizeArchiveError(test.ctx, test.job, test.err))
		})
	}
}
