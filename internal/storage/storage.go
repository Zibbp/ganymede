// Package storage reports archive temp-directory disk availability and the
// low-space state used to pause new archives and warn the user.
package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/utils"
)

// State describes whether new archives may start.
type State string

const (
	StateOK       State = "ok"
	StateWarning  State = "warning"  // New archives are paused; in-flight work drains.
	StateCritical State = "critical" // Additionally, live captures are stopped to protect partial media.
	StateDisabled State = "disabled" // Storage check is turned off.
	StateUnknown  State = "unknown"  // Free space could not be determined; fail open.
)

// Status is a point-in-time snapshot of temp-directory availability.
type Status struct {
	State      State
	FreeBytes  int64
	TotalBytes int64
	// Err is set when the filesystem probe failed; State is StateUnknown then.
	Err error
}

// ErrLowSpace is returned by checkers when new archives must not start.
var ErrLowSpace = errors.New("not enough free space in the temp directory to start new archives")

// Thresholds are the low-space cutoffs in bytes.
type Thresholds struct {
	WarningBytes  int64
	CriticalBytes int64
}

// Checker reports temp-directory availability. The archive gate and the
// storage watchdog both depend on this seam.
type Checker interface {
	Check(ctx context.Context) (Status, error)
}

const cacheTTL = 30 * time.Second

// probeFunc returns free and total bytes for a path. Injectable for tests.
type probeFunc func(path string) (free int64, total int64, err error)

// Service checks TEMP_DIR free space against configured thresholds. Probe
// results are cached briefly so callers can check on every archive creation
// without hammering network mounts.
type Service struct {
	tempDir  string
	probe    probeFunc
	cacheTTL time.Duration

	mu       sync.Mutex
	cached   Status
	cachedAt time.Time
}

// NewService returns a checker for the configured temp directory.
func NewService() *Service {
	return newService(config.GetEnvConfig().TempDir, cacheTTL, defaultProbe)
}

func newService(tempDir string, ttl time.Duration, probe probeFunc) *Service {
	return &Service{tempDir: tempDir, probe: probe, cacheTTL: ttl}
}

func defaultProbe(path string) (int64, int64, error) {
	free, err := utils.GetFreeSpaceOfDirectory(path)
	if err != nil {
		return 0, 0, err
	}
	total, err := utils.GetTotalSpaceOfDirectory(path)
	if err != nil {
		return 0, 0, err
	}
	return free, total, nil
}

// Check returns the current temp-directory status. When the check is
// disabled, stats are still reported (for the system overview) with a
// disabled state that the archive gate ignores. Probe errors are reported in
// the Status rather than returned so callers can deliberately fail open when
// the filesystem cannot be probed (e.g. exotic network mounts).
func (s *Service) Check(_ context.Context) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Since(s.cachedAt) < s.cacheTTL && s.cached.State != StateUnknown {
		return s.cached, nil
	}

	status, err := s.probeUncached()
	if err != nil {
		return Status{State: StateUnknown, Err: err}, nil
	}
	if !config.Get().Archive.StorageCheckEnabled {
		status.State = StateDisabled
	}
	s.cached, s.cachedAt = status, time.Now()
	return status, nil
}

func (s *Service) probeUncached() (Status, error) {
	free, total, err := s.probe(s.tempDir)
	if err != nil {
		return Status{}, fmt.Errorf("probe temp directory %s: %w", s.tempDir, err)
	}

	thresholds := thresholdsFromConfig()
	switch {
	case free <= thresholds.CriticalBytes:
		return Status{State: StateCritical, FreeBytes: free, TotalBytes: total}, nil
	case free <= thresholds.WarningBytes:
		return Status{State: StateWarning, FreeBytes: free, TotalBytes: total}, nil
	default:
		return Status{State: StateOK, FreeBytes: free, TotalBytes: total}, nil
	}
}

func thresholdsFromConfig() Thresholds {
	cfg := config.Get()
	GB := int64(1024 * 1024 * 1024)
	warning := int64(cfg.Archive.TempDirWarningThresholdGB) * GB
	critical := int64(cfg.Archive.TempDirCriticalThresholdGB) * GB
	// A misconfigured critical threshold above the warning threshold would
	// trip the hard stop first; keep the documented ordering.
	if critical > warning {
		critical = warning
	}
	return Thresholds{WarningBytes: warning, CriticalBytes: critical}
}

// Gate blocks when new archives must not start. Probe failures fail open so
// an unreadable filesystem never fully breaks archiving.
func Gate(ctx context.Context, checker Checker) error {
	if checker == nil {
		return nil
	}
	status, _ := checker.Check(ctx)
	switch status.State {
	case StateWarning, StateCritical:
		return fmt.Errorf("%w: %.1fGiB free (state %s)", ErrLowSpace, float64(status.FreeBytes)/(1024*1024*1024), status.State)
	default:
		return nil
	}
}
