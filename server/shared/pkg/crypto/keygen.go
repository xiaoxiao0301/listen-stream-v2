package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// KeyGenerator provides cryptographic key generation utilities.
type KeyGenerator struct {
	// Random source (defaults to crypto/rand.Reader)
	random io.Reader
}

// NewKeyGenerator creates a new key generator.
func NewKeyGenerator() *KeyGenerator {
	return &KeyGenerator{
		random: rand.Reader,
	}
}

// GenerateKey generates a random key of the specified length in bytes.
//
// Example:
//
//	kg := NewKeyGenerator()
//	key, err := kg.GenerateKey(32) // 256-bit key
func (kg *KeyGenerator) GenerateKey(length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("key length must be positive, got %d", length)
	}

	key := make([]byte, length)
	if _, err := io.ReadFull(kg.random, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	return key, nil
}

// GenerateAES128Key generates a 128-bit (16-byte) AES key.
func (kg *KeyGenerator) GenerateAES128Key() ([]byte, error) {
	return kg.GenerateKey(16)
}

// GenerateAES192Key generates a 192-bit (24-byte) AES key.
func (kg *KeyGenerator) GenerateAES192Key() ([]byte, error) {
	return kg.GenerateKey(24)
}

// GenerateAES256Key generates a 256-bit (32-byte) AES key.
func (kg *KeyGenerator) GenerateAES256Key() ([]byte, error) {
	return kg.GenerateKey(32)
}

// GenerateToken generates a random token of the specified length (in bytes).
//
// The token is returned as a hexadecimal string (2x length).
//
// Example:
//
//	token, err := kg.GenerateToken(16) // 32-character hex string
func (kg *KeyGenerator) GenerateToken(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("token length must be positive, got %d", length)
	}

	token := make([]byte, length)
	if _, err := io.ReadFull(kg.random, token); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return hex.EncodeToString(token), nil
}

// GenerateAPIKey generates a random API key with the specified format.
//
// Format: prefix_hextoken
// Example: "lsk_a1b2c3d4e5f6789012345678901234567890abcd"
func (kg *KeyGenerator) GenerateAPIKey(prefix string, length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("API key length must be positive, got %d", length)
	}

	token, err := kg.GenerateToken(length)
	if err != nil {
		return "", err
	}

	if prefix == "" {
		return token, nil
	}

	return fmt.Sprintf("%s_%s", prefix, token), nil
}

// GenerateSalt generates a random salt for password hashing.
//
// Recommended length: 16-32 bytes.
func (kg *KeyGenerator) GenerateSalt(length int) ([]byte, error) {
	if length < 8 {
		return nil, fmt.Errorf("salt length should be at least 8 bytes, got %d", length)
	}

	return kg.GenerateKey(length)
}

// GenerateIV generates a random initialization vector for AES.
//
// AES block size is always 16 bytes.
func (kg *KeyGenerator) GenerateIV() ([]byte, error) {
	return kg.GenerateKey(16)
}

// GenerateNonce generates a random nonce for AES-GCM.
//
// Standard GCM nonce size is 12 bytes (96 bits).
func (kg *KeyGenerator) GenerateNonce() ([]byte, error) {
	return kg.GenerateKey(12)
}

// KeyToHex converts a key to hexadecimal string.
func KeyToHex(key []byte) string {
	return hex.EncodeToString(key)
}

// KeyFromHex converts a hexadecimal string to a key.
func KeyFromHex(hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}
	return key, nil
}

// MustGenerateKey generates a key or panics on error.
//
// Use this only for initialization or testing.
func MustGenerateKey(length int) []byte {
	kg := NewKeyGenerator()
	key, err := kg.GenerateKey(length)
	if err != nil {
		panic(fmt.Sprintf("failed to generate key: %v", err))
	}
	return key
}

// MustGenerateAES256Key generates an AES-256 key or panics.
func MustGenerateAES256Key() []byte {
	return MustGenerateKey(32)
}

// MustGenerateToken generates a token or panics.
func MustGenerateToken(length int) string {
	kg := NewKeyGenerator()
	token, err := kg.GenerateToken(length)
	if err != nil {
		panic(fmt.Sprintf("failed to generate token: %v", err))
	}
	return token
}
