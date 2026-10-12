package ratelimit

import (
	"sync"
	"time"
)

type windowState struct {
	count           int
	windowStartTime time.Time
}

type fixedWindowLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]windowState
}

func newFixedWindowLimiter(
	limit int,
	window time.Duration,
) *fixedWindowLimiter {
	if limit <= 0 {
		panic("limit must be positive")
	}
	if window <= 0 {
		panic("window must be positive")
	}

	return &fixedWindowLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string]windowState),
	}
}

func (l *fixedWindowLimiter) allow(clientID string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	state, exists := l.clients[clientID]

	if !exists || now.Sub(state.windowStartTime) >= l.window {
		state = windowState{
			count:           0,
			windowStartTime: now,
		}
	}

	if state.count > l.limit {
		l.clients[clientID] = state
		return false
	}

	state.count++
	l.clients[clientID] = state

	return true
}
