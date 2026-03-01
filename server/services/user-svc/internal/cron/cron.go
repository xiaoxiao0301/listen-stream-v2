package cron

import (
	"context"
	"log"
	"time"
	"user-svc/internal/service"

	"github.com/robfig/cron/v3"
)

// CronManager 定时任务管理器
type CronManager struct {
	cron           *cron.Cron
	cleanupService *service.CleanupService
}

// NewCronManager 创建定时任务管理器
func NewCronManager(cleanupService *service.CleanupService) *CronManager {
	// 创建带秒级支持的cron（可选）
	// 或使用标准的分钟级: cron.New()
	return &CronManager{
		cron:           cron.New(cron.WithLocation(time.Local)),
		cleanupService: cleanupService,
	}
}

// Start 启动定时任务
func (m *CronManager) Start() error {
	// 每天凌晨2点执行清理任务
	// Cron格式: 分 时 日 月 周
	// "0 2 * * *" = 每天02:00:00
	_, err := m.cron.AddFunc("0 2 * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		log.Println("=== Starting scheduled cleanup job ===")
		startTime := time.Now()

		if err := m.cleanupService.CleanupAllUsers(ctx); err != nil {
			log.Printf("Cleanup job failed: %v", err)
		} else {
			duration := time.Since(startTime)
			log.Printf("Cleanup job completed successfully in %v", duration)
		}

		log.Println("=== Cleanup job finished ===")
	})
	if err != nil {
		return err
	}

	m.cron.Start()
	log.Println("Cron manager started - scheduled cleanup at 02:00 daily")
	return nil
}

// Stop 停止定时任务
func (m *CronManager) Stop() {
	ctx := m.cron.Stop()
	<-ctx.Done() // 等待所有任务完成
	log.Println("Cron manager stopped")
}

// RunCleanupNow 立即执行清理任务（用于测试或手动触发）
func (m *CronManager) RunCleanupNow(ctx context.Context) error {
	log.Println("Running cleanup job immediately...")
	return m.cleanupService.CleanupAllUsers(ctx)
}
