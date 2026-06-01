package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

// Token returns the SHA-256 hex digest of a raw refresh token string.
// We use SHA-256 (not bcrypt) for refresh token storage lookup because:
//   - The raw token is a 144-bit random UUID pair — brute force is infeasible
//   - We need fast constant-time equality at query time, not slow hashing
//   - bcrypt is reserved for passwords where the input space is small
func Token(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
