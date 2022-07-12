package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/user"
	"github.com/zibbp/ganymede/internal/utils"
	"net/http"
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

func (h *Handler) GetUsers(c echo.Context) error {
	users, err := h.Service.UserService.AdminGetUsers(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, users)
}

func (h *Handler) GetUser(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	u, err := h.Service.UserService.AdminGetUser(c, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, u)
}

func (h *Handler) UpdateUser(c echo.Context) error {
	uID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	usr := new(UpdateChannelRequest)
	if err := c.Bind(usr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := c.Validate(usr); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	uDto := user.User{
		ID:       uID,
		Username: usr.Username,
		Role:     utils.Role(usr.Role),
	}
	u, err := h.Service.UserService.AdminUpdateUser(c, uDto)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, u)
}

func (h *Handler) DeleteUser(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	err = h.Service.UserService.AdminDeleteUser(c, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusOK)
}
