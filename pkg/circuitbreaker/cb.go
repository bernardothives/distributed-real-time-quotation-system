package circuitbreaker

import (
	"errors"
	"sync"
	"time"
	"fmt"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu           sync.Mutex
	state        State
	failureCount int
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:    threshold,
		resetTimeout: timeout,
		state:        StateClosed,
	}
}

func (cb *CircuitBreaker) Execute(action func() (interface{}, error)) (interface{}, error) {
	cb.mu.Lock()

	// Verificar Estado
	if cb.state == StateOpen {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			fmt.Println("[CB] Circuit transitioning to HALF-OPEN")
			cb.state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return nil, errors.New("circuit breaker is OPEN")
		}
	}
	cb.mu.Unlock()

	// Executar Ação
	result, err := action()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailure = time.Now()
		fmt.Printf("[CB] Failure detected. Count: %d/%d\n", cb.failureCount, cb.threshold)
		if cb.failureCount >= cb.threshold {
			fmt.Println("[CB] Failure threshold reached. Circuit OPEN.")
			cb.state = StateOpen
		}
		return nil, err
	}

	// Sucesso
	if cb.state == StateHalfOpen {
		fmt.Println("[CB] Success in Half-Open. Circuit CLOSED.")
		cb.state = StateClosed
		cb.failureCount = 0
	} else if cb.state == StateClosed {
		cb.failureCount = 0
	}

	return result, nil
}
