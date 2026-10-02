package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/zibbp/ganymede/ent"
	entqueue "github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/internal/tasks"
	"github.com/zibbp/ganymede/internal/utils"
)

var (
	ErrRestartNotFound = errors.New("queue not found")
	ErrRestartInvalid  = errors.New("invalid queue task restart")
	ErrRestartConflict = errors.New("queue task restart conflict")
)

type StartQueueTaskInput struct {
	QueueId  uuid.UUID
	TaskName string
	Continue bool
}

type RestartResult struct {
	JobID                int64  `json:"job_id"`
	Generation           int    `json:"generation"`
	TaskName             string `json:"task_name"`
	OlderActiveCancelled bool   `json:"older_active_cancelled"`
}

// RestartQueueTask atomically schedules a new executable generation of an
// archive stage and resets its persisted lifecycle state.
func (s *Service) RestartQueueTask(ctx context.Context, input StartQueueTaskInput) (*RestartResult, error) {
	kind, statusTask, err := restartTaskNames(input.TaskName)
	if err != nil {
		return nil, err
	}

	var restartResult *RestartResult
	err = s.Store.WithTx(ctx, func(txClient *ent.Client, tx *sql.Tx) error {
		var lockedID uuid.UUID
		if err := tx.QueryRowContext(ctx, `SELECT id FROM queues WHERE id = $1 FOR UPDATE`, input.QueueId).Scan(&lockedID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: %s", ErrRestartNotFound, input.QueueId)
			}
			return fmt.Errorf("lock queue: %w", err)
		}
		q, err := txClient.Queue.Query().Where(entqueue.ID(input.QueueId)).WithVod().Only(ctx)
		if err != nil {
			return fmt.Errorf("load queue: %w", err)
		}
		if err := validateRestartPrerequisite(q, input.TaskName); err != nil {
			return err
		}

		stages := affectedRestartStages(q, input.TaskName)
		kinds := make([]string, 0, len(stages))
		for _, stage := range stages {
			kinds = append(kinds, stage.kind)
		}
		jobs, err := s.RiverClient.JobsForQueueKindsTx(ctx, tx, input.QueueId, kinds...)
		if err != nil {
			return fmt.Errorf("list existing task generations: %w", err)
		}
		generation := 1
		cancelled := false
		stageStatus := restartStageStatus(q, statusTask)
		for _, job := range jobs {
			var args tasks.RiverJobArgs
			if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
				return fmt.Errorf("decode existing job %d: %w", job.ID, err)
			}
			if job.Kind == kind && args.Input.RecoveryGeneration >= generation {
				generation = args.Input.RecoveryGeneration + 1
			}
			if job.Kind != kind && isActiveJobState(job.State) {
				return fmt.Errorf("%w: downstream stage %s already has active River job %d", ErrRestartConflict, job.Kind, job.ID)
			}
			switch job.State {
			case rivertype.JobStateRunning:
				return fmt.Errorf("%w: %s is already running as River job %d", ErrRestartConflict, input.TaskName, job.ID)
			case rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRetryable, rivertype.JobStateScheduled:
				if stageStatus == utils.Pending || stageStatus == utils.Running {
					return fmt.Errorf("%w: %s already has active River job %d", ErrRestartConflict, input.TaskName, job.ID)
				}
				if err := s.RiverClient.CancelTx(ctx, tx, job.ID); err != nil {
					return fmt.Errorf("cancel existing River job %d: %w", job.ID, err)
				}
				cancelled = true
			}
		}

		resetStages := stages
		if !input.Continue {
			resetStages = stages[:1]
		}
		if err := resetRestartState(ctx, txClient, q, resetStages); err != nil {
			return err
		}
		args, err := restartArgs(input, generation)
		if err != nil {
			return err
		}
		inserted, err := s.RiverClient.InsertTx(ctx, tx, args, nil)
		if err != nil {
			return fmt.Errorf("insert replacement task: %w", err)
		}
		if inserted.UniqueSkippedAsDuplicate {
			return fmt.Errorf("%w: River rejected replacement generation %d as a duplicate", ErrRestartConflict, generation)
		}
		restartResult = &RestartResult{JobID: inserted.Job.ID, Generation: generation, TaskName: input.TaskName, OlderActiveCancelled: cancelled}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return restartResult, nil
}

