package jwt

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	cfg := &Config{
		Secret: "test-secret-key-at-least-32-bytes-long-for-security",
		Issuer: "test-issuer",
	}
	
	mgr := NewManager(cfg)
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
	
	if mgr.issuer != cfg.Issuer {
		t.Errorf("issuer = %v, want %v", mgr.issuer, cfg.Issuer)
	}
	
	// Check default expiry times
	if mgr.tokenExpiry != time.Hour {
		t.Errorf("tokenExpiry = %v, want 1h", mgr.tokenExpiry)
	}
	if mgr.refreshExpiry != 7*24*time.Hour {
		t.Errorf("refreshExpiry = %v, want 7d", mgr.refreshExpiry)
	}
}

func TestNewManager_CustomExpiry(t *testing.T) {
	cfg := &Config{
		Secret:        "test-secret",
		Issuer:        "test",
		TokenExpiry:   2 * time.Hour,
		RefreshExpiry: 14 * 24 * time.Hour,
	}
	
	mgr := NewManager(cfg)
	if mgr.tokenExpiry != 2*time.Hour {
		t.Errorf("tokenExpiry = %v, want 2h", mgr.tokenExpiry)
	}
	if mgr.refreshExpiry != 14*24*time.Hour {
		t.Errorf("refreshExpiry = %v, want 14d", mgr.refreshExpiry)
	}
}

func TestGenerateToken(t *testing.T) {
	mgr := NewManager(&Config{
		Secret: "test-secret-key-at-least-32-bytes-long-for-security",
		Issuer: "test-issuer",
	})
	
	token, err := mgr.GenerateToken("user123", "device456", 1, "192.168.1.1")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	
	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}
}

func TestValidateToken(t *testing.T) {
	mgr := NewManager(&Config{
		Secret: "test-secret-key-at-least-32-bytes-long-for-security",
		Issuer: "test-issuer",
	})
	
	userID := "user123"
	deviceID := "device456"
	tokenVersion := 1
	clientIP := "192.168.1.1"
	
	token, err := mgr.GenerateToken(userID, deviceID, tokenVersion, clientIP)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	
	claims, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.DeviceID != deviceID {
		t.Errorf("DeviceID = %v, want %v", claims.DeviceID, deviceID)
	}
	if claims.TokenVersion != tokenVersion {
		t.Errorf("TokenVersion = %v, want %v", claims.TokenVersion, tokenVersion)
	}
	if claims.ClientIP != clientIP {
		t.Errorf("ClientIP = %v, want %v", claims.ClientIP, clientIP)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	mgr := NewManager(&Config{
		Secret: "test-secret-key-at-least-32-bytes-long-for-security",
		Issuer: "test-issuer",
	})
	
	_, err := mgr.ValidateToken("invalid.token.here")
	if err == nil {
		t.Error("ValidateToken() should return error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	mgr1 := NewManager(&Config{
		Secret: "secret-key-1-at-least-32-bytes-long-here",
		Issuer: "test",
	})
	
	token, _ := mgr1.GenerateToken("user123", "device456", 1, "")
	
	mgr2 := NewManager(&Config{
		Secret: "secret-key-2-different-32-bytes-long-key",
		Issuer: "test",
	})
	
	_, err := mgr2.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should fail with wrong secret")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	mgr := NewManager(&Config{
		Secret: "test-secret-key-at-least-32-bytes-long-for-security",
		Issuer: "test-issuer",
	})
	
	token, err := mgr.GenerateRefreshToken("user123", "device456", 1)
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}
	
	if token == "" {
		t.Error("GenerateRefreshToken() returned empty token")
	}
}

func TestRefreshToken(t *testing.T) {
	mgr := NewManager(&Config{
		Secret: "test-secret-key-at-least-32-bytes-long-for-security",
		Issuer: "test-issuer",
	})
	
	refreshToken, err := mgr.GenerateRefreshToken("user123", "device456", 1)
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}
	
	newToken, err := mgr.RefreshToken(refreshToken, 2)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	
	if newToken == "" {
		t.Error("RefreshToken() returned empty token")
	}
	
	// Validate new token has updated version
	claims, err := mgr.ValidateToken(newToken)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	
	if claims.TokenVersion != 2 {
		t.Errorf("TokenVersion = %v, want 2", claims.TokenVersion)
	}
}

func TestExtractClaims(t *testing.T) {
	mgr := NewManager(&Config{
		Secret: "test-secret-key-at-least-32-bytes-long-for-security",
		Issuer: "test-issuer",
	})
	
	token, _ := mgr.GenerateToken("user123", "device456", 1, "")
	
	claims, err := mgr.ExtractClaims(token)
	if err != nil {
		t.Fatalf("ExtractClaims() error = %v", err)
	}
	
	if claims.UserID != "user123" {
		t.Errorf("UserID = %v, want user123", claims.UserID)
	}
}

func TestGetExpiryTime(t *testing.T) {
	mgr := NewManager(&Config{
		Secret:      "test-secret",
		TokenExpiry: 30 * time.Minute,
	})
	
	if mgr.GetExpiryTime() != 30*time.Minute {
		t.Errorf("GetExpiryTime() = %v, want 30m", mgr.GetExpiryTime())
	}
}

func TestGetRefreshExpiryTime(t *testing.T) {
	mgr := NewManager(&Config{
		Secret:        "test-secret",
		RefreshExpiry: 7 * 24 * time.Hour,
	})
	
	if mgr.GetRefreshExpiryTime() != 7*24*time.Hour {
		t.Errorf("GetRefreshExpiryTime() = %v, want 7d", mgr.GetRefreshExpiryTime())
	}
}
