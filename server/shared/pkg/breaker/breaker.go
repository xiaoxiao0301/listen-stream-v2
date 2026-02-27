// Package breaker provides circuit breaker functionality.
package breaker

import (
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// String returns the string representation of the state.
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

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	name            string
	maxFailures     int
	timeout         time.Duration
	halfOpenMaxReqs int
	
	mu               sync.RWMutex
	state            State
	failures         int
	successes        int
	lastFailureTime  time.Time
	halfOpenRequests int
}

// Config holds circuit breaker configuration.
type Config struct {
	Name            string
	MaxFailures     int           // Max failures before opening
	Timeout         time.Duration // Time to wait before half-open
	HalfOpenMaxReqs int           // Max requests in half-open state
}

// New creates a new circuit breaker.
func New(cfg *Config) *CircuitBreaker {
	if cfg.MaxFailures == 0 {
		cfg.MaxFailures = 5
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HalfOpenMaxReqs == 0 {
		cfg.HalfOpenMaxReqs = 3
	}
	
	return &CircuitBreaker{
		name:            cfg.Name,
		maxFailures:     cfg.MaxFailures,
		timeout:         cfg.Timeout,
		halfOpenMaxReqs: cfg.HalfOpenMaxReqs,
		state:           StateClosed,
	}
}

// Execute runs the given function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.Allow() {
		return fmt.Errorf("circuit breaker %s is open", cb.name)
	}
	
	err := fn()
	if err != nil {
		cb.RecordFailure()
		return err
	}
	
	cb.RecordSuccess()
	return nil
}

// Allow checks if a request is allowed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = StateHalfOpen
			cb.halfOpenRequests = 0
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.halfOpenRequests < cb.halfOpenMaxReqs {
			cb.halfOpenRequests++
			return true
		}
		return false
	default:
		return false
	}
}

// RecordSuccess records a successful execution.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.successes++
	switch cb.state {
	case StateClosed:
		// Reset failures on success
		cb.failures = 0
	case StateHalfOpen:
		// If enough successes in half-open, close the circuit
		if cb.successes >= cb.halfOpenMaxReqs {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		}
	}
}

// RecordFailure records a failed execution.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.failures++
	cb.lastFailureTime = time.Now()
	
	switch cb.state {
	case StateClosed:
		// Open if max failures reached
		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		// Go back to open on any failure in half-open
		cb.state = StateOpen
		cb.successes = 0
	}
}

// GetState returns the current state.
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenRequests = 0
}

// Stats returns circuit breaker statistics.
func (cb *CircuitBreaker) Stats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return Stats{
		Name:            cb.name,
		State:           cb.state,
		Failures:        cb.failures,
		Successes:       cb.successes,
		LastFailureTime: cb.lastFailureTime,
	}
}

// Stats holds circuit breaker statistics.
type Stats struct {
	Name            string    `json:"name"`
	State           State     `json:"state"`
	Failures        int       `json:"failures"`
	Successes       int       `json:"successes"`
	LastFailureTime time.Time `json:"last_failure_time,omitempty"`
}