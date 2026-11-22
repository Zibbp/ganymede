// Package storage reports how much free space is left in the temp directory.
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
	StateWarning  State = "warning"  // New archives are paused.
	StateCritical State = "critical" // New archives are paused and live captures are stopped.
	StateDisabled State = "disabled" // The storage check is turned off.
	StateUnknown  State = "unknown"  // Free space could not be read.
)

// Status is a snapshot of temp-directory free space.
type Status struct {
	State      State
	FreeBytes  int64
	TotalBytes int64
	// Set when free space could not be read; State is StateUnknown then.
	Err error
}

// ErrLowSpace is returned when new archives must not start.
var ErrLowSpace = errors.New("not enough free space in the temp directory to start new archives")

// Thresholds in bytes.
type Thresholds struct {
	WarningBytes  int64
	CriticalBytes int64
}

// Checker reports temp-directory free space.
type Checker interface {
	Check(ctx context.Context) (Status, error)
}

const (
	cacheTTL     = 30 * time.Second
	probeTimeout = 5 * time.Second
	// Oldest reading served while a probe is stuck. Older readings turn
	// unknown so a stale critical value cannot block archives forever.
	readingMaxAge = 5 * time.Minute
)

// probeFunc reads free and total bytes for a path. Swappable in tests.
type probeFunc func(path string) (free int64, total int64, err error)

// probeResult is a raw filesystem reading. Only readings are cached so
// settings changes apply on the next call.
type probeResult struct {
	free  int64
	total int64
	err   error
}

// Service checks TEMP_DIR free space against the configured thresholds.
type Service struct {
	tempDir      string
	probe        probeFunc
	cacheTTL     time.Duration
	probeTimeout time.Duration

	mu       sync.Mutex
	readings probeResult
	probedAt time.Time
	probing  bool
}

// NewService returns a checker for the configured temp directory.
func NewService() *Service {
	return newService(config.GetEnvConfig().TempDir, cacheTTL, defaultProbe)
}

func newService(tempDir string, ttl time.Duration, probe probeFunc) *Service {
	return &Service{tempDir: tempDir, probe: probe, cacheTTL: ttl, probeTimeout: probeTimeout}
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

// Check returns the current status. Stats are still reported when the check
// is disabled. If free space cannot be read, nothing is blocked.
func (s *Service) Check(ctx context.Context) (Status, error) {
	s.mu.Lock()
	if s.probing {
		// A probe is already running; use the last known readings.
		result, ok := s.lastReadingsLocked()
		s.mu.Unlock()
		if !ok {
			return Status{State: StateUnknown}, nil
		}
		return statusFromReadings(result), nil
	}
	if s.readings.err == nil && time.Since(s.probedAt) < s.cacheTTL {
		result := s.readings
		s.mu.Unlock()
		return statusFromReadings(result), nil
	}
	s.probing = true
	s.mu.Unlock()

	done := make(chan probeResult, 1)
	go func() {
		free, total, err := s.probe(s.tempDir)
		result := probeResult{free: free, total: total, err: err}

		s.mu.Lock()
		s.readings = result
		s.probedAt = time.Now()
		s.probing = false
		s.mu.Unlock()

		done <- result
	}()

	select {
	case result := <-done:
		return statusFromReadings(result), nil
	case <-time.After(s.probeTimeout):
		// The probe is stuck (e.g. an NFS mount that never answers).
		// Don't block callers on it; the goroutine caches when it finishes.
		return s.fallback(), nil
	case <-ctx.Done():
		// Caller gave up; same fallback as a stuck probe.
		return s.fallback(), nil
	}
}

func (s *Service) fallback() Status {
	s.mu.Lock()
	result, ok := s.lastReadingsLocked()
	s.mu.Unlock()
	if !ok {
		return Status{State: StateUnknown}
	}
	return statusFromReadings(result)
}

// lastReadingsLocked returns the last stored readings. Callers must hold mu.
func (s *Service) lastReadingsLocked() (probeResult, bool) {
	if s.probedAt.IsZero() || time.Since(s.probedAt) > readingMaxAge {
		return probeResult{}, false
	}
	return s.readings, true
}

// statusFromReadings computes the state from the current settings so config
// changes apply immediately.
func statusFromReadings(r probeResult) Status {
	if r.err != nil {
		return Status{State: StateUnknown, Err: r.err}
	}
	cfg := config.Get()
	if !cfg.Archive.StorageCheckEnabled {
		return Status{State: StateDisabled, FreeBytes: r.free, TotalBytes: r.total}
	}

	t := thresholdsFromConfig()
	switch {
	case r.free <= t.CriticalBytes:
		return Status{State: StateCritical, FreeBytes: r.free, TotalBytes: r.total}
	case r.free <= t.WarningBytes:
		return Status{State: StateWarning, FreeBytes: r.free, TotalBytes: r.total}
	default:
		return Status{State: StateOK, FreeBytes: r.free, TotalBytes: r.total}
	}
}

func thresholdsFromConfig() Thresholds {
	cfg := config.Get()
	GB := int64(1024 * 1024 * 1024)
	warning := int64(cfg.Archive.TempDirWarningThresholdGB) * GB
	critical := int64(cfg.Archive.TempDirCriticalThresholdGB) * GB
	// Critical must never be above warning.
	if critical > warning {
		critical = warning
	}
	return Thresholds{WarningBytes: warning, CriticalBytes: critical}
}

// Gate blocks new archives when the temp directory is low on space.
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
