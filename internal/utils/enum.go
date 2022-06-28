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

type VodPlatform string

const (
	PlatformTwitch  VodPlatform = "twitch"
	PlatformYoutube VodPlatform = "youtube"
)

func (VodPlatform) Values() (kinds []string) {
	for _, s := range []VodPlatform{PlatformTwitch, PlatformYoutube} {
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
	Success    TaskStatus = "success"
	Processing TaskStatus = "processing"
	Waiting    TaskStatus = "waiting"
	Error      TaskStatus = "Error"
)

func (TaskStatus) Values() (kinds []string) {
	for _, s := range []TaskStatus{Success, Processing, Waiting, Error} {
		kinds = append(kinds, string(s))
	}
	return
}
