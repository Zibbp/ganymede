package utils

type Role string

const (
	AdminRole    Role = "admin"
	EditorRole   Role = "editor"
	ArchiverRole Role = "archiver"
	UserRole     Role = "user"
)

func (Role) Values() (kinds []string) {
	for _, s := range []Role{AdminRole, EditorRole, ArchiverRole, UserRole} {
		kinds = append(kinds, string(s))
	}
	return
}

// IsValidRole checks if a string is a valid Role.
func IsValidRole(role string) bool {
	validRoles := map[string]struct{}{
		string(AdminRole):    {},
		string(EditorRole):   {},
		string(ArchiverRole): {},
		string(UserRole):     {},
	}

	_, exists := validRoles[role]
	return exists
}

type VideoPlatform string

const (
	PlatformTwitch  VideoPlatform = "twitch"
	PlatformYoutube VideoPlatform = "youtube"
)

func (VideoPlatform) Values() (kinds []string) {
	for _, s := range []VideoPlatform{PlatformTwitch, PlatformYoutube} {
		kinds = append(kinds, string(s))
	}
	return
}

type VodType string

const (
	Archive   VodType = "archive"
	Live      VodType = "live"
	Highlight VodType = "highlight"
	Upload    VodType = "upload"
	Clip      VodType = "clip"
)

func (VodType) Values() (kinds []string) {
	for _, s := range []VodType{Archive, Live, Highlight, Upload, Clip} {
		kinds = append(kinds, string(s))
	}
	return
}

type TaskStatus string

const (
	Success TaskStatus = "success"
	Running TaskStatus = "running"
	Pending TaskStatus = "pending"
	Failed  TaskStatus = "failed"
)

func (TaskStatus) Values() (kinds []string) {
	for _, s := range []TaskStatus{Success, Running, Pending, Failed} {
		kinds = append(kinds, string(s))
	}
	return
}

type VodQuality string

const (
	Best   VodQuality = "best"
	Source VodQuality = "source"
	R1080  VodQuality = "1080"
	R720   VodQuality = "720"
	R480   VodQuality = "480"
	R360   VodQuality = "360"
	R160   VodQuality = "160"
	Audio  VodQuality = "audio"
)

func (VodQuality) Values() (kinds []string) {
	for _, s := range []VodQuality{Best, Source, R1080, R720, R480, R360, R160, Audio} {
		kinds = append(kinds, string(s))
	}
	return
}

func (q VodQuality) String() string {
	return string(q)
}

type PlaybackStatus string

const (
	InProgress PlaybackStatus = "in_progress"
	Finished   PlaybackStatus = "finished"
)

func (PlaybackStatus) Values() (kinds []string) {
	for _, s := range []PlaybackStatus{InProgress, Finished} {
		kinds = append(kinds, string(s))
	}
	return
}

type TaskName string

const (
	TaskCreateFolder             TaskName = "task_vod_create_folder"
	TaskDownloadThumbnail        TaskName = "task_vod_download_thumbnail"
	TaskSaveInfo                 TaskName = "task_vod_save_info"
	TaskDownloadVideo            TaskName = "task_video_download"
	TaskDownloadLiveVideo        TaskName = "task_live_video_download" // not used queue
	TaskPostProcessVideo         TaskName = "task_video_convert"
	TaskMoveVideo                TaskName = "task_video_move"
	TaskDownloadChat             TaskName = "task_chat_download"
	TaskDownloadLiveChat         TaskName = "task_live_chat_download" // not used queue
	TaskConvertChat              TaskName = "task_chat_convert"
	TaskRenderChat               TaskName = "task_chat_render"
	TaskMoveChat                 TaskName = "task_chat_move"
	TaskUpdateLiveStreamMetadata TaskName = "task_update_live_stream_metadata" // not used queue
)

func (TaskName) Values() (kinds []string) {
	for _, s := range []TaskName{TaskCreateFolder, TaskDownloadThumbnail, TaskSaveInfo, TaskDownloadVideo, TaskPostProcessVideo, TaskMoveVideo, TaskDownloadChat, TaskConvertChat, TaskRenderChat, TaskMoveChat, TaskUpdateLiveStreamMetadata} {
		kinds = append(kinds, string(s))
	}
	return
}

func GetTaskName(s string) TaskName {
	switch s {
	case string(TaskCreateFolder):
		return TaskCreateFolder
	case string(TaskDownloadThumbnail):
		return TaskDownloadThumbnail
	case string(TaskSaveInfo):
		return TaskSaveInfo
	case string(TaskDownloadVideo):
		return TaskDownloadVideo
	case string(TaskPostProcessVideo):
		return TaskPostProcessVideo
	case string(TaskMoveVideo):
		return TaskMoveVideo
	case string(TaskDownloadChat):
		return TaskDownloadChat
	case string(TaskConvertChat):
		return TaskConvertChat
	case string(TaskRenderChat):
		return TaskRenderChat
	case string(TaskMoveChat):
		return TaskMoveChat
	case string(TaskUpdateLiveStreamMetadata):
		return TaskUpdateLiveStreamMetadata
	default:
		return ""
	}
}
