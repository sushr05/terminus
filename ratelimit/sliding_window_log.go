package ratelimit

import (
	"sync"
	"time"
)

type SlidingWindowLogLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string][]time.Time
}

func NewSlidingWindowLogLimiter(
	limit int,
	window time.Duration,
) *SlidingWindowLogLimiter {
	if limit <= 0 {
		panic("limit has to be positive")
	}
	if window <= 0 {
		panic("window has to be positive")
	}

	return &SlidingWindowLogLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string][]time.Time),
	}
}

func (l *SlidingWindowLogLimiter) Allow(clientID string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamps := l.clients[clientID]

	firstValid := 0
	for firstValid < len(timestamps) && !timestamps[firstValid].After(cutoff) {
		firstValid++
	}

	timestamps = timestamps[firstValid:]

	if len(timestamps) >= l.limit {
		l.clients[clientID] = timestamps
		return false
	}

	timestamps = append(timestamps, now)
	l.clients[clientID] = timestamps

	return true
}
