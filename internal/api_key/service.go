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
func (s *Service) Create(ctx context.Context, name, description string, scope utils.ApiKeyScope) (*ent.ApiKey, string, error) {
	if !utils.IsValidApiKeyScope(string(scope)) {
		return nil, "", fmt.Errorf("invalid api key scope: %q", scope)
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
		SetScope(scope).
		Save(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("error creating api key: %w", err)
	}
	return created, full, nil
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
// Errors are non-fatal for the request; the caller fires this in a
// goroutine and discards the error.
func (s *Service) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	row, err := s.Store.Client.ApiKey.Query().
		Where(apikey.ID(id)).
		Select(apikey.FieldLastUsedAt).
		Only(ctx)
	if err != nil {
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
