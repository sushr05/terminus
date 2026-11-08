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

func setupRedisLeakyBucket(t *testing.T, capacity int, leakRate float64) (*RedisLeakyBucketLimiter, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	limiter := NewRedisLeakyBucketLimiter(client, capacity, leakRate)

	return limiter, mr
}

func TestRedisLeakyBucketAllowsRequestsUnderCapacity(t *testing.T) {
	limiter, _ := setupRedisLeakyBucket(t, 3, 1)
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

func TestRedisLeakyBucketRejectsWhenFull(t *testing.T) {
	limiter, _ := setupRedisLeakyBucket(t, 2, 1)
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
		t.Fatal("request should be rejected when bucket is full")
	}
}

func TestRedisLeakyBucketLeaksOverTime(t *testing.T) {
	limiter, _ := setupRedisLeakyBucket(t, 1, 20)
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
		t.Fatal("second request should be rejected immediately")
	}

	time.Sleep(60 * time.Millisecond)

	allowed, err = limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("request should be allowed after the bucket leaks")
	}
}

func TestRedisLeakyBucketSeparatesClients(t *testing.T) {
	limiter, _ := setupRedisLeakyBucket(t, 1, 1)
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
		t.Fatal("second client should have an independent bucket")
	}
}

func TestRedisLeakyBucketConcurrentRequests(t *testing.T) {
	limiter, _ := setupRedisLeakyBucket(t, 10, 1)
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

func TestRedisLeakyBucketCancelledContext(t *testing.T) {
	limiter, _ := setupRedisLeakyBucket(t, 5, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := limiter.Allow(ctx, "user-1")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRedisLeakyBucketRedisDown(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := NewRedisLeakyBucketLimiter(client, 5, 1)

	mr.Close()

	_, err := limiter.Allow(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error when Redis is down")
	}
}

