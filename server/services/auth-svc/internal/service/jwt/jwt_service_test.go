package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
)

// MockUserRepository Mock用户仓储（用于测试）
type MockUserRepository struct {
	users map[string]*domain.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (m *MockUserRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Phone == phone {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *MockUserRepository) UpdateTokenVersion(ctx context.Context, id string, version int) error {
	user, ok := m.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	user.TokenVersion = version
	return nil
}

func (m *MockUserRepository) UpdateActive(ctx context.Context, id string, isActive bool) error {
	user, ok := m.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	user.IsActive = isActive
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return nil, nil
}

func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *MockUserRepository) CountActive(ctx context.Context) (int64, error) {
	count := int64(0)
	for _, user := range m.users {
		if user.IsActive {
			count++
		}
	}
	return count, nil
}

// TestGenerateTokenPair 测试生成Token对
func TestGenerateTokenPair(t *testing.T) {
	userRepo := NewMockUserRepository()
	jwtService := NewJWTService(&JWTConfig{
		Secret:           "test-secret-key",
		Issuer:           "test",
		TokenExpiry:      time.Hour,
		RefreshExpiry:    7 * 24 * time.Hour,
		IPBindingEnabled: false,
	}, userRepo)

	user := domain.NewUser("13800138000")
	ctx := context.Background()
	_ = userRepo.Create(ctx, user)

	tokenPair, err := jwtService.GenerateTokenPair(ctx, user.ID, "device-1", "192.168.1.1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("expected access token")
	}
	if tokenPair.RefreshToken == "" {
		t.Error("expected refresh token")
	}
	if tokenPair.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", tokenPair.TokenType)
	}
}

// TestValidateAccessToken 测试验证Token
func TestValidateAccessToken(t *testing.T) {
	userRepo := NewMockUserRepository()
	jwtService := NewJWTService(&JWTConfig{
		Secret:           "test-secret-key",
		Issuer:           "test",
		TokenExpiry:      time.Hour,
		RefreshExpiry:    7 * 24 * time.Hour,
		IPBindingEnabled: false,
	}, userRepo)

	user := domain.NewUser("13800138000")
	ctx := context.Background()
	_ = userRepo.Create(ctx, user)

	tokenPair, _ := jwtService.GenerateTokenPair(ctx, user.ID, "device-1", "192.168.1.1")

	claims, err := jwtService.ValidateAccessToken(ctx, tokenPair.AccessToken, "192.168.1.1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("expected user ID %s, got %s", user.ID, claims.UserID)
	}
}

// TestTokenVersionMismatch 测试Token版本不匹配
func TestTokenVersionMismatch(t *testing.T) {
	userRepo := NewMockUserRepository()
	jwtService := NewJWTService(&JWTConfig{
		Secret:           "test-secret-key",
		Issuer:           "test",
		TokenExpiry:      time.Hour,
		RefreshExpiry:    7 * 24 * time.Hour,
		IPBindingEnabled: false,
	}, userRepo)

	user := domain.NewUser("13800138000")
	ctx := context.Background()
	_ = userRepo.Create(ctx, user)

	tokenPair, _ := jwtService.GenerateTokenPair(ctx, user.ID, "device-1", "192.168.1.1")

	// 撤销Token
	_ = jwtService.RevokeUserTokens(ctx, user.ID)

	// 验证旧Token应该失败
	_, err := jwtService.ValidateAccessToken(ctx, tokenPair.AccessToken, "192.168.1.1")
	if err != ErrTokenVersionMismatch {
		t.Errorf("expected ErrTokenVersionMismatch, got %v", err)
	}
}

// TestIPBinding 测试IP绑定
func TestIPBinding(t *testing.T) {
	userRepo := NewMockUserRepository()
	jwtService := NewJWTService(&JWTConfig{
		Secret:           "test-secret-key",
		Issuer:           "test",
		TokenExpiry:      time.Hour,
		RefreshExpiry:    7 * 24 * time.Hour,
		IPBindingEnabled: true,
	}, userRepo)

	user := domain.NewUser("13800138000")
	ctx := context.Background()
	_ = userRepo.Create(ctx, user)

	tokenPair, _ := jwtService.GenerateTokenPair(ctx, user.ID, "device-1", "192.168.1.1")

	// 使用相同IP应该成功
	_, err := jwtService.ValidateAccessToken(ctx, tokenPair.AccessToken, "192.168.1.1")
	if err != nil {
		t.Errorf("expected no error for same IP, got %v", err)
	}

	// 使用不同IP应该失败
	_, err = jwtService.ValidateAccessToken(ctx, tokenPair.AccessToken, "192.168.1.2")
	if err != ErrIPMismatch {
		t.Errorf("expected ErrIPMismatch, got %v", err)
	}
}

// TestRefreshToken 测试刷新Token
func TestRefreshToken(t *testing.T) {
	userRepo := NewMockUserRepository()
	jwtService := NewJWTService(&JWTConfig{
		Secret:           "test-secret-key",
		Issuer:           "test",
		TokenExpiry:      time.Hour,
		RefreshExpiry:    7 * 24 * time.Hour,
		IPBindingEnabled: false,
	}, userRepo)

	user := domain.NewUser("13800138000")
	ctx := context.Background()
	_ = userRepo.Create(ctx, user)

	tokenPair, _ := jwtService.GenerateTokenPair(ctx, user.ID, "device-1", "192.168.1.1")

	// 等待一小段时间确保时间戳不同
	time.Sleep(time.Millisecond)

	// 刷新Token
	newTokenPair, err := jwtService.RefreshAccessToken(ctx, tokenPair.RefreshToken, "192.168.1.1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if newTokenPair.AccessToken == "" {
		t.Error("expected new access token")
	}
	if newTokenPair.RefreshToken == "" {
		t.Error("expected new refresh token")
	}
}
