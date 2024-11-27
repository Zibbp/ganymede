package auth

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entUser "github.com/zibbp/ganymede/ent/user"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

type CustomContext struct {
	echo.Context
	User *ent.User
}

func GuardMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check if user has a valid access token
		accessToken, err := c.Cookie("access-token")
		if err != nil {
			// Check if user has oauth2 token
			oauthAccessToken, err := c.Cookie("oauth_access_token")
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid access token")
			}

			if oauthAccessToken.Value == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid access token")
			}

			// Check if user has a valid access token
			userClaims, err := CheckOAuthAccessToken(c, oauthAccessToken.Value)
			if err != nil {
				log.Debug().Err(err).Msg("OAuth access token is invalid")
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid access token")
			}

			c.Set("auth_method", "oauth")
			c.Set("user.sub", userClaims.Sub)
			c.Set("user.username", userClaims.NickName)
			c.Set("access_token", oauthAccessToken.Value)
			return next(c)
		}

		// Check if user has a valid refresh token
		userClaims, err := checkAccessToken(accessToken.Value)
		if err != nil {
			log.Debug().Err(err).Msg("Access token is invalid")
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid access token")
		}

		c.Set("auth_method", "local")
		c.Set("user.id", userClaims.UserID)
		c.Set("user.username", userClaims.Username)
		c.Set("access_token", accessToken.Value)
		return next(c)
	}
}

func GetUserMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authMethod := c.Get("auth_method").(string)
		if authMethod == "local" {
			id := c.Get("user.id").(uuid.UUID)

			user, err := database.DB().Client.User.Query().Where(entUser.ID(id)).Only(c.Request().Context())
			if err != nil {
				// If user not found throw the same error as to not give hints about issue
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid access token")
			}
			cc := &CustomContext{
				Context: c,
				User:    user,
			}
			return next(cc)
		}
		if authMethod == "oauth" {
			sub := c.Get("user.sub").(string)
			// Get user
			user, err := database.DB().Client.User.Query().Where(entUser.Sub(sub)).Only(c.Request().Context())
			if err != nil {
				// If user not found throw the same error as to not give hints about issue
				return echo.NewHTTPError(http.StatusInternalServerError, "Invalid access token")
			}

			cc := &CustomContext{
				Context: c,
				User:    user,
			}
			return next(cc)
		}
		return next(c)
	}
}

func UserRoleMiddleware(role utils.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := c.(*CustomContext)

			switch role {
			case utils.AdminRole:
				if cc.User.Role == utils.AdminRole {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusForbidden, "You are not authorized to access this resource")
			case utils.EditorRole:
				if cc.User.Role == utils.EditorRole || cc.User.Role == utils.AdminRole {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusForbidden, "You are not authorized to access this resource")
			case utils.ArchiverRole:
				if cc.User.Role == utils.ArchiverRole || cc.User.Role == utils.EditorRole || cc.User.Role == utils.AdminRole {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusForbidden, "You are not authorized to access this resource")
			case utils.UserRole:
				if cc.User.Role == utils.UserRole || cc.User.Role == utils.ArchiverRole || cc.User.Role == utils.EditorRole || cc.User.Role == utils.AdminRole {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusForbidden, "You are not authorized to access this resource")
			default:
				return echo.NewHTTPError(http.StatusForbidden, "You are not authorized to access this resource")
			}
		}
	}
}

// GetUserClaims - returns user's claims
func GetUserClaims(c echo.Context) *ent.User {
	cc := c.(*CustomContext)
	return cc.User
}
