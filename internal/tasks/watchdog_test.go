package tasks

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
)

func TestIsLiveHlsCapture(t *testing.T) {
	t.Parallel()

	video := ent.Vod{
		ExtID:                "123",
		TmpVideoHlsPath:      "/tmp/123_uuid-video_hls0",
		TmpVideoDownloadPath: "/tmp/123_uuid-video_hls0/123-video.m3u8",
	}
	if !isLiveHlsCapture(&video) {
		t.Fatal("expected new live layout to be detected as an HLS capture")
	}

	legacy := video
	legacy.TmpVideoDownloadPath = "/tmp/123_uuid-video.ts"
	if isLiveHlsCapture(&legacy) {
		t.Fatal("legacy transport-stream capture must not be detected as an HLS capture")
	}

	if isLiveHlsCapture(&ent.Vod{}) {
		t.Fatal("empty video must not be detected as an HLS capture")
	}
}

func TestValidateRecoverableLiveVideoInput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const extID = "123"
	video := ent.Vod{
		ExtID:                extID,
		TmpVideoHlsPath:      dir,
		TmpVideoDownloadPath: filepath.Join(dir, extID+"-video.m3u8"),
	}
	playlistPath := video.TmpVideoDownloadPath
	initPath := filepath.Join(dir, extID+"_init.mp4")
	segmentPath := filepath.Join(dir, extID+"_segment000000.m4s")

	// A header-only playlist without recoverable segments is not media.
	if err := os.WriteFile(playlistPath, []byte("#EXTM3U\n#EXT-X-VERSION:7\n"), 0o644); err != nil {
		t.Fatalf("write playlist: %v", err)
	}
	if err := validateRecoverableLiveVideoInput(&video); err == nil {
		t.Fatal("expected header-only playlist without segments to be rejected")
	}

	// Segments on disk make a truncated playlist recoverable.
	if err := os.WriteFile(initPath, []byte("init"), 0o644); err != nil {
		t.Fatalf("write init: %v", err)
	}
	if err := os.WriteFile(segmentPath, []byte("segment"), 0o644); err != nil {
		t.Fatalf("write segment: %v", err)
	}
	if err := validateRecoverableLiveVideoInput(&video); err != nil {
		t.Fatalf("expected segments to make the capture recoverable: %v", err)
	}

	// A playlist with media references is recoverable on its own.
	if err := os.WriteFile(playlistPath, []byte("#EXTM3U\n#EXTINF:10.0,\n"+extID+"_segment000000.m4s\n"), 0o644); err != nil {
		t.Fatalf("write playlist: %v", err)
	}
	if err := validateRecoverableLiveVideoInput(&video); err != nil {
		t.Fatalf("expected playlist with EXTINF to be recoverable: %v", err)
	}

	// Nothing on disk fails validation.
	if err := os.Remove(playlistPath); err != nil {
		t.Fatalf("remove playlist: %v", err)
	}
	if err := os.Remove(initPath); err != nil {
		t.Fatalf("remove init: %v", err)
	}
	if err := os.Remove(segmentPath); err != nil {
		t.Fatalf("remove segment: %v", err)
	}
	if err := validateRecoverableLiveVideoInput(&video); err == nil {
		t.Fatal("expected missing capture media to be rejected")
	}

	// Legacy transport streams still validate by file size.
	legacy := ent.Vod{
		ExtID:                extID,
		TmpVideoHlsPath:      dir,
		TmpVideoDownloadPath: filepath.Join(dir, "legacy-video.ts"),
	}
	if err := os.WriteFile(legacy.TmpVideoDownloadPath, []byte("ts"), 0o644); err != nil {
		t.Fatalf("write legacy ts: %v", err)
	}
	if err := validateRecoverableLiveVideoInput(&legacy); err != nil {
		t.Fatalf("expected legacy transport stream to be recoverable: %v", err)
	}
}

