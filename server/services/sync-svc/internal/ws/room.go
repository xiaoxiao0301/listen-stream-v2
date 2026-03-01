package ws

import (
	"encoding/json"
	"log"
	"sync"
)

// Room 房间管理器（按UserID分组连接）
type Room struct {
	// userID -> connections
	rooms map[string]map[string]*Connection
	mu    sync.RWMutex
}

// NewRoom 创建房间管理器
func NewRoom() *Room {
	return &Room{
		rooms: make(map[string]map[string]*Connection),
	}
}

// Join 用户加入房间（添加连接）
func (r *Room) Join(userID string, conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rooms[userID]; !exists {
		r.rooms[userID] = make(map[string]*Connection)
	}

	r.rooms[userID][conn.ID] = conn
	log.Printf("User joined room: user=%s, conn=%s, total_conns=%d",
		userID, conn.ID, len(r.rooms[userID]))
}

// Leave 用户离开房间（移除连接）
func (r *Room) Leave(userID, connID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if conns, exists := r.rooms[userID]; exists {
		delete(conns, connID)
		log.Printf("User left room: user=%s, conn=%s, remaining_conns=%d",
			userID, connID, len(conns))

		// 如果该用户没有任何连接了，删除整个房间
		if len(conns) == 0 {
			delete(r.rooms, userID)
			log.Printf("Room removed: user=%s (no connections left)", userID)
		}
	}
}

// Broadcast 向指定用户的所有连接广播消息
func (r *Room) Broadcast(userID string, message []byte) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conns, exists := r.rooms[userID]
	if !exists {
		return 0
	}

	sentCount := 0
	for _, conn := range conns {
		if conn.Send(message) {
			sentCount++
		}
	}

	log.Printf("Broadcast to user: user=%s, connections=%d, sent=%d",
		userID, len(conns), sentCount)

	return sentCount
}

// BroadcastJSON 向指定用户的所有连接广播JSON消息
func (r *Room) BroadcastJSON(userID string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	r.Broadcast(userID, data)
	return nil
}

// GetUserConnections 获取用户的所有连接
func (r *Room) GetUserConnections(userID string) []*Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conns, exists := r.rooms[userID]
	if !exists {
		return []*Connection{}
	}

	connections := make([]*Connection, 0, len(conns))
	for _, conn := range conns {
		if conn.IsActive() {
			connections = append(connections, conn)
		}
	}

	return connections
}

// GetUserConnectionCount 获取用户的连接数
func (r *Room) GetUserConnectionCount(userID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if conns, exists := r.rooms[userID]; exists {
		count := 0
		for _, conn := range conns {
			if conn.IsActive() {
				count++
			}
		}
		return count
	}

	return 0
}

// IsUserOnline 检查用户是否在线
func (r *Room) IsUserOnline(userID string) bool {
	return r.GetUserConnectionCount(userID) > 0
}

// GetOnlineUsers 获取所有在线用户列表
func (r *Room) GetOnlineUsers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]string, 0, len(r.rooms))
	for userID, conns := range r.rooms {
		// 检查是否有活跃连接
		hasActiveConn := false
		for _, conn := range conns {
			if conn.IsActive() {
				hasActiveConn = true
				break
			}
		}

		if hasActiveConn {
			users = append(users, userID)
		}
	}

	return users
}

// GetStats 获取房间统计信息
func (r *Room) GetStats() RoomStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RoomStats{
		TotalRooms: len(r.rooms),
	}

	for _, conns := range r.rooms {
		activeConns := 0
		for _, conn := range conns {
			if conn.IsActive() {
				activeConns++
			}
		}
		stats.TotalConnections += activeConns

		if activeConns > stats.MaxConnectionsPerUser {
			stats.MaxConnectionsPerUser = activeConns
		}
	}

	if stats.TotalRooms > 0 {
		stats.AvgConnectionsPerUser = float64(stats.TotalConnections) / float64(stats.TotalRooms)
	}

	return stats
}

// RoomStats 房间统计信息
type RoomStats struct {
	TotalRooms             int
	TotalConnections       int
	MaxConnectionsPerUser  int
	AvgConnectionsPerUser  float64
}
