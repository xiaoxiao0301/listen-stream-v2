package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: These tests require a Redis instance running on localhost:6379
// Run with: go test -v
// Skip integration tests: go test -v -short

func setupTestClient(t *testing.T) (*Client, func()) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &Config{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           15, // Use DB 15 for testing
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		MaxRetries:   3,
	}

	client, err := NewClient(cfg)
	require.NoError(t, err, "failed to create redis client")

	cleanup := func() {
		ctx := context.Background()
		client.universal.FlushDB(ctx)
		client.Close()
	}

	return client, cleanup
}

func TestNewClient_Success(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	assert.NotNil(t, client)
	assert.NotNil(t, client.universal)
}

func TestClient_SetAndGet(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()

	err := client.Set(ctx, "test:key", "test-value", time.Minute)
	require.NoError(t, err)

	result, err := client.Get(ctx, "test:key")
	require.NoError(t, err)
	assert.Equal(t, "test-value", result)
}

func TestClient_GetNonExistent(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()
	_, err := client.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClient_SetNX(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()

	ok, err := client.SetNX(ctx, "test:setnx", "value1", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = client.SetNX(ctx, "test:setnx", "value2", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestClient_IncrDecr(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()

	val, err := client.Incr(ctx, "test:counter")
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)

	val, err = client.IncrBy(ctx, "test:counter", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(6), val)

	val, err = client.Decr(ctx, "test:counter")
	require.NoError(t, err)
	assert.Equal(t, int64(5), val)
}

func TestClient_Hash(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()

	err := client.HSet(ctx, "test:hash", "field1", "value1")
	require.NoError(t, err)

	val, err := client.HGet(ctx, "test:hash", "field1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val)

	all, err := client.HGetAll(ctx, "test:hash")
	require.NoError(t, err)
	assert.Equal(t, "value1", all["field1"])
}

func TestClient_List(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()

	err := client.RPush(ctx, "test:list", "item1", "item2")
	require.NoError(t, err)

	length, err := client.LLen(ctx, "test:list")
	require.NoError(t, err)
	assert.Equal(t, int64(2), length)

	items, err := client.LRange(ctx, "test:list", 0, -1)
	require.NoError(t, err)
	assert.Equal(t, []string{"item1", "item2"}, items)
}

func TestClient_Set(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()

	err := client.SAdd(ctx, "test:set", "member1", "member2")
	require.NoError(t, err)

	isMember, err := client.SIsMember(ctx, "test:set", "member1")
	require.NoError(t, err)
	assert.True(t, isMember)

	members, err := client.SMembers(ctx, "test:set")
	require.NoError(t, err)
	assert.Equal(t, 2, len(members))
}

func TestClient_Ping(t *testing.T) {
	client, cleanup := setupTestClient(t)
	defer cleanup()

	ctx := context.Background()
	err := client.Ping(ctx)
	assert.NoError(t, err)
}
