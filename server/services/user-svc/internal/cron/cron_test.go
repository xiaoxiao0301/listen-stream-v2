package cron

import (
	"context"
	"testing"
	"time"

	"user-svc/internal/domain"
	"user-svc/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPlayHistoryRepository 模拟仓储
type MockPlayHistoryRepository struct {
	mock.Mock
}

func (m *MockPlayHistoryRepository) Create(ctx context.Context, history *domain.PlayHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockPlayHistoryRepository) GetByID(ctx context.Context, id string) (*domain.PlayHistory, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PlayHistory), args.Error(1)
}

func (m *MockPlayHistoryRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.PlayHistory, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PlayHistory), args.Error(1)
}

func (m *MockPlayHistoryRepository) Count(ctx context.Context, userID string) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPlayHistoryRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPlayHistoryRepository) DeleteOldest(ctx context.Context, userID string, count int) error {
	args := m.Called(ctx, userID, count)
	return args.Error(0)
}

func (m *MockPlayHistoryRepository) Cleanup(ctx context.Context, userID string, keepCount int) error {
	args := m.Called(ctx, userID, keepCount)
	return args.Error(0)
}

func (m *MockPlayHistoryRepository) GetAllUserIDs(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestCronManager_Start(t *testing.T) {
	mockRepo := new(MockPlayHistoryRepository)
	cleanupService := service.NewCleanupService(mockRepo)
	cronManager := NewCronManager(cleanupService)

	err := cronManager.Start()
	assert.NoError(t, err)

	// 清理
	cronManager.Stop()
}

func TestCronManager_RunCleanupNow(t *testing.T) {
	mockRepo := new(MockPlayHistoryRepository)
	
	// 模拟3个用户
	userIDs := []string{"user1", "user2", "user3"}
	mockRepo.On("GetAllUserIDs", mock.Anything).Return(userIDs, nil)
	
	// user1: 600条记录，需要清理
	mockRepo.On("Count", mock.Anything, "user1").Return(int64(600), nil)
	mockRepo.On("Cleanup", mock.Anything, "user1", 500).Return(nil)
	
	// user2: 400条记录，不需要清理
	mockRepo.On("Count", mock.Anything, "user2").Return(int64(400), nil)
	
	// user3: 1000条记录，需要清理
	mockRepo.On("Count", mock.Anything, "user3").Return(int64(1000), nil)
	mockRepo.On("Cleanup", mock.Anything, "user3", 500).Return(nil)

	cleanupService := service.NewCleanupService(mockRepo)
	cronManager := NewCronManager(cleanupService)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := cronManager.RunCleanupNow(ctx)
	assert.NoError(t, err)

	// 验证mock被调用
	mockRepo.AssertExpectations(t)
}

func TestCleanupService_CleanupAllUserHistories(t *testing.T) {
	mockRepo := new(MockPlayHistoryRepository)
	
	// 模拟用户数据
	userIDs := []string{"user1", "user2"}
	mockRepo.On("GetAllUserIDs", mock.Anything).Return(userIDs, nil)
	
	// user1: 600条记录
	mockRepo.On("Count", mock.Anything, "user1").Return(int64(600), nil)
	mockRepo.On("Cleanup", mock.Anything, "user1", 500).Return(nil)
	
	// user2: 300条记录
	mockRepo.On("Count", mock.Anything, "user2").Return(int64(300), nil)

	cleanupService := service.NewCleanupService(mockRepo)
	
	ctx := context.Background()
	err := cleanupService.CleanupAllUserHistories(ctx)
	assert.NoError(t, err)

	// 验证mock被调用
	mockRepo.AssertExpectations(t)
}
