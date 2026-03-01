package ws

import (
	"errors"
	"sync"
	"sync/atomic"
)

const (
	// DefaultMaxConnections 默认最大连接数
	DefaultMaxConnections = 10000
)

var (
	// ErrConnectionLimitExceeded 连接数超限错误
	ErrConnectionLimitExceeded = errors.New("connection limit exceeded")
)

// ConnectionLimiter 连接数限制器
type ConnectionLimiter struct {
	maxConnections int32
	currentCount   int32
	mu             sync.RWMutex
	semaphore      chan struct{}
}

// NewConnectionLimiter 创建连接限制器
func NewConnectionLimiter(maxConnections int) *ConnectionLimiter {
	if maxConnections <= 0 {
		maxConnections = DefaultMaxConnections
	}
	return &ConnectionLimiter{
		maxConnections: int32(maxConnections),
		currentCount:   0,
		semaphore:      make(chan struct{}, maxConnections),
	}
}

// Acquire 获取连接许可
func (l *ConnectionLimiter) Acquire() error {
	select {
	case l.semaphore <- struct{}{}:
		atomic.AddInt32(&l.currentCount, 1)
		return nil
	default:
		return ErrConnectionLimitExceeded
	}
}

// Release 释放连接许可
func (l *ConnectionLimiter) Release() {
	select {
	case <-l.semaphore:
		atomic.AddInt32(&l.currentCount, -1)
	default:
		// 防止重复释放
	}
}

// CurrentCount 获取当前连接数
func (l *ConnectionLimiter) CurrentCount() int32 {
	return atomic.LoadInt32(&l.currentCount)
}

// MaxConnections 获取最大连接数
func (l *ConnectionLimiter) MaxConnections() int32 {
	return l.maxConnections
}

// Available 获取可用连接数
func (l *ConnectionLimiter) Available() int32 {
	return l.maxConnections - l.CurrentCount()
}
