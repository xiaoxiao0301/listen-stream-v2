package service

import (
	"context"
	"log"
	"time"

	"user-svc/internal/repository"
)

const (
	// MaxHistoryCount 每个用户保留的最大历史记录数
	MaxHistoryCount = 500
)

// CleanupService 清理服务
type CleanupService struct {
	historyRepo repository.PlayHistoryRepository
}

// NewCleanupService 创建清理服务
func NewCleanupService(historyRepo repository.PlayHistoryRepository) *CleanupService {
	return &CleanupService{
		historyRepo: historyRepo,
	}
}

// CleanupStats 清理统计
type CleanupStats struct {
	StartTime      time.Time
	EndTime        time.Time
	TotalUsers     int
	CleanedUsers   int
	FailedUsers    int
	TotalRecords   int64
	DeletedRecords int64
	Errors         []string
}

// CleanupAllUserHistories 清理所有用户的历史记录
func (s *CleanupService) CleanupAllUserHistories(ctx context.Context) error {
	stats := &CleanupStats{
		StartTime: time.Now(),
		Errors:    make([]string, 0),
	}

	log.Println("Starting cleanup of all user play histories...")

	// 获取所有有播放历史的用户ID
	userIDs, err := s.historyRepo.GetAllUserIDs(ctx)
	if err != nil {
		log.Printf("Failed to get user IDs: %v", err)
		return err
	}

	stats.TotalUsers = len(userIDs)
	log.Printf("Found %d users with play histories", stats.TotalUsers)

	// 遍历每个用户，执行清理
	for _, userID := range userIDs {
		// 获取用户当前的历史记录数量
		count, err := s.historyRepo.Count(ctx, userID)
		if err != nil {
			log.Printf("Failed to count histories for user %s: %v", userID, err)
			stats.FailedUsers++
			stats.Errors = append(stats.Errors, err.Error())
			continue
		}

		stats.TotalRecords += count

		// 如果超过500条，执行清理
		if count > MaxHistoryCount {
			if err := s.historyRepo.Cleanup(ctx, userID, MaxHistoryCount); err != nil {
				log.Printf("Failed to cleanup histories for user %s: %v", userID, err)
				stats.FailedUsers++
				stats.Errors = append(stats.Errors, err.Error())
				continue
			}

			deleted := count - MaxHistoryCount
			stats.DeletedRecords += deleted
			stats.CleanedUsers++
			log.Printf("Cleaned up %d records for user %s (kept %d)", deleted, userID, MaxHistoryCount)
		}
	}

	stats.EndTime = time.Now()
	duration := stats.EndTime.Sub(stats.StartTime)

	// 打印统计信息
	log.Printf("Cleanup completed in %v", duration)
	log.Printf("Statistics:")
	log.Printf("  - Total users: %d", stats.TotalUsers)
	log.Printf("  - Cleaned users: %d", stats.CleanedUsers)
	log.Printf("  - Failed users: %d", stats.FailedUsers)
	log.Printf("  - Total records: %d", stats.TotalRecords)
	log.Printf("  - Deleted records: %d", stats.DeletedRecords)

	if len(stats.Errors) > 0 {
		log.Printf("  - Errors: %d", len(stats.Errors))
		for i, errMsg := range stats.Errors {
			if i < 5 { // 只打印前5个错误
				log.Printf("    - %s", errMsg)
			}
		}
	}

	// 如果有失败的用户，返回错误信息
	if stats.FailedUsers > 0 {
		log.Printf("Cleanup completed with %d failures", stats.FailedUsers)
	} else {
		log.Println("Cleanup completed successfully")
	}

	return nil
}

// CleanupUserHistory 清理指定用户的历史记录
func (s *CleanupService) CleanupUserHistory(ctx context.Context, userID string, keepCount int) error {
	return s.historyRepo.Cleanup(ctx, userID, keepCount)
}

// CleanupAllUsers 清理所有用户的历史记录（别名方法）
func (s *CleanupService) CleanupAllUsers(ctx context.Context) error {
	return s.CleanupAllUserHistories(ctx)
}
