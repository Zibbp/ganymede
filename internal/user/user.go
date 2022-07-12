package user

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/user"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

type User struct {
	ID        uuid.UUID  `json:"id"`
	Username  string     `json:"username"`
	Password  string     `json:"password"`
	Role      utils.Role `json:"role"`
	Webhook   string     `json:"webhook"`
	UpdatedAt string     `json:"updated_at"`
	CreatedAt string     `json:"created_at"`
}

func (s *Service) AdminGetUsers(c echo.Context) ([]*ent.User, error) {
	users, err := s.Store.Client.User.Query().All(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error getting users: %v", err)
	}
	return users, nil
}

func (s *Service) AdminGetUser(c echo.Context, uID uuid.UUID) (*ent.User, error) {
	u, err := s.Store.Client.User.Query().Where(user.ID(uID)).Only(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error getting user: %v", err)
	}
	return u, nil
}

func (s *Service) AdminUpdateUser(c echo.Context, uDto User) (*ent.User, error) {
	u, err := s.Store.Client.User.UpdateOneID(uDto.ID).SetUsername(uDto.Username).SetRole(uDto.Role).Save(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error updating user: %v", err)
	}
	return u, nil
}

func (s *Service) AdminDeleteUser(c echo.Context, uID uuid.UUID) error {
	err := s.Store.Client.User.DeleteOneID(uID).Exec(c.Request().Context())
	if err != nil {
		return fmt.Errorf("error deleting user: %v", err)
	}
	return nil
}
