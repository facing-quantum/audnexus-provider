package resilience

import (
	"errors"
	"sync"
	"time"
)

// State represents the current circuit state
type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateHalfOpen:
		return "HALF-OPEN"
	case StateOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// ErrCircuitOpen is returned when an action is rejected because the circuit is open
var ErrCircuitOpen = errors.New("circuit breaker is open; failing fast")

// CircuitBreaker guards against cascading failures to failing upstreams
type CircuitBreaker struct {
	mu sync.Mutex

	name             string
	failureThreshold int
	coolDown         time.Duration

	state            State
	failureCount     int
	lastFailureTime  time.Time
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, failureThreshold int, coolDown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		failureThreshold: failureThreshold,
		coolDown:         coolDown,
		state:            StateClosed,
	}
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.checkStateLocked()
	return cb.state
}

func (cb *CircuitBreaker) checkStateLocked() {
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) >= cb.coolDown {
			cb.state = StateHalfOpen
		}
	}
}

// Execute runs the provided action if the circuit is not open
func (cb *CircuitBreaker) Execute(action func() error) error {
	cb.mu.Lock()
	cb.checkStateLocked()

	if cb.state == StateOpen {
		cb.mu.Unlock()
		return ErrCircuitOpen
	}

	isHalfOpen := (cb.state == StateHalfOpen)
	cb.mu.Unlock()

	// Execute action
	err := action()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()
		if isHalfOpen || cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
		}
		return err
	}

	// Success resets circuit
	cb.failureCount = 0
	cb.state = StateClosed
	return nil
}
