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

func setupRedisSlidingCounter(t *testing.T, limit int, window time.Duration) (*RedisSlidingCounterLimiter, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	limiter := NewRedisSlidingCounterLimiter(client, limit, window)

	return limiter, mr
}

func TestRedisSlidingCounterAllowsRequestsUnderLimit(t *testing.T) {
	limiter, _ := setupRedisSlidingCounter(t, 3, time.Minute)
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

func TestRedisSlidingCounterRejectsRequestsOverLimit(t *testing.T) {
	limiter, _ := setupRedisSlidingCounter(t, 3, time.Minute)
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

func TestRedisSlidingCounterSeparatesClients(t *testing.T) {
	limiter, _ := setupRedisSlidingCounter(t, 1, time.Minute)
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

func TestRedisSlidingCounterAdvancesWindow(t *testing.T) {
	limiter, _ := setupRedisSlidingCounter(t, 2, 100*time.Millisecond)
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

func TestRedisSlidingCounterConcurrentRequests(t *testing.T) {
	limiter, _ := setupRedisSlidingCounter(t, 10, time.Minute)
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

func TestRedisSlidingCounterCancelledContext(t *testing.T) {
	limiter, _ := setupRedisSlidingCounter(t, 5, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := limiter.Allow(ctx, "user-1")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRedisSlidingCounterRedisDown(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := NewRedisSlidingCounterLimiter(client, 5, time.Minute)

	mr.Close()

	_, err := limiter.Allow(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error when Redis is down")
	}
}

