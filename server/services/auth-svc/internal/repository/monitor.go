package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	pool *pgxpool.Pool
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(pool *pgxpool.Pool) *HealthChecker {
	return &HealthChecker{pool: pool}
}

// HealthStatus 健康状态
type HealthStatus struct {
	Healthy          bool          `json:"healthy"`
	Timestamp        time.Time     `json:"timestamp"`
	ResponseTime     time.Duration `json:"response_time_ms"`
	Error            string        `json:"error,omitempty"`
	PoolStats        PoolStatistics `json:"pool_stats"`
}

// PoolStatistics 连接池统计信息
type PoolStatistics struct {
	AcquireCount         int64   `json:"acquire_count"`          // 获取连接的总次数
	AcquireDuration      int64   `json:"acquire_duration_ns"`    // 获取连接的总耗时（纳秒）
	AcquiredConns        int32   `json:"acquired_conns"`         // 当前已获取的连接数
	CanceledAcquireCount int64   `json:"canceled_acquire_count"` // 取消获取连接的次数
	ConstructingConns    int32   `json:"constructing_conns"`     // 正在创建的连接数
	EmptyAcquireCount    int64   `json:"empty_acquire_count"`    // 空闲连接池获取次数
	IdleConns            int32   `json:"idle_conns"`             // 空闲连接数
	MaxConns             int32   `json:"max_conns"`              // 最大连接数
	TotalConns           int32   `json:"total_conns"`            // 总连接数
	NewConnsCount        int64   `json:"new_conns_count"`        // 新建连接的次数
	MaxLifetimeDestroyCount int64 `json:"max_lifetime_destroy_count"` // 因超过生命周期而销毁的连接数
	MaxIdleDestroyCount  int64   `json:"max_idle_destroy_count"` // 因超过空闲时间而销毁的连接数
	UtilizationRate      float64 `json:"utilization_rate"`       // 连接池利用率 (acquired/max)
}

// Check 执行健康检查
func (h *HealthChecker) Check(ctx context.Context) *HealthStatus {
	start := time.Now()
	status := &HealthStatus{
		Timestamp: start,
	}

	// 执行简单的查询测试数据库连接
	var result int
	err := h.pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	
	status.ResponseTime = time.Since(start)
	
	if err != nil {
		status.Healthy = false
		status.Error = err.Error()
		return status
	}

	if result != 1 {
		status.Healthy = false
		status.Error = "unexpected query result"
		return status
	}

	// 获取连接池统计信息
	stats := h.pool.Stat()
	status.PoolStats = PoolStatistics{
		AcquireCount:            stats.AcquireCount(),
		AcquireDuration:         stats.AcquireDuration().Nanoseconds(),
		AcquiredConns:           stats.AcquiredConns(),
		CanceledAcquireCount:    stats.CanceledAcquireCount(),
		ConstructingConns:       stats.ConstructingConns(),
		EmptyAcquireCount:       stats.EmptyAcquireCount(),
		IdleConns:               stats.IdleConns(),
		MaxConns:                stats.MaxConns(),
		TotalConns:              stats.TotalConns(),
		NewConnsCount:           stats.NewConnsCount(),
		MaxLifetimeDestroyCount: stats.MaxLifetimeDestroyCount(),
		MaxIdleDestroyCount:     stats.MaxIdleDestroyCount(),
	}

	// 计算连接池利用率
	if stats.MaxConns() > 0 {
		status.PoolStats.UtilizationRate = float64(stats.AcquiredConns()) / float64(stats.MaxConns())
	}

	status.Healthy = true
	return status
}

// CheckWithTimeout 带超时的健康检查
func (h *HealthChecker) CheckWithTimeout(timeout time.Duration) *HealthStatus {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return h.Check(ctx)
}

// Monitor 监控器
type Monitor struct {
	pool *pgxpool.Pool
}

// NewMonitor 创建监控器
func NewMonitor(pool *pgxpool.Pool) *Monitor {
	return &Monitor{pool: pool}
}

