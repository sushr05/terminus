package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSlidingWindowCounterAllowsRequestsUnderLimit(t *testing.T) {
	limiter := NewSlidingWindowCounterLimiter(3, time.Minute)
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

func TestSlidingWindowCounterRejectsRequestsOverLimit(t *testing.T) {
	limiter := NewSlidingWindowCounterLimiter(3, time.Minute)
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

func TestSlidingWindowCounterSeparatesClients(t *testing.T) {
	limiter := NewSlidingWindowCounterLimiter(1, time.Minute)
	ctx := context.Background()

	allowed, err := limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("first client's request should be allowed")
	}

	allowed, err = limiter.Allow(ctx, "user-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("second client should have an independent quota")
	}
}

func TestSlidingWindowCounterAdvancesWindow(t *testing.T) {
	limiter := NewSlidingWindowCounterLimiter(2, 100*time.Millisecond)
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
	if !allowed {
		t.Fatal("second request should be allowed")
	}

	allowed, err = limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("third request should be rejected")
	}

	time.Sleep(120 * time.Millisecond)

	allowed, err = limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("request should be allowed after the previous window expires")
	}
}

func TestSlidingWindowCounterConcurrentRequests(t *testing.T) {
	limiter := NewSlidingWindowCounterLimiter(10, time.Minute)
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

	if got := allowed.Load(); got != 10 {
		t.Fatalf("expected 10 allowed requests, got %d", got)
	}
}
