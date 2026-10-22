package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLeakyBucketAllowsRequestsUnderCapacity(t *testing.T) {
	limiter := NewLeakyBucketLimiter(3, 1)

	for i := 0; i < 3; i++ {
		if !limiter.Allow("user-1") {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}
}

func TestLeakyBucketRejectsWhenFull(t *testing.T) {
	limiter := NewLeakyBucketLimiter(2, 1)

	if !limiter.Allow("user-1") {
		t.Fatal("first request should be allowed")
	}
	if !limiter.Allow("user-1") {
		t.Fatal("second request should be allowed")
	}
	if limiter.Allow("user-1") {
		t.Fatal("request should be rejected when bucket is full")
	}
}

func TestLeakyBucketLeaksOverTime(t *testing.T) {
	limiter := NewLeakyBucketLimiter(1, 20)

	if !limiter.Allow("user-1") {
		t.Fatal("first request should be allowed")
	}
	if limiter.Allow("user-1") {
		t.Fatal("second request should be rejected immediately")
	}

	time.Sleep(60 * time.Millisecond)

	if !limiter.Allow("user-1") {
		t.Fatal("request should be allowed after the bucket leaks")
	}
}

func TestLeakyBucketSeparatesClients(t *testing.T) {
	limiter := NewLeakyBucketLimiter(1, 1)

	if !limiter.Allow("user-1") {
		t.Fatal("first client's request should be allowed")
	}
	if !limiter.Allow("user-2") {
		t.Fatal("second client should have an independent bucket")
	}
}

func TestLeakyBucketConcurrentRequests(t *testing.T) {
	limiter := NewLeakyBucketLimiter(10, 1)

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
