package ratelimit

import (
	"sync"
	"time"
)

type bucketState struct {
	tokens     float64
	lastRefill time.Time
}

type TokenBucketLimiter struct {
	mu         sync.Mutex
	capacity   float64
	refillRate float64
	clients    map[string]bucketState
}

func NewTokenBucketLimiter(
	capacity int, refillRate float64,
) *TokenBucketLimiter {
	if capacity <= 0 {
		panic("capacity must be positive")
	}
	if refillRate <= 0 {
		panic("refill rate must be positive")
	}

	return &TokenBucketLimiter{
		capacity:   float64(capacity),
		refillRate: refillRate,
		clients:    make(map[string]bucketState),
	}
}

func (l *TokenBucketLimiter) Allow(clientID string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	state, exists := l.clients[clientID]
	if !exists {
		state = bucketState{
			tokens:     l.capacity,
			lastRefill: now,
		}
	}

	elapsed := now.Sub(state.lastRefill).Seconds()
	state.tokens += elapsed * l.refillRate

	if state.tokens > l.capacity {
		state.tokens = l.capacity
	}

	state.lastRefill = now

	if state.tokens < 1 {
		l.clients[clientID] = state
		return false
	}

	state.tokens--
	l.clients[clientID] = state

	return true

}
