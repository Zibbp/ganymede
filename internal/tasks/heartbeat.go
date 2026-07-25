package tasks

// RiverJobArgs is the common JSON shape used when inspecting archive jobs
// without knowing their concrete argument type.
type RiverJobArgs struct {
	VideoId  string            `json:"video_id"`
	Input    ArchiveVideoInput `json:"input"`
	Continue bool              `json:"continue"`
}
