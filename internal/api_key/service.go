package api_key

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/apikey"
	"github.com/zibbp/ganymede/ent/user"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

// Sentinel errors so the HTTP layer can distinguish input-validation
// failures (400 Bad Request) from internal failures (500). Use
// errors.Is to detect; the wrapping fmt.Errorf preserves the underlying
// cause for logs while keeping the typed sentinel for matching.
var (
	// ErrInvalidScope is returned when Create/Update sees a scope
	// string that doesn't parse as a known resource:tier in the
	// catalog. Wrapped with fmt.Errorf("...: %w", ErrInvalidScope) so
	// the caller still gets the offending value in the message.
	ErrInvalidScope = errors.New("unknown scope")

	// ErrEmptyScopes is returned when Create/Update is called with an
	// empty (or all-duplicate) scope list. The HTTP validator already
	// catches the JSON-level "min=1" case; this guards the service
	// layer for in-process callers and post-dedup empty lists.
	ErrEmptyScopes = errors.New("at least one scope is required")

	// ErrNotFound is returned when an Update or revoke target either
	// doesn't exist or has been soft-deleted. The HTTP layer maps
	// both this and ent.IsNotFound to 404. Used by the conditional-
	// update path so we don't need to construct ent's internal
	// NotFoundError type from outside the ent package.
	ErrNotFound = errors.New("api key not found")
)

// SystemUserUsername is the reserved username of the singleton service-account
// user injected into request context whenever a request is authenticated via
// API key. Handlers that read userFromContext continue to work unchanged;
// scope-based permission decisions are made by the request middleware, not
// by this user's role.
const SystemUserUsername = "system:api"

// touchDebounceWindow is the minimum interval between last_used_at writes
// for a single key. Without it a busy key would generate one DB write per
// request; with it we get at most one update per key per minute.
const touchDebounceWindow = 60 * time.Second

// Service is the storage and lifecycle service for API keys.
type Service struct {
	Store *database.Database
	Cache *VerificationCache
}

// NewService constructs a Service backed by the given database. The
// returned service owns its own VerificationCache.
func NewService(store *database.Database) *Service {
	return &Service{
		Store: store,
		Cache: NewVerificationCache(0),
	}
}

// Create mints a fresh API key, persists the prefix and bcrypt-hashed
// secret, and returns the new ent row alongside the full secret token to
// surface to the admin exactly once.
//
// createdBy records which session user minted the key for audit
// purposes. The HTTP layer reads it from request context (the admin's
// session) and passes it here. uuid.Nil disables attribution — only
// useful for tests; production callers should always pass a real id.
//
// The scopes slice must be non-empty and every entry must be a valid
// utils.ApiKeyScope (resource:tier with both halves in the catalog).
// Duplicates are silently de-duplicated; the validation reports the
// first invalid entry so an admin gets a precise 400 message.
func (s *Service) Create(ctx context.Context, name, description string, scopes []utils.ApiKeyScope, createdBy uuid.UUID) (*ent.ApiKey, string, error) {
	normalized, err := normalizeScopes(scopes)
	if err != nil {
		return nil, "", err
	}

	full, prefix, secret, err := Generate()
	if err != nil {
		return nil, "", fmt.Errorf("error generating api key: %w", err)
	}

	hashed, err := HashSecret(secret)
	if err != nil {
		return nil, "", fmt.Errorf("error hashing api key secret: %w", err)
	}

	builder := s.Store.Client.ApiKey.Create().
		SetName(name).
		SetDescription(description).
		SetPrefix(prefix).
		SetHashedSecret(hashed).
		SetScopes(normalized.Strings())
	if createdBy != uuid.Nil {
		builder = builder.SetCreatedByID(createdBy)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("error creating api key: %w", err)
	}
	return created, full, nil
}

