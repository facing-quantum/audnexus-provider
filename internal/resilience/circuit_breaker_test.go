package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_SuccessKeepsClosed(t *testing.T) {
	cb := NewCircuitBreaker("test-service", 3, 50*time.Millisecond)

	for i := 0; i < 10; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("expected circuit to be StateClosed, got %v", cb.State())
	}
}

func TestCircuitBreaker_TripOnConsecutiveFailures(t *testing.T) {
	cb := NewCircuitBreaker("test-service", 3, 50*time.Millisecond)

	failingErr := errors.New("upstream timeout")

	// 1st and 2nd failures
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			return failingErr
		})
		if !errors.Is(err, failingErr) {
			t.Fatalf("expected failingErr, got %v", err)
		}
		if cb.State() != StateClosed {
			t.Errorf("expected circuit to still be StateClosed after %d failures", i+1)
		}
	}

	// 3rd failure trips the circuit
	err := cb.Execute(func() error {
		return failingErr
	})
	if !errors.Is(err, failingErr) {
		t.Fatalf("expected failingErr on 3rd failure, got %v", err)
	}
	if cb.State() != StateOpen {
		t.Errorf("expected circuit to be StateOpen after 3 failures, got %v", cb.State())
	}

	// Immediate next call should fail-fast without executing the action
	executed := false
	err = cb.Execute(func() error {
		executed = true
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
	if executed {
		t.Error("expected action NOT to be executed when circuit is open")
	}
}

func TestCircuitBreaker_HalfOpenCanaryRecovery(t *testing.T) {
	coolDown := 40 * time.Millisecond
	cb := NewCircuitBreaker("test-service", 2, coolDown)

	// Trip open with 2 failures
	_ = cb.Execute(func() error { return errors.New("err") })
	_ = cb.Execute(func() error { return errors.New("err") })
	if cb.State() != StateOpen {
		t.Fatalf("expected circuit to be open")
	}

	// Wait for cooldown to expire
	time.Sleep(coolDown + 10*time.Millisecond)

	// Next call should be in HalfOpen state; canary succeeds
	canaryExecuted := false
	err := cb.Execute(func() error {
		canaryExecuted = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected canary error: %v", err)
	}
	if !canaryExecuted {
		t.Fatal("expected canary to execute")
	}

	// Successful canary should reset to Closed
	if cb.State() != StateClosed {
		t.Errorf("expected circuit to recover to StateClosed, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenCanaryFails(t *testing.T) {
	coolDown := 40 * time.Millisecond
	cb := NewCircuitBreaker("test-service", 2, coolDown)

	// Trip open
	_ = cb.Execute(func() error { return errors.New("err") })
	_ = cb.Execute(func() error { return errors.New("err") })

	time.Sleep(coolDown + 10*time.Millisecond)

	// Canary fails
	canaryErr := errors.New("canary failed")
	err := cb.Execute(func() error {
		return canaryErr
	})
	if !errors.Is(err, canaryErr) {
		t.Fatalf("expected canaryErr, got %v", err)
	}

	// Should trip back to Open
	if cb.State() != StateOpen {
		t.Errorf("expected circuit to re-open after failed canary, got %v", cb.State())
	}
}
