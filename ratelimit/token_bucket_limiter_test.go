package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucketAllowsBurst(t *testing.T) {
	limiter := NewTokenBucketLimiter(3, 1)

	for i := 0; i < 3; i++ {
		if !limiter.Allow("user-1") {
			t.Fatalf("request %d should have been allowed", i+1)
		}
	}

	if limiter.Allow("user-1") {
		t.Fatal("request beyond bucket capacity should be rejected")
	}
}

func TestTokenBucketRefills(t *testing.T) {
	limiter := NewTokenBucketLimiter(1, 20)

	if !limiter.Allow("user-1") {
		t.Fatal("first request should be allowed")
	}

	if limiter.Allow("user-1") {
		t.Fatal("bucket should be empty immediately after first request")
	}

	time.Sleep(60 * time.Millisecond)

	if !limiter.Allow("user-1") {
		t.Fatal("request should be allowed after a token refills")
	}
}

func TestTokenBucketSeparatesClients(t *testing.T) {
	limiter := NewTokenBucketLimiter(1, 1)

	if !limiter.Allow("user-1") {
		t.Fatal("first client's request should be allowed")
	}

	if !limiter.Allow("user-2") {
		t.Fatal("second client should have an independent bucket")
	}
}

func TestTokenBucketConcurrentRequests(t *testing.T) {
	limiter := NewTokenBucketLimiter(10, 1)

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
