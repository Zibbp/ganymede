package auth

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/user"
	"github.com/zibbp/ganymede/internal/utils"
)

const (
	accessTokenCookieName  = "access-token"
	refreshTokenCookieName = "refresh-token"
)

type Claims struct {
	UserID   uuid.UUID  `json:"user_id"`
	Username string     `json:"username"`
	Role     utils.Role `json:"role"`
	jwt.RegisteredClaims
}

func GetJWTSecret() string {
	env := config.GetEnvApplicationConfig()
	jwtSecret := env.JWTSecret
	return jwtSecret
}
func GetJWTRefreshSecret() string {
	env := config.GetEnvApplicationConfig()
	jwtRefreshSecret := env.JWTRefreshSecret
	return jwtRefreshSecret
}

// generateJWTToken generates a new JWT token for the user.
func generateJWTToken(user *user.User, expirationTime time.Time, secret []byte) (string, time.Time, error) {
	// Create the JWT claims, which includes the username and expiry time.
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			// In JWT, the expiry time is expressed as unix milliseconds.
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	// Declare the token with the HS256 algorithm used for signing, and the claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Create the JWT string.
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", time.Now(), err
	}

	return tokenString, expirationTime, nil
}

// setTokenCookie sets the cookie with the token.
func setTokenCookie(c echo.Context, name string, token string, expiration time.Time) {
	// Get optional cookie domain name
	env := config.GetEnvConfig()
	cookieDomain := env.CookieDomain
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = token
	cookie.Expires = expiration
	cookie.Path = "/"
	// Frontend uses the contents of the cookie - not the best but it works.
	cookie.HttpOnly = false
	cookie.SameSite = http.SameSiteLaxMode
	if cookieDomain != "" {
		cookie.Domain = cookieDomain
	}

	c.SetCookie(cookie)
}

// checkAccessToken checks if the JWT access token is valid.
func checkAccessToken(accessToken string) (*Claims, error) {
	// Parse the token.
	token, err := jwt.ParseWithClaims(accessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(GetJWTSecret()), nil
	})
	if err != nil {
		return nil, err
	}

	// Validate the token and return the custom claims.
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, err
	}

	return claims, nil
}

// JWTErrorChecker will be executed when user try to access a protected path.
func JWTErrorChecker(err error, c echo.Context) error {
	// Redirects to the signIn form.
	return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
}
