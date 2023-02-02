package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/enttest"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/database"
	httpTransport "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/utils"
)

var (
	// The following are used for testing.
	testUserJson = `{
		"username": "test",
		"password": "test1234"
		}`
)

// * TestRegister tests the Register function.
// Test registers a new user.
func TestRegister(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	viper.Set("registration_enabled", true)

	h := &httpTransport.Handler{
		Server: echo.New(),
		Service: httpTransport.Services{
			AuthService: auth.NewService(&database.Database{Client: client}),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(testUserJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)

	if assert.NoError(t, h.Register(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test", response["username"])
	}
}

// * TestLogin tests the Login function.
// Test logs in a user.
func TestLogin(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	viper.Set("registration_enabled", true)
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("JWT_REFRESH_SECRET", "test")

	h := &httpTransport.Handler{
		Server: echo.New(),
		Service: httpTransport.Services{
			AuthService: auth.NewService(&database.Database{Client: client}),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Register a new user
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(testUserJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)
	err := h.Register(c)
	assert.NoError(t, err)

	// Login the user
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(testUserJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = h.Server.NewContext(req, rec)

	if assert.NoError(t, h.Login(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check response body
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "test", response["username"])
	}
}

// * TestRefresh tests the Refresh function.
// Test refreshes a user's access token.
func TestRefresh(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}

	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	viper.Set("registration_enabled", true)
	os.Setenv("JWT_SECRET", "test")
	os.Setenv("JWT_REFRESH", "test")

	h := &httpTransport.Handler{
		Server: echo.New(),
		Service: httpTransport.Services{
			AuthService: auth.NewService(&database.Database{Client: client}),
		},
	}

	h.Server.Validator = &utils.CustomValidator{Validator: validator.New()}

	// Register a new user
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(testUserJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := h.Server.NewContext(req, rec)
	err := h.Register(c)
	assert.NoError(t, err)

	// Login the user
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(testUserJson))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = h.Server.NewContext(req, rec)
	err = h.Login(c)
	assert.NoError(t, err)

	// Refresh the user's access token

	// Get the refresh token from the response cookie
	cookies := rec.Result().Cookies()
	var refreshToken string
	for _, cookie := range cookies {
		if cookie.Name == "refresh-token" {
			refreshToken = cookie.Value
		}
	}

	// Create a new request with the refresh token
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  "refresh-token",
		Value: refreshToken,
	})
	rec = httptest.NewRecorder()
	c = h.Server.NewContext(req, rec)

	if assert.NoError(t, h.Refresh(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}
