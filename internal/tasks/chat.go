package tasks

import (
	"context"
	"time"

	"github.com/riverqueue/river"
	"github.com/zibbp/ganymede/internal/errors"
	"github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/utils"
)

// //////////////////////
// Download Chat (VOD) //
// //////////////////////
type DownloadChatArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (DownloadChatArgs) Kind() string { return string(utils.TaskDownloadChat) }

func (args DownloadChatArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Queue:       QueueChatDownload,
		Tags:        []string{"archive"},
		UniqueOpts:  archiveUniqueOpts(),
	}
}

func (w *DownloadChatWorker) Timeout(job *river.Job[DownloadChatArgs]) time.Duration {
	return 49 * time.Hour
}

type DownloadChatWorker struct {
	river.WorkerDefaults[DownloadChatArgs]
}

func (w DownloadChatWorker) Work(ctx context.Context, job *river.Job[DownloadChatArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadChat,
	})
	if err != nil {
		return err
	}

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	// download video
	err = exec.DownloadTwitchChat(ctx, dbItems.Video)
	if err != nil {
		return err
	}

	next := []transactionalJob{}
	if job.Args.Continue {
		if dbItems.Queue.RenderChat {
			next = append(next, transactionalJob{Args: &RenderChatArgs{Continue: true, Input: nextArchiveInput(job.Args.Input)}})
		} else {
			next = append(next, transactionalJob{Args: &MoveChatArgs{Continue: true, Input: nextArchiveInput(job.Args.Input)}})
		}
	}
	err = setQueueStatusAndEnqueue(ctx, store, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskDownloadChat,
	}, next...)
	if err != nil {
		return err
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}

	return nil
}

// ////////////////////
// Render Chat (VOD) //
// ////////////////////
type RenderChatArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (RenderChatArgs) Kind() string { return string(utils.TaskRenderChat) }

func (args RenderChatArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Queue:       QueueChatRender,
		Tags:        []string{"archive"},
		UniqueOpts:  archiveUniqueOpts(),
	}
}

func (w *RenderChatWorker) Timeout(job *river.Job[RenderChatArgs]) time.Duration {
	return 49 * time.Hour
}

type RenderChatWorker struct {
	river.WorkerDefaults[RenderChatArgs]
}

func (w RenderChatWorker) Work(ctx context.Context, job *river.Job[RenderChatArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskRenderChat,
	})
	if err != nil {
		return err
	}

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	continueArchive := true

	// download video
	err = exec.RenderTwitchChat(ctx, dbItems.Video)
	if err != nil {

		// check if chat render has no messages
		// not a real error - continue with next job
		if errors.Is(err, errors.ErrNoChatMessages) {
			continueArchive = false
			// set video chat path to empty
			_, err = store.Client.Vod.UpdateOneID(dbItems.Video.ID).SetChatPath("").SetChatVideoPath("").Save(ctx)
			if err != nil {
				return err
			}
			// set queue chat to completed
			_, err = store.Client.Queue.UpdateOneID(job.Args.Input.QueueId).SetChatProcessing(false).SetTaskChatMove(utils.Success).Save(ctx)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	next := []transactionalJob{}
	if job.Args.Continue && continueArchive {
		next = append(next, transactionalJob{Args: &MoveChatArgs{Continue: true, Input: nextArchiveInput(job.Args.Input)}})
	}
	err = setQueueStatusAndEnqueue(ctx, store, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskRenderChat,
	}, next...)
	if err != nil {
		return err
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}

	return nil
}

// ////////////
// Move Chat //
// ///////////
type MoveChatArgs struct {
	Continue bool              `json:"continue"`
	Input    ArchiveVideoInput `json:"input"`
}

func (MoveChatArgs) Kind() string { return string(utils.TaskMoveChat) }

func (args MoveChatArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
		Tags:        []string{"archive"},
		UniqueOpts:  archiveUniqueOpts(),
	}
}

func (w *MoveChatWorker) Timeout(job *river.Job[MoveChatArgs]) time.Duration {
	return 49 * time.Hour
}

type MoveChatWorker struct {
	river.WorkerDefaults[MoveChatArgs]
}

func (w MoveChatWorker) Work(ctx context.Context, job *river.Job[MoveChatArgs]) error {
	// get store from context
	store, err := StoreFromContext(ctx)
	if err != nil {
		return err
	}

	// set queue status to running
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Running,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskMoveChat,
	})
	if err != nil {
		return err
	}

	dbItems, err := getDatabaseItems(ctx, store.Client, job.Args.Input.QueueId)
	if err != nil {
		return err
	}

	err = utils.MoveFile(ctx, dbItems.Video.TmpChatDownloadPath, dbItems.Video.ChatPath)
	if err != nil {
		return err
	}

	if dbItems.Queue.LiveArchive {
		err = utils.MoveFile(ctx, dbItems.Video.TmpLiveChatDownloadPath, dbItems.Video.LiveChatPath)
		if err != nil {
			return err
		}
		err = utils.MoveFile(ctx, dbItems.Video.TmpLiveChatConvertPath, dbItems.Video.LiveChatConvertPath)
		if err != nil {
			return err
		}
	}

	if dbItems.Queue.RenderChat {
		err = utils.MoveFile(ctx, dbItems.Video.TmpChatRenderPath, dbItems.Video.ChatVideoPath)
		if err != nil {
			return err
		}
	}

	// set queue status to completed
	err = setQueueStatus(ctx, store.Client, QueueStatusInput{
		Status:  utils.Success,
		QueueId: job.Args.Input.QueueId,
		Task:    utils.TaskMoveChat,
	})
	if err != nil {
		return err
	}

	// check if tasks are done
	if err := checkIfTasksAreDone(ctx, store.Client, job.Args.Input); err != nil {
		return err
	}

	return nil
}
