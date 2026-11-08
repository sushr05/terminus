package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sushr05/terminus/scripts"
)

type RedisSlidingCounterLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
	script *redis.Script
}

func NewRedisSlidingCounterLimiter(
	client *redis.Client,
	limit int,
	window time.Duration,
) *RedisSlidingCounterLimiter {
	return &RedisSlidingCounterLimiter{
		client: client,
		limit:  limit,
		window: window,
		script: redis.NewScript(scripts.RedisSlidingCounter),
	}
}

func (l *RedisSlidingCounterLimiter) Allow(
	ctx context.Context,
	clientID string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	key := fmt.Sprintf(
		"terminus:ratelimit:sliding_counter:{%s}",
		clientID,
	)

	nowMs := time.Now().UnixMilli()

	result, err := l.script.Run(
		ctx,
		l.client,
		[]string{key},
		l.window.Milliseconds(),
		l.limit,
		nowMs,
	).Int()

	if err != nil {
		return false, fmt.Errorf("redis sliding counter: %w", err)
	}

	return result == 1, nil
}

