package sms

import (
	"context"
	"sync"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/repository"
)

// Stats SMS发送统计
type Stats struct {
	repo  repository.SMSRecordRepository
	mu    sync.RWMutex
	cache *StatsCache
}

// StatsCache 统计缓存（避免频繁查询数据库）
type StatsCache struct {
	TotalSent       int64           // 总发送数
	TotalSuccess    int64           // 总成功数
	TotalFailed     int64           // 总失败数
	ProviderStats   map[string]int64 // 各提供商发送数
	LastUpdate      time.Time       // 最后更新时间
	CacheDuration   time.Duration   // 缓存有效期
}

// NewStats 创建统计服务
func NewStats(repo repository.SMSRecordRepository) *Stats {
	return &Stats{
		repo: repo,
		cache: &StatsCache{
			ProviderStats: make(map[string]int64),
			CacheDuration: 5 * time.Minute, // 缓存5分钟
		},
	}
}

// Record 记录发送结果
func (s *Stats) Record(ctx context.Context, phone, provider string, success bool, errorMsg string) error {
	// 创建记录
	var record *domain.SMSRecord
	if success {
		record = domain.NewSuccessSMSRecord(phone, provider)
	} else {
		record = domain.NewFailedSMSRecord(phone, provider, errorMsg)
	}

	// 验证
	if err := record.Validate(); err != nil {
		return err
	}

	// 保存到数据库
	if err := s.repo.Create(ctx, record); err != nil {
		return err
	}

	// 更新缓存
	s.updateCache(provider, success)

	return nil
}

// updateCache 更新内存缓存
func (s *Stats) updateCache(provider string, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache.TotalSent++
	if success {
		s.cache.TotalSuccess++
	} else {
		s.cache.TotalFailed++
	}

	s.cache.ProviderStats[provider]++
}

// GetStats 获取统计信息
func (s *Stats) GetStats(ctx context.Context, refresh bool) (*StatsInfo, error) {
	// 检查缓存是否有效
	s.mu.RLock()
	if !refresh && time.Since(s.cache.LastUpdate) < s.cache.CacheDuration {
		info := &StatsInfo{
			TotalSent:     s.cache.TotalSent,
			TotalSuccess:  s.cache.TotalSuccess,
			TotalFailed:   s.cache.TotalFailed,
			ProviderStats: s.copyProviderStats(s.cache.ProviderStats),
			SuccessRate:   s.calculateSuccessRate(s.cache.TotalSuccess, s.cache.TotalSent),
			CachedAt:      s.cache.LastUpdate,
		}
		s.mu.RUnlock()
		return info, nil
	}
	s.mu.RUnlock()

	// 从数据库刷新统计
	return s.refreshStats(ctx)
}

// refreshStats 从数据库刷新统计
func (s *Stats) refreshStats(ctx context.Context) (*StatsInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查询最近24小时的数据
	after := time.Now().Add(-24 * time.Hour)

	// 查询成功数
	successCount, err := s.repo.CountSuccess(ctx, after)
	if err != nil {
		return nil, err
	}

	// 查询失败数
	failedCount, err := s.repo.CountFailed(ctx, after)
	if err != nil {
		return nil, err
	}

	totalCount := successCount + failedCount

	// 查询各提供商统计
	providerStats := make(map[string]int64)
	providers := []string{domain.ProviderAliyun, domain.ProviderTencent, domain.ProviderTwilio}
	for _, provider := range providers {
		count, err := s.repo.CountByProvider(ctx, provider, after)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			providerStats[provider] = count
		}
	}

	// 更新缓存
	s.cache.TotalSent = totalCount
	s.cache.TotalSuccess = successCount
	s.cache.TotalFailed = failedCount
	s.cache.ProviderStats = providerStats
	s.cache.LastUpdate = time.Now()

	return &StatsInfo{
		TotalSent:     totalCount,
		TotalSuccess:  successCount,
		TotalFailed:   failedCount,
		ProviderStats: s.copyProviderStats(providerStats),
		SuccessRate:   s.calculateSuccessRate(successCount, totalCount),
		CachedAt:      s.cache.LastUpdate,
	}, nil
}

// copyProviderStats 复制提供商统计（避免外部修改）
func (s *Stats) copyProviderStats(stats map[string]int64) map[string]int64 {
	copy := make(map[string]int64, len(stats))
	for k, v := range stats {
		copy[k] = v
	}
	return copy
}

// calculateSuccessRate 计算成功率
func (s *Stats) calculateSuccessRate(success, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(success) / float64(total) * 100
}

// StatsInfo 统计信息
type StatsInfo struct {
	TotalSent     int64            `json:"total_sent"`     // 总发送数
	TotalSuccess  int64            `json:"total_success"`  // 总成功数
	TotalFailed   int64            `json:"total_failed"`   // 总失败数
	ProviderStats map[string]int64 `json:"provider_stats"` // 各提供商统计
	SuccessRate   float64          `json:"success_rate"`   // 成功率(%)
	CachedAt      time.Time        `json:"cached_at"`      // 缓存时间
}

// GetProviderName 获取提供商友好名称
func (i *StatsInfo) GetProviderName(provider string) string {
	record := &domain.SMSRecord{Provider: provider}
	return record.GetProviderName()
}
