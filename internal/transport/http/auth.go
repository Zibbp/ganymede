package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/user"
)

type AuthService interface {
	Register(ctx context.Context, userDto user.User) (*ent.User, error)
	Login(ctx context.Context, userDto user.User) (*ent.User, error)
	Refresh(c echo.Context, refreshToken string) error
	ChangePassword(ctx context.Context, userId uuid.UUID, oldPassword, newPassword string) error
	OAuthRedirect(c echo.Context) error
	OAuthCallback(c echo.Context) error
	OAuthTokenRefresh(c echo.Context, refreshToken string) error
	OAuthLogout(c echo.Context) error
}

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
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

// Register godoc
//
//	@Summary		Register a user
//	@Description	Register a user (does not log in)
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			register	body		RegisterRequest	true	"Register"
//	@Success		200			{object}	ent.User
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		403			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/auth/register [post]
func (h *Handler) Register(c echo.Context) error {
	rr := new(RegisterRequest)
	if err := c.Bind(rr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(rr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	userDto := user.User{
		Username: rr.Username,
		Password: rr.Password,
	}

	u, err := h.Service.AuthService.Register(c.Request().Context(), userDto)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, u, "successfully registered")
}

// Login godoc
//
//	@Summary		Login a user
//	@Description	Login a user (sets access-token and refresh-token cookies). Access token lasts for 1 hour. Refresh token lasts for 1 month.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			login	body		LoginRequest	true	"Login"
//	@Success		200		{object}	ent.User
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		401		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/auth/login [post]
func (h *Handler) Login(c echo.Context) error {
	lr := new(LoginRequest)
	if err := c.Bind(lr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(lr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	userDto := user.User{
		Username: lr.Username,
		Password: lr.Password,
	}

	u, err := h.Service.AuthService.Login(c.Request().Context(), userDto)
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	h.SessionManager.Put(c.Request().Context(), "user_id", u.ID.String())

	return SuccessResponse(c, u, "successfully logged in")
}

func (h *Handler) Logout(c echo.Context) error {
	if err := h.SessionManager.Destroy(c.Request().Context()); err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, "error deleting session")
	}

	return SuccessResponse(c, "", "logged out")
}

// OAuthLogin godoc
//
//	@Summary		Login a user with OAuth
//	@Description	Login a user with OAuth (sets access-token and refresh-token cookies)
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	ent.User
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		401	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/auth/oauth/login [get]
func (h *Handler) OAuthLogin(c echo.Context) error {
	env := config.GetEnvConfig()
	if !env.OAuthEnabled {
		return echo.NewHTTPError(http.StatusForbidden, "OAuth is disabled")
	}
	// Redirect to OAuth provider
	err := h.Service.AuthService.OAuthRedirect(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, "oAuth redirect")
}

// Refresh godoc
//
//	@Summary		Refresh access-token and refresh-token
//	@Description	Refresh access-token and refresh-token (sets access-token and refresh-token cookies)
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		401	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/auth/refresh [post]
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

// Me godoc
//
//	@Summary		Get current user
//	@Description	Get current user
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	ent.User
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		401	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/auth/me [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) Me(c echo.Context) error {
	user := userFromContext(c)

	return SuccessResponse(c, user, "you")
}

// ChangePassword godoc
//
//	@Summary		Change password
//	@Description	Change password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			change-password	body		ChangePasswordRequest	true	"Change password"
//	@Success		200				{object}	string
//	@Failure		400				{object}	utils.ErrorResponse
//	@Failure		401				{object}	utils.ErrorResponse
//	@Failure		500				{object}	utils.ErrorResponse
//	@Router			/auth/change-password [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) ChangePassword(c echo.Context) error {
	user := userFromContext(c)

	changePasswordRequest := new(ChangePasswordRequest)
	if err := c.Bind(changePasswordRequest); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(changePasswordRequest); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	err := h.Service.AuthService.ChangePassword(c.Request().Context(), user.ID, changePasswordRequest.OldPassword, changePasswordRequest.NewPassword)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	return SuccessResponse(c, "", "password changed")
}

// OAuthCallback godoc
//
//	@Summary		OAuth callback
//	@Description	OAuth callback for OAuth provider
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		401	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/auth/oauth/callback [get]
func (h *Handler) OAuthCallback(c echo.Context) error {
	env := config.GetEnvApplicationConfig()
	err := h.Service.AuthService.OAuthCallback(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.Redirect(http.StatusFound, env.FrontendHost)
}

// OAuthTokenRefresh godoc
//
//	@Summary		Refresh access-token and refresh-token
//	@Description	Refresh access-token and refresh-token (sets access-token and refresh-token cookies)
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		401	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/auth/oauth/refresh [get]
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

// OAuthLogout godoc
//
//	@Summary		Logout
//	@Description	Logout
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	string
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		401	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/auth/oauth/logout [get]
func (h *Handler) OAuthLogout(c echo.Context) error {
	env := config.GetEnvApplicationConfig()
	err := h.Service.AuthService.OAuthLogout(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.Redirect(http.StatusFound, env.FrontendHost)
}
