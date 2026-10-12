package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFixedWindowAllowsRequestsUnderLimit(t *testing.T) {
	limiter := newFixedWindowLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !limiter.allow("user-1") {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}
}

func TestFixedWindowRejectsRequestsOverLimit(t *testing.T) {
	limiter := newFixedWindowLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		limiter.allow("user-1")
	}

	if limiter.allow("user-1") {
		t.Fatal("request beyond the limit should have been rejected")
	}
}

func TestFixedWindowSeparatesClients(t *testing.T) {
	limiter := newFixedWindowLimiter(1, time.Minute)

	if !limiter.allow("user-1") {
		t.Fatal("first request should be allowed")
	}

	if !limiter.allow("user-2") {
		t.Fatal("different client should have its own quota")
	}
}

func TestFixedWindowResetsAfterWindow(t *testing.T) {
	limiter := newFixedWindowLimiter(1, 50*time.Millisecond)

	if !limiter.allow("user-1") {
		t.Fatal("first request should be allowed")
	}

	if limiter.allow("user-1") {
		t.Fatal("second request should be rejected")
	}

	time.Sleep(60 * time.Millisecond)

	if !limiter.allow("user-1") {
		t.Fatal("request should be allowed after the window expires")
	}
}

func TestFixedWindowConcurrentRequests(t *testing.T) {
	limiter := newFixedWindowLimiter(10, time.Minute)

	var wg sync.WaitGroup
	var allowed atomic.Int32

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if limiter.allow("user-1") {
				allowed.Add(1)
			}
		}()
	}

	wg.Wait()

	if allowed.Load() != 10 {
		t.Fatalf("expected 10 allowed requests, got %d", allowed.Load())
	}
}
