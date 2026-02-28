package jwt

import (
	"context"
	"fmt"
	"time"

	sharedJWT "github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/jwt"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/repository"
)

// JWTService JWT服务接口
type JWTService interface {
	// GenerateTokenPair 生成Token对（AccessToken + RefreshToken）
	GenerateTokenPair(ctx context.Context, userID, deviceID, clientIP string) (*TokenPair, error)

	// ValidateAccessToken 验证AccessToken
	ValidateAccessToken(ctx context.Context, token, clientIP string) (*TokenClaims, error)

	// ValidateRefreshToken 验证RefreshToken
	ValidateRefreshToken(ctx context.Context, token string) (*TokenClaims, error)

	// RefreshAccessToken 使用RefreshToken刷新AccessToken
	RefreshAccessToken(ctx context.Context, refreshToken, clientIP string) (*TokenPair, error)

	// RevokeUserTokens 撤销用户的所有Token（递增版本号）
	RevokeUserTokens(ctx context.Context, userID string) error

	// GetTokenExpiry 获取Token过期时间
	GetTokenExpiry() time.Duration
}

// jwtService JWT服务实现
type jwtService struct {
	jwtManager     *sharedJWT.Manager
	userRepo       repository.UserRepository
	ipBindingEnabled bool
}

// JWTConfig JWT服务配置
type JWTConfig struct {
	Secret           string
	Issuer           string
	TokenExpiry      time.Duration
	RefreshExpiry    time.Duration
	IPBindingEnabled bool // 是否启用IP绑定验证
}

// NewJWTService 创建JWT服务
func NewJWTService(
	config *JWTConfig,
	userRepo repository.UserRepository,
) JWTService {
	jwtManager := sharedJWT.NewManager(&sharedJWT.Config{
		Secret:        config.Secret,
		Issuer:        config.Issuer,
		TokenExpiry:   config.TokenExpiry,
		RefreshExpiry: config.RefreshExpiry,
	})

	return &jwtService{
		jwtManager:       jwtManager,
		userRepo:         userRepo,
		ipBindingEnabled: config.IPBindingEnabled,
	}
}

// TokenPair Token对
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// TokenClaims Token声明（验证后返回）
type TokenClaims struct {
	UserID       string
	DeviceID     string
	TokenVersion int
	ClientIP     string
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

// GenerateTokenPair 生成Token对
func (s *jwtService) GenerateTokenPair(ctx context.Context, userID, deviceID, clientIP string) (*TokenPair, error) {
	// 1. 获取用户信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// 2. 检查用户是否可以登录
	if !user.CanLogin() {
		return nil, domain.ErrUserInactive
	}

	// 3. 生成AccessToken
	accessToken, err := s.jwtManager.GenerateToken(userID, deviceID, user.TokenVersion, clientIP)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 4. 生成RefreshToken
	refreshToken, err := s.jwtManager.GenerateRefreshToken(userID, deviceID, user.TokenVersion)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// 5. 计算过期时间
	expiresAt := time.Now().Add(s.jwtManager.GetExpiryTime())

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// ValidateAccessToken 验证AccessToken
func (s *jwtService) ValidateAccessToken(ctx context.Context, token, clientIP string) (*TokenClaims, error) {
	// 1. 解析Token
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}

	// 2. 获取用户信息
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	// 3. 检查用户是否激活
	if !user.CanLogin() {
		return nil, domain.ErrUserInactive
	}

	// 4. 检查Token版本（重要：支持全局撤销）
	if !user.IsTokenVersionValid(claims.TokenVersion) {
		return nil, ErrTokenVersionMismatch
	}

	// 5. 检查IP绑定（可选）
	if s.ipBindingEnabled && claims.ClientIP != "" && claims.ClientIP != clientIP {
		return nil, ErrIPMismatch
	}

	// 6. 返回Token声明
	return &TokenClaims{
		UserID:       claims.UserID,
		DeviceID:     claims.DeviceID,
		TokenVersion: claims.TokenVersion,
		ClientIP:     claims.ClientIP,
		IssuedAt:     claims.IssuedAt.Time,
		ExpiresAt:    claims.ExpiresAt.Time,
	}, nil
}

// ValidateRefreshToken 验证RefreshToken
func (s *jwtService) ValidateRefreshToken(ctx context.Context, token string) (*TokenClaims, error) {
	// 1. 解析Token
	claims, err := s.jwtManager.ValidateRefreshToken(token)
	if err != nil {
		return nil, fmt.Errorf("validate refresh token: %w", err)
	}

	// 2. 获取用户信息
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	// 3. 检查用户是否激活
	if !user.CanLogin() {
		return nil, domain.ErrUserInactive
	}

	// 4. 检查Token版本
	if !user.IsTokenVersionValid(claims.TokenVersion) {
		return nil, ErrTokenVersionMismatch
	}

	// 5. 返回Token声明
	return &TokenClaims{
		UserID:       claims.UserID,
		DeviceID:     claims.DeviceID,
		TokenVersion: claims.TokenVersion,
		ClientIP:     claims.ClientIP,
		IssuedAt:     claims.IssuedAt.Time,
		ExpiresAt:    claims.ExpiresAt.Time,
	}, nil
}

// RefreshAccessToken 刷新AccessToken
func (s *jwtService) RefreshAccessToken(ctx context.Context, refreshToken, clientIP string) (*TokenPair, error) {
	// 1. 验证RefreshToken
	claims, err := s.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("validate refresh token: %w", err)
	}

	// 2. 生成新的Token对
	return s.GenerateTokenPair(ctx, claims.UserID, claims.DeviceID, clientIP)
}

// RevokeUserTokens 撤销用户的所有Token
func (s *jwtService) RevokeUserTokens(ctx context.Context, userID string) error {
	// 1. 获取用户信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	// 2. 递增Token版本号
	user.IncrementTokenVersion()

	// 3. 更新到数据库
	if err := s.userRepo.UpdateTokenVersion(ctx, userID, user.TokenVersion); err != nil {
		return fmt.Errorf("update token version: %w", err)
	}

	return nil
}

// GetTokenExpiry 获取Token过期时间
func (s *jwtService) GetTokenExpiry() time.Duration {
	return s.jwtManager.GetExpiryTime()
}

// 自定义错误
var (
	// ErrTokenVersionMismatch Token版本不匹配（Token已被撤销）
	ErrTokenVersionMismatch = fmt.Errorf("token version mismatch")

	// ErrIPMismatch IP地址不匹配（可能是Token被盗用）
	ErrIPMismatch = fmt.Errorf("client IP mismatch")
)
