package auth

import (
	jwtv3 "github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/utils"
	"net/http"
)

type CustomContext struct {
	echo.Context
	UserClaims *Claims
}

func GetUserMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		u := c.Get("user").(*jwtv3.Token)
		claims := u.Claims.(*Claims)
		//c.Set("UserClaims", claims)
		cc := &CustomContext{Context: c, UserClaims: claims}
		return next(cc)
	}
}

func UserRoleMiddleware(role utils.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := c.(*CustomContext)

			switch role {
			case utils.AdminRole:
				if cc.UserClaims.Role == utils.AdminRole {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusForbidden, "You are not authorized to access this resource")
			case utils.EditorRole:
				if cc.UserClaims.Role == utils.EditorRole || cc.UserClaims.Role == utils.AdminRole {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusForbidden, "You are not authorized to access this resource")
			case utils.ArchiverRole:
				if cc.UserClaims.Role == utils.ArchiverRole || cc.UserClaims.Role == utils.EditorRole || cc.UserClaims.Role == utils.AdminRole {
					return next(c)
				}
				return echo.NewHTTPError(http.StatusForbidden, "You are not authorized to access this resource")
			case utils.UserRole:
				if cc.UserClaims.Role == utils.UserRole || cc.UserClaims.Role == utils.ArchiverRole || cc.UserClaims.Role == utils.EditorRole || cc.UserClaims.Role == utils.AdminRole {
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
func GetUserClaims(c echo.Context) *Claims {
	cc := c.(*CustomContext)
	return cc.UserClaims
}
