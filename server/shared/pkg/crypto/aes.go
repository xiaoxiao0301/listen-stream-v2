package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidKeySize indicates the AES key size is invalid
	ErrInvalidKeySize = errors.New("crypto: invalid key size, must be 16, 24, or 32 bytes")
	
	// ErrInvalidCiphertext indicates the ciphertext is too short or invalid
	ErrInvalidCiphertext = errors.New("crypto: invalid ciphertext")
	
	// ErrDecryptionFailed indicates decryption failed
	ErrDecryptionFailed = errors.New("crypto: decryption failed")
)

// AESCipher provides AES-256-GCM encryption and decryption
type AESCipher struct {
	key []byte
}

// NewAESCipher creates a new AES cipher with the given key
// Key must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256
func NewAESCipher(key []byte) (*AESCipher, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKeySize
	}
	
	return &AESCipher{
		key: key,
	}, nil
}

// NewAES256Cipher creates a new AES-256 cipher with a 32-byte key
func NewAES256Cipher(key []byte) (*AESCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("crypto: AES-256 requires 32-byte key, got %d bytes", len(key))
	}
	
	return &AESCipher{
		key: key,
	}, nil
}

// Encrypt encrypts plaintext using AES-GCM
// Returns base64-encoded ciphertext with nonce prepended
func (c *AESCipher) Encrypt(plaintext []byte) (string, error) {
	if len(plaintext) == 0 {
		return "", errors.New("crypto: plaintext cannot be empty")
	}
	
	// Create AES cipher block
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("crypto: failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: failed to create GCM: %w", err)
	}
	
	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: failed to generate nonce: %w", err)
	}
	
	// Encrypt and seal
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	
	// Encode to base64 for safe storage/transmission
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// EncryptString is a convenience method for encrypting strings
func (c *AESCipher) EncryptString(plaintext string) (string, error) {
	return c.Encrypt([]byte(plaintext))
}

// Decrypt decrypts base64-encoded ciphertext using AES-GCM
func (c *AESCipher) Decrypt(ciphertext string) ([]byte, error) {
	if ciphertext == "" {
		return nil, errors.New("crypto: ciphertext cannot be empty")
	}
	
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("crypto: invalid base64: %w", err)
	}
	
	// Create AES cipher block
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create GCM: %w", err)
	}
	
	// Check ciphertext length
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrInvalidCiphertext
	}
	
	// Extract nonce and ciphertext
	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	
	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	
	return plaintext, nil
}

// DecryptString is a convenience method for decrypting to strings
func (c *AESCipher) DecryptString(ciphertext string) (string, error) {
	plaintext, err := c.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptWithKey is a convenience function for one-time encryption
func EncryptWithKey(plaintext, key []byte) (string, error) {
	cipher, err := NewAESCipher(key)
	if err != nil {
		return "", err
	}
	return cipher.Encrypt(plaintext)
}

// DecryptWithKey is a convenience function for one-time decryption
func DecryptWithKey(ciphertext string, key []byte) ([]byte, error) {
	cipher, err := NewAESCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.Decrypt(ciphertext)
}

// RotateKey re-encrypts data with a new key
// This is useful for key rotation scenarios
func RotateKey(ciphertext string, oldKey, newKey []byte) (string, error) {
	// Decrypt with old key
	oldCipher, err := NewAESCipher(oldKey)
	if err != nil {
		return "", fmt.Errorf("crypto: invalid old key: %w", err)
	}
	
	plaintext, err := oldCipher.Decrypt(ciphertext)
	if err != nil {
		return "", fmt.Errorf("crypto: decryption failed: %w", err)
	}
	
	// Encrypt with new key
	newCipher, err := NewAESCipher(newKey)
	if err != nil {
		return "", fmt.Errorf("crypto: invalid new key: %w", err)
	}
	
	return newCipher.Encrypt(plaintext)
}
