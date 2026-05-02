package api_key

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zibbp/ganymede/internal/utils"
)

// cacheEntry is a positive verification result for a presented token.
// It is keyed by sha256(full token) so the raw secret is never held in
// memory once the bcrypt verify has succeeded.
type cacheEntry struct {
	apiKeyID  uuid.UUID
	scope     utils.ApiKeyScope
	expiresAt time.Time
}

// VerificationCache is a small in-process TTL cache that absorbs repeated
// verifications of the same token. The bcrypt step is only paid the first
// time a token is presented within a TTL window.
//
// The cache is intentionally simple: a map guarded by a RWMutex with a
// fixed TTL. The expected working set is "number of API keys actively
// hitting the server in the last minute", which is tiny in practice
// (typically <100), so a plain map without LRU eviction is fine. Expired
// entries are reaped opportunistically on Get and Set.
type VerificationCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	now     func() time.Time
}

// NewVerificationCache constructs a cache with the given TTL. Pass 0 to
// use the default (60 s).
func NewVerificationCache(ttl time.Duration) *VerificationCache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &VerificationCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		now:     time.Now,
	}
}

// keyFor returns the SHA-256 hex digest of the full token, used as the
// internal cache key. Hashing means we never retain the raw secret in
// memory after the bcrypt verify.
func keyFor(fullToken string) string {
	sum := sha256.Sum256([]byte(fullToken))
	return hex.EncodeToString(sum[:])
}

// Get returns a cached positive result for fullToken if one exists and
// has not expired. The boolean second return is false on miss/expired.
func (c *VerificationCache) Get(fullToken string) (uuid.UUID, utils.ApiKeyScope, bool) {
	k := keyFor(fullToken)
	c.mu.RLock()
	entry, ok := c.entries[k]
	c.mu.RUnlock()
	if !ok {
		return uuid.Nil, "", false
	}
	if c.now().After(entry.expiresAt) {
		// Opportunistic eviction; safe to do under write lock.
		c.mu.Lock()
		delete(c.entries, k)
		c.mu.Unlock()
		return uuid.Nil, "", false
	}
	return entry.apiKeyID, entry.scope, true
}

// Set stores a positive verification result keyed by sha256(fullToken).
// Existing entries for the same token are overwritten and their TTL
// reset.
func (c *VerificationCache) Set(fullToken string, apiKeyID uuid.UUID, scope utils.ApiKeyScope) {
	k := keyFor(fullToken)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[k] = cacheEntry{
		apiKeyID:  apiKeyID,
		scope:     scope,
		expiresAt: c.now().Add(c.ttl),
	}
}

// InvalidateByID removes any cache entries whose stored API key id matches
// the given id. Called on revocation so a revoked key stops authenticating
// within the same request rather than after the TTL elapses.
//
// Linear scan over the map is acceptable because the working set is
// small (see type doc).
func (c *VerificationCache) InvalidateByID(id uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.entries {
		if e.apiKeyID == id {
			delete(c.entries, k)
		}
	}
}

// Clear empties the cache. Mainly useful for tests.
func (c *VerificationCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
}
