package api_key_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
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
		key, full, err := svc.Create(ctx, "create-test", "first key", []utils.ApiKeyScope{utils.ApiKeyScopeVodRead}, uuid.Nil)
		require.NoError(t, err)
		require.NotEmpty(t, full)
		assert.NotEqual(t, full, key.HashedSecret, "hashed_secret must not equal plaintext token")
		assert.NotEmpty(t, key.Prefix)
		assert.Equal(t, []string{string(utils.ApiKeyScopeVodRead)}, key.Scopes)

		_, secret, err := api_key.Parse(full)
		require.NoError(t, err)
		assert.NoError(t, api_key.Verify(key.HashedSecret, secret), "stored hash must verify against the issued secret")
	})

	t.Run("Create rejects unknown scope", func(t *testing.T) {
		_, _, err := svc.Create(ctx, "bogus-scope", "", []utils.ApiKeyScope{utils.ApiKeyScope("bogus:read")}, uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("Create rejects empty scope list", func(t *testing.T) {
		_, _, err := svc.Create(ctx, "no-scope", "", nil, uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("Create records the creator id for audit attribution", func(t *testing.T) {
		// The edge.To(User) generates a real foreign key, so we can't
		// pass a random uuid — use the seeded admin user's id (from
		// database.seedDatabase).
		admins, err := app.Database.Client.User.Query().All(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, admins, "seedDatabase should have created at least one user")
		creator := admins[0].ID

		key, _, err := svc.Create(ctx, "with-creator", "", []utils.ApiKeyScope{utils.ApiKeyScopeVodRead}, creator)
		require.NoError(t, err)
		require.NotNil(t, key.CreatedByID)
		assert.Equal(t, creator, *key.CreatedByID)
	})

	t.Run("Create de-duplicates repeated scopes", func(t *testing.T) {
		key, _, err := svc.Create(ctx, "dedup", "", []utils.ApiKeyScope{
			utils.ApiKeyScopeVodWrite, utils.ApiKeyScopeVodWrite, utils.ApiKeyScopePlaylistRead,
		}, uuid.Nil)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{
			string(utils.ApiKeyScopeVodWrite), string(utils.ApiKeyScopePlaylistRead),
		}, key.Scopes)
	})

	t.Run("List returns non-revoked keys, newest first", func(t *testing.T) {
		toRevoke, _, err := svc.Create(ctx, "list-revoke-target", "", []utils.ApiKeyScope{utils.ApiKeyScopeVodWrite}, uuid.Nil)
		require.NoError(t, err)
		_, _, err = svc.Create(ctx, "list-keep", "", []utils.ApiKeyScope{utils.ApiKeyScopeVodWrite}, uuid.Nil)
		require.NoError(t, err)

		require.NoError(t, svc.Revoke(ctx, toRevoke.ID))

		keys, err := svc.List(ctx, false)
		require.NoError(t, err)
		for _, k := range keys {
			assert.NotEqual(t, toRevoke.ID, k.ID, "revoked key must not be in default list")
		}

		// includeRevoked=true brings the revoked row back, with
		// revoked_at populated.
		all, err := svc.List(ctx, true)
		require.NoError(t, err)
		var foundRevoked bool
		for _, k := range all {
			if k.ID == toRevoke.ID {
				foundRevoked = true
				require.NotNil(t, k.RevokedAt, "revoked row must surface RevokedAt when included")
			}
		}
		assert.True(t, foundRevoked, "List(includeRevoked=true) must return the revoked row")
	})

	t.Run("GetByPrefix excludes revoked", func(t *testing.T) {
		key, _, err := svc.Create(ctx, "revoke-prefix-target", "", []utils.ApiKeyScope{utils.ApiKeyScopeAllAdmin}, uuid.Nil)
		require.NoError(t, err)
		require.NoError(t, svc.Revoke(ctx, key.ID))

		_, err = svc.GetByPrefix(ctx, key.Prefix)
		assert.Error(t, err, "GetByPrefix must not return revoked key")
	})

	t.Run("Revoke flushes the verification cache", func(t *testing.T) {
		key, full, err := svc.Create(ctx, "cache-flush", "", []utils.ApiKeyScope{utils.ApiKeyScopeVodRead}, uuid.Nil)
		require.NoError(t, err)

		// Prime the cache.
		svc.Cache.Set(full, key.ID, utils.ApiKeyScopesFromStrings(key.Scopes))
		_, _, hit := svc.Cache.Get(full)
		require.True(t, hit, "cache should be primed before revoke")

		require.NoError(t, svc.Revoke(ctx, key.ID))

		_, _, hit = svc.Cache.Get(full)
		assert.False(t, hit, "cache must be flushed after revoke")
	})

	t.Run("TouchLastUsed updates after debounce window", func(t *testing.T) {
		key, _, err := svc.Create(ctx, "touch-test", "", []utils.ApiKeyScope{utils.ApiKeyScopeVodRead}, uuid.Nil)
		require.NoError(t, err)

		require.NoError(t, svc.TouchLastUsed(ctx, key.ID))

		fresh, err := svc.GetByPrefix(ctx, key.Prefix)
		require.NoError(t, err)
		require.NotNil(t, fresh.LastUsedAt)
		assert.WithinDuration(t, time.Now(), *fresh.LastUsedAt, 5*time.Second)
	})

	t.Run("TouchLastUsed silently no-ops on a revoked key", func(t *testing.T) {
		key, _, err := svc.Create(ctx, "touch-after-revoke", "", []utils.ApiKeyScope{utils.ApiKeyScopeVodRead}, uuid.Nil)
		require.NoError(t, err)
		require.NoError(t, svc.Revoke(ctx, key.ID))

		// Conditional UPDATE matches no rows (WHERE includes
		// RevokedAtIsNil), so the call returns nil with zero rows
		// affected. Verify last_used_at stays nil by reading the row
		// directly through the ent client (GetByPrefix already filters
		// revoked).
		require.NoError(t, svc.TouchLastUsed(ctx, key.ID))

		row, err := app.Database.Client.ApiKey.Get(ctx, key.ID)
		require.NoError(t, err)
		assert.Nil(t, row.LastUsedAt, "last_used_at must remain nil on a revoked key")
	})

	t.Run("Update changes editable fields and flushes cache", func(t *testing.T) {
		key, full, err := svc.Create(ctx, "update-target", "before", []utils.ApiKeyScope{utils.ApiKeyScopeVodRead}, uuid.Nil)
		require.NoError(t, err)

		// Prime cache with the original scopes so we can assert flush.
		svc.Cache.Set(full, key.ID, utils.ApiKeyScopesFromStrings(key.Scopes))
		_, _, hit := svc.Cache.Get(full)
		require.True(t, hit, "cache primed before update")

		updated, err := svc.Update(ctx, key.ID, "update-target-renamed", "after", []utils.ApiKeyScope{
			utils.ApiKeyScopeVodWrite, utils.ApiKeyScopePlaylistRead,
		})
		require.NoError(t, err)
		assert.Equal(t, "update-target-renamed", updated.Name)
		assert.Equal(t, "after", updated.Description)
		assert.ElementsMatch(t, []string{
			string(utils.ApiKeyScopeVodWrite), string(utils.ApiKeyScopePlaylistRead),
		}, updated.Scopes)

		_, _, hit = svc.Cache.Get(full)
		assert.False(t, hit, "cache must be flushed after update so new scopes take effect")
	})

	t.Run("Update rejects unknown scopes", func(t *testing.T) {
		key, _, err := svc.Create(ctx, "update-bad-scope", "", []utils.ApiKeyScope{utils.ApiKeyScopeVodRead}, uuid.Nil)
		require.NoError(t, err)
		_, err = svc.Update(ctx, key.ID, "x", "", []utils.ApiKeyScope{utils.ApiKeyScope("bogus:read")})
		assert.Error(t, err)
	})

	t.Run("Update on revoked key returns not-found", func(t *testing.T) {
		key, _, err := svc.Create(ctx, "update-revoked", "", []utils.ApiKeyScope{utils.ApiKeyScopeVodRead}, uuid.Nil)
		require.NoError(t, err)
		require.NoError(t, svc.Revoke(ctx, key.ID))
		_, err = svc.Update(ctx, key.ID, "x", "", []utils.ApiKeyScope{utils.ApiKeyScopeVodRead})
		assert.Error(t, err)
	})

	t.Run("EnsureSystemUser is idempotent and uses SystemRole", func(t *testing.T) {
		first, err := svc.EnsureSystemUser(ctx)
		require.NoError(t, err)
		second, err := svc.EnsureSystemUser(ctx)
		require.NoError(t, err)
		assert.Equal(t, first.ID, second.ID)
		assert.Equal(t, api_key.SystemUserUsername, first.Username)
		assert.Equal(t, utils.SystemRole, first.Role,
			"system user must have SystemRole so legacy AuthUserRoleMiddleware fails closed for API keys")
	})
}
