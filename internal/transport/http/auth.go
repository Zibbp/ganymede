package http

import (
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/user"
	"net/http"
	"os"
)

type AuthService interface {
	Register(c echo.Context, userDto user.User) (*ent.User, error)
	Login(c echo.Context, userDto user.User) (*ent.User, error)
	Refresh(c echo.Context, refreshToken string) error
	Me(c *auth.CustomContext) (*ent.User, error)
	ChangePassword(c *auth.CustomContext, passwordDto auth.ChangePassword) error
	OAuthRedirect(c echo.Context) error
	OAuthCallback(c echo.Context) error
	OAuthTokenRefresh(c echo.Context, refreshToken string) error
	OAuthLogout(c echo.Context) error
}

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=20"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type ChangePasswordRequest struct {
	OldPassword        string `json:"old_password" validate:"required"`
	NewPassword        string `json:"new_password" validate:"required,min=8"`
	ConfirmNewPassword string `json:"confirm_new_password" validate:"required,eqfield=NewPassword"`
}

func (h *Handler) Register(c echo.Context) error {
	// Check if registration is enabled
	if !viper.Get("registration_enabled").(bool) {
		return echo.NewHTTPError(http.StatusForbidden, "registration is disabled")
	}
	rr := new(RegisterRequest)
	if err := c.Bind(rr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(rr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	userDto := user.User{
		Username: rr.Username,
		Password: rr.Password,
	}

	u, err := h.Service.AuthService.Register(c, userDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, u)
}

func (h *Handler) Login(c echo.Context) error {
	lr := new(LoginRequest)
	if err := c.Bind(lr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(lr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	userDto := user.User{
		Username: lr.Username,
		Password: lr.Password,
	}

	u, err := h.Service.AuthService.Login(c, userDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}
	return c.JSON(http.StatusOK, u)
}

func (h *Handler) OAuthLogin(c echo.Context) error {
	oAuthEnabled := viper.GetBool("oauth_enabled")
	if !oAuthEnabled {
		return echo.NewHTTPError(http.StatusForbidden, "OAuth is disabled")
	}
	// Redirect to OAuth provider
	err := h.Service.AuthService.OAuthRedirect(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "oAuth redirect")
}

func (h *Handler) Refresh(c echo.Context) error {

	refreshCookie, err := c.Cookie("refresh-token")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}
	refreshToken := refreshCookie.Value

	err = h.Service.AuthService.Refresh(c, refreshToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, "tokens refreshed")
}

func (h *Handler) Me(c echo.Context) error {
	cc := c.(*auth.CustomContext)

	u, err := h.Service.AuthService.Me(cc)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}
	return c.JSON(http.StatusOK, u)
}

func (h *Handler) ChangePassword(c echo.Context) error {
	cc := c.(*auth.CustomContext)
	cp := new(ChangePasswordRequest)
	if err := c.Bind(cp); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(cp); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if cp.OldPassword == cp.NewPassword {
		return echo.NewHTTPError(http.StatusBadRequest, "new password must be different from old password")
	}

	passwordDto := auth.ChangePassword{
		OldPassword: cp.OldPassword,
		NewPassword: cp.NewPassword,
	}

	err := h.Service.AuthService.ChangePassword(cc, passwordDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, "password changed")
}

func (h *Handler) OAuthCallback(c echo.Context) error {
	err := h.Service.AuthService.OAuthCallback(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.Redirect(http.StatusFound, os.Getenv("FRONTEND_HOST"))
}

func (h *Handler) OAuthTokenRefresh(c echo.Context) error {
	refreshCookie, err := c.Cookie("oauth_refresh_token")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}
	refreshToken := refreshCookie.Value

	err = h.Service.AuthService.OAuthTokenRefresh(c, refreshToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, "tokens refreshed")
}

func (h *Handler) OAuthLogout(c echo.Context) error {

	err := h.Service.AuthService.OAuthLogout(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.Redirect(http.StatusFound, os.Getenv("FRONTEND_HOST"))
}
