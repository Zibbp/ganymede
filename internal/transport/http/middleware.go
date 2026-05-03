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
//	api_key.id      (uuid.UUID,           only when auth_method = api_key)
//	api_key.scopes  (utils.ApiKeyScopes,  only when auth_method = api_key)
//	user.id         (string,              only when auth_method = local)
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
			id, scopes, err := authenticateAPIKey(c.Request().Context(), token)
			if err != nil {
				return ErrorInvalidAccessTokenResponse(c)
			}
			c.Set("auth_method", authMethodAPIKey)
			c.Set("api_key.id", id)
			c.Set("api_key.scopes", scopes)
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

// authenticateAPIKey resolves a presented Bearer token to a (key id, scopes)
// pair, hitting the verification cache first and falling back to a DB
// lookup + bcrypt verify. To prevent prefix enumeration via response
// timing it always pays a bcrypt cost, even when the prefix is unknown.
func authenticateAPIKey(ctx context.Context, token string) (uuid.UUID, utils.ApiKeyScopes, error) {
	// Fast path: cached positive verification.
	if id, scopes, hit := apiKeyService.Cache.Get(token); hit {
		touchAsync(id)
		return id, scopes, nil
	}

	prefix, secret, err := api_key.Parse(token)
	if err != nil {
		// Run a constant-time bcrypt anyway so a malformed token cannot
		// be distinguished from a wrong-secret one via timing.
		api_key.VerifyDummy()
		return uuid.Nil, nil, err
	}

	row, err := apiKeyService.GetByPrefix(ctx, prefix)
	if err != nil {
		api_key.VerifyDummy()
		return uuid.Nil, nil, err
	}

	if err := api_key.Verify(row.HashedSecret, secret); err != nil {
		return uuid.Nil, nil, err
	}

	scopes := utils.ApiKeyScopesFromStrings(row.Scopes)
	apiKeyService.Cache.Set(token, row.ID, scopes)
	touchAsync(row.ID)
	return row.ID, scopes, nil
}

// touchAsync fires last_used_at update in a background goroutine using a
// fresh context (the request context dies after the response).
//
// Two-stage debounce so a hot key (10k+ RPS during a script run) does not
// generate one goroutine + SQL UPDATE per request:
//  1. In-memory hint via VerificationCache.ShouldTouch — atomically asks
//     "has this id been touched within the window?" and records the
//     attempt. Skips the goroutine entirely on subsequent hits.
//  2. SQL guard inside Service.TouchLastUsed — conditional UPDATE WHERE
//     last_used_at < cutoff so even if two processes race past stage (1)
//     (separate caches), only the first UPDATE lands.
//
// Stage (1) is best-effort and lost on restart; stage (2) is the
// authoritative debounce that survives across processes.
func touchAsync(id uuid.UUID) {
	if !apiKeyService.Cache.ShouldTouch(id, api_key.TouchDebounceWindow) {
		return
	}
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
//
// When auth_method == "api_key" the middleware injects the singleton
// system service-account user (see api_key.SystemUserUsername). This
// keeps handlers that read userFromContext working unchanged under API
// key authentication; the actual permission decision is made by
// RequireRoleOrScope based on the API key's scope, not this user's role.
func AuthGetUserMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authMethod, _ := c.Get("auth_method").(string)

		switch authMethod {
		case authMethodLocal:
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

			u, err := database.DB().Client.User.Query().Where(user.ID(id)).Only(c.Request().Context())
			if err != nil {
				return ErrorInvalidAccessTokenResponse(c)
			}

			c.Set("user", u)
		case authMethodAPIKey:
			if apiKeyService == nil {
				log.Error().Msg("api key service not configured")
				return ErrorInvalidAccessTokenResponse(c)
			}
			sysUser, err := apiKeyService.GetSystemUser(c.Request().Context())
			if err != nil {
				log.Error().Err(err).Msg("error fetching api system user")
				return ErrorInvalidAccessTokenResponse(c)
			}
			c.Set("user", sysUser)
		default:
			// Reject unknown / missing auth_method instead of falling
			// through. This keeps the chain fail-closed if a route is
			// ever wired with AuthGetUserMiddleware but no upstream
			// AuthGuardMiddleware / AuthAPIKeyOrSessionMiddleware to
			// set auth_method — handlers that read userFromContext
			// would otherwise see a nil user and behave unpredictably.
			log.Error().Str("auth_method", authMethod).Msg("missing or unknown auth_method in context")
			return ErrorInvalidAccessTokenResponse(c)
		}

		return next(c)
	}
}

func AuthUserRoleMiddleware(role utils.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := userFromContext(c)
			if user == nil {
				return ErrorUnauthorizedResponse(c)
			}
			if roleSatisfies(user.Role, role) {
				return next(c)
			}
			return ErrorUnauthorizedResponse(c)
		}
	}
}

// roleSatisfies reports whether actual is at least required, using the
// existing hierarchy admin > editor > archiver > user.
func roleSatisfies(actual, required utils.Role) bool {
	switch required {
	case utils.AdminRole:
		return actual == utils.AdminRole
	case utils.EditorRole:
		return actual == utils.EditorRole || actual == utils.AdminRole
	case utils.ArchiverRole:
		return actual == utils.ArchiverRole || actual == utils.EditorRole || actual == utils.AdminRole
	case utils.UserRole:
		return actual == utils.UserRole || actual == utils.ArchiverRole || actual == utils.EditorRole || actual == utils.AdminRole
	default:
		return false
	}
}

// RequireRoleOrScope enforces permission for routes that accept both
// session and API key authentication. Session requests are checked
// against the role hierarchy (matching AuthUserRoleMiddleware); API key
// requests are checked against the scope hierarchy: the key's scopes
// list satisfies the requirement if any element Includes the required
// scope (resource match-or-wildcard + tier hierarchy).
//
// Call this *after* AuthAPIKeyOrSessionMiddleware and
// AuthGetUserMiddleware in the route's middleware chain.
func RequireRoleOrScope(role utils.Role, scope utils.ApiKeyScope) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authMethod, _ := c.Get("auth_method").(string)
			switch authMethod {
			case authMethodAPIKey:
				scopes, _ := c.Get("api_key.scopes").(utils.ApiKeyScopes)
				if scopes.Includes(scope) {
					return next(c)
				}
				return ErrorUnauthorizedResponse(c)
			case authMethodLocal:
				user := userFromContext(c)
				if user == nil {
					return ErrorUnauthorizedResponse(c)
				}
				if roleSatisfies(user.Role, role) {
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
