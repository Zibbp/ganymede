package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/user"
	"github.com/zibbp/ganymede/internal/utils"
)

type UserService interface {
	AdminGetUsers(c echo.Context) ([]*ent.User, error)
	AdminGetUser(c echo.Context, id uuid.UUID) (*ent.User, error)
	AdminUpdateUser(c echo.Context, uDto user.User) (*ent.User, error)
	AdminDeleteUser(c echo.Context, id uuid.UUID) error
}

type UpdateChannelRequest struct {
	Username string `json:"username" validate:"required,min=2,max=50"`
	Role     string `json:"role" validate:"required,oneof=admin editor archiver user"`
}

// GetUsers godoc
//
//	@Summary		Get all users
//	@Description	Get all users
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]ent.User
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/user [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetUsers(c echo.Context) error {
	users, err := h.Service.UserService.AdminGetUsers(c)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, users, "users")
}

// GetUser godoc
//
//	@Summary		Get user by id
//	@Description	Get user by id
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	ent.User
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/user/{id} [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetUser(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	u, err := h.Service.UserService.AdminGetUser(c, id)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, u, "user")
}

// UpdateUser godoc
//
//	@Summary		Update user
//	@Description	Update user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"User ID"
//	@Param			body	body		UpdateChannelRequest	true	"User data"
//	@Success		200		{object}	ent.User
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/user/{id} [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateUser(c echo.Context) error {
	uID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	usr := new(UpdateChannelRequest)
	if err := c.Bind(usr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(usr); err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	uDto := user.User{
		ID:       uID,
		Username: usr.Username,
		Role:     utils.Role(usr.Role),
	}
	u, err := h.Service.UserService.AdminUpdateUser(c, uDto)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}
	return SuccessResponse(c, u, "user updated")
}

// DeleteUser godoc
//
//	@Summary		Delete user
//	@Description	Delete user
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"User ID"
//	@Success		200
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/user/{id} [delete]
//	@Security		ApiKeyCookieAuth
func (h *Handler) DeleteUser(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return ErrorResponse(c, http.StatusBadRequest, err.Error())
	}
	err = h.Service.UserService.AdminDeleteUser(c, id)
	if err != nil {
		return ErrorResponse(c, http.StatusInternalServerError, err.Error())
	}

	// destroy sessions
	err = h.SessionManager.Iterate(c.Request().Context(), func(ctx context.Context) error {
		userID := sessionManager.GetString(ctx, "user_id")

		if userID == id.String() {
			return sessionManager.Destroy(ctx)
		}

		return nil
	})
	if err != nil {
		return ErrorResponse(c, 500, "error deleting user sessions")
	}

	return c.NoContent(http.StatusOK)
}
