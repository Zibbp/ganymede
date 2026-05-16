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
	scopes    utils.ApiKeyScopes
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
//
// The cache also tracks an in-memory last-touched timestamp per key id
// so the middleware's touchAsync can short-circuit before spawning a
// goroutine + hitting the DB on a key that was just touched. Persistent
// state still lives in api_keys.last_used_at; this is purely a debounce
// hint that's lost on process restart.
type VerificationCache struct {
	mu          sync.RWMutex
	entries     map[string]cacheEntry
	lastTouched map[uuid.UUID]time.Time
	ttl         time.Duration
	now         func() time.Time
}

// NewVerificationCache constructs a cache with the given TTL. Pass 0 to
// use the default (60 s).
func NewVerificationCache(ttl time.Duration) *VerificationCache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &VerificationCache{
		entries:     make(map[string]cacheEntry),
		lastTouched: make(map[uuid.UUID]time.Time),
		ttl:         ttl,
		now:         time.Now,
	}
}

// ShouldTouch reports whether last_used_at for this key id should be
// refreshed in the database. Returns true at most once per id per
// debounce window; concurrent callers see only the first true. When
// it returns true the cache atomically records the current time as the
// last touch, so the caller doesn't need a follow-up "I touched it"
// call — failing or succeeding, the next caller is debounced for
// `debounce` from now.
//
// Used by the middleware's touchAsync to skip spawning a goroutine
// (and the DB UPDATE that follows) on hot keys that were just touched.
func (c *VerificationCache) ShouldTouch(id uuid.UUID, debounce time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if last, ok := c.lastTouched[id]; ok && now.Sub(last) < debounce {
		return false
	}
	c.lastTouched[id] = now
	return true
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
//
// Returns a copy of the scopes slice so callers can't mutate the cache
// entry through the returned reference. Symmetric with Set, which
// already takes a defensive copy on insert.
func (c *VerificationCache) Get(fullToken string) (uuid.UUID, utils.ApiKeyScopes, bool) {
	k := keyFor(fullToken)
	c.mu.RLock()
	entry, ok := c.entries[k]
	c.mu.RUnlock()
	if !ok {
		return uuid.Nil, nil, false
	}
	if c.now().After(entry.expiresAt) {
		// Opportunistic eviction. Re-check under the write lock
		// because a concurrent Set between the RUnlock above and the
		// Lock below could have stored a fresh, non-expired entry —
		// blindly deleting would discard it. Re-read entries[k] and
		// only evict if it's still the expired version.
		c.mu.Lock()
		current, stillThere := c.entries[k]
		if stillThere && !c.now().After(current.expiresAt) {
			// Replaced under us; return the fresh entry instead of
			// evicting.
			c.mu.Unlock()
			return current.apiKeyID, copyScopes(current.scopes), true
		}
		delete(c.entries, k)
		c.mu.Unlock()
		return uuid.Nil, nil, false
	}
	return entry.apiKeyID, copyScopes(entry.scopes), true
}

// copyScopes is the defensive-copy helper shared by Get and Set so the
// cache and its callers never alias the same slice.
func copyScopes(scopes utils.ApiKeyScopes) utils.ApiKeyScopes {
	cp := make(utils.ApiKeyScopes, len(scopes))
	copy(cp, scopes)
	return cp
}

// Set stores a positive verification result keyed by sha256(fullToken).
// Existing entries for the same token are overwritten and their TTL
// reset.
func (c *VerificationCache) Set(fullToken string, apiKeyID uuid.UUID, scopes utils.ApiKeyScopes) {
	k := keyFor(fullToken)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[k] = cacheEntry{
		apiKeyID: apiKeyID,
		// Defensive copy so a later mutation of the caller's slice
		// cannot silently change the cached scopes. Symmetric with
		// the copy on Get.
		scopes:    copyScopes(scopes),
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
	// Also drop the touch debounce hint so a re-mint of this id (or the
	// next touch on a not-yet-revoked key after a stale entry is cleared)
	// is not silently suppressed by a stale timestamp.
	delete(c.lastTouched, id)
}

// Clear empties the cache. Mainly useful for tests.
func (c *VerificationCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
	c.lastTouched = make(map[uuid.UUID]time.Time)
}
