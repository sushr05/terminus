package ratelimit

import (
	"context"
	"fmt"
	"log"
)

// FallbackLimiter wraps a primary (typically Redis-backed distributed) limiter
// and a fallback (typically in-memory local) limiter. If the primary limiter returns
// an error (e.g. Redis connectivity/timeout error), FallbackLimiter logs a warning
// and degrades gracefully by delegating the decision to the fallback limiter.
type FallbackLimiter struct {
	primary  RateLimiter
	fallback RateLimiter
}

func NewFallbackLimiter(primary, fallback RateLimiter) (*FallbackLimiter, error) {
	if primary == nil {
		return nil, fmt.Errorf("primary rate limiter cannot be nil")
	}
	if fallback == nil {
		return nil, fmt.Errorf("fallback rate limiter cannot be nil")
	}

	return &FallbackLimiter{
		primary:  primary,
		fallback: fallback,
	}, nil
}

func (l *FallbackLimiter) Allow(ctx context.Context, clientID string) (bool, error) {
	allowed, err := l.primary.Allow(ctx, clientID)
	if err == nil {
		return allowed, nil
	}

	log.Printf("[WARN] primary rate limiter failed (%v); falling back to in-memory limiter for client %q", err, clientID)

	return l.fallback.Allow(ctx, clientID)
}

