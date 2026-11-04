package ratelimit

import (
	"context"
	"sync"
	"time"
)

type counterWindowState struct {
	previousCount int
	currentCount  int
	windowStart   time.Time
}

type SlidingWindowCounterLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]counterWindowState
}

func NewSlidingWindowCounterLimiter(
	limit int,
	window time.Duration,
) *SlidingWindowCounterLimiter {
	if limit <= 0 {
		panic("limit should be positive")
	}
	if window <= 0 {
		panic("winodw should be positive")
	}

	return &SlidingWindowCounterLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string]counterWindowState),
	}
}

func (l *SlidingWindowCounterLimiter) Allow(_ context.Context, clientID string) (bool, error) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	state, exists := l.clients[clientID]

	if !exists {
		state = counterWindowState{windowStart: now}
	}

	elapsed := now.Sub(state.windowStart)

	if elapsed >= l.window {
		windowsPassed := int64(elapsed / l.window)

		if windowsPassed == 1 {
			state.previousCount = state.currentCount
		} else {
			state.previousCount = 0
		}

		state.currentCount = 0
		state.windowStart = state.windowStart.Add(
			time.Duration(windowsPassed) * l.window,
		)
	}

	elapsed = now.Sub(state.windowStart)
	elapsedFraction := float64(elapsed) / float64(l.window)

	estimatedCount := float64(state.previousCount)*(1-elapsedFraction) + float64(state.currentCount)

	if estimatedCount >= float64(l.limit) {
		l.clients[clientID] = state
		return false, nil
	}

	state.currentCount++
	l.clients[clientID] = state

	return true, nil
}
