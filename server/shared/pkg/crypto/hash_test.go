package crypto

import (
	"strings"
	"testing"
	"time"
)

func TestPasswordHasher_HashAndVerify(t *testing.T) {
	hasher := NewPasswordHasher()
	
	tests := []struct {
		name     string
		password string
	}{
		{"simple password", "password123"},
		{"complex password", "P@ssw0rd!2023#Complex"},
		{"unicode password", "密码123🔐"},
		{"long password", strings.Repeat("longpass", 20)},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Hash password
			hash, err := hasher.Hash(tt.password)
			if err != nil {
				t.Fatalf("hashing failed: %v", err)
			}
			
			if hash == "" {
				t.Error("hash is empty")
			}
			
			// Verify hash format
			if !strings.HasPrefix(hash, "$argon2id$") {
				t.Error("hash doesn't start with $argon2id$")
			}
			
			// Verify correct password
			match, err := hasher.Verify(tt.password, hash)
			if err != nil {
				t.Fatalf("verification failed: %v", err)
			}
			if !match {
				t.Error("correct password should match hash")
			}
			
			// Verify incorrect password
			wrongPassword := tt.password + "wrong"
			match, err = hasher.Verify(wrongPassword, hash)
			if err != nil {
				t.Fatalf("verification failed: %v", err)
			}
			if match {
				t.Error("wrong password should not match hash")
			}
		})
	}
}

func TestPasswordHasher_EmptyPassword(t *testing.T) {
	hasher := NewPasswordHasher()
	
	// Hashing empty password should fail
	_, err := hasher.Hash("")
	if err == nil {
		t.Error("expected error for empty password")
	}
	
	// Verifying empty password should fail
	_, err = hasher.Verify("", "somehash")
	if err == nil {
		t.Error("expected error for empty password")
	}
}

func TestPasswordHasher_DifferentHashes(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "samepassword"
	
	// Hash same password twice
	hash1, _ := hasher.Hash(password)
	hash2, _ := hasher.Hash(password)
	
	// Should produce different hashes due to random salts
	if hash1 == hash2 {
		t.Error("hashing same password twice should produce different hashes")
	}
	
	// But both should verify correctly
	match1, _ := hasher.Verify(password, hash1)
	match2, _ := hasher.Verify(password, hash2)
	
	if !match1 || !match2 {
		t.Error("both hashes should verify correctly")
	}
}

func TestPasswordHasher_VerifyOrError(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "testpassword"
	hash, _ := hasher.Hash(password)
	
	// Correct password should not return error
	err := hasher.VerifyOrError(password, hash)
	if err != nil {
		t.Errorf("correct password should not error: %v", err)
	}
	
	// Wrong password should return ErrMismatchedHash
	err = hasher.VerifyOrError("wrongpassword", hash)
	if err != ErrMismatchedHash {
		t.Errorf("expected ErrMismatchedHash, got %v", err)
	}
}

func TestPasswordHasher_InvalidHash(t *testing.T) {
	hasher := NewPasswordHasher()
	
	invalidHashes := []string{
		"",
		"invalid",
		"$argon2id$invalid",
		"$invalid$v=19$m=65536,t=3,p=2$salt$hash",
		"$argon2id$v=9999$m=65536,t=3,p=2$salt$hash",
		"$argon2id$v=19$m=invalid$salt$hash",
	}
	
	for _, hash := range invalidHashes {
		t.Run(hash, func(t *testing.T) {
			_, err := hasher.Verify("password", hash)
			if err == nil {
				t.Error("expected error for invalid hash")
			}
		})
	}
}

func TestPasswordHasher_NeedsRehash(t *testing.T) {
	// Create hasher with default params
	hasher := NewPasswordHasher()
	password := "testpassword"
	hash, _ := hasher.Hash(password)
	
	// Same params should not need rehash
	needsRehash, err := hasher.NeedsRehash(hash)
	if err != nil {
		t.Fatalf("NeedsRehash failed: %v", err)
	}
	if needsRehash {
		t.Error("hash with same params should not need rehash")
	}
	
	// Create hasher with different params
	highSecHasher := NewPasswordHasherWithParams(HighSecurityArgon2Params())
	
	// Should need rehash with different params
	needsRehash, err = highSecHasher.NeedsRehash(hash)
	if err != nil {
		t.Fatalf("NeedsRehash failed: %v", err)
	}
	if !needsRehash {
		t.Error("hash with different params should need rehash")
	}
}

func TestPasswordHasher_CustomParams(t *testing.T) {
	params := &Argon2Params{
		Memory:      32 * 1024,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
	
	hasher := NewPasswordHasherWithParams(params)
	password := "testpassword"
	
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("hashing failed: %v", err)
	}
	
	match, err := hasher.Verify(password, hash)
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}
	if !match {
		t.Error("password should match hash")
	}
}

func TestHashPassword_ConvenienceFunction(t *testing.T) {
	password := "testpassword"
	
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	
	match, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !match {
		t.Error("password should match hash")
	}
}

func TestPasswordHasher_Timing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing test in short mode")
	}
	
	hasher := NewPasswordHasher()
	password := "testpassword"
	
	// Hash should complete within reasonable time (< 500ms)
	start := time.Now()
	hash, err := hasher.Hash(password)
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("hashing failed: %v", err)
	}
	
	t.Logf("Hashing took: %v", duration)
	
	if duration > 500*time.Millisecond {
		t.Errorf("hashing took too long: %v (should be < 500ms)", duration)
	}
	
	// Verify should also be reasonably fast
	start = time.Now()
	_, err = hasher.Verify(password, hash)
	duration = time.Since(start)
	
	if err != nil {
		t.Fatalf("verification failed: %v", err)
	}
	
	t.Logf("Verification took: %v", duration)
	
	if duration > 500*time.Millisecond {
		t.Errorf("verification took too long: %v (should be < 500ms)", duration)
	}
}

// Benchmarks

func BenchmarkPasswordHash(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "benchmarkpassword"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.Hash(password)
	}
}

func BenchmarkPasswordVerify(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "benchmarkpassword"
	hash, _ := hasher.Hash(password)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.Verify(password, hash)
	}
}

func BenchmarkPasswordHashHighSecurity(b *testing.B) {
	hasher := NewPasswordHasherWithParams(HighSecurityArgon2Params())
	password := "benchmarkpassword"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.Hash(password)
	}
}
