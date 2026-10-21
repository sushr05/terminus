package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSlidingWindowCounterAllowsRequestsUnderLimit(t *testing.T) {
	limiter := NewSlidingWindowCounterLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !limiter.Allow("user-1") {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}
}

func TestSlidingWindowCounterRejectsRequestsOverLimit(t *testing.T) {
	limiter := NewSlidingWindowCounterLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		limiter.Allow("user-1")
	}

	if limiter.Allow("user-1") {
		t.Fatal("request beyond the limit should have been rejected")
	}
}

func TestSlidingWindowCounterSeparatesClients(t *testing.T) {
	limiter := NewSlidingWindowCounterLimiter(1, time.Minute)

	if !limiter.Allow("user-1") {
		t.Fatal("first client's request should be allowed")
	}

	if !limiter.Allow("user-2") {
		t.Fatal("second client should have an independent quota")
	}
}

func TestSlidingWindowCounterAdvancesWindow(t *testing.T) {
	limiter := NewSlidingWindowCounterLimiter(2, 100*time.Millisecond)

	if !limiter.Allow("user-1") {
		t.Fatal("first request should be allowed")
	}
	if !limiter.Allow("user-1") {
		t.Fatal("second request should be allowed")
	}
	if limiter.Allow("user-1") {
		t.Fatal("third request should be rejected")
	}

	time.Sleep(120 * time.Millisecond)

	if !limiter.Allow("user-1") {
		t.Fatal("request should be allowed after the previous window expires")
	}
}

func TestSlidingWindowCounterConcurrentRequests(t *testing.T) {
	limiter := NewSlidingWindowCounterLimiter(10, time.Minute)

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
