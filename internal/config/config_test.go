package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetDefaultsEnablesNFOGeneration(t *testing.T) {
	t.Parallel()

	var cfg Config
	cfg.SetDefaults()

	require.True(t, cfg.Archive.GenerateNFOFiles)
}

func TestSetDefaultsStorageCheck(t *testing.T) {
	t.Parallel()

	var cfg Config
	cfg.SetDefaults()

	require.True(t, cfg.Archive.StorageCheckEnabled)
	require.Equal(t, 50, cfg.Archive.TempDirWarningThresholdGB)
	require.Equal(t, 5, cfg.Archive.TempDirCriticalThresholdGB)
}