type restartStage struct {
	kind   string
	status utils.TaskName
}

func affectedRestartStages(q *ent.Queue, task string) []restartStage {
	kind, status, _ := restartTaskNames(task)
	stages := []restartStage{{kind: kind, status: status}}

	video := []restartStage{
		{kind: string(utils.TaskDownloadVideo), status: utils.TaskDownloadVideo},
		{kind: string(utils.TaskPostProcessVideo), status: utils.TaskPostProcessVideo},
		{kind: string(utils.TaskMoveVideo), status: utils.TaskMoveVideo},
	}
	if q.LiveArchive {
		video[0].kind = string(utils.TaskDownloadLiveVideo)
	}
	chat := []restartStage{{kind: string(utils.TaskDownloadChat), status: utils.TaskDownloadChat}}
	if q.LiveArchive {
		chat[0].kind = string(utils.TaskDownloadLiveChat)
		chat = append(chat, restartStage{kind: string(utils.TaskConvertChat), status: utils.TaskConvertChat})
	}
	if q.RenderChat {
		chat = append(chat, restartStage{kind: string(utils.TaskRenderChat), status: utils.TaskRenderChat})
	}
	chat = append(chat, restartStage{kind: string(utils.TaskMoveChat), status: utils.TaskMoveChat})

	appendArchiveBranches := func() {
		stages = append(stages, video...)
		if q.ArchiveChat {
			stages = append(stages, chat...)
		}
	}
	switch task {
	case string(utils.TaskCreateFolder):
		stages = append(stages,
			restartStage{kind: string(utils.TaskSaveInfo), status: utils.TaskSaveInfo},
			restartStage{kind: string(utils.TaskDownloadThumbnail), status: utils.TaskDownloadThumbnail},
		)
		appendArchiveBranches()
	case string(utils.TaskSaveInfo):
		stages = append(stages, restartStage{kind: string(utils.TaskDownloadThumbnail), status: utils.TaskDownloadThumbnail})
		appendArchiveBranches()
	case string(utils.TaskDownloadThumbnail):
		appendArchiveBranches()
	case string(utils.TaskDownloadVideo), string(utils.TaskDownloadLiveVideo):
		stages = append(stages, video[1:]...)
	case string(utils.TaskPostProcessVideo):
		stages = append(stages, video[2])
	case string(utils.TaskDownloadChat), string(utils.TaskDownloadLiveChat):
		stages = append(stages, chat[1:]...)
	case string(utils.TaskConvertChat):
		for _, stage := range chat {
			if stage.status == utils.TaskRenderChat || stage.status == utils.TaskMoveChat {
				stages = append(stages, stage)
			}
		}
	case string(utils.TaskRenderChat):
		stages = append(stages, restartStage{kind: string(utils.TaskMoveChat), status: utils.TaskMoveChat})
	}
	return stages
}

func isActiveJobState(state rivertype.JobState) bool {
	switch state {
	case rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRetryable, rivertype.JobStateRunning, rivertype.JobStateScheduled:
		return true
	default:
		return false
	}
}

