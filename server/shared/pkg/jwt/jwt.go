// Package jwt provides JWT token generation and validation.
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/errors"
)

// Claims represents JWT claims.
type Claims struct {
	UserID       string `json:"user_id"`
	DeviceID     string `json:"device_id"`
	TokenVersion int    `json:"token_version"`
	ClientIP     string `json:"client_ip,omitempty"` // Optional: for IP binding
	jwt.RegisteredClaims
}

// Manager handles JWT operations.
type Manager struct {
	secret        []byte
	issuer        string
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
}

// Config holds JWT manager configuration.
type Config struct {
	Secret        string
	Issuer        string
	TokenExpiry   time.Duration // Default: 1 hour
	RefreshExpiry time.Duration // Default: 7 days
}

// NewManager creates a new JWT manager.
func NewManager(cfg *Config) *Manager {
	tokenExpiry := cfg.TokenExpiry
	if tokenExpiry == 0 {
		tokenExpiry = time.Hour
	}
	
	refreshExpiry := cfg.RefreshExpiry
	if refreshExpiry == 0 {
		refreshExpiry = 7 * 24 * time.Hour
	}
	
	return &Manager{
		secret:        []byte(cfg.Secret),
		issuer:        cfg.Issuer,
		tokenExpiry:   tokenExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// GenerateToken generates a new access token.
func (m *Manager) GenerateToken(userID, deviceID string, tokenVersion int, clientIP string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:       userID,
		DeviceID:     deviceID,
		TokenVersion: tokenVersion,
		ClientIP:     clientIP,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.tokenExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	
	return tokenString, nil
}

// GenerateRefreshToken generates a new refresh token.
func (m *Manager) GenerateRefreshToken(userID, deviceID string, tokenVersion int) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:       userID,
		DeviceID:     deviceID,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}
	
	return tokenString, nil
}

// ValidateToken validates a token and returns its claims.
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	
	if err != nil {
		return nil, errors.ErrTokenInvalid.WithError(err)
	}
	
	if !token.Valid {
		return nil, errors.ErrTokenInvalid
	}
	
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.ErrTokenInvalid
	}
	
	return claims, nil
}

// ValidateRefreshToken validates a refresh token.
func (m *Manager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return m.ValidateToken(tokenString)
}

// ExtractClaims extracts claims without validation (for debugging).
func (m *Manager) ExtractClaims(tokenString string) (*Claims, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}
	
	return claims, nil
}

// RefreshToken generates a new token using a refresh token.
func (m *Manager) RefreshToken(refreshToken string, newTokenVersion int) (string, error) {
	// Validate refresh token
	claims, err := m.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", err
	}
	
	// Generate new access token with updated version
	return m.GenerateToken(claims.UserID, claims.DeviceID, newTokenVersion, claims.ClientIP)
}

// GetExpiryTime returns the expiry time for access tokens.
func (m *Manager) GetExpiryTime() time.Duration {
	return m.tokenExpiry
}

// GetRefreshExpiryTime returns the expiry time for refresh tokens.
func (m *Manager) GetRefreshExpiryTime() time.Duration {
	return m.refreshExpiry
}