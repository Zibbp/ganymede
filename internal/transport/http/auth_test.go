package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/auth"
	httpHandler "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/user"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(c echo.Context, userDto user.User) (*ent.User, error) {
	args := m.Called(c, userDto)
	return args.Get(0).(*ent.User), args.Error(1)
}

func (m *MockAuthService) Login(c echo.Context, userDto user.User) (*ent.User, error) {
	args := m.Called(c, userDto)
	return args.Get(0).(*ent.User), args.Error(1)
}

func (m *MockAuthService) Refresh(c echo.Context, refreshToken string) error {
	args := m.Called(c, refreshToken)
	return args.Error(0)
}

func (m *MockAuthService) Me(c *auth.CustomContext) (*ent.User, error) {
	args := m.Called(c)
	return args.Get(0).(*ent.User), args.Error(1)
}

func (m *MockAuthService) ChangePassword(c *auth.CustomContext, passwordDto auth.ChangePassword) error {
	args := m.Called(c, passwordDto)
	return args.Error(0)
}

func (m *MockAuthService) OAuthRedirect(c echo.Context) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *MockAuthService) OAuthCallback(c echo.Context) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *MockAuthService) OAuthTokenRefresh(c echo.Context, refreshToken string) error {
	args := m.Called(c, refreshToken)
	return args.Error(0)
}

func (m *MockAuthService) OAuthLogout(c echo.Context) error {
	args := m.Called(c)
	return args.Error(0)
}

func setupAuthHandler() *httpHandler.Handler {
	e := setupEcho()
	mockAuthService := new(MockAuthService)

	services := httpHandler.Services{
		AuthService: mockAuthService,
	}

	handler := &httpHandler.Handler{
		Server:  e,
		Service: services,
	}

	return handler
}

func TestRegister(t *testing.T) {
	handler := setupAuthHandler()
	e := handler.Server
	mockService := handler.Service.AuthService.(*MockAuthService)

	// test register
	registerBody := httpHandler.RegisterRequest{
		Username: "username",
		Password: "password",
	}

	expectedInput := user.User{
		Username: "username",
		Password: "password",
	}

	expectedOutput := &ent.User{
		Username: "username",
	}

	mockService.On("Register", mock.Anything, expectedInput).Return(expectedOutput, nil)

	b, err := json.Marshal(registerBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.Register(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var response *ent.User
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, response)
	}
}

// TestLogin is a test function for login.
func TestLogin(t *testing.T) {
	handler := setupAuthHandler()
	e := handler.Server
	mockService := handler.Service.AuthService.(*MockAuthService)

	// test login
	loginBody := httpHandler.LoginRequest{
		Username: "username",
		Password: "password",
	}

	expectedInput := user.User{
		Username: "username",
		Password: "password",
	}

	expectedOutput := &ent.User{
		Username: "username",
	}

	mockService.On("Login", mock.Anything, expectedInput).Return(expectedOutput, nil)

	b, err := json.Marshal(loginBody)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(t, handler.Login(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var response *ent.User
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, response)
	}
}
