package crypto

import (
	"encoding/hex"
	"testing"
)

func TestAESCipher_EncryptDecrypt(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes for AES-256
	cipher, err := NewAESCipher(key)
	if err != nil {
		t.Fatalf("failed to create cipher: %v", err)
	}

	plaintext := "Hello, World! This is a test message."

	// Encrypt
	ciphertext, err := cipher.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if ciphertext == "" {
		t.Error("ciphertext is empty")
	}

	// Decrypt
	decrypted, err := cipher.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted text doesn't match:\nwant: %s\ngot:  %s", plaintext, decrypted)
	}
}

func TestAESCipher_InvalidKey(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
		wantErr bool
	}{
		{"16 bytes (AES-128)", 16, false},
		{"24 bytes (AES-192)", 24, false},
		{"32 bytes (AES-256)", 32, false},
		{"invalid 15 bytes", 15, true},
		{"invalid 20 bytes", 20, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			_, err := NewAESCipher(key)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestAESCipher_DecryptInvalidData(t *testing.T) {
	key := make([]byte, 32)
	cipher, _ := NewAESCipher(key)

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"empty string", ""},
		{"invalid base64", "not-base64!!!"},
		{"too short", "YWJjZA=="}, // "abcd" in base64 - too short for nonce+tag
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cipher.Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("expected error for invalid ciphertext")
			}
		})
	}
}

func TestAESCipher_EmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	cipher, _ := NewAESCipher(key)

	// Empty plaintext should error
	_, err := cipher.EncryptString("")
	if err == nil {
		t.Error("expected error for empty plaintext")
	}
}

func TestAESCipher_LongText(t *testing.T) {
	key := make([]byte, 32)
	cipher, _ := NewAESCipher(key)

	// Create a long text (10KB)
	longText := ""
	for i := 0; i < 10240; i++ {
		longText += "a"
	}

	ciphertext, err := cipher.EncryptString(longText)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := cipher.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != longText {
		t.Error("decrypted long text doesn't match")
	}
}

func TestAESCipher_UnicodePlaintext(t *testing.T) {
	key := make([]byte, 32)
	cipher, _ := NewAESCipher(key)

	plaintext := "Hello 世界! 🌍 Привет مرحبا"

	ciphertext, err := cipher.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := cipher.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("unicode text doesn't match:\nwant: %s\ngot:  %s", plaintext, decrypted)
	}
}

func TestAESCipher_RotateKey(t *testing.T) {
	oldKey := make([]byte, 32)
	newKey := make([]byte, 32)
	copy(newKey, []byte("new-key-0123456789abcdef0123456"))

	cipher, _ := NewAESCipher(oldKey)

	plaintext := "Secret message"

	// Encrypt with old key
	ciphertext, _ := cipher.EncryptString(plaintext)

	// Rotate key (using standalone function)
	newCiphertext, err := RotateKey(ciphertext, oldKey, newKey)
	if err != nil {
		t.Fatalf("key rotation failed: %v", err)
	}

	// Old ciphertext should not decrypt with new key
	newCipher, _ := NewAESCipher(newKey)
	_, err = newCipher.DecryptString(ciphertext)
	if err == nil {
		t.Error("old ciphertext should not decrypt with new key")
	}

	// New ciphertext should decrypt with new key
	decrypted, err := newCipher.DecryptString(newCiphertext)
	if err != nil {
		t.Fatalf("decryption with new key failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted text doesn't match after rotation")
	}
}

func TestAESCipher_NonceUniqueness(t *testing.T) {
	key := make([]byte, 32)
	cipher, _ := NewAESCipher(key)

	plaintext := "Same plaintext"

	// Encrypt multiple times
	ciphertexts := make([]string, 100)
	for i := 0; i < 100; i++ {
		ct, err := cipher.EncryptString(plaintext)
		if err != nil {
			t.Fatalf("encryption %d failed: %v", i, err)
		}
		ciphertexts[i] = ct
	}

	// All ciphertexts should be different (due to different nonces)
	seen := make(map[string]bool)
	for _, ct := range ciphertexts {
		if seen[ct] {
			t.Error("found duplicate ciphertext - nonce not unique")
		}
		seen[ct] = true
	}
}

func TestAESCipher_ConcurrentEncryption(t *testing.T) {
	key := make([]byte, 32)
	cipher, _ := NewAESCipher(key)

	plaintext := "Concurrent test"

	// Run 100 concurrent encryptions
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			_, err := cipher.EncryptString(plaintext)
			if err != nil {
				t.Errorf("concurrent encryption failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestHexEncodeDecode(t *testing.T) {
	original := []byte("test key 123456")

	// Encode
	hexStr := hex.EncodeToString(original)

	// Decode
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatalf("hex decode failed: %v", err)
	}

	if string(decoded) != string(original) {
		t.Errorf("decoded key doesn't match:\nwant: %v\ngot:  %v", original, decoded)
	}
}

// Benchmarks

func BenchmarkAESEncrypt_Small(b *testing.B) {
	key := make([]byte, 32)
	cipher, _ := NewAESCipher(key)
	plaintext := "Small message for benchmarking"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cipher.EncryptString(plaintext)
	}
}

func BenchmarkAESEncrypt_Large(b *testing.B) {
	key := make([]byte, 32)
	cipher, _ := NewAESCipher(key)

	// 1MB plaintext
	plaintext := ""
	for i := 0; i < 1024*1024; i++ {
		plaintext += "a"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cipher.EncryptString(plaintext)
	}
}

func BenchmarkAESDecrypt(b *testing.B) {
	key := make([]byte, 32)
	cipher, _ := NewAESCipher(key)
	plaintext := "Message for decryption benchmark"

	ciphertext, _ := cipher.EncryptString(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cipher.DecryptString(ciphertext)
	}
}
