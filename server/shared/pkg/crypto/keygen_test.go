package crypto

import (
	"encoding/hex"
	"testing"
)

func TestKeyGenerator_GenerateKey(t *testing.T) {
	kg := NewKeyGenerator()

	tests := []struct {
		name   string
		length int
		want   int
	}{
		{"16 bytes", 16, 16},
		{"32 bytes", 32, 32},
		{"64 bytes", 64, 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := kg.GenerateKey(tt.length)
			if err != nil {
				t.Fatalf("GenerateKey failed: %v", err)
			}

			if len(key) != tt.want {
				t.Errorf("expected key length %d, got %d", tt.want, len(key))
			}
		})
	}
}

func TestKeyGenerator_InvalidLength(t *testing.T) {
	kg := NewKeyGenerator()

	_, err := kg.GenerateKey(0)
	if err == nil {
		t.Error("expected error for zero length")
	}

	_, err = kg.GenerateKey(-1)
	if err == nil {
		t.Error("expected error for negative length")
	}
}

func TestKeyGenerator_AESKeys(t *testing.T) {
	kg := NewKeyGenerator()

	tests := []struct {
		name     string
		generate func() ([]byte, error)
		want     int
	}{
		{"AES-128", kg.GenerateAES128Key, 16},
		{"AES-192", kg.GenerateAES192Key, 24},
		{"AES-256", kg.GenerateAES256Key, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := tt.generate()
			if err != nil {
				t.Fatalf("failed to generate key: %v", err)
			}

			if len(key) != tt.want {
				t.Errorf("expected %d bytes, got %d", tt.want, len(key))
			}
		})
	}
}

func TestKeyGenerator_GenerateToken(t *testing.T) {
	kg := NewKeyGenerator()

	tests := []struct {
		name   string
		length int
		want   int // hex string length (2x input)
	}{
		{"16 bytes", 16, 32},
		{"32 bytes", 32, 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := kg.GenerateToken(tt.length)
			if err != nil {
				t.Fatalf("GenerateToken failed: %v", err)
			}

			if len(token) != tt.want {
				t.Errorf("expected token length %d, got %d", tt.want, len(token))
			}

			// Verify it's valid hex
			_, err = hex.DecodeString(token)
			if err != nil {
				t.Errorf("token is not valid hex: %v", err)
			}
		})
	}
}

func TestKeyGenerator_GenerateAPIKey(t *testing.T) {
	kg := NewKeyGenerator()

	tests := []struct {
		name   string
		prefix string
		length int
	}{
		{"with prefix", "lsk", 16},
		{"no prefix", "", 16},
		{"long prefix", "test_api_key", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, err := kg.GenerateAPIKey(tt.prefix, tt.length)
			if err != nil {
				t.Fatalf("GenerateAPIKey failed: %v", err)
			}

			if tt.prefix != "" {
				expectedPrefix := tt.prefix + "_"
				if len(apiKey) < len(expectedPrefix) {
					t.Errorf("API key too short")
				}
				if apiKey[:len(expectedPrefix)] != expectedPrefix {
					t.Errorf("expected prefix %s, got %s", expectedPrefix, apiKey[:len(expectedPrefix)])
				}
			}
		})
	}
}

func TestKeyGenerator_GenerateSalt(t *testing.T) {
	kg := NewKeyGenerator()

	// Valid lengths
	lengths := []int{8, 16, 32}
	for _, length := range lengths {
		salt, err := kg.GenerateSalt(length)
		if err != nil {
			t.Errorf("GenerateSalt(%d) failed: %v", length, err)
		}
		if len(salt) != length {
			t.Errorf("expected salt length %d, got %d", length, len(salt))
		}
	}

	// Invalid length (too short)
	_, err := kg.GenerateSalt(4)
	if err == nil {
		t.Error("expected error for salt length < 8")
	}
}

