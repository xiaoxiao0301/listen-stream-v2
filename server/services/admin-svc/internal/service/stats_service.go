package service

import (
	"context"
	"fmt"
	"time"

	"admin-svc/internal/domain"

	"github.com/redis/go-redis/v9"
)

// StatsService 数据统计服务（聚合+实时）
type StatsService struct {
	redis *redis.Client
}

// NewStatsService 创建统计服务
func NewStatsService(redis *redis.Client) *StatsService {
	return &StatsService{
		redis: redis,
	}
}

// GetRealtimeStats 获取实时统计
func (s *StatsService) GetRealtimeStats(ctx context.Context) (*domain.RealtimeStats, error) {
	pipe := s.redis.Pipeline()
	
	onlineUsersCmd := pipe.Get(ctx, "stats:realtime:online_users")
	activeSessionsCmd := pipe.Get(ctx, "stats:realtime:active_sessions")
	requestsPerMinCmd := pipe.Get(ctx, "stats:realtime:requests_per_min")
	errorsPerMinCmd := pipe.Get(ctx, "stats:realtime:errors_per_min")
	avgResponseTimeCmd := pipe.Get(ctx, "stats:realtime:avg_response_time")
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("get realtime stats: %w", err)
	}
	
	stats := &domain.RealtimeStats{
		Timestamp: time.Now(),
	}
	
	if val, err := onlineUsersCmd.Int64(); err == nil {
		stats.OnlineUsers = val
	}
	if val, err := activeSessionsCmd.Int64(); err == nil {
		stats.ActiveSessions = val
	}
	if val, err := requestsPerMinCmd.Int64(); err == nil {
		stats.RequestsPerMin = val
	}
	if val, err := errorsPerMinCmd.Int64(); err == nil {
		stats.ErrorsPerMin = val
	}
	if val, err := avgResponseTimeCmd.Int64(); err == nil {
		stats.AvgResponseTime = val
	}
	
	return stats, nil
}

// UpdateRealtimeStats 更新实时统计
func (s *StatsService) UpdateRealtimeStats(ctx context.Context, key string, value int64) error {
	fullKey := fmt.Sprintf("stats:realtime:%s", key)
	return s.redis.Set(ctx, fullKey, value, 5*time.Minute).Err()
}

// IncrementCounter 增加计数器
func (s *StatsService) IncrementCounter(ctx context.Context, key string) error {
	fullKey := fmt.Sprintf("stats:realtime:%s", key)
	return s.redis.Incr(ctx, fullKey).Err()
}

// RecordResponseTime 记录响应时间（使用滑动窗口计算平均值）
func (s *StatsService) RecordResponseTime(ctx context.Context, duration int64) error {
	key := "stats:realtime:response_times"
	now := time.Now().Unix()
	
	// 添加到Sorted Set
	err := s.redis.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: duration,
	}).Err()
	if err != nil {
		return err
	}
	
	// 清理1分钟之前的数据
	minScore := now - 60
	_ = s.redis.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", minScore)).Err()
	
	// 计算平均值
	return s.calculateAvgResponseTime(ctx)
}

// calculateAvgResponseTime 计算平均响应时间
func (s *StatsService) calculateAvgResponseTime(ctx context.Context) error {
	key := "stats:realtime:response_times"
	
	// 获取所有响应时间
	members, err := s.redis.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		return err
	}
	
	if len(members) == 0 {
		return s.redis.Set(ctx, "stats:realtime:avg_response_time", 0, 5*time.Minute).Err()
	}
	
	var sum int64
	for _, member := range members {
		var duration int64
		fmt.Sscanf(member, "%d", &duration)
		sum += duration
	}
	
	avg := sum / int64(len(members))
	return s.redis.Set(ctx, "stats:realtime:avg_response_time", avg, 5*time.Minute).Err()
}

