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
	Source  VodQuality = "source"
	R720P60 VodQuality = "720p60"
	R480P30 VodQuality = "480p30"
	R360P30 VodQuality = "360p30"
	R160P30 VodQuality = "160p30"
)

func (VodQuality) Values() (kinds []string) {
	for _, s := range []VodQuality{Source, R720P60, R480P30, R360P30, R160P30} {
		kinds = append(kinds, string(s))
	}
	return
}