// GetPoolMetrics 获取连接池指标
func (m *Monitor) GetPoolMetrics() map[string]interface{} {
	stats := m.pool.Stat()
	
	metrics := map[string]interface{}{
		// 连接数指标
		"pool.acquired_conns":      stats.AcquiredConns(),
		"pool.idle_conns":          stats.IdleConns(),
		"pool.total_conns":         stats.TotalConns(),
		"pool.max_conns":           stats.MaxConns(),
		"pool.constructing_conns":  stats.ConstructingConns(),
		
		// 性能指标
		"pool.acquire_count":       stats.AcquireCount(),
		"pool.acquire_duration_ms": stats.AcquireDuration().Milliseconds(),
		"pool.empty_acquire_count": stats.EmptyAcquireCount(),
		"pool.canceled_acquire_count": stats.CanceledAcquireCount(),
		
		// 连接生命周期指标
		"pool.new_conns_count":     stats.NewConnsCount(),
		"pool.max_lifetime_destroy_count": stats.MaxLifetimeDestroyCount(),
		"pool.max_idle_destroy_count": stats.MaxIdleDestroyCount(),
	}

	// 计算衍生指标
	if stats.MaxConns() > 0 {
		metrics["pool.utilization_rate"] = float64(stats.AcquiredConns()) / float64(stats.MaxConns())
	}
	
	if stats.AcquireCount() > 0 {
		metrics["pool.avg_acquire_duration_ms"] = float64(stats.AcquireDuration().Milliseconds()) / float64(stats.AcquireCount())
	}

	return metrics
}

// CheckSlowQueries 检查慢查询（需要在PostgreSQL中启用pg_stat_statements扩展）
func (m *Monitor) CheckSlowQueries(ctx context.Context, thresholdMs int64) ([]SlowQuery, error) {
	query := `
		SELECT 
			query,
			calls,
			total_exec_time,
			mean_exec_time,
			max_exec_time
		FROM pg_stat_statements
		WHERE mean_exec_time > $1
		ORDER BY mean_exec_time DESC
		LIMIT 10
	`

	rows, err := m.pool.Query(ctx, query, thresholdMs)
	if err != nil {
		// pg_stat_statements可能未启用，返回空结果
		return nil, nil
	}
	defer rows.Close()

	var slowQueries []SlowQuery
	for rows.Next() {
		var sq SlowQuery
		err := rows.Scan(
			&sq.Query,
			&sq.Calls,
			&sq.TotalExecTime,
			&sq.MeanExecTime,
			&sq.MaxExecTime,
		)
		if err != nil {
			return nil, fmt.Errorf("scan slow query: %w", err)
		}
		slowQueries = append(slowQueries, sq)
	}

	return slowQueries, nil
}

// SlowQuery 慢查询信息
type SlowQuery struct {
	Query         string  `json:"query"`
	Calls         int64   `json:"calls"`
	TotalExecTime float64 `json:"total_exec_time_ms"`
	MeanExecTime  float64 `json:"mean_exec_time_ms"`
	MaxExecTime   float64 `json:"max_exec_time_ms"`
}

// GetTableSizes 获取表大小统计
func (m *Monitor) GetTableSizes(ctx context.Context) ([]TableSize, error) {
	query := `
		SELECT 
			tablename,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size,
			pg_total_relation_size(schemaname||'.'||tablename) AS size_bytes
		FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
	`

	rows, err := m.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query table sizes: %w", err)
	}
	defer rows.Close()

	var sizes []TableSize
	for rows.Next() {
		var ts TableSize
		err := rows.Scan(&ts.TableName, &ts.Size, &ts.SizeBytes)
		if err != nil {
			return nil, fmt.Errorf("scan table size: %w", err)
		}
		sizes = append(sizes, ts)
	}

	return sizes, nil
}

// TableSize 表大小信息
type TableSize struct {
	TableName string `json:"table_name"`
	Size      string `json:"size"`
	SizeBytes int64  `json:"size_bytes"`
}
