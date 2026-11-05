package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sushr05/terminus/scripts"
)

type RedisFixedWindowLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
	script *redis.Script
}

func NewRedisFixedWindowLimiter(
	client *redis.Client,
	limit int,
	window time.Duration,
) *RedisFixedWindowLimiter {
	return &RedisFixedWindowLimiter{
		client: client,
		limit:  limit,
		window: window,
		script: redis.NewScript(scripts.RedisFixedWindow),
	}
}

func (l *RedisFixedWindowLimiter) Allow(
	ctx context.Context,
	clientID string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	key := fmt.Sprintf(
		"terminus:ratelimit:fixed_window:{%s}",
		clientID,
	)

	result, err := l.script.Run(
		ctx,
		l.client,
		[]string{key},
		l.window.Milliseconds(),
		l.limit,
	).Int()

	if err != nil {
		return false, fmt.Errorf("redis fixed window: %w", err)
	}

	return result == 1, nil
}
