package tasks

import (
	"testing"
	"time"

	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
)

func TestStaleLiveArchiveRecoveryAction(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 22, 20, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		metadata      string
		kind          string
		wantAction    staleLiveArchiveAction
		wantRemaining time.Duration
		wantErr       string
	}{
		{
			name:       "requests cancellation when River has no cancellation marker",
			metadata:   `{"output":{"heartbeat_at":"2026-07-22T19:58:00Z"}}`,
			kind:       string(utils.TaskDownloadLiveVideo),
			wantAction: staleLiveArchiveActionCancel,
		},
		{
			name:          "waits for a worker inside its finalization window",
			metadata:      `{"cancel_attempted_at":"2026-07-22T19:59:00Z"}`,
			kind:          string(utils.TaskDownloadLiveVideo),
			wantAction:    staleLiveArchiveActionWait,
			wantRemaining: liveArchiveVideoCancellationGrace - time.Minute,
		},
		{
			name:       "recovers after the worker missed its finalization window",
			metadata:   `{"cancel_attempted_at":"2026-07-22T19:50:00Z"}`,
			kind:       string(utils.TaskDownloadLiveVideo),
			wantAction: staleLiveArchiveActionRecover,
		},
		{
			name:       "uses the shorter chat finalization window",
			metadata:   `{"cancel_attempted_at":"2026-07-22T19:57:00Z"}`,
			kind:       string(utils.TaskDownloadLiveChat),
			wantAction: staleLiveArchiveActionRecover,
		},
		{
			name:     "rejects malformed River metadata",
			metadata: `{`,
			kind:     string(utils.TaskDownloadLiveVideo),
			wantErr:  "decode archive job 42 cancellation metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			job := &rivertype.JobRow{ID: 42, Kind: tt.kind, Metadata: []byte(tt.metadata)}
			action, remaining, err := staleLiveArchiveRecoveryAction(job, now)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantAction, action)
			require.Equal(t, tt.wantRemaining, remaining)
		})
	}
}

func TestArchiveJobNeedsWatchdog(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 22, 20, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Minute)
	stale := now.Add(-archiveHeartbeatTimeout - time.Second)
	liveVideo := string(utils.TaskDownloadLiveVideo)

	require.False(t, archiveJobNeedsWatchdog(liveVideo, fresh, now))
	require.True(t, archiveJobNeedsWatchdog(string(utils.TaskDownloadVideo), stale, now))
	require.False(t, archiveJobNeedsWatchdog(string(utils.TaskDownloadVideo), time.Time{}, now))
}

func TestLiveArchiveDownloadNeedsRecovery(t *testing.T) {
	t.Parallel()

	queue := &ent.Queue{
		TaskVideoDownload: utils.Running,
		TaskChatDownload:  utils.Success,
	}

	require.True(t, liveArchiveDownloadNeedsRecovery(queue, string(utils.TaskDownloadLiveVideo)))
	require.False(t, liveArchiveDownloadNeedsRecovery(queue, string(utils.TaskDownloadLiveChat)))
	require.False(t, liveArchiveDownloadNeedsRecovery(queue, string(utils.TaskDownloadVideo)))

	queue.TaskVideoDownload = utils.Success
	require.False(t, liveArchiveDownloadNeedsRecovery(queue, string(utils.TaskDownloadLiveVideo)))
}

func TestArchiveQueueStageStatusNeedsRecovery(t *testing.T) {
	t.Parallel()

	queue := &ent.Queue{
		Processing:               true,
		TaskVodCreateFolder:      utils.Success,
		TaskVodDownloadThumbnail: utils.Success,
		TaskVodSaveInfo:          utils.Success,
		TaskVideoDownload:        utils.Running,
		TaskVideoConvert:         utils.Pending,
		TaskVideoMove:            utils.Pending,
		TaskChatDownload:         utils.Success,
		TaskChatConvert:          utils.Success,
		TaskChatRender:           utils.Success,
		TaskChatMove:             utils.Success,
	}

	require.True(t, archiveQueueStageStatusNeedsRecovery(queue, string(utils.TaskDownloadVideo)))
	require.True(t, archiveQueueStageStatusNeedsRecovery(queue, string(utils.TaskDownloadLiveVideo)))
	require.False(t, archiveQueueStageStatusNeedsRecovery(queue, string(utils.TaskPostProcessVideo)))
	require.False(t, archiveQueueStageStatusNeedsRecovery(queue, "unknown_archive_task"))

	queue.TaskVideoDownload = utils.Success
	require.False(t, archiveQueueStageStatusNeedsRecovery(queue, string(utils.TaskDownloadVideo)))

	queue.TaskVideoDownload = utils.Running
	queue.Processing = false
	require.False(t, archiveQueueStageStatusNeedsRecovery(queue, string(utils.TaskDownloadVideo)))
}
