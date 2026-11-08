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

func setupRedisTokenBucket(t *testing.T, capacity int, refillRate float64) (*RedisTokenBucketLimiter, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	limiter := NewRedisTokenBucketLimiter(client, capacity, refillRate)

	return limiter, mr
}

func TestRedisTokenBucketAllowsBurst(t *testing.T) {
	limiter, _ := setupRedisTokenBucket(t, 3, 1)
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

	allowed, err := limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("request beyond bucket capacity should be rejected")
	}
}

func TestRedisTokenBucketRefills(t *testing.T) {
	limiter, _ := setupRedisTokenBucket(t, 1, 20)
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
		t.Fatal("bucket should be empty immediately after first request")
	}

	time.Sleep(60 * time.Millisecond)

	allowed, err = limiter.Allow(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("request should be allowed after a token refills")
	}
}

func TestRedisTokenBucketSeparatesClients(t *testing.T) {
	limiter, _ := setupRedisTokenBucket(t, 1, 1)
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

func TestRedisTokenBucketConcurrentRequests(t *testing.T) {
	limiter, _ := setupRedisTokenBucket(t, 10, 1)
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

func TestRedisTokenBucketCancelledContext(t *testing.T) {
	limiter, _ := setupRedisTokenBucket(t, 5, 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := limiter.Allow(ctx, "user-1")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRedisTokenBucketRedisDown(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := NewRedisTokenBucketLimiter(client, 5, 1)

	mr.Close()

	_, err := limiter.Allow(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error when Redis is down")
	}
}

