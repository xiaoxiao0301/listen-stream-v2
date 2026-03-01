package ws

import (
	"context"
	"log"
	"time"
)

const (
	// HeartbeatCheckInterval 心跳检查间隔
	HeartbeatCheckInterval = 15 * time.Second
	// HeartbeatTimeout 心跳超时时间（如果超过这个时间没有收到Pong，则断开连接）
	HeartbeatTimeout = 90 * time.Second
)

// HeartbeatChecker 心跳检查器
type HeartbeatChecker struct {
	manager  *Manager
	interval time.Duration
	timeout  time.Duration
	stopChan chan struct{}
}

// NewHeartbeatChecker 创建心跳检查器
func NewHeartbeatChecker(manager *Manager) *HeartbeatChecker {
	return &HeartbeatChecker{
		manager:  manager,
		interval: HeartbeatCheckInterval,
		timeout:  HeartbeatTimeout,
		stopChan: make(chan struct{}),
	}
}

// Start 启动心跳检查
func (h *HeartbeatChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	log.Printf("Heartbeat checker started (interval=%v, timeout=%v)", h.interval, h.timeout)

	for {
		select {
		case <-ctx.Done():
			log.Println("Heartbeat checker stopped by context")
			return
		case <-h.stopChan:
			log.Println("Heartbeat checker stopped")
			return
		case <-ticker.C:
			h.checkConnections()
		}
	}
}

// Stop 停止心跳检查
func (h *HeartbeatChecker) Stop() {
	close(h.stopChan)
}

// checkConnections 检查所有连接的心跳状态
func (h *HeartbeatChecker) checkConnections() {
	now := time.Now()
	deadConnections := make([]*Connection, 0)

	// 获取所有连接并检查心跳
	h.manager.mu.RLock()
	for _, conn := range h.manager.connections {
		if !conn.IsActive() {
			continue
		}

		lastPongTime := conn.GetLastPongTime()
		elapsed := now.Sub(lastPongTime)

		if elapsed > h.timeout {
			deadConnections = append(deadConnections, conn)
			log.Printf("Connection heartbeat timeout: id=%s, user=%s, last_pong=%v ago",
				conn.ID, conn.UserID, elapsed)
		}
	}
	h.manager.mu.RUnlock()

	// 关闭超时的连接
	for _, conn := range deadConnections {
		conn.Close("heartbeat timeout")
		h.manager.Unregister(conn)
	}

	if len(deadConnections) > 0 {
		log.Printf("Closed %d connections due to heartbeat timeout", len(deadConnections))
	}
}

// GetStats 获取心跳统计信息
func (h *HeartbeatChecker) GetStats() HeartbeatStats {
	now := time.Now()
	stats := HeartbeatStats{
		CheckedAt: now,
	}

	h.manager.mu.RLock()
	defer h.manager.mu.RUnlock()

	for _, conn := range h.manager.connections {
		if !conn.IsActive() {
			continue
		}

		lastPongTime := conn.GetLastPongTime()
		elapsed := now.Sub(lastPongTime)

		stats.TotalConnections++

		if elapsed < 30*time.Second {
			stats.HealthyConnections++
		} else if elapsed < 60*time.Second {
			stats.WarningConnections++
		} else {
			stats.UnhealthyConnections++
		}

		if elapsed > stats.MaxPongDelay {
			stats.MaxPongDelay = elapsed
		}
		if stats.MinPongDelay == 0 || elapsed < stats.MinPongDelay {
			stats.MinPongDelay = elapsed
		}
		stats.AvgPongDelay += elapsed
	}

	if stats.TotalConnections > 0 {
		stats.AvgPongDelay = stats.AvgPongDelay / time.Duration(stats.TotalConnections)
	}

	return stats
}

// HeartbeatStats 心跳统计信息
type HeartbeatStats struct {
	CheckedAt            time.Time
	TotalConnections     int
	HealthyConnections   int  // < 30s
	WarningConnections   int  // 30s-60s
	UnhealthyConnections int  // > 60s
	MaxPongDelay         time.Duration
	MinPongDelay         time.Duration
	AvgPongDelay         time.Duration
}
