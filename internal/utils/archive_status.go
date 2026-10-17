package utils

// ArchiveStatus describes processing, independently of capture completeness.
type ArchiveStatus string

const (
	ArchiveQueued     ArchiveStatus = "queued"
	ArchiveRunning    ArchiveStatus = "running"
	ArchiveFinalizing ArchiveStatus = "finalizing"
	ArchiveCompleted  ArchiveStatus = "completed"
	ArchiveFailed     ArchiveStatus = "failed"
)

func (ArchiveStatus) Values() []string {
	return []string{string(ArchiveQueued), string(ArchiveRunning), string(ArchiveFinalizing), string(ArchiveCompleted), string(ArchiveFailed)}
}

func (s ArchiveStatus) Valid() bool {
	switch s {
	case ArchiveQueued, ArchiveRunning, ArchiveFinalizing, ArchiveCompleted, ArchiveFailed:
		return true
	default:
		return false
	}
}

func (s ArchiveStatus) Active() bool {
	return s == ArchiveQueued || s == ArchiveRunning || s == ArchiveFinalizing
}

func ActiveArchiveStatuses() []ArchiveStatus {
	return []ArchiveStatus{ArchiveQueued, ArchiveRunning, ArchiveFinalizing}
}
