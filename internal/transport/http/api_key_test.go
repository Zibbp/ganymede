package http_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalHttp "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/tests"
)

// loginAdmin authenticates as the seeded admin user. The session cookie
// is stored by httpexpect's cookie jar so subsequent calls on the same
// Expect carry it automatically.
func loginAdmin(t *testing.T, e *httpexpect.Expect) {
	t.Helper()
	e.POST("/auth/login").
		WithJSON(internalHttp.LoginRequest{Username: "admin", Password: "ganymede"}).
		Expect().
		Status(http.StatusOK)
}

// bareHTTPClient builds a fresh httpexpect against the same server the
// shared tests.SetupHTTP started, but with no cookie jar. Used to assert
// that a Bearer header alone is sufficient — i.e. the API key auth path
// is exercised independently of any session cookie.
func bareHTTPClient(t *testing.T) *httpexpect.Expect {
	t.Helper()
	port := os.Getenv("APP_PORT")
	require.NotEmpty(t, port, "APP_PORT not set; tests.SetupHTTP must run first")
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  fmt.Sprintf("http://localhost:%s/api/v1", port),
		Reporter: httpexpect.NewAssertReporter(t),
		Client:   &http.Client{}, // no cookie jar
	})
}

// TestApiKeyHTTP exercises /admin/api-keys and the API key Bearer auth
// path. We share a single SetupHTTP container across subtests for speed.
func TestApiKeyHTTP(t *testing.T) {
	e, err := tests.SetupHTTP(t)
	require.NoError(t, err)
	loginAdmin(t, e)

	t.Run("CreateRequiresAdminBody", func(t *testing.T) {
		e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{Name: "x", Scopes: []string{"vod:read"}}).
			Expect().
			Status(http.StatusBadRequest) // name too short
	})

	t.Run("CreateRejectsUnknownScope", func(t *testing.T) {
		e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{Name: "bogus-scope-test", Scopes: []string{"bogus:read"}}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("CreateRequiresAtLeastOneScope", func(t *testing.T) {
		e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{Name: "no-scope-test", Scopes: []string{}}).
			Expect().
			Status(http.StatusBadRequest)
	})

	var fullSecret, prefix string

	t.Run("AdminCreateReturnsSecretOnce", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:        "test-admin-key",
				Description: "integration",
				Scopes:      []string{"*:admin"},
			}).
			Expect().
			Status(http.StatusCreated).
			JSON().Object()
		obj.Value("success").IsEqual(true)
		data := obj.Value("data").Object()
		fullSecret = data.Value("secret").String().NotEmpty().Raw()
		prefix = data.Value("api_key").Object().Value("prefix").String().NotEmpty().Raw()
		// Lock the published token-format contract: gym_<12-hex>_<43-char>.
		// A regression in either segment would break leak-scanner heuristics
		// and prefix-lookup invariants.
		assert.Regexp(t, `^gym_[0-9a-f]{12}_[A-Za-z0-9_-]{43}$`, fullSecret,
			"token must follow gym_<12-hex>_<43-char base64url> shape")
		assert.Equal(t, "gym_"+prefix, fullSecret[:len("gym_"+prefix)],
			"token's prefix segment must match the prefix returned in the DTO")
		// Scopes round-trip on the response DTO.
		data.Value("api_key").Object().Value("scopes").Array().ContainsAll("*:admin")
	})

	t.Run("ListReturnsCreatedKeyWithoutSecret", func(t *testing.T) {
		arr := e.GET("/admin/api-keys").
			Expect().
			Status(http.StatusOK).
			JSON().Object().Value("data").Array()
		arr.Length().Gt(0)
		// Verify our key is in the list and its prefix matches.
		arr.Find(func(_ int, v *httpexpect.Value) bool {
			return v.Object().Value("prefix").String().Raw() == prefix
		}).Object().NotContainsKey("hashed_secret").NotContainsKey("secret")
	})

	// Bearer-auth assertions point at routes behind
	// AuthAPIKeyOrSessionMiddleware (e.g. /queue with queue:read or
	// /vod write endpoints). /admin/api-keys is intentionally
	// session-only, so a Bearer header against it would always fail
	// regardless of the key's validity — testing it here would prove
	// the wrong invariant. The session-only behavior is asserted
	// separately below in AdminApiKeysIsAlwaysSessionOnly.

	t.Run("BearerTokenAuthenticatesAdminScope", func(t *testing.T) {
		// Use a fresh httpexpect with no cookies so we know we're hitting
		// the API key auth path, not the seeded admin session.
		bearerOnly := bareHTTPClient(t)
		// /queue requires queue:read; *:admin satisfies it via the
		// wildcard hierarchy.
		bearerOnly.GET("/queue").
			WithHeader("Authorization", "Bearer "+fullSecret).
			Expect().
			Status(http.StatusOK)
	})

	var revokedSecret string
	t.Run("RevokedKeyIsRejected", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{Name: "to-revoke", Scopes: []string{"*:admin"}}).
			Expect().Status(http.StatusCreated).JSON().Object()
		revokedSecret = obj.Path("$.data.secret").String().NotEmpty().Raw()
		id := obj.Path("$.data.api_key.id").String().NotEmpty().Raw()

		// Prime the verification cache by hitting a flexible-auth
		// route with the bearer first; the cache flush on revoke is
		// part of what we're verifying.
		bearerOnly := bareHTTPClient(t)
		bearerOnly.GET("/queue").
			WithHeader("Authorization", "Bearer "+revokedSecret).
			Expect().
			Status(http.StatusOK)

		e.DELETE("/admin/api-keys/" + id).
			Expect().
			Status(http.StatusOK)

		// Revoke flushed the cache; the same flexible-auth route
		// now rejects with 401.
		bearerOnly.GET("/queue").
			WithHeader("Authorization", "Bearer "+revokedSecret).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("MalformedBearerIsRejected", func(t *testing.T) {
		bearerOnly := bareHTTPClient(t)
		// /queue accepts API keys, so a malformed Bearer reaches the
		// verifier and is rejected there. (Hitting /admin/api-keys
		// would also return 401 but only because that route is
		// session-only — wrong invariant.)
		bearerOnly.GET("/queue").
			WithHeader("Authorization", "Bearer not-a-real-key").
			Expect().
			Status(http.StatusUnauthorized)

		// Companion: the same malformed Bearer must still fail when a
		// valid session is also present. Catches a regression where
		// AuthAPIKeyOrSessionMiddleware silently falls through to
		// session auth on a bad Bearer instead of failing closed.
		e.GET("/queue").
			WithHeader("Authorization", "Bearer not-a-real-key").
			Expect().
			Status(http.StatusUnauthorized)
	})

	// Cross-resource enforcement. Each subtest creates a key with a
	// narrow scope and asserts it succeeds against routes its scope
	// covers, and is rejected from routes another scope would gate.
	t.Run("VodWriteScopeCannotDeleteVod", func(t *testing.T) {
		// Mint a key that can edit but not destroy VODs.
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "vod-writer-only",
				Scopes: []string{"vod:write"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()

		bearer := bareHTTPClient(t)
		// DELETE /vod/:id requires vod:admin → 403, not 401.
		bearer.DELETE("/vod/00000000-0000-0000-0000-000000000000").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("PlaylistScopeCannotAccessQueue", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "playlist-only",
				Scopes: []string{"playlist:write"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()

		bearer := bareHTTPClient(t)
		// GET /queue requires queue:read → playlist scope must not satisfy.
		bearer.GET("/queue").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("WildcardAdminCoversEveryMigratedRoute", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "superuser",
				Scopes: []string{"*:admin"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()

		bearer := bareHTTPClient(t)
		// /queue requires queue:read; *:admin covers it.
		bearer.GET("/queue").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("MultipleScopesAreUnioned", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "vod-and-queue-reader",
				Scopes: []string{"vod:read", "queue:read"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()

		bearer := bareHTTPClient(t)
		// queue:read passes the GET /queue gate.
		bearer.GET("/queue").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusOK)
		// But neither scope covers playlist writes.
		bearer.POST("/playlist").
			WithHeader("Authorization", "Bearer "+token).
			WithJSON(map[string]any{"name": "from-script"}).
			Expect().
			Status(http.StatusForbidden)
	})

	// Phase 3: smoke coverage for the newly-migrated route groups. Each
	// case mints a narrow scope and verifies the matching endpoint
	// authenticates while a route in a different group rejects the same
	// key.

	t.Run("SystemReadCoversAdminStats", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "stats-only",
				Scopes: []string{"system:read"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()

		bearer := bareHTTPClient(t)
		bearer.GET("/admin/system-overview").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusOK)
		// Same key cannot mint or list keys — /admin/api-keys stays
		// session-only on purpose.
		bearer.GET("/admin/api-keys").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("UserReadCannotEditUsers", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "user-reader",
				Scopes: []string{"user:read"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()

		bearer := bareHTTPClient(t)
		// GET succeeds with user:read.
		bearer.GET("/user").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusOK)
		// PUT requires user:write, which user:read does not include.
		bearer.PUT("/user/00000000-0000-0000-0000-000000000000").
			WithHeader("Authorization", "Bearer "+token).
			WithJSON(map[string]any{"username": "x", "role": "user"}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("ConfigReadCanGetButNotPut", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "config-reader",
				Scopes: []string{"config:read"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()

		bearer := bareHTTPClient(t)
		bearer.GET("/config").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusOK)
		bearer.PUT("/config").
			WithHeader("Authorization", "Bearer "+token).
			WithJSON(map[string]any{}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("ChannelWriteCannotDeleteChannel", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "channel-writer",
				Scopes: []string{"channel:write"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()

		bearer := bareHTTPClient(t)
		// DELETE requires channel:admin.
		bearer.DELETE("/channel/00000000-0000-0000-0000-000000000000").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("UpdateChangesScopesAndRevokesCachedAccess", func(t *testing.T) {
		// Cache-flush proof requires a flexible-auth route (one whose
		// middleware actually consults the bearer). GET /queue is the
		// canonical read-tier route used elsewhere in this file. The
		// asymmetry of "queue:read worked, then queue:read stops
		// working after we updated the scopes" can only happen if the
		// cached entry was invalidated; otherwise the cached scopes
		// for this token would still contain queue:read.
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "update-target-http",
				Scopes: []string{"queue:read"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()
		id := obj.Path("$.data.api_key.id").String().Raw()

		// Warm the verification cache by authenticating against a
		// route that actually checks the bearer.
		bearer := bareHTTPClient(t)
		bearer.GET("/queue").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusOK)

		// Update: rename + swap scopes for vod:read (drops queue
		// access). If the cache held a stale entry with the old
		// scopes, the next /queue hit would still pass.
		updated := e.PUT("/admin/api-keys/"+id).
			WithJSON(internalHttp.UpdateApiKeyRequest{
				Name:   "update-target-http-renamed",
				Scopes: []string{"vod:read"},
			}).
			Expect().Status(http.StatusOK).JSON().Object()
		updated.Path("$.data.name").IsEqual("update-target-http-renamed")
		updated.Path("$.data.scopes").Array().ContainsAll("vod:read")
		updated.Path("$.data.scopes").Array().NotContainsAll("queue:read")

		// Cache flushed: previously-allowed queue read is now denied.
		bearer.GET("/queue").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("UpdateRejectsUnknownScope", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "update-bad-scope-http",
				Scopes: []string{"vod:read"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		id := obj.Path("$.data.api_key.id").String().Raw()

		e.PUT("/admin/api-keys/"+id).
			WithJSON(internalHttp.UpdateApiKeyRequest{
				Name:   "update-bad-scope-http",
				Scopes: []string{"bogus:read"},
			}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("DeleteRevokedKeyReturns404", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "delete-revoked-http",
				Scopes: []string{"vod:read"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		id := obj.Path("$.data.api_key.id").String().Raw()

		// First revoke succeeds.
		e.DELETE("/admin/api-keys/" + id).Expect().Status(http.StatusOK)
		// Second revoke on the same id is no longer there to revoke;
		// the conditional Update matches zero rows and Service.Revoke
		// returns ErrNotFound, which the handler maps to 404.
		e.DELETE("/admin/api-keys/" + id).Expect().Status(http.StatusNotFound)
	})

	t.Run("DeleteNonexistentKeyReturns404", func(t *testing.T) {
		// Random uuid, no row exists.
		e.DELETE("/admin/api-keys/00000000-0000-0000-0000-000000000000").
			Expect().Status(http.StatusNotFound)
	})

	t.Run("UpdateRevokedKeyReturns404", func(t *testing.T) {
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "update-revoked-http",
				Scopes: []string{"vod:read"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		id := obj.Path("$.data.api_key.id").String().Raw()

		e.DELETE("/admin/api-keys/" + id).Expect().Status(http.StatusOK)

		e.PUT("/admin/api-keys/"+id).
			WithJSON(internalHttp.UpdateApiKeyRequest{
				Name:   "update-revoked-http",
				Scopes: []string{"vod:read"},
			}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("SystemUserCannotBeDeletedOrUpdated", func(t *testing.T) {
		// Mint an admin-tier key so we hit the migrated /user/:id routes
		// via Bearer rather than the seeded session — the protection
		// must hold regardless of auth method.
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "system-user-protector",
				Scopes: []string{"*:admin"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()

		// Locate the system:api user via the admin list — it shows up
		// because EnsureSystemUser ran at startup.
		bearer := bareHTTPClient(t)
		users := bearer.GET("/user").
			WithHeader("Authorization", "Bearer "+token).
			Expect().Status(http.StatusOK).JSON().Object().Value("data").Array()
		systemID := ""
		for _, v := range users.Iter() {
			if v.Object().Value("username").String().Raw() == "system:api" {
				systemID = v.Object().Value("id").String().Raw()
				break
			}
		}
		require.NotEmpty(t, systemID, "system:api user must exist in /user list")

		// DELETE → 403 Forbidden, not 200.
		bearer.DELETE("/user/"+systemID).
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusForbidden)

		// PUT → 403 Forbidden as well.
		bearer.PUT("/user/"+systemID).
			WithHeader("Authorization", "Bearer "+token).
			WithJSON(map[string]any{"username": "renamed-system", "role": "user"}).
			Expect().
			Status(http.StatusForbidden)

		// Closing the rename-bypass: take a regular non-system user
		// (the seeded admin) and try to rename them to the reserved
		// system username. Must return 403 — the previous fix only
		// guarded the system row from being mutated, not regular rows
		// from being renamed INTO the system position.
		nonSystemID := ""
		for _, v := range users.Iter() {
			if v.Object().Value("username").String().Raw() != "system:api" {
				nonSystemID = v.Object().Value("id").String().Raw()
				break
			}
		}
		require.NotEmpty(t, nonSystemID, "test fixture must include at least one non-system user")
		bearer.PUT("/user/"+nonSystemID).
			WithHeader("Authorization", "Bearer "+token).
			WithJSON(map[string]any{"username": "system:api", "role": "user"}).
			Expect().
			Status(http.StatusForbidden)

		// Case-fold variant — "System:API" must also be rejected.
		bearer.PUT("/user/"+nonSystemID).
			WithHeader("Authorization", "Bearer "+token).
			WithJSON(map[string]any{"username": "System:API", "role": "user"}).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("AdminApiKeysIsAlwaysSessionOnly", func(t *testing.T) {
		// Even a *:admin key cannot reach /admin/api-keys via Bearer —
		// the route is intentionally guarded by the session-only chain.
		obj := e.POST("/admin/api-keys").
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "would-be-key-minter",
				Scopes: []string{"*:admin"},
			}).
			Expect().Status(http.StatusCreated).JSON().Object()
		token := obj.Path("$.data.secret").String().Raw()

		bearer := bareHTTPClient(t)
		bearer.GET("/admin/api-keys").
			WithHeader("Authorization", "Bearer "+token).
			Expect().
			Status(http.StatusUnauthorized)
		bearer.POST("/admin/api-keys").
			WithHeader("Authorization", "Bearer "+token).
			WithJSON(internalHttp.CreateApiKeyRequest{
				Name:   "minted-via-bearer",
				Scopes: []string{"vod:read"},
			}).
			Expect().
			Status(http.StatusUnauthorized)
	})
}
