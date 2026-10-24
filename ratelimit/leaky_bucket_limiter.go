package ratelimit

import (
	"sync"
	"time"
)

type leakyBucketState struct {
	level     float64 //why float because tokens are generated over time not in a fixed interval
	lastCheck time.Time
}

type LeakyBucketLimiter struct {
	mu       sync.Mutex
	capacity float64
	leakRate float64
	clients  map[string]leakyBucketState
}

func NewLeakyBucketLimiter(
	capacity int,
	leakRate float64,
) *LeakyBucketLimiter {
	if capacity <= 0 {
		panic("capacity should be positive")
	}
	if leakRate <= 0 {
		panic("rate should be positive")
	}

	return &LeakyBucketLimiter{
		capacity: float64(capacity),
		leakRate: leakRate,
		clients:  make(map[string]leakyBucketState),
	}
}

func (l *LeakyBucketLimiter) Allow(clientID string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	state, exists := l.clients[clientID]

	if !exists {
		state = leakyBucketState{lastCheck: now}
	}

	elapsed := now.Sub(state.lastCheck).Seconds()

	state.level -= elapsed * l.leakRate //we calculate the leak when a request arrives, rather than running a background goroutine to empty the bucket continuously.

	if state.level < 0 {
		state.level = 0
	}

	state.lastCheck = now

	if state.level+1 > l.capacity {
		l.clients[clientID] = state
		return false
	}

	state.level++
	l.clients[clientID] = state

	return true
}
