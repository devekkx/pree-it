package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Errors returned by Verify  callers switch on these.
var (
	ErrMismatch    = errors.New("password does not match")
	ErrInvalidHash = errors.New("stored hash is malformed")
)

// params holds the Argon2id tuning parameters.
// These are stored inside every hash string so old hashes remain
// verifiable after a parameter upgrade.
type params struct {
	memory      uint32 // KiB of memory to use
	iterations  uint32 // number of passes over memory
	parallelism uint8  // degree of parallelism (threads)
	saltLen     uint32 // bytes of random salt
	keyLen      uint32 // bytes of derived key
}

// defaultParams meets OWASP 2024 Argon2id minimums:
//
//	memory      ≥ 19 MiB  (we use 64 MiB  more is better)
//	iterations  ≥ 2
//	parallelism ≥ 1
//
// Benchmark on your target hardware and increase memory/iterations
// until hashing takes ~300 ms under expected load.
var defaultParams = params{
	memory:      64 * 1024, // 64 MiB
	iterations:  3,
	parallelism: 2,
	saltLen:     16, // 128-bit salt  unique per password
	keyLen:      32, // 256-bit derived key
}

// Hash derives an Argon2id hash from plaintext and returns an encoded
// string in the format:
//
//	$argon2id$v=19$m=65536,t=3,p=2$<salt_b64>$<hash_b64>
//
// This format is self-describing  Verify never needs out-of-band params.
func Hash(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("password.Hash: plaintext must not be empty")
	}

	salt := make([]byte, defaultParams.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password.Hash: generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(plaintext),
		salt,
		defaultParams.iterations,
		defaultParams.memory,
		defaultParams.parallelism,
		defaultParams.keyLen,
	)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		defaultParams.memory,
		defaultParams.iterations,
		defaultParams.parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// Verify checks plaintext against an encoded Argon2id hash string.
// Returns nil on match, ErrMismatch on wrong password, ErrInvalidHash
// if the stored string is malformed.
//
// The comparison is performed in constant time to prevent timing attacks.
func Verify(plaintext, encoded string) error {
	p, salt, storedHash, err := decodeHash(encoded)
	if err != nil {
		return ErrInvalidHash
	}

	candidateHash := argon2.IDKey(
		[]byte(plaintext),
		salt,
		p.iterations,
		p.memory,
		p.parallelism,
		p.keyLen,
	)

	// subtle.ConstantTimeCompare prevents timing-based password oracle attacks.
	if subtle.ConstantTimeCompare(storedHash, candidateHash) != 1 {
		return ErrMismatch
	}

	return nil
}

// NeedsRehash reports whether the stored hash was produced with parameters
// that differ from the current defaults. Call this after a successful Verify
// and rehash if it returns true  enables transparent parameter upgrades.
func NeedsRehash(encoded string) bool {
	p, _, _, err := decodeHash(encoded)
	if err != nil {
		return true
	}
	return p.memory != defaultParams.memory ||
		p.iterations != defaultParams.iterations ||
		p.parallelism != defaultParams.parallelism ||
		p.keyLen != defaultParams.keyLen
}

// decodeHash parses a hash string produced by Hash.
// Format: $argon2id$v=<ver>$m=<mem>,t=<iter>,p=<par>$<salt>$<hash>
func decodeHash(encoded string) (p params, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	// parts[0] = "" (before leading $)
	// parts[1] = "argon2id"
	// parts[2] = "v=19"
	// parts[3] = "m=65536,t=3,p=2"
	// parts[4] = <salt_b64>
	// parts[5] = <hash_b64>
	if len(parts) != 6 {
		err = fmt.Errorf("decode: expected 6 fields, got %d", len(parts))
		return
	}

	if parts[1] != "argon2id" {
		err = fmt.Errorf("decode: unsupported algorithm %q", parts[1])
		return
	}

	var version int
	if _, scanErr := fmt.Sscanf(parts[2], "v=%d", &version); scanErr != nil {
		err = fmt.Errorf("decode: parse version: %w", scanErr)
		return
	}
	if version != argon2.Version {
		err = fmt.Errorf("decode: unsupported argon2 version %d", version)
		return
	}

	_, scanErr := fmt.Sscanf(
		parts[3], "m=%d,t=%d,p=%d",
		&p.memory, &p.iterations, &p.parallelism,
	)
	if scanErr != nil {
		err = fmt.Errorf("decode: parse params: %w", scanErr)
		return
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		err = fmt.Errorf("decode: decode salt: %w", err)
		return
	}
	p.saltLen = uint32(len(salt))

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		err = fmt.Errorf("decode: decode hash: %w", err)
		return
	}
	p.keyLen = uint32(len(hash))

	return
}
