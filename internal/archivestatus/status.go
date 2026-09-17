package archivestatus

import (
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
)

func VideoDone(q *ent.Queue) bool {
	return q.TaskVideoDownload == utils.Success && q.TaskVideoConvert == utils.Success && q.TaskVideoMove == utils.Success
}

func ChatDone(q *ent.Queue) bool {
	if !q.ArchiveChat {
		return true
	}
	// A terminal chat failure must not leave finished video processing forever.
	steps := []utils.TaskStatus{q.TaskChatDownload, q.TaskChatMove}
	if q.LiveArchive {
		steps = append(steps, q.TaskChatConvert)
	}
	if q.RenderChat {
		steps = append(steps, q.TaskChatRender)
	}
	allSucceeded := true
	for _, status := range steps {
		if status == utils.Failed {
			return true
		}
		allSucceeded = allSucceeded && status == utils.Success
	}
	return allSucceeded
}

// FromQueue summarizes the pipeline; task errors remain available on the queue.
func FromQueue(q *ent.Queue) utils.ArchiveStatus {
	if VideoDone(q) {
		if ChatDone(q) {
			return utils.ArchiveCompleted
		}
		return utils.ArchiveFinalizing
	}
	for _, status := range []utils.TaskStatus{
		q.TaskVodCreateFolder, q.TaskVodSaveInfo, q.TaskVodDownloadThumbnail,
		q.TaskVideoDownload, q.TaskVideoConvert, q.TaskVideoMove,
	} {
		if status == utils.Failed {
			return utils.ArchiveFailed
		}
	}
	if q.TaskVideoDownload == utils.Running {
		return utils.ArchiveRunning
	}
	if q.TaskVideoDownload == utils.Success {
		return utils.ArchiveFinalizing
	}
	return utils.ArchiveQueued
}
