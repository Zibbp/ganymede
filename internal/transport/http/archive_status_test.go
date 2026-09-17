package http

import (
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/utils"
	"testing"
)

func TestParseArchiveStatuses(t *testing.T) {
	statuses, err := parseArchiveStatuses("")
	require.NoError(t, err)
	require.Empty(t, statuses)
	statuses, err = parseArchiveStatuses("queued,failed")
	require.NoError(t, err)
	require.Equal(t, []utils.ArchiveStatus{utils.ArchiveQueued, utils.ArchiveFailed}, statuses)
	for _, value := range []string{"processing", "true", "completed,", "running,unknown"} {
		_, err := parseArchiveStatuses(value)
		require.Error(t, err, value)
	}
}