func TestStaleLiveArchiveRecoveryAction(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 22, 20, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		metadata      string
		wantAction    staleLiveArchiveAction
		wantRemaining time.Duration
		wantErr       string
	}{
		{
			name:       "requests cancellation when River has no cancellation marker",
			metadata:   `{"output":{"heartbeat_at":"2026-07-22T19:58:00Z"}}`,
			wantAction: staleLiveArchiveActionCancel,
		},
		{
			name:          "waits for a worker inside its finalization window",
			metadata:      `{"cancel_attempted_at":"2026-07-22T19:59:00Z"}`,
			wantAction:    staleLiveArchiveActionWait,
			wantRemaining: liveArchiveCancellationGrace - time.Minute,
		},
		{
			name:       "recovers after the worker missed its finalization window",
			metadata:   `{"cancel_attempted_at":"2026-07-22T19:55:00Z"}`,
			wantAction: staleLiveArchiveActionRecover,
		},
		{
			name:     "rejects malformed River metadata",
			metadata: `{`,
			wantErr:  "decode archive job 42 cancellation metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			job := &rivertype.JobRow{ID: 42, Metadata: []byte(tt.metadata)}
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

func TestFileIsQuiet(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 22, 20, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "capture.ts")
	require.NoError(t, os.WriteFile(path, []byte("partial media"), 0o600))

	require.NoError(t, os.Chtimes(path, now.Add(-liveArchiveMediaQuietPeriod+time.Second), now.Add(-liveArchiveMediaQuietPeriod+time.Second)))
	quiet, err := fileIsQuiet(path, now)
	require.NoError(t, err)
	require.False(t, quiet)

	require.NoError(t, os.Chtimes(path, now.Add(-liveArchiveMediaQuietPeriod), now.Add(-liveArchiveMediaQuietPeriod)))
	quiet, err = fileIsQuiet(path, now)
	require.NoError(t, err)
	require.True(t, quiet)

	quiet, err = fileIsQuiet(filepath.Join(t.TempDir(), "missing.ts"), now)
	require.NoError(t, err)
	require.True(t, quiet)
}

func TestFileIsStalled(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 22, 20, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "capture.ts")
	require.NoError(t, os.WriteFile(path, []byte("partial media"), 0o600))

	require.NoError(t, os.Chtimes(path, now.Add(-liveArchiveMediaStallTimeout+time.Second), now.Add(-liveArchiveMediaStallTimeout+time.Second)))
	stalled, err := fileIsStalled(path, nil, now)
	require.NoError(t, err)
	require.False(t, stalled)

	require.NoError(t, os.Chtimes(path, now.Add(-liveArchiveMediaStallTimeout), now.Add(-liveArchiveMediaStallTimeout)))
	stalled, err = fileIsStalled(path, nil, now)
	require.NoError(t, err)
	require.True(t, stalled)

	stalled, err = fileIsStalled(filepath.Join(t.TempDir(), "missing.ts"), nil, now)
	require.NoError(t, err)
	require.False(t, stalled)

	startedAt := now.Add(-liveArchiveMediaStallTimeout)
	stalled, err = fileIsStalled(filepath.Join(t.TempDir(), "missing.ts"), &startedAt, now)
	require.NoError(t, err)
	require.True(t, stalled)
}

func TestArchiveJobNeedsWatchdog(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 22, 20, 0, 0, 0, time.UTC)
	fresh := now.Add(-time.Minute)
	stale := now.Add(-archiveHeartbeatTimeout - time.Second)
	liveVideo := string(utils.TaskDownloadLiveVideo)

	require.False(t, archiveJobNeedsWatchdog(liveVideo, fresh, false, now))
	require.True(t, archiveJobNeedsWatchdog(liveVideo, fresh, true, now))
	require.True(t, archiveJobNeedsWatchdog(string(utils.TaskDownloadVideo), stale, false, now))
	require.False(t, archiveJobNeedsWatchdog(string(utils.TaskDownloadVideo), time.Time{}, false, now))
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
