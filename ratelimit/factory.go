package ratelimit

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/sushr05/terminus/internal/config"
)

func NewLimiter(cfg config.Config, client *redis.Client) (RateLimiter, error) {
	switch cfg.Mode {
	case "local":
		return newLocalLimiter(cfg)
	case "distributed":
		if client == nil {
			return nil, fmt.Errorf("redis client is required for distributed mode")
		}
		return newDistributedLimiter(cfg, client)
	default:
		return nil, fmt.Errorf("unsupported mode: %q", cfg.Mode)
	}
}

func newLocalLimiter(cfg config.Config) (RateLimiter, error) {
	switch cfg.Algorithm {
	case "fixed_window":
		return NewFixedWindowLimiter(
			cfg.FixedWindowLimit,
			cfg.FixedWindowDuration,
		), nil
	case "sliding_log":
		return NewSlidingWindowLogLimiter(
			cfg.SlidingLogLimit,
			cfg.SlidingLogWindow,
		), nil
	case "sliding_counter":
		return NewSlidingWindowCounterLimiter(
			cfg.SlidingCounterLimit,
			cfg.SlidingCounterWindow,
		), nil
	case "token_bucket":
		return NewTokenBucketLimiter(
			cfg.TokenBucketCapacity,
			cfg.TokenBucketRefillRate,
		), nil
	case "leaky_bucket":
		return NewLeakyBucketLimiter(
			cfg.LeakyBucketCapacity,
			cfg.LeakyBucketLeakRate,
		), nil
	default:
		return nil, fmt.Errorf(
			"unsupported rate limiter algorithm: %q",
			cfg.Algorithm,
		)
	}
}

func newDistributedLimiter(cfg config.Config, client *redis.Client) (RateLimiter, error) {
	switch cfg.Algorithm {
	case "fixed_window":
		return NewRedisFixedWindowLimiter(
			client,
			cfg.FixedWindowLimit,
			cfg.FixedWindowDuration,
		), nil
	case "sliding_log":
		return NewRedisSlidingLogLimiter(
			client,
			cfg.SlidingLogLimit,
			cfg.SlidingLogWindow,
		), nil
	case "sliding_counter":
		return NewRedisSlidingCounterLimiter(
			client,
			cfg.SlidingCounterLimit,
			cfg.SlidingCounterWindow,
		), nil
	case "token_bucket":
		return NewRedisTokenBucketLimiter(
			client,
			cfg.TokenBucketCapacity,
			cfg.TokenBucketRefillRate,
		), nil
	case "leaky_bucket":
		return NewRedisLeakyBucketLimiter(
			client,
			cfg.LeakyBucketCapacity,
			cfg.LeakyBucketLeakRate,
		), nil
	default:
		return nil, fmt.Errorf(
			"unsupported rate limiter algorithm: %q",
			cfg.Algorithm,
		)
	}
}