func restartStageStatus(q *ent.Queue, task utils.TaskName) utils.TaskStatus {
	switch task {
	case utils.TaskCreateFolder:
		return q.TaskVodCreateFolder
	case utils.TaskDownloadThumbnail:
		return q.TaskVodDownloadThumbnail
	case utils.TaskSaveInfo:
		return q.TaskVodSaveInfo
	case utils.TaskDownloadVideo:
		return q.TaskVideoDownload
	case utils.TaskPostProcessVideo:
		return q.TaskVideoConvert
	case utils.TaskMoveVideo:
		return q.TaskVideoMove
	case utils.TaskDownloadChat:
		return q.TaskChatDownload
	case utils.TaskConvertChat:
		return q.TaskChatConvert
	case utils.TaskRenderChat:
		return q.TaskChatRender
	case utils.TaskMoveChat:
		return q.TaskChatMove
	default:
		return ""
	}
}

// StartQueueTask preserves the existing service API for callers outside the
// HTTP transport while applying the new restart semantics.
func (s *Service) StartQueueTask(ctx context.Context, input StartQueueTaskInput) (*RestartResult, error) {
	return s.RestartQueueTask(ctx, input)
}

func restartTaskNames(name string) (string, utils.TaskName, error) {
	switch name {
	case string(utils.TaskCreateFolder), string(utils.TaskDownloadThumbnail), string(utils.TaskSaveInfo), string(utils.TaskDownloadVideo), string(utils.TaskPostProcessVideo), string(utils.TaskMoveVideo), string(utils.TaskDownloadChat), string(utils.TaskConvertChat), string(utils.TaskRenderChat), string(utils.TaskMoveChat):
		return name, utils.GetTaskName(name), nil
	case string(utils.TaskDownloadLiveVideo):
		return name, utils.TaskDownloadVideo, nil
	case string(utils.TaskDownloadLiveChat):
		return name, utils.TaskDownloadChat, nil
	default:
		return "", "", fmt.Errorf("%w: unknown task %q", ErrRestartInvalid, name)
	}
}

func restartArgs(input StartQueueTaskInput, generation int) (river.JobArgs, error) {
	taskInput := tasks.ArchiveVideoInput{QueueId: input.QueueId, RecoveryGeneration: generation}
	switch input.TaskName {
	case string(utils.TaskCreateFolder):
		return tasks.CreateDirectoryArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskDownloadThumbnail):
		return tasks.DownloadThumbnailArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskSaveInfo):
		return tasks.SaveVideoInfoArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskDownloadVideo):
		return tasks.DownloadVideoArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskDownloadLiveVideo):
		return tasks.DownloadLiveVideoArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskPostProcessVideo):
		return tasks.PostProcessVideoArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskMoveVideo):
		return tasks.MoveVideoArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskDownloadChat):
		return tasks.DownloadChatArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskDownloadLiveChat):
		return tasks.DownloadLiveChatArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskConvertChat):
		return tasks.ConvertLiveChatArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskRenderChat):
		return tasks.RenderChatArgs{Continue: input.Continue, Input: taskInput}, nil
	case string(utils.TaskMoveChat):
		return tasks.MoveChatArgs{Continue: input.Continue, Input: taskInput}, nil
	default:
		return nil, fmt.Errorf("%w: unknown task %q", ErrRestartInvalid, input.TaskName)
	}
}

