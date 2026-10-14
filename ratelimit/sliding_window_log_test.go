package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSlidingWindowLogLimit(t *testing.T) {
	limiter := NewSlidingWindowLogLimiter(2, time.Minute)
	if !limiter.Allow("user-1") {
		t.Fatal("first request should be allowed")
	}
	if !limiter.Allow("user-1") {
		t.Fatal("second request should be allowed")
	}
	if limiter.Allow("user-1") {
		t.Fatal("third request should not be allowed")
	}
}

func TestSlidingWindowLogSeparatesClients(t *testing.T) {
	limiter := NewSlidingWindowLogLimiter(1, time.Minute)

	if !limiter.Allow("user-1") {
		t.Fatal("first client's request should be allowed")
	}
	if !limiter.Allow("user-2") {
		t.Fatal("second client's request should be allowed")
	}
}

func TestSlidingWindowLogExpiresRequests(t *testing.T) {
	limiter := NewSlidingWindowLogLimiter(1, 50*time.Millisecond)

	if !limiter.Allow("user-1") {
		t.Fatal("first request should be allowed")
	}

	time.Sleep(70 * time.Millisecond)

	if !limiter.Allow("user-1") {
		t.Fatal("request should be allowed after the window expires")
	}
}

func TestSlidingWindowLogConcurrentRequests(t *testing.T) {
	limiter := NewSlidingWindowLogLimiter(10, time.Minute)

	var wg sync.WaitGroup
	var allowed atomic.Int32

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if limiter.Allow("user-1") {
				allowed.Add(1)
			}
		}()
	}

	wg.Wait()

	if got := allowed.Load(); got != 10 {
		t.Fatalf("expected 10 allowed requests, got %d", got)
	}
}
