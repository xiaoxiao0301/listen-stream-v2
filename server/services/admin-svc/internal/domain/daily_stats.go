package domain

import "time"

// DailyStats 每日统计实体（定时聚合）
type DailyStats struct {
	Date            time.Time `json:"date" db:"date"` // PK: 日期（YYYY-MM-DD）
	TotalUsers      int64     `json:"total_users" db:"total_users"` // 总用户数
	NewUsers        int64     `json:"new_users" db:"new_users"` // 新增用户数
	ActiveUsers     int64     `json:"active_users" db:"active_users"` // 活跃用户数（当日有操作）
	TotalRequests   int64     `json:"total_requests" db:"total_requests"` // 总请求数
	SuccessRequests int64     `json:"success_requests" db:"success_requests"` // 成功请求数
	FailedRequests  int64     `json:"failed_requests" db:"failed_requests"` // 失败请求数
	ErrorRate       float64   `json:"error_rate" db:"error_rate"` // 错误率
	AvgResponseTime int64     `json:"avg_response_time" db:"avg_response_time"` // 平均响应时间（ms）
	TotalFavorites  int64     `json:"total_favorites" db:"total_favorites"` // 总收藏数
	TotalPlaylists  int64     `json:"total_playlists" db:"total_playlists"` // 总歌单数
	TotalPlays      int64     `json:"total_plays" db:"total_plays"` // 总播放次数
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// CalculateErrorRate 计算错误率
func (s *DailyStats) CalculateErrorRate() {
	if s.TotalRequests > 0 {
		s.ErrorRate = float64(s.FailedRequests) / float64(s.TotalRequests) * 100
	} else {
		s.ErrorRate = 0
	}
}

// RealtimeStats 实时统计（存储在Redis）
type RealtimeStats struct {
	OnlineUsers     int64     `json:"online_users"`
	ActiveSessions  int64     `json:"active_sessions"`
	RequestsPerMin  int64     `json:"requests_per_min"`
	ErrorsPerMin    int64     `json:"errors_per_min"`
	AvgResponseTime int64     `json:"avg_response_time"`
	Timestamp       time.Time `json:"timestamp"`
}
