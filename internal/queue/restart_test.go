package queue

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/tasks"
	"github.com/zibbp/ganymede/internal/utils"
)

func TestRestartArgsUsesRequestedGeneration(t *testing.T) {
	queueID := uuid.New()
	args, err := restartArgs(StartQueueTaskInput{
		QueueId:  queueID,
		TaskName: string(utils.TaskDownloadVideo),
		Continue: true,
	}, 4)
	require.NoError(t, err)

	download, ok := args.(tasks.DownloadVideoArgs)
	require.True(t, ok)
	require.Equal(t, queueID, download.Input.QueueId)
	require.Equal(t, 4, download.Input.RecoveryGeneration)
	require.True(t, download.Continue)
}

func TestValidateRestartPrerequisite(t *testing.T) {
	tempDir := t.TempDir()
	downloaded := filepath.Join(tempDir, "download.mp4")
	require.NoError(t, os.WriteFile(downloaded, []byte("media"), 0o600))

	tests := []struct {
		name    string
		queue   *ent.Queue
		task    string
		wantErr error
	}{
		{
			name:  "vod conversion with input",
			queue: &ent.Queue{Edges: ent.QueueEdges{Vod: &ent.Vod{TmpVideoDownloadPath: downloaded}}},
			task:  string(utils.TaskPostProcessVideo),
		},
		{
			name:    "vod conversion without input",
			queue:   &ent.Queue{Edges: ent.QueueEdges{Vod: &ent.Vod{}}},
			task:    string(utils.TaskPostProcessVideo),
			wantErr: ErrRestartConflict,
		},
		{
			name:    "live task on vod",
			queue:   &ent.Queue{Edges: ent.QueueEdges{Vod: &ent.Vod{}}},
			task:    string(utils.TaskDownloadLiveVideo),
			wantErr: ErrRestartInvalid,
		},
		{
			name:    "chat disabled",
			queue:   &ent.Queue{ArchiveChat: false, Edges: ent.QueueEdges{Vod: &ent.Vod{}}},
			task:    string(utils.TaskRenderChat),
			wantErr: ErrRestartInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRestartPrerequisite(tt.queue, tt.task)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.True(t, errors.Is(err, tt.wantErr))
		})
	}
}

func TestRestartTaskNamesRejectsUnknownTask(t *testing.T) {
	_, _, err := restartTaskNames("not_a_task")
	require.ErrorIs(t, err, ErrRestartInvalid)
}
