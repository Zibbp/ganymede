// Package api_key provides primitives, storage, and a verification cache for
// admin-managed API keys used to authenticate external scripts against the
// Ganymede HTTP API.
//
// Token format: "gym_<12-hex-prefix>_<43-char-base64url-secret>".
// The prefix is stored in the DB unhashed and indexed for O(log n) lookup;
// the secret half is bcrypt-hashed and verified per request.
package api_key

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// TokenBrand is the constant prefix every Ganymede API key starts with.
	// It lets clients (and grep) recognise leaked keys instantly.
	TokenBrand = "gym"

	// PrefixBytes is the random portion of the prefix (in bytes, hex-encoded
	// so the on-the-wire prefix is 12 hex chars).
	PrefixBytes = 6

	// SecretBytes is the entropy of the secret half. base64url-encoded
	// without padding this becomes 43 chars (256 bits of entropy).
	SecretBytes = 32

	// BcryptCost balances per-request latency against brute-force resistance.
	// 12 yields ~80–150 ms per verify on modern hardware; the LRU cache
	// makes this a one-time cost per key per minute.
	BcryptCost = 12
)

var (
	ErrMalformedToken = errors.New("malformed api key token")
	ErrInvalidSecret  = errors.New("invalid api key secret")
)

// Generate produces a fresh random API key. Returns the full token (to be
// shown once to the admin), the prefix (stored in the DB for lookup), and
// the secret (passed to HashSecret for storage).
func Generate() (full, prefix, secret string, err error) {
	prefixBuf := make([]byte, PrefixBytes)
	if _, err = rand.Read(prefixBuf); err != nil {
		return "", "", "", err
	}
	secretBuf := make([]byte, SecretBytes)
	if _, err = rand.Read(secretBuf); err != nil {
		return "", "", "", err
	}

	prefix = hex.EncodeToString(prefixBuf)
	secret = base64.RawURLEncoding.EncodeToString(secretBuf)
	full = TokenBrand + "_" + prefix + "_" + secret
	return full, prefix, secret, nil
}

// HashSecret bcrypt-hashes the secret half of the token at the package's
// configured cost. Only the secret is hashed; the prefix is stored in
// plaintext so we can find the row before paying the bcrypt cost.
func HashSecret(secret string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(secret), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// Parse splits a full token into its prefix and secret components and
// validates the brand/shape. It does NOT verify the secret against any
// stored hash — pair with Verify for that.
//
// The secret is base64url-encoded and so may itself contain '_'. Only the
// first two underscores are treated as separators.
func Parse(full string) (prefix, secret string, err error) {
	parts := strings.SplitN(full, "_", 3)
	if len(parts) != 3 {
		return "", "", ErrMalformedToken
	}
	if parts[0] != TokenBrand {
		return "", "", ErrMalformedToken
	}
	if len(parts[1]) != PrefixBytes*2 {
		return "", "", ErrMalformedToken
	}
	// Sanity-check that the prefix is hex; this prevents a malicious value
	// from probing the prefix index with non-hex strings.
	if _, decodeErr := hex.DecodeString(parts[1]); decodeErr != nil {
		return "", "", ErrMalformedToken
	}
	if parts[2] == "" {
		return "", "", ErrMalformedToken
	}
	return parts[1], parts[2], nil
}

// Verify compares a presented secret against a stored bcrypt hash.
// Returns ErrInvalidSecret on mismatch.
func Verify(hashedSecret, secret string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedSecret), []byte(secret)); err != nil {
		return ErrInvalidSecret
	}
	return nil
}

// dummyHash is used as a constant-time decoy when a presented prefix is
// not found in the database. Performing the same bcrypt compare on a
// successful prefix lookup vs a missing one prevents a timing oracle that
// would let an attacker enumerate valid prefixes.
//
// Computed lazily and cached for the lifetime of the process.
var dummyHash = func() string {
	h, _ := bcrypt.GenerateFromPassword([]byte("dummy-not-a-real-secret"), BcryptCost)
	return string(h)
}()

// VerifyDummy performs a bcrypt comparison against an internal sentinel
// hash. Use this when the DB lookup misses to keep request timing
// indistinguishable from a wrong-secret response.
func VerifyDummy() {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte("dummy-not-a-real-secret-mismatch"))
}
