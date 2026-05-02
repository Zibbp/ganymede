package user

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/user"
	"github.com/zibbp/ganymede/internal/api_key"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

// ErrSystemUserProtected is returned by AdminUpdateUser/AdminDeleteUser
// when the target row is the API system user. Callers should surface
// this as a 403 Forbidden. Mutating or deleting that row would break
// API key auth (the middleware injects it into request context for
// every keyed request), so the admin API treats it as immutable.
var ErrSystemUserProtected = errors.New("the system api user cannot be modified or deleted")

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
	if err := s.assertNotSystemUser(c, uDto.ID); err != nil {
		return nil, err
	}
	u, err := s.Store.Client.User.UpdateOneID(uDto.ID).SetUsername(uDto.Username).SetRole(uDto.Role).Save(c.Request().Context())
	if err != nil {
		return nil, fmt.Errorf("error updating user: %v", err)
	}
	return u, nil
}

func (s *Service) AdminDeleteUser(c echo.Context, uID uuid.UUID) error {
	if err := s.assertNotSystemUser(c, uID); err != nil {
		return err
	}
	err := s.Store.Client.User.DeleteOneID(uID).Exec(c.Request().Context())
	if err != nil {
		return fmt.Errorf("error deleting user: %v", err)
	}
	return nil
}

// assertNotSystemUser fails with ErrSystemUserProtected if uID points
// at the singleton system api user. Mutations on that row would break
// API key auth (the middleware injects this user into context for every
// keyed request), so the admin API treats it as immutable.
//
// A row that doesn't exist is not a violation here — the underlying
// Update/Delete will surface its own NotFoundError.
func (s *Service) assertNotSystemUser(c echo.Context, uID uuid.UUID) error {
	u, err := s.Store.Client.User.Query().Where(user.ID(uID)).Only(c.Request().Context())
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("error checking user: %v", err)
	}
	if u.Username == api_key.SystemUserUsername {
		return ErrSystemUserProtected
	}
	return nil
}
