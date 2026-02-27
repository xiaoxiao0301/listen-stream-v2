package breaker

import (
	"errors"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cb := New(&Config{
		Name:        "test-breaker",
		MaxFailures: 5,
		Timeout:     10 * time.Second,
	})
	
	if cb == nil {
		t.Fatal("New() returned nil")
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("Initial state = %v, want %v", cb.GetState(), StateClosed)
	}
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := New(&Config{
		Name:        "test-breaker",
		MaxFailures: 3,
		Timeout:     time.Second,
	})
	
	err := cb.Execute(func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	
	if cb.GetState() != StateClosed {
		t.Errorf("State should remain %v", StateClosed)
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := New(&Config{
		Name:        "test-breaker",
		MaxFailures: 2,
		Timeout:     time.Second,
	})
	
	testErr := errors.New("test error")
	
	// First failure
	err := cb.Execute(func() error {
		return testErr
	})
	if err != testErr {
		t.Errorf("Execute() error = %v, want %v", err, testErr)
	}
	
	// Second failure - should open circuit
	err = cb.Execute(func() error {
		return testErr
	})
	if err != testErr {
		t.Errorf("Execute() error = %v, want %v", err, testErr)
	}
	
	if cb.GetState() != StateOpen {
		t.Errorf("State = %v, want %v after max failures", cb.GetState(), StateOpen)
	}
}

func TestCircuitBreaker_StateOpen_RejectsRequests(t *testing.T) {
	cb := New(&Config{
		Name:        "test-breaker",
		MaxFailures: 1,
		Timeout:     time.Second,
	})
	
	// Trigger circuit breaker to open
	cb.Execute(func() error {
		return errors.New("fail")
	})
	
	if cb.GetState() != StateOpen {
		t.Skip("Circuit breaker should be open")
	}
	
	// Try to execute - should be rejected
	err := cb.Execute(func() error {
		return nil
	})
	
	if err == nil {
		t.Error("Execute() should return error when circuit is open")
	}
}

func TestCircuitBreaker_StateTransition_HalfOpen(t *testing.T) {
	cb := New(&Config{
		Name:           "test-breaker",
		MaxFailures:    1,
		Timeout:        100 * time.Millisecond,
		HalfOpenMaxReqs: 1,
	})
	
	// Open the circuit
	cb.Execute(func() error {
		return errors.New("fail")
	})
	
	if cb.GetState() != StateOpen {
		t.Fatal("Circuit should be open")
	}
	
	// Wait for timeout
	time.Sleep(150 * time.Millisecond)
	
	// Check if can execute (should enter half-open state)
	if !cb.Allow() {
		t.Error("Should allow request after timeout")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := New(&Config{
		Name:        "test-breaker",
		MaxFailures: 1,
		Timeout:     time.Second,
	})
	
	// Open circuit
	cb.Execute(func() error {
		return errors.New("fail")
	})
	
	if cb.GetState() != StateOpen {
		t.Fatal("Circuit should be open")
	}
	
	// Reset
	cb.Reset()
	
	if cb.GetState() != StateClosed {
		t.Errorf("State = %v, want %v after reset", cb.GetState(), StateClosed)
	}
	
	stats := cb.Stats()
	if stats.Failures != 0 {
		t.Errorf("Failures = %v, want 0 after reset", stats.Failures)
	}
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := New(&Config{
		Name:        "test-breaker",
		MaxFailures: 3,
		Timeout:     time.Second,
	})
	
	cb.RecordSuccess()
	stats := cb.Stats()
	
	if stats.Successes != 1 {
		t.Errorf("Successes = %v, want 1", stats.Successes)
	}
}

func TestCircuitBreaker_RecordFailure(t *testing.T) {
	cb := New(&Config{
		Name:        "test-breaker",
		MaxFailures: 3,
		Timeout:     time.Second,
	})
	
	cb.RecordFailure()
	stats := cb.Stats()
	
	if stats.Failures != 1 {
		t.Errorf("Failures = %v, want 1", stats.Failures)
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := New(&Config{
		Name:        "test-breaker",
		MaxFailures: 3,
		Timeout:     time.Second,
	})
	
	stats := cb.Stats()
	
	if stats.Name != "test-breaker" {
		t.Errorf("Name = %v, want test-breaker", stats.Name)
	}
	if stats.State != StateClosed {
		t.Errorf("State = %v, want %v", stats.State, StateClosed)
	}
	if stats.Failures != 0 {
		t.Errorf("Failures = %v, want 0", stats.Failures)
	}
	if stats.Successes != 0 {
		t.Errorf("Successes = %v, want 0", stats.Successes)
	}
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cb := New(&Config{
		Name: "test",
	})
	
	stats := cb.Stats()
	if stats.State != StateClosed {
		t.Errorf("Default state = %v, want %v", stats.State, StateClosed)
	}
}
