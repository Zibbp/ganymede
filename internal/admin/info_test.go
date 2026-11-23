package admin

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionOrUnavailable(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "1.2.3", versionOrUnavailable(func() (string, error) {
		return "1.2.3", nil
	}))
	assert.Equal(t, "unavailable", versionOrUnavailable(func() (string, error) {
		return "", errors.New("binary not found")
	}))
}
