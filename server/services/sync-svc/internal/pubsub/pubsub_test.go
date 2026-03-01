package pubsub

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"sync-svc/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestPublisher(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	testMsg := &domain.SyncMessage{
		ID:     "test-msg",
		Type:   domain.MessageTypeFavoriteAdded,
		UserID: "user-1",
		Data:   map[string]interface{}{"test": "data"},
		Timestamp: time.Now(),
	}

	t.Run("PublishToUser", func(t *testing.T) {
		pub := NewPublisher(client, "instance-1")
		err := pub.PublishToUser(context.Background(), "user-1", testMsg)
		require.NoError(t, err)

		stats := pub.GetStats()
		assert.Equal(t, int64(1), stats.TotalPublished)
		assert.Equal(t, int64(1), stats.UserPublished)
	})

	t.Run("PublishBroadcast", func(t *testing.T) {
		pub := NewPublisher(client, "instance-1")
		err := pub.PublishBroadcast(context.Background(), testMsg)
		require.NoError(t, err)

		stats := pub.GetStats()
		assert.Equal(t, int64(1), stats.BroadcastPublished)
	})

	t.Run("BatchPublish", func(t *testing.T) {
		pub := NewPublisher(client, "instance-1")
		users := []string{"user-1", "user-2", "user-3"}
		err := pub.BatchPublishToUsers(context.Background(), users, testMsg)
		require.NoError(t, err)

		stats := pub.GetStats()
		assert.Equal(t, int64(3), stats.UserPublished)
	})
}

func TestSubscriber(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	t.Run("SubscribeAndReceive", func(t *testing.T) {
		sub := NewSubscriber(client, DefaultSubscriberConfig("instance-2"))
		var received int64

		sub.Subscribe("sync:user:*", func(msg *domain.SyncMessage, channel string) error {
			atomic.AddInt64(&received, 1)
			return nil
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		require.NoError(t, sub.Start(ctx))
		time.Sleep(100 * time.Millisecond)

		pub := NewPublisher(client, "instance-1")
		msg := &domain.SyncMessage{
			ID:     "test",
			Type:   domain.MessageTypeFavoriteAdded,
			UserID: "user-1",
			Data:   map[string]interface{}{},
			Timestamp: time.Now(),
		}

		require.NoError(t, pub.PublishToUser(ctx, "user-1", msg))
		time.Sleep(200 * time.Millisecond)

		assert.Equal(t, int64(1), atomic.LoadInt64(&received))
		sub.Stop()
	})

	t.Run("FilterOwnInstance", func(t *testing.T) {
		sub := NewSubscriber(client, DefaultSubscriberConfig("instance-1"))
		var received int64

		sub.Subscribe("sync:user:*", func(msg *domain.SyncMessage, channel string) error {
			atomic.AddInt64(&received, 1)
			return nil
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		require.NoError(t, sub.Start(ctx))
		time.Sleep(100 * time.Millisecond)

		// 发布来自同一实例的消息（应该被过滤）
		pub := NewPublisher(client, "instance-1")
		msg := &domain.SyncMessage{
			ID:     "test",
			Type:   domain.MessageTypeFavoriteAdded,
			UserID: "user-1",
			Data:   map[string]interface{}{},
			Timestamp: time.Now(),
		}

		require.NoError(t, pub.PublishToUser(ctx, "user-1", msg))
		time.Sleep(200 * time.Millisecond)

		// 不应该接收到消息
		assert.Equal(t, int64(0), atomic.LoadInt64(&received))
		
		// 但统计应该显示被丢弃
		stats := sub.GetStats()
		assert.Equal(t, int64(1), stats.DroppedMessages)

		sub.Stop()
	})
}

func TestCrossInstance(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	t.Run("MessageFlow", func(t *testing.T) {
		// Instance A订阅
		subA := NewSubscriber(client, DefaultSubscriberConfig("instance-A"))
		var receivedMsg *domain.SyncMessage

		subA.Subscribe("sync:user:*", func(msg *domain.SyncMessage, channel string) error {
			receivedMsg = msg
			return nil
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		require.NoError(t, subA.Start(ctx))
		time.Sleep(100 * time.Millisecond)

		// Instance B发布
		pubB := NewPublisher(client, "instance-B")
		msg := &domain.SyncMessage{
			ID:     "cross-msg",
			Type:   domain.MessageTypePlaylistCreated,
			UserID: "user-123",
			Data:   map[string]interface{}{"playlist_id": "pl-456"},
			Timestamp: time.Now(),
		}

		require.NoError(t, pubB.PublishToUser(ctx, "user-123", msg))
		time.Sleep(200 * time.Millisecond)

		// 验证接收
		require.NotNil(t, receivedMsg)
		assert.Equal(t, "cross-msg", receivedMsg.ID)
		assert.Equal(t, "instance-B", receivedMsg.InstanceID)

		subA.Stop()
	})
}
