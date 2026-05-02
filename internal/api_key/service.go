package api_key

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/apikey"
	"github.com/zibbp/ganymede/ent/user"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
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
// The scopes slice must be non-empty and every entry must be a valid
// utils.ApiKeyScope (resource:tier with both halves in the catalog).
// Duplicates are silently de-duplicated; the validation reports the
// first invalid entry so an admin gets a precise 400 message.
func (s *Service) Create(ctx context.Context, name, description string, scopes []utils.ApiKeyScope) (*ent.ApiKey, string, error) {
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

	created, err := s.Store.Client.ApiKey.Create().
		SetName(name).
		SetDescription(description).
		SetPrefix(prefix).
		SetHashedSecret(hashed).
		SetScopes(normalized.Strings()).
		Save(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("error creating api key: %w", err)
	}
	return created, full, nil
}

// normalizeScopes validates a presented scope list and returns a typed,
// de-duplicated slice in input order. Returns a 400-friendly error on
// the first invalid scope or if the list is empty.
func normalizeScopes(in []utils.ApiKeyScope) (utils.ApiKeyScopes, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("at least one scope is required")
	}
	seen := make(map[utils.ApiKeyScope]struct{}, len(in))
	out := make(utils.ApiKeyScopes, 0, len(in))
	for _, s := range in {
		if !s.IsValid() {
			return nil, fmt.Errorf("unknown scope: %q", s)
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out, nil
}

// List returns every non-revoked API key, ordered most-recently-created
// first.
func (s *Service) List(ctx context.Context) ([]*ent.ApiKey, error) {
	keys, err := s.Store.Client.ApiKey.Query().
		Where(apikey.RevokedAtIsNil()).
		Order(ent.Desc(apikey.FieldCreatedAt)).
		All(ctx)
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

	// Reject updates on revoked keys: a revoked key shouldn't be
	// editable, otherwise admins could "un-revoke" by editing it back
	// into circulation. Combined Where on (id, RevokedAtIsNil) returns
	// not-found rather than fetching+checking in two steps.
	row, err := s.Store.Client.ApiKey.Query().
		Where(apikey.ID(id), apikey.RevokedAtIsNil()).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	updated, err := s.Store.Client.ApiKey.UpdateOneID(row.ID).
		SetName(name).
		SetDescription(description).
		SetScopes(normalized.Strings()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("error updating api key: %w", err)
	}

	if s.Cache != nil {
		s.Cache.InvalidateByID(id)
	}
	return updated, nil
}

// Revoke marks the API key as revoked (soft delete). Subsequent
// authentication attempts with this key are rejected. The verification
// cache is flushed so the change takes effect immediately rather than
// after the cache TTL elapses.
func (s *Service) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := s.Store.Client.ApiKey.UpdateOneID(id).
		SetRevokedAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("error revoking api key: %w", err)
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
// Filters on revoked_at IS NULL so a request that was already in flight
// when an admin revoked a key (cache flushed AFTER the DB commit on
// Revoke) doesn't bump last_used_at on the now-revoked row, which would
// otherwise leave audit data with last_used_at > revoked_at.
//
// Errors are non-fatal for the request; the caller fires this in a
// goroutine and discards the error.
func (s *Service) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	row, err := s.Store.Client.ApiKey.Query().
		Where(apikey.ID(id), apikey.RevokedAtIsNil()).
		Select(apikey.FieldLastUsedAt).
		Only(ctx)
	if err != nil {
		// Includes the case where the row exists but is revoked — Only
		// returns NotFoundError because the WHERE filter excluded it.
		// That's the intended behavior: silently no-op rather than
		// updating audit columns on a revoked key.
		return err
	}
	now := time.Now()
	if row.LastUsedAt != nil && now.Sub(*row.LastUsedAt) < touchDebounceWindow {
		return nil
	}
	_, err = s.Store.Client.ApiKey.UpdateOneID(id).
		SetLastUsedAt(now).
		Save(ctx)
	return err
}

// EnsureSystemUser idempotently creates the singleton service-account
// user used to satisfy handlers that call userFromContext under API key
// authentication. The user has the admin role so any handler-level role
// check that runs before scope enforcement does not reject it; the
// actual permission decision is made by RequireRoleOrScope based on the
// API key's scope.
//
// The function is safe to call repeatedly; if the row already exists it
// is returned unchanged.
func (s *Service) EnsureSystemUser(ctx context.Context) (*ent.User, error) {
	existing, err := s.Store.Client.User.Query().
		Where(user.Username(SystemUserUsername)).
		Only(ctx)
	if err == nil {
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("error querying system user: %w", err)
	}

	created, err := s.Store.Client.User.Create().
		SetUsername(SystemUserUsername).
		SetOauth(true).
		SetRole(utils.AdminRole).
		Save(ctx)
	if err != nil {
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
