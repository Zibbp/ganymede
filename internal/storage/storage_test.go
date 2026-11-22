package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/config"
)

func initTestConfig(t *testing.T) {
	t.Helper()
	t.Setenv("CONFIG_DIR", t.TempDir())
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "ganymede")
	t.Setenv("DB_PASS", "ganymede")
	t.Setenv("DB_NAME", "ganymede")
	t.Setenv("TWITCH_CLIENT_ID", "test-client-id")
	t.Setenv("TWITCH_CLIENT_SECRET", "test-client-secret")
	_, err := config.Init()
	require.NoError(t, err)
}

func setThresholds(t *testing.T, enabled bool, warningGB, criticalGB int) {
	t.Helper()
	cfg := config.Get()
	cfg.SetDefaults()
	cfg.Archive.StorageCheckEnabled = enabled
	cfg.Archive.TempDirWarningThresholdGB = warningGB
	cfg.Archive.TempDirCriticalThresholdGB = criticalGB
	require.NoError(t, config.UpdateConfig(cfg))
}

func fakeProbe(free, total int64) probeFunc {
	return func(string) (int64, int64, error) { return free, total, nil }
}

const gb = int64(1024 * 1024 * 1024)

func TestCheckStates(t *testing.T) {
	initTestConfig(t)

	cases := []struct {
		name     string
		warning  int
		critical int
		free     int64
		want     State
	}{
		{"above warning is ok", 50, 5, 60 * gb, StateOK},
		{"at warning is warning", 50, 5, 50 * gb, StateWarning},
		{"below warning is warning", 50, 5, 49 * gb, StateWarning},
		{"at critical is critical", 50, 5, 5 * gb, StateCritical},
		{"below critical is critical", 50, 5, 1 * gb, StateCritical},
		{"zero thresholds never trip", 0, 0, 100 * gb, StateOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setThresholds(t, true, tc.warning, tc.critical)
			svc := newService("/tmp", 0, fakeProbe(tc.free, 100*gb))
			status, err := svc.Check(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tc.want, status.State)
			assert.Equal(t, tc.free, status.FreeBytes)
		})
	}
}

func TestCheckDisabledStillReportsStats(t *testing.T) {
	initTestConfig(t)
	setThresholds(t, false, 50, 5)

	svc := newService("/tmp", 0, fakeProbe(10*gb, 100*gb))
	status, err := svc.Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StateDisabled, status.State)
	assert.Equal(t, int64(10*gb), status.FreeBytes)
}

func TestCheckCriticalClampedToWarning(t *testing.T) {
	initTestConfig(t)
	// Critical above warning is treated as equal to warning.
	setThresholds(t, true, 5, 50)

	svc := newService("/tmp", 0, fakeProbe(30*gb, 100*gb))
	status, err := svc.Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StateOK, status.State)
}

func TestCheckProbeErrorFailsOpen(t *testing.T) {
	initTestConfig(t)
	setThresholds(t, true, 50, 5)

	probeErr := errors.New("statfs: stale mount")
	svc := newService("/tmp", 0, func(string) (int64, int64, error) { return 0, 0, probeErr })
	status, err := svc.Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StateUnknown, status.State)
	require.ErrorIs(t, status.Err, probeErr)

	// The gate lets archives through when free space is unknown.
	assert.NoError(t, Gate(context.Background(), svc))
}

func TestCheckCachesProbeResults(t *testing.T) {
	initTestConfig(t)
	setThresholds(t, true, 50, 5)

	calls := 0
	svc := newService("/tmp", time.Minute, func(string) (int64, int64, error) {
		calls++
		return 60 * gb, 100 * gb, nil
	})
	for i := 0; i < 3; i++ {
		_, err := svc.Check(context.Background())
		require.NoError(t, err)
	}
	assert.Equal(t, 1, calls)
}

func TestCheckAppliesSettingsImmediately(t *testing.T) {
	initTestConfig(t)

	calls := 0
	svc := newService("/tmp", time.Minute, func(string) (int64, int64, error) {
		calls++
		return 60 * gb, 100 * gb, nil
	})
	setThresholds(t, true, 50, 5)

	status, err := svc.Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StateOK, status.State)

	// Raising the warning threshold takes effect without a new probe.
	setThresholds(t, true, 65, 5)
	status, err = svc.Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StateWarning, status.State)
	assert.Equal(t, 1, calls)

	// Disabling the check also applies immediately.
	setThresholds(t, false, 65, 5)
	status, err = svc.Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StateDisabled, status.State)
}

func TestCheckProbeTimeout(t *testing.T) {
	initTestConfig(t)
	setThresholds(t, true, 50, 5)

	blocked := make(chan struct{})
	svc := newService("/tmp", 0, func(string) (int64, int64, error) {
		<-blocked
		return 0, 0, nil
	})
	svc.probeTimeout = 10 * time.Millisecond

	// Never probed: unknown instead of a fake zero reading.
	status, err := svc.Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StateUnknown, status.State)

	// Once readings exist, a stuck probe serves the last known value.
	svc2 := newService("/tmp", time.Hour, fakeProbe(60*gb, 100*gb))
	_, err = svc2.Check(context.Background())
	require.NoError(t, err)
	svc2.cacheTTL = 0
	svc2.probe = func(string) (int64, int64, error) { <-blocked; return 0, 0, nil }
	svc2.probeTimeout = 10 * time.Millisecond

	status, err = svc2.Check(context.Background())
	require.NoError(t, err)
	assert.Equal(t, StateOK, status.State)

	close(blocked)
}

func TestGate(t *testing.T) {
	ctx := context.Background()

	t.Run("nil checker passes", func(t *testing.T) {
		assert.NoError(t, Gate(ctx, nil))
	})

	t.Run("ok passes", func(t *testing.T) {
		assert.NoError(t, Gate(ctx, staticChecker{Status{State: StateOK, FreeBytes: 100 * gb}}))
	})

	t.Run("warning blocks", func(t *testing.T) {
		err := Gate(ctx, staticChecker{Status{State: StateWarning, FreeBytes: 49 * gb}})
		require.ErrorIs(t, err, ErrLowSpace)
		assert.Contains(t, err.Error(), "warning")
	})

	t.Run("critical blocks", func(t *testing.T) {
		err := Gate(ctx, staticChecker{Status{State: StateCritical, FreeBytes: 1 * gb}})
		require.ErrorIs(t, err, ErrLowSpace)
		assert.Contains(t, err.Error(), "critical")
	})

	t.Run("disabled passes", func(t *testing.T) {
		assert.NoError(t, Gate(ctx, staticChecker{Status{State: StateDisabled}}))
	})

	t.Run("unknown fails open", func(t *testing.T) {
		assert.NoError(t, Gate(ctx, staticChecker{Status{State: StateUnknown, Err: errors.New("probe failed")}}))
	})
}

type staticChecker struct{ status Status }

func (c staticChecker) Check(context.Context) (Status, error) { return c.status, nil }
