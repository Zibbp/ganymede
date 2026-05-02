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
		assert.NotEqual(t, fullSecret, prefix)
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

	t.Run("BearerTokenAuthenticatesAdminScope", func(t *testing.T) {
		// Use a fresh httpexpect with no cookies so we know we're hitting
		// the API key auth path, not the seeded admin session.
		bearerOnly := bareHTTPClient(t)
		bearerOnly.GET("/admin/api-keys").
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

		e.DELETE("/admin/api-keys/" + id).
			Expect().
			Status(http.StatusOK)

		bearerOnly := bareHTTPClient(t)
		bearerOnly.GET("/admin/api-keys").
			WithHeader("Authorization", "Bearer "+revokedSecret).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("MalformedBearerIsRejected", func(t *testing.T) {
		bearerOnly := bareHTTPClient(t)
		bearerOnly.GET("/admin/api-keys").
			WithHeader("Authorization", "Bearer not-a-real-key").
			Expect().
			Status(http.StatusUnauthorized)
	})
}