func validateRestartPrerequisite(q *ent.Queue, task string) error {
	v := q.Edges.Vod
	conflict := func(message string) error { return fmt.Errorf("%w: %s", ErrRestartConflict, message) }
	if (task == string(utils.TaskDownloadLiveVideo) || task == string(utils.TaskDownloadLiveChat)) && !q.LiveArchive {
		return fmt.Errorf("%w: %s only applies to live archives", ErrRestartInvalid, task)
	}
	if (task == string(utils.TaskDownloadVideo) || task == string(utils.TaskDownloadChat)) && q.LiveArchive {
		return fmt.Errorf("%w: use the live-download task for a live archive", ErrRestartInvalid)
	}
	if (task == string(utils.TaskDownloadChat) || task == string(utils.TaskDownloadLiveChat) || task == string(utils.TaskConvertChat) || task == string(utils.TaskRenderChat) || task == string(utils.TaskMoveChat)) && !q.ArchiveChat {
		return fmt.Errorf("%w: chat archiving is disabled for this queue", ErrRestartInvalid)
	}
	switch task {
	case string(utils.TaskPostProcessVideo):
		if q.LiveArchive {
			if !directoryHasNonEmptyFile(v.TmpVideoHlsPath) {
				return conflict("video conversion requires recoverable live HLS segments")
			}
		} else if !nonEmptyFile(v.TmpVideoDownloadPath) {
			return conflict("video conversion requires a nonempty downloaded video")
		}
	case string(utils.TaskMoveVideo):
		if v.VideoHlsPath != "" {
			if !directoryHasNonEmptyFile(v.TmpVideoHlsPath) && !directoryHasNonEmptyFile(v.VideoHlsPath) {
				return conflict("video move requires converted HLS output")
			}
		} else if !nonEmptyFile(v.TmpVideoConvertPath) && !nonEmptyFile(v.TmpVideoDownloadPath) && !nonEmptyFile(v.VideoPath) {
			return conflict("video move requires converted or downloaded video output")
		}
	case string(utils.TaskConvertChat):
		if !q.LiveArchive {
			return fmt.Errorf("%w: chat conversion only applies to live archives", ErrRestartInvalid)
		}
		if !nonEmptyFile(v.TmpLiveChatDownloadPath) {
			return conflict("chat conversion requires raw live chat")
		}
	case string(utils.TaskRenderChat):
		if !nonEmptyFile(v.TmpChatDownloadPath) {
			return conflict("chat rendering requires converted or downloaded chat")
		}
	case string(utils.TaskMoveChat):
		if !nonEmptyFile(v.TmpChatDownloadPath) && !nonEmptyFile(v.ChatPath) {
			return conflict("chat move requires downloaded chat output")
		}
	}
	return nil
}

func resetRestartState(ctx context.Context, client *ent.Client, q *ent.Queue, stages []restartStage) error {
	u := client.Queue.UpdateOneID(q.ID).SetProcessing(true).SetOnHold(false)
	for _, stage := range stages {
		switch stage.status {
		case utils.TaskCreateFolder:
			u.SetTaskVodCreateFolder(utils.Pending)
		case utils.TaskDownloadThumbnail:
			u.SetTaskVodDownloadThumbnail(utils.Pending)
		case utils.TaskSaveInfo:
			u.SetTaskVodSaveInfo(utils.Pending)
		case utils.TaskDownloadVideo:
			u.SetTaskVideoDownload(utils.Pending).SetVideoProcessing(true)
		case utils.TaskPostProcessVideo:
			u.SetTaskVideoConvert(utils.Pending).SetVideoProcessing(true)
		case utils.TaskMoveVideo:
			u.SetTaskVideoMove(utils.Pending).SetVideoProcessing(true)
		case utils.TaskDownloadChat:
			u.SetTaskChatDownload(utils.Pending).SetChatProcessing(true)
		case utils.TaskConvertChat:
			u.SetTaskChatConvert(utils.Pending).SetChatProcessing(true)
		case utils.TaskRenderChat:
			u.SetTaskChatRender(utils.Pending).SetChatProcessing(true)
		case utils.TaskMoveChat:
			u.SetTaskChatMove(utils.Pending).SetChatProcessing(true)
		}
	}
	if _, err := u.Save(ctx); err != nil {
		return fmt.Errorf("reset queue task: %w", err)
	}
	if _, err := client.Vod.UpdateOneID(q.Edges.Vod.ID).SetProcessing(true).Save(ctx); err != nil {
		return fmt.Errorf("reset archive lifecycle: %w", err)
	}
	return nil
}

func nonEmptyFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}
func directoryHasNonEmptyFile(path string) bool {
	if path == "" {
		return false
	}
	found := false
	_ = filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		info, statErr := entry.Info()
		if statErr == nil && info.Size() > 0 {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
