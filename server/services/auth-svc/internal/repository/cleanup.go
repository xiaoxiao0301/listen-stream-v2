package repository

import (
	"context"
	"fmt"
	"log"
	"time"
)

// CleanupService 数据清理服务
type CleanupService struct {
	smsVerifyRepo SMSVerificationRepository
	deviceRepo    DeviceRepository
	smsRecordRepo SMSRecordRepository
}

// NewCleanupService 创建清理服务
func NewCleanupService(
	smsVerifyRepo SMSVerificationRepository,
	deviceRepo DeviceRepository,
	smsRecordRepo SMSRecordRepository,
) *CleanupService {
	return &CleanupService{
		smsVerifyRepo: smsVerifyRepo,
		deviceRepo:    deviceRepo,
		smsRecordRepo: smsRecordRepo,
	}
}

// CleanExpiredSMSVerifications 清理过期的短信验证码
// 推荐：每小时执行一次
func (s *CleanupService) CleanExpiredSMSVerifications(ctx context.Context) error {
	// 删除1小时前过期的验证码
	before := time.Now().Add(-1 * time.Hour)
	err := s.smsVerifyRepo.DeleteExpired(ctx, before)
	if err != nil {
		return fmt.Errorf("delete expired sms verifications: %w", err)
	}
	log.Printf("Cleaned expired SMS verifications before %v", before)
	return nil
}

// CleanInactiveDevices 清理不活跃设备
// 推荐：每天执行一次
func (s *CleanupService) CleanInactiveDevices(ctx context.Context) error {
	// 删除90天未登录的设备
	before := time.Now().Add(-90 * 24 * time.Hour)
	err := s.deviceRepo.DeleteInactive(ctx, before)
	if err != nil {
		return fmt.Errorf("delete inactive devices: %w", err)
	}
	log.Printf("Cleaned inactive devices before %v", before)
	return nil
}

// ArchiveOldSMSRecords 归档旧的短信记录
// 推荐：每月执行一次
func (s *CleanupService) ArchiveOldSMSRecords(ctx context.Context) error {
	// 删除90天前的短信记录
	before := time.Now().Add(-90 * 24 * time.Hour)
	err := s.smsRecordRepo.DeleteOld(ctx, before)
	if err != nil {
		return fmt.Errorf("delete old sms records: %w", err)
	}
	log.Printf("Archived old SMS records before %v", before)
	return nil
}

// RunAllCleanupTasks 执行所有清理任务
func (s *CleanupService) RunAllCleanupTasks(ctx context.Context) error {
	// 清理过期验证码
	if err := s.CleanExpiredSMSVerifications(ctx); err != nil {
		log.Printf("Error cleaning expired SMS verifications: %v", err)
	}

	// 清理不活跃设备
	if err := s.CleanInactiveDevices(ctx); err != nil {
		log.Printf("Error cleaning inactive devices: %v", err)
	}

	// 归档旧短信记录
	if err := s.ArchiveOldSMSRecords(ctx); err != nil {
		log.Printf("Error archiving old SMS records: %v", err)
	}

	return nil
}

// StartScheduledCleanup 启动定期清理任务
// 这个函数会阻塞，应该在goroutine中运行
func (s *CleanupService) StartScheduledCleanup(ctx context.Context) {
	// 每小时清理过期验证码
	hourlyTicker := time.NewTicker(time.Hour)
	defer hourlyTicker.Stop()

	// 每天清理不活跃设备（凌晨2点）
	dailyTicker := time.NewTicker(24 * time.Hour)
	defer dailyTicker.Stop()

	// 每30天归档旧短信记录
	monthlyTicker := time.NewTicker(30 * 24 * time.Hour)
	defer monthlyTicker.Stop()

	log.Println("Started scheduled cleanup service")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopped scheduled cleanup service")
			return

		case <-hourlyTicker.C:
			if err := s.CleanExpiredSMSVerifications(ctx); err != nil {
				log.Printf("Hourly cleanup error: %v", err)
			}

		case <-dailyTicker.C:
			if err := s.CleanInactiveDevices(ctx); err != nil {
				log.Printf("Daily cleanup error: %v", err)
			}

		case <-monthlyTicker.C:
			if err := s.ArchiveOldSMSRecords(ctx); err != nil {
				log.Printf("Monthly cleanup error: %v", err)
			}
		}
	}
}