// AggregateDailyStats 聚合每日统计（由Cron任务调用）
// 这个方法返回需要聚合的数据，实际写入由Repository完成
func (s *StatsService) AggregateDailyStats(ctx context.Context, date time.Time) (*domain.DailyStats, error) {
	stats := &domain.DailyStats{
		Date:      date,
		CreatedAt: time.Now(),
	}
	
	// 从Redis获取今日统计数据
	dateKey := date.Format("2006-01-02")
	
	pipe := s.redis.Pipeline()
	
	totalUsersCmd := pipe.Get(ctx, fmt.Sprintf("daily:%s:total_users", dateKey))
	newUsersCmd := pipe.Get(ctx, fmt.Sprintf("daily:%s:new_users", dateKey))
	activeUsersCmd := pipe.Get(ctx, fmt.Sprintf("daily:%s:active_users", dateKey))
	totalRequestsCmd := pipe.Get(ctx, fmt.Sprintf("daily:%s:total_requests", dateKey))
	successRequestsCmd := pipe.Get(ctx, fmt.Sprintf("daily:%s:success_requests", dateKey))
	failedRequestsCmd := pipe.Get(ctx, fmt.Sprintf("daily:%s:failed_requests", dateKey))
	totalFavoritesCmd := pipe.Get(ctx, fmt.Sprintf("daily:%s:total_favorites", dateKey))
	totalPlaylistsCmd := pipe.Get(ctx, fmt.Sprintf("daily:%s:total_playlists", dateKey))
	totalPlaysCmd := pipe.Get(ctx, fmt.Sprintf("daily:%s:total_plays", dateKey))
	
	// 响应时间需要从Sorted Set计算
	avgResponseTimeKey := fmt.Sprintf("daily:%s:response_times", dateKey)
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("aggregate daily stats: %w", err)
	}
	
	// 读取数据
	if val, err := totalUsersCmd.Int64(); err == nil {
		stats.TotalUsers = val
	}
	if val, err := newUsersCmd.Int64(); err == nil {
		stats.NewUsers = val
	}
	if val, err := activeUsersCmd.Int64(); err == nil {
		stats.ActiveUsers = val
	}
	if val, err := totalRequestsCmd.Int64(); err == nil {
		stats.TotalRequests = val
	}
	if val, err := successRequestsCmd.Int64(); err == nil {
		stats.SuccessRequests = val
	}
	if val, err := failedRequestsCmd.Int64(); err == nil {
		stats.FailedRequests = val
	}
	if val, err := totalFavoritesCmd.Int64(); err == nil {
		stats.TotalFavorites = val
	}
	if val, err := totalPlaylistsCmd.Int64(); err == nil {
		stats.TotalPlaylists = val
	}
	if val, err := totalPlaysCmd.Int64(); err == nil {
		stats.TotalPlays = val
	}
	
	// 计算平均响应时间
	responseTimeMembers, err := s.redis.ZRange(ctx, avgResponseTimeKey, 0, -1).Result()
	if err == nil && len(responseTimeMembers) > 0 {
		var sum int64
		for _, member := range responseTimeMembers {
			var duration int64
			fmt.Sscanf(member, "%d", &duration)
			sum += duration
		}
		stats.AvgResponseTime = sum / int64(len(responseTimeMembers))
	}
	
	// 计算错误率
	stats.CalculateErrorRate()
	
	return stats, nil
}

// IncrementDailyCounter 增加每日计数器
func (s *StatsService) IncrementDailyCounter(ctx context.Context, date time.Time, metric string) error {
	dateKey := date.Format("2006-01-02")
	key := fmt.Sprintf("daily:%s:%s", dateKey, metric)
	
	err := s.redis.Incr(ctx, key).Err()
	if err != nil {
		return err
	}
	
	// 设置过期时间（保留7天）
	return s.redis.Expire(ctx, key, 7*24*time.Hour).Err()
}

// RecordDailyResponseTime 记录每日响应时间
func (s *StatsService) RecordDailyResponseTime(ctx context.Context, date time.Time, duration int64) error {
	dateKey := date.Format("2006-01-02")
	key := fmt.Sprintf("daily:%s:response_times", dateKey)
	
	err := s.redis.ZAdd(ctx, key, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: duration,
	}).Err()
	if err != nil {
		return err
	}
	
	// 设置过期时间
	return s.redis.Expire(ctx, key, 7*24*time.Hour).Err()
}
