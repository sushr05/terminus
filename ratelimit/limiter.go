package ratelimit

import "context"

type RateLimiter interface {
	Allow(ctx context.Context, clientID string) (bool, error)
}
