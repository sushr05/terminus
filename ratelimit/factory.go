package ratelimit

import (
	"fmt"

	"github.com/sushr05/terminus/internal/config"
)

func NewLimiter(cfg config.Config) (RateLimiter, error) {
	//if err := cfg.Validate(); err != nil {
	//return nil, fmt.Errorf("Invalid rate limiter config:%w", err)
	//} not required since when we load cfg its already being validated.

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