func TestKeyGenerator_GenerateIV(t *testing.T) {
	kg := NewKeyGenerator()

	iv, err := kg.GenerateIV()
	if err != nil {
		t.Fatalf("GenerateIV failed: %v", err)
	}

	if len(iv) != 16 {
		t.Errorf("expected IV length 16, got %d", len(iv))
	}
}

func TestKeyGenerator_GenerateNonce(t *testing.T) {
	kg := NewKeyGenerator()

	nonce, err := kg.GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}

	if len(nonce) != 12 {
		t.Errorf("expected nonce length 12, got %d", len(nonce))
	}
}

func TestKeyToHexAndBack(t *testing.T) {
	kg := NewKeyGenerator()
	key, _ := kg.GenerateKey(32)

	// Convert to hex
	hexKey := KeyToHex(key)

	// Convert back
	decoded, err := KeyFromHex(hexKey)
	if err != nil {
		t.Fatalf("KeyFromHex failed: %v", err)
	}

	if string(decoded) != string(key) {
		t.Error("key doesn't match after hex conversion")
	}
}

func TestKeyFromHex_InvalidInput(t *testing.T) {
	tests := []string{
		"not-hex",
		"gg",
		"abc",      // odd length
		"12 34 56", // spaces
	}

	for _, input := range tests {
		_, err := KeyFromHex(input)
		if err == nil {
			t.Errorf("expected error for invalid hex: %s", input)
		}
	}
}

func TestKeyGenerator_Uniqueness(t *testing.T) {
	kg := NewKeyGenerator()

	// Generate 1000 keys and check for duplicates
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		key, err := kg.GenerateKey(32)
		if err != nil {
			t.Fatalf("GenerateKey failed: %v", err)
		}

		keyStr := string(key)
		if seen[keyStr] {
			t.Error("found duplicate key")
		}
		seen[keyStr] = true
	}

	if len(seen) != 1000 {
		t.Errorf("expected 1000 unique keys, got %d", len(seen))
	}
}

func TestKeyGenerator_TokenUniqueness(t *testing.T) {
	kg := NewKeyGenerator()

	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		token, err := kg.GenerateToken(16)
		if err != nil {
			t.Fatalf("GenerateToken failed: %v", err)
		}

		if seen[token] {
			t.Error("found duplicate token")
		}
		seen[token] = true
	}
}

func TestMustGenerateKey(t *testing.T) {
	// Should not panic
	key := MustGenerateKey(32)
	if len(key) != 32 {
		t.Errorf("expected key length 32, got %d", len(key))
	}
}

func TestMustGenerateAES256Key(t *testing.T) {
	key := MustGenerateAES256Key()
	if len(key) != 32 {
		t.Errorf("expected key length 32, got %d", len(key))
	}
}

func TestMustGenerateToken(t *testing.T) {
	token := MustGenerateToken(16)
	if len(token) != 32 { // hex string is 2x
		t.Errorf("expected token length 32, got %d", len(token))
	}
}

func TestKeyGenerator_ConcurrentGeneration(t *testing.T) {
	kg := NewKeyGenerator()

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			_, err := kg.GenerateKey(32)
			if err != nil {
				t.Errorf("concurrent key generation failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

// Benchmarks

func BenchmarkGenerateKey_16(b *testing.B) {
	kg := NewKeyGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = kg.GenerateKey(16)
	}
}

func BenchmarkGenerateKey_32(b *testing.B) {
	kg := NewKeyGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = kg.GenerateKey(32)
	}
}

func BenchmarkGenerateAES256Key(b *testing.B) {
	kg := NewKeyGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = kg.GenerateAES256Key()
	}
}

func BenchmarkGenerateToken(b *testing.B) {
	kg := NewKeyGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = kg.GenerateToken(16)
	}
}

func BenchmarkGenerateAPIKey(b *testing.B) {
	kg := NewKeyGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = kg.GenerateAPIKey("lsk", 16)
	}
}

func BenchmarkKeyToHex(b *testing.B) {
	key := make([]byte, 32)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = KeyToHex(key)
	}
}

func BenchmarkKeyFromHex(b *testing.B) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = KeyFromHex(hexKey)
	}
}