// normalizeScopes validates a presented scope list and returns a typed,
// de-duplicated slice in input order. Returns ErrEmptyScopes for an
// empty list and a wrapped ErrInvalidScope for the first unknown
// entry; the HTTP layer uses errors.Is to map these to 400.
func normalizeScopes(in []utils.ApiKeyScope) (utils.ApiKeyScopes, error) {
	if len(in) == 0 {
		return nil, ErrEmptyScopes
	}
	seen := make(map[utils.ApiKeyScope]struct{}, len(in))
	out := make(utils.ApiKeyScopes, 0, len(in))
	for _, s := range in {
		if !s.IsValid() {
			return nil, fmt.Errorf("%w: %q", ErrInvalidScope, s)
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out, nil
}

// List returns API keys, ordered most-recently-created first.
// Defaults to active keys only; pass includeRevoked=true to include
// soft-deleted rows for forensic / audit views.
func (s *Service) List(ctx context.Context, includeRevoked bool) ([]*ent.ApiKey, error) {
	q := s.Store.Client.ApiKey.Query().
		Order(ent.Desc(apikey.FieldCreatedAt))
	if !includeRevoked {
		q = q.Where(apikey.RevokedAtIsNil())
	}
	keys, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing api keys: %w", err)
	}
	return keys, nil
}

// Update replaces the editable fields of a non-revoked API key (name,
// description, scopes). The prefix and hashed_secret are immutable —
// rotating a key means revoking it and creating a fresh one. Returns
// ent.NotFoundError when the id doesn't exist or the key has been
// revoked.
//
// Like Create, scopes are normalized (validated, deduped) before
// persistence so a 400-friendly error is returned for unknown scopes.
//
// The verification cache is flushed for this key id so the new scopes
// take effect on the next request rather than after the cache TTL
// elapses — same pattern Revoke uses.
func (s *Service) Update(ctx context.Context, id uuid.UUID, name, description string, scopes []utils.ApiKeyScope) (*ent.ApiKey, error) {
	normalized, err := normalizeScopes(scopes)
	if err != nil {
		return nil, err
	}

	// Single conditional UPDATE rather than read-then-update: if a
	// concurrent Revoke commits between a SELECT and the subsequent
	// UpdateOneID, the second write would land on a now-revoked row.
	// Update().Where(ID, RevokedAtIsNil()).Save returns the count of
	// matched rows; zero means the row doesn't exist or is revoked, in
	// which case we surface the same NotFoundError the read-then-update
	// path used to.
	affected, err := s.Store.Client.ApiKey.Update().
		Where(apikey.ID(id), apikey.RevokedAtIsNil()).
		SetName(name).
		SetDescription(description).
		SetScopes(normalized.Strings()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("error updating api key: %w", err)
	}
	if affected == 0 {
		return nil, ErrNotFound
	}

	// Flush the cache before the re-read. The scope change has already
	// committed; if the post-update Only() fails for a transient reason
	// (DB blip, context cancel) and we haven't invalidated yet, an
	// existing cache entry holding the previous broader scopes would
	// stay valid until the 60s TTL elapses. Invalidating here means
	// the new scopes take effect on the next request even if we bail
	// out of this call.
	if s.Cache != nil {
		s.Cache.InvalidateByID(id)
	}

	// Re-read to return the fresh row to the caller.
	updated, err := s.Store.Client.ApiKey.Query().Where(apikey.ID(id)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("error reloading updated api key: %w", err)
	}
	return updated, nil
}

// Revoke marks the API key as revoked (soft delete). Subsequent
// authentication attempts with this key are rejected. The verification
// cache is flushed so the change takes effect immediately rather than
// after the cache TTL elapses.
//
// Returns ErrNotFound when the id either doesn't exist or has already
// been revoked. Conditional UPDATE (RevokedAtIsNil filter) makes the
// "first revoke wins" semantics explicit and atomic against double-
// revoke, parallel to the Update path.
func (s *Service) Revoke(ctx context.Context, id uuid.UUID) error {
	affected, err := s.Store.Client.ApiKey.Update().
		Where(apikey.ID(id), apikey.RevokedAtIsNil()).
		SetRevokedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("error revoking api key: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	if s.Cache != nil {
		s.Cache.InvalidateByID(id)
	}
	return nil
}

// GetByPrefix looks up a non-revoked API key by its public prefix. Used
// by the auth middleware to fetch the row before paying the bcrypt cost.
// Returns *ent.NotFoundError when no matching live key exists.
func (s *Service) GetByPrefix(ctx context.Context, prefix string) (*ent.ApiKey, error) {
	return s.Store.Client.ApiKey.Query().
		Where(
			apikey.Prefix(prefix),
			apikey.RevokedAtIsNil(),
		).
		Only(ctx)
}

// TouchLastUsed updates the last_used_at column, but only if the existing
// value is older than the debounce window. This avoids a write storm on
// busy keys while keeping the timestamp accurate to within ~60 s.
//
// Single conditional UPDATE rather than read-then-update: a concurrent
// Revoke between the SELECT and the UpdateOneID would otherwise let
// the touch land on the now-revoked row. The WHERE clause includes
// revoked_at IS NULL and "last_used_at is null OR last_used_at <
// (now - debounce)" so the debounce check is part of the same atomic
// statement.
//
// Errors are non-fatal for the request; the caller fires this in a
// goroutine and discards the error. Zero affected rows (revoked,
// missing, or still inside the debounce window) is not surfaced as
// an error — there's nothing for the caller to do about it.
func (s *Service) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	cutoff := now.Add(-touchDebounceWindow)
	_, err := s.Store.Client.ApiKey.Update().
		Where(
			apikey.ID(id),
			apikey.RevokedAtIsNil(),
			apikey.Or(
				apikey.LastUsedAtIsNil(),
				apikey.LastUsedAtLT(cutoff),
			),
		).
		SetLastUsedAt(now).
		Save(ctx)
	return err
}

