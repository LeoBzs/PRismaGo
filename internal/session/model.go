// Package session answers "who is making this request?". It issues opaque
// bearer tokens, stores only their SHA-256 hashes, and provides the
// RequireUser middleware that puts the caller's user id into the context.
//
// It imports nothing from other features, so user, lift and auth can all
// depend on it without import cycles.
package session

import (
	"crypto/sha256"
	"errors"
	"time"
)

var ErrUnauthenticated = errors.New("authentication required")

// TTL is how long a login lasts. Fixed, not sliding: the user logs in again after it.
const TTL = 30 * 24 * time.Hour

// Token is what the client receives at login and sends back as
// "Authorization: Bearer <token>".
type Token struct {
	Value     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// hash is what gets stored. SHA-256 (not bcrypt) is right here: the token is
// already 128+ random bits, so there's nothing to brute-force, and lookups
// happen on every request so they must be fast.
func hash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
