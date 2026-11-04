package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFixedWindowAllowsRequestsUnderLimit(t *testing.T) {
	limiter := NewFixedWindowLimiter(3, time.Minute)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		allowed, err := limiter.Allow(ctx, "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}
}

func TestFixedWindowRejectsRequestsOverLimit(t *testing.T) {
	limiter := NewFixedWindowLimiter(3, time.Minute)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		limiter.Allow(ctx, "user-1")
	}

	allowed, err := limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("request beyond the limit should have been rejected")
	}
}

func TestFixedWindowSeparatesClients(t *testing.T) {
	limiter := NewFixedWindowLimiter(1, time.Minute)
	ctx := context.Background()

	allowed, err := limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("first request should be allowed")
	}

	allowed, err = limiter.Allow(ctx, "user-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("different client should have its own quota")
	}
}

func TestFixedWindowResetsAfterWindow(t *testing.T) {
	limiter := NewFixedWindowLimiter(1, 50*time.Millisecond)
	ctx := context.Background()

	allowed, err := limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("first request should be allowed")
	}

	allowed, err = limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("second request should be rejected")
	}

	time.Sleep(60 * time.Millisecond)

	allowed, err = limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("request should be allowed after the window expires")
	}
}

func TestFixedWindowConcurrentRequests(t *testing.T) {
	limiter := NewFixedWindowLimiter(10, time.Minute)
	ctx := context.Background()

	var wg sync.WaitGroup
	var allowed atomic.Int32

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			ok, err := limiter.Allow(ctx, "user-1")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if ok {
				allowed.Add(1)
			}
		}()
	}

	wg.Wait()

	if allowed.Load() != 10 {
		t.Fatalf("expected 10 allowed requests, got %d", allowed.Load())
	}
}
