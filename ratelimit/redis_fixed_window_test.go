package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupRedisFixedWindow(t *testing.T, limit int, window time.Duration) (*RedisFixedWindowLimiter, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	limiter := NewRedisFixedWindowLimiter(client, limit, window)

	return limiter, mr
}

func TestRedisFixedWindowAllowsRequestsUnderLimit(t *testing.T) {
	limiter, _ := setupRedisFixedWindow(t, 3, time.Minute)
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

func TestRedisFixedWindowRejectsRequestsOverLimit(t *testing.T) {
	limiter, _ := setupRedisFixedWindow(t, 3, time.Minute)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := limiter.Allow(ctx, "user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	allowed, err := limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("request beyond the limit should have been rejected")
	}
}

func TestRedisFixedWindowSeparatesClients(t *testing.T) {
	limiter, _ := setupRedisFixedWindow(t, 1, time.Minute)
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
		t.Fatal("second client should have its own quota")
	}
}

func TestRedisFixedWindowResetsAfterExpiry(t *testing.T) {
	limiter, mr := setupRedisFixedWindow(t, 1, 2*time.Second)
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

	// Fast-forward time in miniredis instead of sleeping.
	mr.FastForward(3 * time.Second)

	allowed, err = limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("request should be allowed after the window expires")
	}
}

func TestRedisFixedWindowConcurrentRequests(t *testing.T) {
	limiter, _ := setupRedisFixedWindow(t, 10, time.Minute)
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

func TestRedisFixedWindowCancelledContext(t *testing.T) {
	limiter, _ := setupRedisFixedWindow(t, 5, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := limiter.Allow(ctx, "user-1")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRedisFixedWindowRedisDown(t *testing.T) {
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	limiter := NewRedisFixedWindowLimiter(client, 5, time.Minute)

	// Shut down Redis.
	mr.Close()

	_, err := limiter.Allow(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error when Redis is down")
	}
}

