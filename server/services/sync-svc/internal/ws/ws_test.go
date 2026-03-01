package ws

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// setupTestRedis 创建测试用Redis
func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

func TestConnectionLimiter(t *testing.T) {
	t.Run("Acquire and Release", func(t *testing.T) {
		limiter := NewConnectionLimiter(3)

		// 获取3个连接
		assert.NoError(t, limiter.Acquire())
		assert.NoError(t, limiter.Acquire())
		assert.NoError(t, limiter.Acquire())
		assert.Equal(t, int32(3), limiter.CurrentCount())

		// 第4个连接应该失败
		err := limiter.Acquire()
		assert.Error(t, err)
		assert.Equal(t, ErrConnectionLimitExceeded, err)

		// 释放一个连接
		limiter.Release()
		assert.Equal(t, int32(2), limiter.CurrentCount())

		// 现在应该可以获取新连接
		assert.NoError(t, limiter.Acquire())
		assert.Equal(t, int32(3), limiter.CurrentCount())
	})

	t.Run("Available connections", func(t *testing.T) {
		limiter := NewConnectionLimiter(10)

		assert.Equal(t, int32(10), limiter.Available())
		
		limiter.Acquire()
		assert.Equal(t, int32(9), limiter.Available())

		limiter.Acquire()
		limiter.Acquire()
		assert.Equal(t, int32(7), limiter.Available())
	})

	t.Run("Default max connections", func(t *testing.T) {
		limiter := NewConnectionLimiter(0)
		assert.Equal(t, int32(DefaultMaxConnections), limiter.MaxConnections())

		limiter = NewConnectionLimiter(-1)
		assert.Equal(t, int32(DefaultMaxConnections), limiter.MaxConnections())
	})
}

func TestRoom(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	t.Run("Join and Leave", func(t *testing.T) {
		room := NewRoom()
		manager := NewManager(10, client, "test-instance")

		conn1 := &Connection{ID: "conn1", UserID: "user1", isActive: 1, manager: manager}
		conn2 := &Connection{ID: "conn2", UserID: "user1", isActive: 1, manager: manager}
		conn3 := &Connection{ID: "conn3", UserID: "user2", isActive: 1, manager: manager}

		room.Join("user1", conn1)
		room.Join("user1", conn2)
		room.Join("user2", conn3)

		assert.Equal(t, 2, room.GetUserConnectionCount("user1"))
		assert.Equal(t, 1, room.GetUserConnectionCount("user2"))
		assert.True(t, room.IsUserOnline("user1"))
		assert.True(t, room.IsUserOnline("user2"))

		room.Leave("user1", "conn1")
		assert.Equal(t, 1, room.GetUserConnectionCount("user1"))

		room.Leave("user1", "conn2")
		assert.Equal(t, 0, room.GetUserConnectionCount("user1"))
		assert.False(t, room.IsUserOnline("user1"))
	})

	t.Run("Get online users", func(t *testing.T) {
		room := NewRoom()
		manager := NewManager(10, client, "test-instance")

		conn1 := &Connection{ID: "conn1", UserID: "user1", isActive: 1, manager: manager}
		conn2 := &Connection{ID: "conn2", UserID: "user2", isActive: 1, manager: manager}
		conn3 := &Connection{ID: "conn3", UserID: "user3", isActive: 1, manager: manager}

		room.Join("user1", conn1)
		room.Join("user2", conn2)
		room.Join("user3", conn3)

		users := room.GetOnlineUsers()
		assert.Len(t, users, 3)
		assert.Contains(t, users, "user1")
		assert.Contains(t, users, "user2")
		assert.Contains(t, users, "user3")
	})

	t.Run("Room stats", func(t *testing.T) {
		room := NewRoom()
		manager := NewManager(10, client, "test-instance")

		conn1 := &Connection{ID: "conn1", UserID: "user1", isActive: 1, manager: manager}
		conn2 := &Connection{ID: "conn2", UserID: "user1", isActive: 1, manager: manager}
		conn3 := &Connection{ID: "conn3", UserID: "user2", isActive: 1, manager: manager}

		room.Join("user1", conn1)
		room.Join("user1", conn2)
		room.Join("user2", conn3)

		stats := room.GetStats()
		assert.Equal(t, 2, stats.TotalRooms)
		assert.Equal(t, 3, stats.TotalConnections)
		assert.Equal(t, 2, stats.MaxConnectionsPerUser)
		assert.Equal(t, 1.5, stats.AvgConnectionsPerUser)
	})
}

func TestConnection(t *testing.T) {
	t.Run("IsActive", func(t *testing.T) {
		conn := &Connection{
			ID:       "test",
			UserID:   "user1",
			isActive: 1,
		}

		assert.True(t, conn.IsActive())

		conn.isActive = 0
		assert.False(t, conn.IsActive())
	})

	t.Run("Update ping/pong times", func(t *testing.T) {
		conn := &Connection{
			ID:           "test",
			UserID:       "user1",
			lastPingTime: time.Now().Add(-1 * time.Minute),
			lastPongTime: time.Now().Add(-1 * time.Minute),
		}

		oldPingTime := conn.lastPingTime
		oldPongTime := conn.lastPongTime

		time.Sleep(10 * time.Millisecond)

		conn.UpdatePingTime()
		assert.True(t, conn.lastPingTime.After(oldPingTime))

		conn.UpdatePongTime()
		assert.True(t, conn.lastPongTime.After(oldPongTime))
	})
}

func TestHeartbeatStats(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	manager := NewManager(10, client, "test-instance")
	checker := NewHeartbeatChecker(manager)

	now := time.Now()

	// 添加一些模拟连接
	conn1 := &Connection{
		ID:           "conn1",
		UserID:       "user1",
		isActive:     1,
		lastPongTime: now.Add(-10 * time.Second), // 健康
		manager:      manager,
	}
	conn2 := &Connection{
		ID:           "conn2",
		UserID:       "user2",
		isActive:     1,
		lastPongTime: now.Add(-45 * time.Second), // 警告
		manager:      manager,
	}
	conn3 := &Connection{
		ID:           "conn3",
		UserID:       "user3",
		isActive:     1,
		lastPongTime: now.Add(-70 * time.Second), // 不健康
		manager:      manager,
	}

	manager.connections["conn1"] = conn1
	manager.connections["conn2"] = conn2
	manager.connections["conn3"] = conn3

	stats := checker.GetStats()

	assert.Equal(t, 3, stats.TotalConnections)
	assert.Equal(t, 1, stats.HealthyConnections)
	assert.Equal(t, 1, stats.WarningConnections)
	assert.Equal(t, 1, stats.UnhealthyConnections)
}
