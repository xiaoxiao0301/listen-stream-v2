package offline

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis 创建测试用Redis
func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
}

func TestOfflineMessage(t *testing.T) {
	msg := &OfflineMessage{
		ID:        "msg-123",
		UserID:    "user-1",
		Type:      "favorite.added",
		Data:      map[string]interface{}{"song_id": "song-1"},
		AckToken:  "token-123",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// 序列化
	jsonStr, err := msg.ToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonStr)

	// 反序列化
	parsed, err := FromJSON(jsonStr)
	assert.NoError(t, err)
	assert.Equal(t, msg.ID, parsed.ID)
	assert.Equal(t, msg.UserID, parsed.UserID)
}

func TestQueuePushPull(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	queue := NewQueue(client, nil)

	// 推送消息
	msg1, err := queue.Push(ctx, "user1", "favorite.added", map[string]interface{}{"song_id": "song1"})
	assert.NoError(t, err)
	assert.NotEmpty(t, msg1.ID)
	assert.NotEmpty(t, msg1.AckToken)

	// 拉取消息
	messages, err := queue.Pull(ctx, "user1", 10)
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, msg1.ID, messages[0].ID)
}

func TestQueueAck(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	queue := NewQueue(client, nil)

	// 推送消息
	msg, err := queue.Push(ctx, "user1", "test", map[string]interface{}{})
	assert.NoError(t, err)

	// 确认消息
	err = queue.Ack(ctx, "user1", msg.ID, msg.AckToken)
	assert.NoError(t, err)

	// 确认后消息应该被删除
	count, err := queue.Count(ctx, "user1")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestService(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	queue := NewQueue(client, nil)
	service := NewService(queue)

	// 推送消息
	msg, err := service.Push(ctx, "user1", "test", map[string]interface{}{})
	assert.NoError(t, err)
	assert.NotEmpty(t, msg.ID)

	// 拉取消息
	messages, err := service.Pull(ctx, "user1", 10)
	assert.NoError(t, err)
	assert.Len(t, messages, 1)

	// 确认消息
	err = service.Ack(ctx, "user1", msg.ID, msg.AckToken)
	assert.NoError(t, err)

	// 检查统计
	stats, err := service.GetStats(ctx)
	assert.NoError(t, err)
	assert.Greater(t, stats.TotalPushed, int64(0))
	assert.Greater(t, stats.TotalPulled, int64(0))
	assert.Greater(t, stats.TotalAcked, int64(0))
}
