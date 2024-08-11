package auth_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/internal/user"
	"github.com/zibbp/ganymede/tests"
)

func TestRegister(t *testing.T) {
	ctx := context.Background()
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// test Register
	usr, err := app.AuthService.Register(ctx, user.User{Username: "test_user", Password: "password"})
	assert.NoError(t, err)
	assert.Equal(t, "test_user", usr.Username)
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	rec := httptest.NewRecorder()

	echoCtx := e.NewContext(req, rec)

	_, err = app.AuthService.Register(ctx, user.User{Username: "test_user", Password: "password"})
	assert.NoError(t, err)

	// test Login
	usr, err := app.AuthService.Login(echoCtx, user.User{Username: "admin", Password: "ganymede"})
	assert.NoError(t, err)
	assert.Equal(t, "admin", usr.Username)

	setCookies := rec.Header().Values("Set-Cookie")

	// test cookies are valid jwt tokens
	for _, cookie := range setCookies {
		// example cookie:
		// refresh-token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiNmRmNWFiNDctMzNiOC00ZWFjLWE2M2QtYjlhZjhlMmRiNWRjIiwidXNlcm5hbWUiOiJhZG1pbiIsInJvbGUiOiJhZG1pbiIsImV4cCI6MTcyNTg1NDM0NX0.wltMCYWMwbV6BqU2PM7PLIWIy9uqJmGN5N50oNLpWSY; Path=/; Expires=Mon, 09 Sep 2024 03:59:05 GMT; SameSite=Lax
		split := strings.Split(cookie, ";")
		token := strings.Split(split[0], "=")[1]

		assert.NotEmpty(t, token)

		parts := strings.Split(token, ".")

		assert.Equal(t, 3, len(parts))

		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		assert.NoError(t, err)

		var claims jwt.MapClaims
		err = json.Unmarshal(payload, &claims)
		assert.NoError(t, err)
		assert.Equal(t, usr.ID.String(), claims["user_id"])
		assert.Equal(t, usr.Username, claims["username"])
		assert.Equal(t, string(usr.Role), claims["role"])
	}
}