// EnsureSystemUser idempotently creates the singleton service-account
// user used by AuthGetUserMiddleware to satisfy handlers that read
// userFromContext under API key authentication.
//
// The user has utils.SystemRole — a sentinel role that roleSatisfies
// rejects unconditionally. This makes every legacy
// AuthUserRoleMiddleware check fail closed for API key requests; the
// only middleware that can authorise a keyed request is
// RequireRoleOrScope (which keys off api_key.scopes, not user.role).
// If a future route is mistakenly registered with the legacy chain,
// API keys are rejected rather than silently granted.
//
// Existing deployments may have a row with role == admin from before
// SystemRole was introduced; this function self-heals such rows on
// every boot by setting the role to SystemRole.
//
// Safe to call repeatedly.
func (s *Service) EnsureSystemUser(ctx context.Context) (*ent.User, error) {
	existing, err := s.Store.Client.User.Query().
		Where(user.Username(SystemUserUsername)).
		Only(ctx)
	if err == nil {
		if existing.Role != utils.SystemRole {
			// Self-heal: legacy row from before the sentinel role was
			// introduced. Bump it to SystemRole so the middleware
			// fail-closed property holds going forward.
			updated, updErr := s.Store.Client.User.UpdateOneID(existing.ID).
				SetRole(utils.SystemRole).
				Save(ctx)
			if updErr != nil {
				return nil, fmt.Errorf("error migrating system user role: %w", updErr)
			}
			return updated, nil
		}
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("error querying system user: %w", err)
	}

	created, err := s.Store.Client.User.Create().
		SetUsername(SystemUserUsername).
		SetOauth(true).
		SetRole(utils.SystemRole).
		Save(ctx)
	if err != nil {
		// Bootstrap race: two replicas booting at once can both pass
		// the Query miss above and both call Create. The unique
		// constraint on username then makes one of them fail. Catch
		// the constraint violation and re-query for the row that the
		// other replica just created — Postgres has guaranteed it
		// exists by the time the constraint fires.
		if _, isConstraint := err.(*ent.ConstraintError); isConstraint {
			existing, qErr := s.Store.Client.User.Query().
				Where(user.Username(SystemUserUsername)).
				Only(ctx)
			if qErr == nil {
				return existing, nil
			}
			return nil, fmt.Errorf("system user create lost a race but could not re-query: %w", qErr)
		}
		return nil, fmt.Errorf("error creating system user: %w", err)
	}
	return created, nil
}

// GetSystemUser returns the singleton system user. Returns ErrSystemUserMissing
// (wrapping ent.NotFoundError) if EnsureSystemUser has not been called yet.
func (s *Service) GetSystemUser(ctx context.Context) (*ent.User, error) {
	return s.Store.Client.User.Query().
		Where(user.Username(SystemUserUsername)).
		Only(ctx)
}
