package upstream

import (
	"sync"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// State 熔断器状态
type State int

const (
	// StateClosed 关闭状态（正常）
	StateClosed State = iota
	// StateOpen 打开状态（熔断）
	StateOpen
	// StateHalfOpen 半开状态（尝试恢复）
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// BreakerSettings 熔断器设置
type BreakerSettings struct {
	MaxFailures  int           // 最大失败次数
	Timeout      time.Duration // 熔断超时时间
	MaxRequests  int           // 半开状态最大请求数
	ReadyToTrip  func(counts) bool
	OnStateChange func(from, to State)
}

// DefaultBreakerSettings 默认熔断器设置
func DefaultBreakerSettings() BreakerSettings {
	return BreakerSettings{
		MaxFailures: 5,
		Timeout:     30 * time.Second,
		MaxRequests: 3,
		ReadyToTrip: func(c counts) bool {
			return c.ConsecutiveFailures >= 5
		},
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	settings BreakerSettings
	state    State
	counts   counts
	expiry   time.Time
	mu       sync.RWMutex
	logger   logger.Logger
}

// counts 计数器
type counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(settings BreakerSettings, log logger.Logger) *CircuitBreaker {
	if settings.MaxFailures == 0 {
		settings.MaxFailures = 5
	}
	if settings.Timeout == 0 {
		settings.Timeout = 30 * time.Second
	}
	if settings.MaxRequests == 0 {
		settings.MaxRequests = 3
	}
	if settings.ReadyToTrip == nil {
		settings.ReadyToTrip = func(c counts) bool {
			return c.ConsecutiveFailures >= uint32(settings.MaxFailures)
		}
	}

	return &CircuitBreaker{
		settings: settings,
		state:    StateClosed,
		logger:   log,
	}
}

// Execute 执行函数，带熔断保护
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	// 检查是否允许执行
	if err := cb.beforeRequest(); err != nil {
		return nil, err
	}

	// 执行函数
	result, err := fn()

	// 记录结果
	cb.afterRequest(err == nil)

	return result, err
}

// beforeRequest 请求前检查
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateOpen:
		// 检查是否到达超时时间
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen)
			cb.counts = counts{}
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// 半开状态限制请求数
		if cb.counts.Requests >= uint32(cb.settings.MaxRequests) {
			return ErrCircuitOpen
		}

	case StateClosed:
		// 关闭状态正常执行
	}

	cb.counts.Requests++
	return nil
}

// afterRequest 请求后处理
func (cb *CircuitBreaker) afterRequest(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

// onSuccess 成功处理
func (cb *CircuitBreaker) onSuccess() {
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0

	if cb.state == StateHalfOpen {
		// 半开状态连续成功，恢复到关闭状态
		if cb.counts.ConsecutiveSuccesses >= uint32(cb.settings.MaxRequests) {
			cb.setState(StateClosed)
			cb.counts = counts{}
		}
	}
}

// onFailure 失败处理
func (cb *CircuitBreaker) onFailure() {
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0

	if cb.settings.ReadyToTrip(cb.counts) {
		cb.setState(StateOpen)
		cb.expiry = time.Now().Add(cb.settings.Timeout)
	}
}

// setState 设置状态
func (cb *CircuitBreaker) setState(state State) {
	if cb.state == state {
		return
	}

	oldState := cb.state
	cb.state = state

	cb.logger.Info("Circuit breaker state changed",
		logger.String("from", oldState.String()),
		logger.String("to", state.String()),
	)

	if cb.settings.OnStateChange != nil {
		cb.settings.OnStateChange(oldState, state)
	}
}

// State 获取当前状态
func (cb *CircuitBreaker) State() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state.String()
}

// Stats 获取统计信息
func (cb *CircuitBreaker) Stats() BreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return BreakerStats{
		State:                cb.state.String(),
		Requests:             cb.counts.Requests,
		TotalSuccesses:       cb.counts.TotalSuccesses,
		TotalFailures:        cb.counts.TotalFailures,
		ConsecutiveSuccesses: cb.counts.ConsecutiveSuccesses,
		ConsecutiveFailures:  cb.counts.ConsecutiveFailures,
	}
}

// BreakerStats 熔断器统计
type BreakerStats struct {
	State                string
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.counts = counts{}
	cb.expiry = time.Time{}
}
