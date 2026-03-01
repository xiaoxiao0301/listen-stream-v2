package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sync-svc/internal/handler"
	"sync-svc/internal/ws"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*gin.Engine, *ws.Manager, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	manager := ws.NewManager(100, client, "test-instance")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		mr.Close()
	})
	go manager.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	wsHandler := handler.NewWSHandler(manager)
	eventHandler := handler.NewEventHandler(manager)

	api := router.Group("/api/v1")
	{
		api.GET("/stats", wsHandler.GetStats)
		api.GET("/online-users", wsHandler.GetOnlineUsers)
		api.GET("/users/:user_id/online", wsHandler.CheckUserOnline)
		api.GET("/offline/stats", wsHandler.GetOfflineStats)
		api.GET("/stats/pubsub", eventHandler.GetPubSubStats)
	}

	events := router.Group("/api/v1/events")
	events.Use(mockJWTMiddleware("test-user"))
	{
		events.POST("", eventHandler.PublishEvent)
		events.POST("/batch", eventHandler.BatchPublishEvent)
		events.POST("/broadcast", eventHandler.BroadcastEvent)
	}

	offline := router.Group("/api/v1/offline")
	offline.Use(mockJWTMiddleware("test-user"))
	{
		offline.GET("/messages", wsHandler.GetOfflineMessages)
		offline.POST("/ack", wsHandler.AckOfflineMessage)
		offline.POST("/ack/batch", wsHandler.BatchAckOfflineMessages)
		offline.GET("/count", wsHandler.GetOfflineMessageCount)
	}

	return router, manager, mr
}

func mockJWTMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func TestGetStats(t *testing.T) {
	router, _, _ := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "current_connections")
	assert.Contains(t, response, "instance_id")
	assert.Equal(t, "test-instance", response["instance_id"])
}

func TestGetOnlineUsers(t *testing.T) {
	router, _, _ := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/online-users", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "online_users")
	assert.Contains(t, response, "count")
}

func TestCheckUserOnline(t *testing.T) {
	router, _, _ := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/users/test-user/online", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-user", response["user_id"])
	assert.Contains(t, response, "online")
	assert.Contains(t, response, "connection_count")
}

func TestPublishEvent(t *testing.T) {
	router, _, _ := setupTestServer(t)

	payload := `{
		"user_id": "test-user",
		"type": "favorite.added",
		"data": {
			"item_id": "song-123"
		}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, "test-user", response["user_id"])
	assert.Equal(t, "favorite.added", response["type"])
	assert.Contains(t, response, "message_id")
}

func TestPublishEventInvalidType(t *testing.T) {
	router, _, _ := setupTestServer(t)

	payload := `{
		"user_id": "test-user",
		"type": "invalid.type",
		"data": {}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "invalid_message_type", response["error"])
}

func TestBatchPublishEvent(t *testing.T) {
	router, _, _ := setupTestServer(t)

	payload := `{
		"user_ids": ["user-1", "user-2", "user-3"],
		"type": "playlist.created",
		"data": {
			"playlist_id": "pl-123"
		}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events/batch", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, float64(3), response["user_count"])
	assert.Equal(t, "playlist.created", response["type"])
}

func TestBroadcastEvent(t *testing.T) {
	router, _, _ := setupTestServer(t)

	payload := `{
		"type": "favorite.added",
		"data": {
			"announcement": "System maintenance"
		}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/events/broadcast", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Contains(t, response, "online_users")
}

func TestGetOfflineMessageCount(t *testing.T) {
	router, _, _ := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/offline/count", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-user", response["user_id"])
	assert.Contains(t, response, "count")
}

func TestGetOfflineMessages(t *testing.T) {
	router, manager, _ := setupTestServer(t)

	ctx := context.Background()
	_, err := manager.GetOfflineService().Push(ctx, "test-user", "favorite.added", map[string]interface{}{
		"item_id": "song-123",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/offline/messages", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Greater(t, response["count"].(float64), float64(0))
	messages := response["messages"].([]interface{})
	assert.Greater(t, len(messages), 0)
}

func TestAckOfflineMessage(t *testing.T) {
	router, manager, _ := setupTestServer(t)

	ctx := context.Background()
	offlineMsg, err := manager.GetOfflineService().Push(ctx, "test-user", "favorite.added", map[string]interface{}{
		"item_id": "song-123",
	})
	require.NoError(t, err)

	payload := map[string]string{
		"message_id": offlineMsg.ID,
		"ack_token":  offlineMsg.AckToken,
	}
	payloadBytes, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/offline/ack", strings.NewReader(string(payloadBytes)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "acknowledged", response["message"])
}

func TestGetPubSubStats(t *testing.T) {
	router, _, _ := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stats/pubsub", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-instance", response["instance_id"])
	assert.Contains(t, response, "publisher")
	assert.Contains(t, response, "subscriber")
}

func TestGetOfflineStats(t *testing.T) {
	router, _, _ := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/offline/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "total_pushed")
	assert.Contains(t, response, "current_pending")
}

func TestMessageTypeValidation(t *testing.T) {
	validTypes := []string{
		"favorite.added",
		"favorite.removed",
		"playlist.created",
		"playlist.updated",
		"playlist.deleted",
		"playlist.song.added",
		"playlist.song.removed",
		"history.added",
	}

	router, _, _ := setupTestServer(t)

	for _, msgType := range validTypes {
		t.Run(msgType, func(t *testing.T) {
			payload := map[string]interface{}{
				"user_id": "test-user",
				"type":    msgType,
				"data":    map[string]interface{}{},
			}
			payloadBytes, _ := json.Marshal(payload)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/events", strings.NewReader(string(payloadBytes)))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.True(t, response["success"].(bool))
			assert.Equal(t, msgType, response["type"])
		})
	}
}
