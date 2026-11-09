package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sushr05/terminus/internal/config"
)

func TestNewLimiterLocal(t *testing.T) {
	algorithms := []string{
		"fixed_window",
		"sliding_log",
		"sliding_counter",
		"token_bucket",
		"leaky_bucket",
	}

	for _, algo := range algorithms {
		t.Run(algo, func(t *testing.T) {
			cfg := config.Config{
				Mode:                  "local",
				Algorithm:             algo,
				FixedWindowLimit:      5,
				FixedWindowDuration:   time.Minute,
				SlidingLogLimit:       5,
				SlidingLogWindow:      time.Minute,
				SlidingCounterLimit:   5,
				SlidingCounterWindow:  time.Minute,
				TokenBucketCapacity:   5,
				TokenBucketRefillRate: 1,
				LeakyBucketCapacity:   5,
				LeakyBucketLeakRate:   1,
			}

			limiter, err := NewLimiter(cfg, nil)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if limiter == nil {
				t.Fatal("expected non-nil limiter")
			}

			allowed, err := limiter.Allow(context.Background(), "user-1")
			if err != nil {
				t.Fatalf("unexpected error on Allow: %v", err)
			}
			if !allowed {
				t.Fatal("expected first request to be allowed")
			}
		})
	}
}

func TestNewLimiterDistributed(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	algorithms := []string{
		"fixed_window",
		"sliding_log",
		"sliding_counter",
		"token_bucket",
		"leaky_bucket",
	}

	for _, algo := range algorithms {
		t.Run(algo, func(t *testing.T) {
			cfg := config.Config{
				Mode:                  "distributed",
				Algorithm:             algo,
				FixedWindowLimit:      5,
				FixedWindowDuration:   time.Minute,
				SlidingLogLimit:       5,
				SlidingLogWindow:      time.Minute,
				SlidingCounterLimit:   5,
				SlidingCounterWindow:  time.Minute,
				TokenBucketCapacity:   5,
				TokenBucketRefillRate: 1,
				LeakyBucketCapacity:   5,
				LeakyBucketLeakRate:   1,
			}

			limiter, err := NewLimiter(cfg, client)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if limiter == nil {
				t.Fatal("expected non-nil limiter")
			}

			allowed, err := limiter.Allow(context.Background(), "user-1")
			if err != nil {
				t.Fatalf("unexpected error on Allow: %v", err)
			}
			if !allowed {
				t.Fatal("expected first request to be allowed")
			}
		})
	}
}

func TestNewLimiterDistributedRequiresClient(t *testing.T) {
	cfg := config.Config{
		Mode:      "distributed",
		Algorithm: "token_bucket",
	}

	_, err := NewLimiter(cfg, nil)
	if err == nil {
		t.Fatal("expected error when client is nil in distributed mode")
	}
}

func TestNewLimiterInvalidMode(t *testing.T) {
	cfg := config.Config{
		Mode: "unknown_mode",
	}

	_, err := NewLimiter(cfg, nil)
	if err == nil {
		t.Fatal("expected error for unsupported mode")
	}
}

func TestNewLimiterInvalidAlgorithm(t *testing.T) {
	cfgLocal := config.Config{
		Mode:      "local",
		Algorithm: "magic_algo",
	}
	if _, err := NewLimiter(cfgLocal, nil); err == nil {
		t.Fatal("expected error for unsupported local algorithm")
	}

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cfgDistributed := config.Config{
		Mode:      "distributed",
		Algorithm: "magic_algo",
	}
	if _, err := NewLimiter(cfgDistributed, client); err == nil {
		t.Fatal("expected error for unsupported distributed algorithm")
	}
}

