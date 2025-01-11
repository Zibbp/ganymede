package http_test

import (
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/ent"
	internalHttp "github.com/zibbp/ganymede/internal/transport/http"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/tests"
)

type HttpTestService struct {
	E *httpexpect.Expect
}

// AuthTest runs all the auth tests
func TestAuth(t *testing.T) {
	e, err := tests.SetupHTTP(t)
	assert.NoError(t, err)

	s := &HttpTestService{E: e}

	t.Run("AuthLoginInvalid", s.AuthLoginInvalid)
	t.Run("AuthLogin", s.AuthLogin)
	t.Run("AuthGetUser", s.AuthGetUser)
	t.Run("AuthChangePasswordInvalid", s.AuthChangePasswordInvalid)
	t.Run("AuthChangePasswordNoMatch", s.AuthChangePasswordNoMatch)
	t.Run("AuthChangePassword", s.AuthChangePassword)
	t.Run("AuthChangePasswordBack", s.AuthChangePasswordBack)
	t.Run("AuthRegisterInvalid", s.AuthRegisterInvalid)
	t.Run("AuthRegister", s.AuthRegister)
	t.Run("AuthLogout", s.AuthLogout)
	t.Run("AuthLogin", s.AuthLogin)
}

// AuthLoginInvalid tests the /auth/login endpoint with invalid credentials
func (s *HttpTestService) AuthLoginInvalid(t *testing.T) {
	obj := s.E.POST("/auth/login").WithJSON(internalHttp.LoginRequest{Username: "foo", Password: "foobar123"}).Expect().Status(http.StatusBadRequest).JSON().Object()

	obj.Value("success").IsEqual(false)
}

// AuthLogin tests the /auth/login endpoint with valid credentials
func (s *HttpTestService) AuthLogin(t *testing.T) {
	var user *ent.User

	obj := s.E.POST("/auth/login").WithJSON(internalHttp.LoginRequest{Username: "admin", Password: "ganymede"}).Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("success").IsEqual(true)
	obj.Value("data").Decode(&user)

	assert.Equal(t, "admin", user.Username)
	assert.Equal(t, utils.AdminRole, user.Role)
	assert.NotNil(t, user.ID)
}

// AuthGetUser tests the /auth/me endpoint
func (s *HttpTestService) AuthGetUser(t *testing.T) {
	var user *ent.User

	obj := s.E.GET("/auth/me").Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("success").IsEqual(true)
	obj.Value("data").Decode(&user)

	assert.Equal(t, "admin", user.Username)
	assert.Equal(t, utils.AdminRole, user.Role)
	assert.NotNil(t, user.ID)
}

// AuthChangePasswordInvalid tests the /auth/change-password endpoint with invalid credentials
func (s *HttpTestService) AuthChangePasswordInvalid(t *testing.T) {
	obj := s.E.POST("/auth/change-password").WithJSON(internalHttp.ChangePasswordRequest{OldPassword: "ganymede1", NewPassword: "ganymede1", ConfirmNewPassword: "ganymede1"}).Expect().Status(http.StatusInternalServerError).JSON().Object()

	obj.Value("success").IsEqual(false)
}

// AuthChangePasswordNoMatch tests the /auth/change-password endpoint with non-matching new passwords
func (s *HttpTestService) AuthChangePasswordNoMatch(t *testing.T) {
	obj := s.E.POST("/auth/change-password").WithJSON(internalHttp.ChangePasswordRequest{OldPassword: "ganymede", NewPassword: "ganymede1", ConfirmNewPassword: "ganymede2"}).Expect().Status(http.StatusBadRequest).JSON().Object()

	obj.Value("success").IsEqual(false)
}

// AuthChangePassword tests the /auth/change-password endpoint with valid credentials
func (s *HttpTestService) AuthChangePassword(t *testing.T) {
	obj := s.E.POST("/auth/change-password").WithJSON(internalHttp.ChangePasswordRequest{OldPassword: "ganymede", NewPassword: "ganymede1", ConfirmNewPassword: "ganymede1"}).Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("success").IsEqual(true)
}

// AuthChangePassword tests the /auth/change-password endpoint changing the password back to the original
func (s *HttpTestService) AuthChangePasswordBack(t *testing.T) {
	obj := s.E.POST("/auth/change-password").WithJSON(internalHttp.ChangePasswordRequest{OldPassword: "ganymede1", NewPassword: "ganymede", ConfirmNewPassword: "ganymede"}).Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("success").IsEqual(true)
}

// AuthRegisterInvalid tests the /auth/register endpoint with invalid credentials
func (s *HttpTestService) AuthRegisterInvalid(t *testing.T) {
	obj := s.E.POST("/auth/register").WithJSON(internalHttp.RegisterRequest{Username: "t", Password: "short"}).Expect().Status(http.StatusBadRequest).JSON().Object()

	obj.Value("success").IsEqual(false)
}

// AuthRegister tests the /auth/register endpoint with valid credentials
func (s *HttpTestService) AuthRegister(t *testing.T) {
	var user *ent.User

	obj := s.E.POST("/auth/register").WithJSON(internalHttp.RegisterRequest{Username: "testing", Password: "testing123"}).Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("success").IsEqual(true)
	obj.Value("data").Decode(&user)

	assert.Equal(t, "testing", user.Username)
	assert.Equal(t, utils.UserRole, user.Role)
	assert.NotNil(t, user.ID)
}

// AuthLogout tests the /auth/logout endpoint
func (s *HttpTestService) AuthLogout(t *testing.T) {

	obj := s.E.POST("/auth/logout").Expect().Status(http.StatusOK).JSON().Object()

	obj.Value("success").IsEqual(true)
}
