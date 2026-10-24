package ratelimit

type RateLimiter interface {
	Allow(clientID string) bool
}
