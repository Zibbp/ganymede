package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/internal/utils"
)

func TestRoleFromGroups(t *testing.T) {
	t.Run("returns role when valid ganymede group exists", func(t *testing.T) {
		role, ok := roleFromGroups([]string{"users", "ganymede-admin"})
		assert.True(t, ok)
		assert.Equal(t, utils.AdminRole, role)
	})

	t.Run("returns role when valid ganymede underscore group exists", func(t *testing.T) {
		role, ok := roleFromGroups([]string{"users", "ganymede_editor"})
		assert.True(t, ok)
		assert.Equal(t, utils.EditorRole, role)
	})

	t.Run("skips invalid role groups", func(t *testing.T) {
		role, ok := roleFromGroups([]string{"ganymede-invalid"})
		assert.False(t, ok)
		assert.Empty(t, role)
	})

	t.Run("returns false when no ganymede role group exists", func(t *testing.T) {
		role, ok := roleFromGroups([]string{"users", "staff"})
		assert.False(t, ok)
		assert.Empty(t, role)
	})
}
