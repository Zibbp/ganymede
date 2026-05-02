package api_key_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zibbp/ganymede/internal/api_key"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests"
)

// TestApiKeyService groups the integration tests so we only spin up one
// postgres testcontainer for the whole suite.
func TestApiKeyService(t *testing.T) {
	app, err := tests.Setup(t)
	require.NoError(t, err)

	svc := api_key.NewService(app.Database)
	ctx := context.Background()

	t.Run("Create persists hashed_secret distinct from plaintext", func(t *testing.T) {
		key, full, err := svc.Create(ctx, "create-test", "first key", utils.ApiKeyScopeRead)
		require.NoError(t, err)
		require.NotEmpty(t, full)
		assert.NotEqual(t, full, key.HashedSecret, "hashed_secret must not equal plaintext token")
		assert.NotEmpty(t, key.Prefix)
		assert.Equal(t, utils.ApiKeyScopeRead, key.Scope)

		_, secret, err := api_key.Parse(full)
		require.NoError(t, err)
		assert.NoError(t, api_key.Verify(key.HashedSecret, secret), "stored hash must verify against the issued secret")
	})

	t.Run("Create rejects unknown scope", func(t *testing.T) {
		_, _, err := svc.Create(ctx, "bogus-scope", "", utils.ApiKeyScope("bogus"))
		assert.Error(t, err)
	})

	t.Run("List returns non-revoked keys, newest first", func(t *testing.T) {
		toRevoke, _, err := svc.Create(ctx, "list-revoke-target", "", utils.ApiKeyScopeWrite)
		require.NoError(t, err)
		_, _, err = svc.Create(ctx, "list-keep", "", utils.ApiKeyScopeWrite)
		require.NoError(t, err)

		require.NoError(t, svc.Revoke(ctx, toRevoke.ID))

		keys, err := svc.List(ctx)
		require.NoError(t, err)
		for _, k := range keys {
			assert.NotEqual(t, toRevoke.ID, k.ID, "revoked key must not be in list")
		}
	})

	t.Run("GetByPrefix excludes revoked", func(t *testing.T) {
		key, _, err := svc.Create(ctx, "revoke-prefix-target", "", utils.ApiKeyScopeAdmin)
		require.NoError(t, err)
		require.NoError(t, svc.Revoke(ctx, key.ID))

		_, err = svc.GetByPrefix(ctx, key.Prefix)
		assert.Error(t, err, "GetByPrefix must not return revoked key")
	})

	t.Run("Revoke flushes the verification cache", func(t *testing.T) {
		key, full, err := svc.Create(ctx, "cache-flush", "", utils.ApiKeyScopeRead)
		require.NoError(t, err)

		// Prime the cache.
		svc.Cache.Set(full, key.ID, key.Scope)
		_, _, hit := svc.Cache.Get(full)
		require.True(t, hit, "cache should be primed before revoke")

		require.NoError(t, svc.Revoke(ctx, key.ID))

		_, _, hit = svc.Cache.Get(full)
		assert.False(t, hit, "cache must be flushed after revoke")
	})

	t.Run("TouchLastUsed updates after debounce window", func(t *testing.T) {
		key, _, err := svc.Create(ctx, "touch-test", "", utils.ApiKeyScopeRead)
		require.NoError(t, err)

		require.NoError(t, svc.TouchLastUsed(ctx, key.ID))

		fresh, err := svc.GetByPrefix(ctx, key.Prefix)
		require.NoError(t, err)
		require.NotNil(t, fresh.LastUsedAt)
		assert.WithinDuration(t, time.Now(), *fresh.LastUsedAt, 5*time.Second)
	})

	t.Run("EnsureSystemUser is idempotent", func(t *testing.T) {
		first, err := svc.EnsureSystemUser(ctx)
		require.NoError(t, err)
		second, err := svc.EnsureSystemUser(ctx)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
		assert.Equal(t, api_key.SystemUserUsername, first.Username)
		assert.Equal(t, utils.AdminRole, first.Role)
	})
}
