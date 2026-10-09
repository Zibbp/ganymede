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

// HEVC recordings do not play on Apple devices without the hvc1 tag, so an
// installation that never touches the setting gets the fix.
func TestSetDefaultsEnablesHevcRetagging(t *testing.T) {
	t.Parallel()

	var cfg Config
	cfg.SetDefaults()

	require.True(t, cfg.Archive.TagHevcAsHvc1)
}
