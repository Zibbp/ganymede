package archivestatus

import (
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
	"testing"
)

func TestFromQueue(t *testing.T) {
	tests := []struct {
		name  string
		queue ent.Queue
		want  utils.ArchiveStatus
	}{
		{"waiting", ent.Queue{}, utils.ArchiveQueued},
		{"download", ent.Queue{TaskVideoDownload: utils.Running}, utils.ArchiveRunning},
		{"download retry", ent.Queue{TaskVideoDownload: utils.Pending}, utils.ArchiveQueued},
		{"waiting to remux", ent.Queue{TaskVideoDownload: utils.Success}, utils.ArchiveFinalizing},
		{"preparation failed", ent.Queue{TaskVodSaveInfo: utils.Failed}, utils.ArchiveFailed},
		{"download failed", ent.Queue{TaskVideoDownload: utils.Failed}, utils.ArchiveFailed},
		{"remux failed", ent.Queue{TaskVideoDownload: utils.Success, TaskVideoConvert: utils.Failed}, utils.ArchiveFailed},
		{"move failed", ent.Queue{TaskVideoDownload: utils.Success, TaskVideoConvert: utils.Success, TaskVideoMove: utils.Failed}, utils.ArchiveFailed},
		{"chat failed during video", ent.Queue{ArchiveChat: true, TaskChatDownload: utils.Failed, TaskVideoDownload: utils.Running}, utils.ArchiveRunning},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.want, FromQueue(&tc.queue)) })
	}
}

func TestCompletionWithOptionalChat(t *testing.T) {
	base := ent.Queue{TaskVideoDownload: utils.Success, TaskVideoConvert: utils.Success, TaskVideoMove: utils.Success}
	require.Equal(t, utils.ArchiveCompleted, FromQueue(&base))
	base.ArchiveChat = true
	require.Equal(t, utils.ArchiveFinalizing, FromQueue(&base))
	base.TaskChatDownload, base.TaskChatMove = utils.Success, utils.Success
	require.Equal(t, utils.ArchiveCompleted, FromQueue(&base))
	base.LiveArchive = true
	require.Equal(t, utils.ArchiveFinalizing, FromQueue(&base))
	base.TaskChatConvert = utils.Success
	base.RenderChat = true
	require.Equal(t, utils.ArchiveFinalizing, FromQueue(&base))
	base.TaskChatRender = utils.Failed
	require.Equal(t, utils.ArchiveCompleted, FromQueue(&base), "terminal chat failure does not invalidate the video")
	base.TaskChatRender = utils.Running
	require.Equal(t, utils.ArchiveFinalizing, FromQueue(&base), "explicit chat retry reopens finalization")
	base.TaskChatRender = utils.Success
	require.Equal(t, utils.ArchiveCompleted, FromQueue(&base))
}
