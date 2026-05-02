package http

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/user"
	"github.com/zibbp/ganymede/internal/api_key"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

// authMethodAPIKey identifies a request authenticated by an API key.
const authMethodAPIKey = "api_key"

// authMethodLocal identifies a request authenticated by a session cookie.
const authMethodLocal = "local"

// extractBearerToken returns the bearer token from the Authorization
// header, or "" if no Bearer-prefixed value is present.
func extractBearerToken(c echo.Context) string {
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(auth, prefix))
}

// AuthAPIKeyOrSessionMiddleware authenticates a request using either an
// API key (Authorization: Bearer …) or a session cookie. API keys are
// tried first; if no Bearer header is present, the request falls back to
// the existing session check.
//
// On success the middleware sets:
//
//	auth_method     = "api_key" | "local"
//	api_key.id      (uuid.UUID, only when auth_method = api_key)
//	api_key.scope   (string,    only when auth_method = api_key)
//	user.id         (string,    only when auth_method = local)
//
// When Config.ApiKeysEnabled is false the Authorization header is
// silently ignored — the request falls through to the session check —
// so flipping the toggle disables external scripts without breaking
// admins managing existing keys via the web UI.
func AuthAPIKeyOrSessionMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := extractBearerToken(c)
		cfg := config.Get()

		if token != "" && cfg != nil && cfg.ApiKeysEnabled && apiKeyService != nil {
			id, scope, err := authenticateAPIKey(c.Request().Context(), token)
			if err != nil {
				return ErrorInvalidAccessTokenResponse(c)
			}
			c.Set("auth_method", authMethodAPIKey)
			c.Set("api_key.id", id)
			c.Set("api_key.scope", string(scope))
			return next(c)
		}

		// Fall back to the existing session-based check.
		userID, ok := sessionManager.Get(c.Request().Context(), "user_id").(string)
		if !ok {
			return ErrorInvalidAccessTokenResponse(c)
		}
		c.Set("auth_method", authMethodLocal)
		c.Set("user.id", userID)
		return next(c)
	}
}

// authenticateAPIKey resolves a presented Bearer token to a (key id, scope)
// pair, hitting the verification cache first and falling back to a DB
// lookup + bcrypt verify. To prevent prefix enumeration via response
// timing it always pays a bcrypt cost, even when the prefix is unknown.
func authenticateAPIKey(ctx context.Context, token string) (uuid.UUID, utils.ApiKeyScope, error) {
	// Fast path: cached positive verification.
	if id, scope, hit := apiKeyService.Cache.Get(token); hit {
		touchAsync(id)
		return id, scope, nil
	}

	prefix, secret, err := api_key.Parse(token)
	if err != nil {
		// Run a constant-time bcrypt anyway so a malformed token cannot
		// be distinguished from a wrong-secret one via timing.
		api_key.VerifyDummy()
		return uuid.Nil, "", err
	}

	row, err := apiKeyService.GetByPrefix(ctx, prefix)
	if err != nil {
		api_key.VerifyDummy()
		return uuid.Nil, "", err
	}

	if err := api_key.Verify(row.HashedSecret, secret); err != nil {
		return uuid.Nil, "", err
	}

	apiKeyService.Cache.Set(token, row.ID, row.Scope)
	touchAsync(row.ID)
	return row.ID, row.Scope, nil
}

// touchAsync fires last_used_at update in a background goroutine using a
// fresh context (the request context dies after the response). The
// service-level debounce keeps this from generating one DB write per
// request on busy keys.
func touchAsync(id uuid.UUID) {
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := apiKeyService.TouchLastUsed(bgCtx, id); err != nil {
			log.Debug().Err(err).Str("api_key_id", id.String()).Msg("touch last_used_at failed")
		}
	}()
}

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
		err := ErrorInvalidAccessTokenResponse(c)
		if err != nil {
			return nil
		}
	}
	return user
}
