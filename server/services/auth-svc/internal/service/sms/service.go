package sms

import (
	"context"
	"fmt"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/repository"
)

// Service SMS服务（整合所有功能）
type Service struct {
	chain              *FallbackChain
	stats              *Stats
	verificationRepo   repository.SMSVerificationRepository
	config             *Config
}

// NewService 创建SMS服务
func NewService(
	config *Config,
	verificationRepo repository.SMSVerificationRepository,
	recordRepo repository.SMSRecordRepository,
) *Service {
	// 创建提供商
	providers := []Provider{
		NewAliyunProvider(config.Aliyun),
		NewTencentProvider(config.Tencent),
		NewTwilioProvider(config.Twilio),
	}

	// 创建Fallback链
	chain := NewFallbackChain(providers, config.FallbackEnabled)

	// 创建统计服务
	stats := NewStats(recordRepo)

	return &Service{
		chain:            chain,
		stats:            stats,
		verificationRepo: verificationRepo,
		config:           config,
	}
}

// SendVerificationCode 发送验证码
func (s *Service) SendVerificationCode(ctx context.Context, phone string) (*SendCodeResult, error) {
	// 1. 检查发送频率限制（60秒内不能重复发送）
	recentCount, err := s.verificationRepo.CountRecent(ctx, phone, time.Now().Add(-domain.SMSCodeRateLimit))
	if err != nil {
		return nil, fmt.Errorf("check rate limit: %w", err)
	}
	if recentCount > 0 {
		return nil, domain.ErrSMSTooFrequent
	}

	// 2. 创建验证码
	verification, err := domain.NewSMSVerification(phone)
	if err != nil {
		return nil, fmt.Errorf("create verification: %w", err)
	}

	// 3. 发送短信（带Fallback）
	sendResult, err := s.chain.Send(ctx, phone, verification.Code)
	
	// 4. 记录发送结果
	if sendResult != nil {
		// 记录统计（异步，不阻塞主流程）
		go func() {
			recordCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var errorMsg string
			if err != nil {
				errorMsg = err.Error()
			} else if len(sendResult.Errors) > 0 {
				errorMsg = fmt.Sprintf("fallback attempts: %v", sendResult.Errors)
			}

			_ = s.stats.Record(recordCtx, phone, sendResult.Provider, sendResult.Success, errorMsg)
		}()
	}

	// 5. 处理发送失败
	if err != nil {
		return &SendCodeResult{
			Success:      false,
			Provider:     sendResult.Provider,
			Attempts:     sendResult.Attempts,
			TotalLatency: sendResult.TotalLatency,
			Errors:       sendResult.Errors,
		}, err
	}

	// 6. 保存验证码到数据库
	if err := s.verificationRepo.Create(ctx, verification); err != nil {
		return nil, fmt.Errorf("save verification: %w", err)
	}

	return &SendCodeResult{
		Success:      true,
		Provider:     sendResult.Provider,
		Attempts:     sendResult.Attempts,
		TotalLatency: sendResult.TotalLatency,
		ExpiresAt:    verification.ExpiresAt,
	}, nil
}

// VerifyCode 验证验证码
func (s *Service) VerifyCode(ctx context.Context, phone, code string) error {
	// 1. 获取最新的验证码
	verification, err := s.verificationRepo.GetLatest(ctx, phone)
	if err != nil {
		return domain.ErrInvalidSMSCode
	}

	// 2. 验证验证码
	if err := verification.Verify(code); err != nil {
		return err
	}

	// 3. 标记为已使用
	if err := verification.MarkAsUsed(); err != nil {
		return err
	}

	// 4. 更新数据库
	if err := s.verificationRepo.MarkAsUsed(ctx, verification.ID, *verification.UsedAt); err != nil {
		return fmt.Errorf("mark as used: %w", err)
	}

	return nil
}

// GetStats 获取统计信息
func (s *Service) GetStats(ctx context.Context, refresh bool) (*StatsInfo, error) {
	return s.stats.GetStats(ctx, refresh)
}

// GetAvailableProviders 获取可用的提供商列表
func (s *Service) GetAvailableProviders() []string {
	return s.chain.GetAvailableProviders()
}

// CleanupExpired 清理过期的验证码
func (s *Service) CleanupExpired(ctx context.Context) error {
	before := time.Now().Add(-24 * time.Hour) // 清理24小时前的
	return s.verificationRepo.DeleteExpired(ctx, before)
}

// SendCodeResult 发送验证码结果
type SendCodeResult struct {
	Success      bool          `json:"success"`                 // 是否成功
	Provider     string        `json:"provider"`                // 使用的提供商
	Attempts     int           `json:"attempts"`                // 尝试次数
	TotalLatency time.Duration `json:"total_latency"`           // 总延迟
	ExpiresAt    time.Time     `json:"expires_at,omitempty"`    // 过期时间（成功时）
	Errors       []string      `json:"errors,omitempty"`        // 错误信息（失败时）
}

// IsSuccess 是否成功
func (r *SendCodeResult) IsSuccess() bool {
	return r.Success
}

// GetProviderName 获取提供商友好名称
func (r *SendCodeResult) GetProviderName() string {
	record := &domain.SMSRecord{Provider: r.Provider}
	return record.GetProviderName()
}
