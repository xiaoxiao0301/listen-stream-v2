package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	// ErrInvalidHash indicates the hash string format is invalid
	ErrInvalidHash = errors.New("crypto: invalid hash format")
	
	// ErrMismatchedHash indicates password verification failed
	ErrMismatchedHash = errors.New("crypto: password does not match hash")
)

// Argon2Params defines parameters for Argon2id algorithm
type Argon2Params struct {
	// Memory in KiB
	Memory uint32
	
	// Number of iterations
	Iterations uint32
	
	// Number of parallel threads
	Parallelism uint8
	
	// Salt length in bytes
	SaltLength uint32
	
	// Key length in bytes
	KeyLength uint32
}

// DefaultArgon2Params returns recommended parameters for Argon2id
// These provide a good balance between security and performance
// Hashing time: ~100-200ms on modern hardware
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// HighSecurityArgon2Params returns parameters for high-security scenarios
// Hashing time: ~500ms on modern hardware
func HighSecurityArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      256 * 1024, // 256 MB
		Iterations:  4,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// PasswordHasher provides password hashing using Argon2id
type PasswordHasher struct {
	params *Argon2Params
}

// NewPasswordHasher creates a new password hasher with default parameters
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		params: DefaultArgon2Params(),
	}
}

// NewPasswordHasherWithParams creates a password hasher with custom parameters
func NewPasswordHasherWithParams(params *Argon2Params) *PasswordHasher {
	return &PasswordHasher{
		params: params,
	}
}

// Hash generates an Argon2id hash of the password
// Returns a formatted string: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
func (h *PasswordHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("crypto: password cannot be empty")
	}
	
	// Generate random salt
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("crypto: failed to generate salt: %w", err)
	}
	
	// Generate hash
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.params.Iterations,
		h.params.Memory,
		h.params.Parallelism,
		h.params.KeyLength,
	)
	
	// Encode to base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	
	// Format: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.params.Memory,
		h.params.Iterations,
		h.params.Parallelism,
		b64Salt,
		b64Hash,
	)
	
	return encoded, nil
}

// Verify checks if the password matches the hash
func (h *PasswordHasher) Verify(password, encodedHash string) (bool, error) {
	if password == "" {
		return false, errors.New("crypto: password cannot be empty")
	}
	
	if encodedHash == "" {
		return false, errors.New("crypto: hash cannot be empty")
	}
	
	// Parse the encoded hash
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}
	
	// Generate hash with the same parameters
	computedHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)
	
	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(hash, computedHash) == 1 {
		return true, nil
	}
	
	return false, nil
}

// VerifyOrError is like Verify but returns an error if verification fails
func (h *PasswordHasher) VerifyOrError(password, encodedHash string) error {
	match, err := h.Verify(password, encodedHash)
	if err != nil {
		return err
	}
	
	if !match {
		return ErrMismatchedHash
	}
	
	return nil
}

// NeedsRehash checks if the hash needs to be regenerated with current parameters
// Returns true if the hash uses different parameters than the current hasher
func (h *PasswordHasher) NeedsRehash(encodedHash string) (bool, error) {
	params, _, _, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}
	
	// Compare parameters
	if params.Memory != h.params.Memory ||
		params.Iterations != h.params.Iterations ||
		params.Parallelism != h.params.Parallelism ||
		params.KeyLength != h.params.KeyLength {
		return true, nil
	}
	
	return false, nil
}

// decodeHash parses an encoded hash string
func decodeHash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}
	
	if parts[1] != "argon2id" {
		return nil, nil, nil, errors.New("crypto: unsupported algorithm")
	}
	
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, fmt.Errorf("crypto: invalid version: %w", err)
	}
	
	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("crypto: incompatible version %d", version)
	}
	
	params := &Argon2Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return nil, nil, nil, fmt.Errorf("crypto: invalid parameters: %w", err)
	}
	
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("crypto: invalid salt: %w", err)
	}
	params.SaltLength = uint32(len(salt))
	
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("crypto: invalid hash: %w", err)
	}
	params.KeyLength = uint32(len(hash))
	
	return params, salt, hash, nil
}

// HashPassword is a convenience function using default parameters
func HashPassword(password string) (string, error) {
	hasher := NewPasswordHasher()
	return hasher.Hash(password)
}

// VerifyPassword is a convenience function for password verification
func VerifyPassword(password, hash string) (bool, error) {
	hasher := NewPasswordHasher()
	return hasher.Verify(password, hash)
}
