package http

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/user"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

// AuthGuardMiddleware is a middleware that enforces authentication. If the request does not contain a vaild session a 403 error is returned.
func AuthGuardMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		userID, ok := sessionManager.Get(c.Request().Context(), "user_id").(string)
		if !ok {
			return ErrorInvalidAccessTokenResponse(c)
		}

		c.Set("auth_method", "local")
		c.Set("user.id", userID)

		return next(c)
	}
}

// AuthGetUserMiddleware is a middleware that fetches the user from the database and sets it in the request context. AuthGuardMiddleware is expected to run before this to set the user ID from a session token.
func AuthGetUserMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authMethod := c.Get("auth_method").(string)
		if authMethod == "local" {
			idStr, ok := c.Get("user.id").(string)
			if !ok {
				log.Error().Msg("user id missing from context")
				return ErrorInvalidAccessTokenResponse(c)
			}

			id, err := uuid.Parse(idStr)
			if err != nil {
				log.Error().Err(err).Msg("error parsing user id as uuid")
				return ErrorInvalidAccessTokenResponse(c)
			}

			user, err := database.DB().Client.User.Query().Where(user.ID(id)).Only(c.Request().Context())
			if err != nil {
				return ErrorInvalidAccessTokenResponse(c)
			}

			c.Set("user", user)

			return next(c)
		}

		return next(c)
	}
}

func AuthUserRoleMiddleware(role utils.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := userFromContext(c)

			switch role {
			case utils.AdminRole:
				if user.Role == utils.AdminRole {
					return next(c)
				}
				return ErrorUnauthorizedResponse(c)
			case utils.EditorRole:
				if user.Role == utils.EditorRole || user.Role == utils.AdminRole {
					return next(c)
				}
				return ErrorUnauthorizedResponse(c)
			case utils.ArchiverRole:
				if user.Role == utils.ArchiverRole || user.Role == utils.EditorRole || user.Role == utils.AdminRole {
					return next(c)
				}
				return ErrorUnauthorizedResponse(c)
			case utils.UserRole:
				if user.Role == utils.UserRole || user.Role == utils.ArchiverRole || user.Role == utils.EditorRole || user.Role == utils.AdminRole {
					return next(c)
				}
				return ErrorUnauthorizedResponse(c)
			default:
				return ErrorUnauthorizedResponse(c)
			}
		}
	}
}

// userFromContext parses the user from the request context as a *ent.User.
func userFromContext(c echo.Context) *ent.User {
	userStr := c.Get("user")
	user, ok := userStr.(*ent.User)
	if !ok {
		ErrorInvalidAccessTokenResponse(c)
	}
	return user
}
