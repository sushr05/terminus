package ratelimit

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sushr05/terminus/scripts"
)

type RedisSlidingLogLimiter struct {
	client     *redis.Client
	limit      int
	window     time.Duration
	script     *redis.Script
	reqCounter atomic.Uint64
}

func NewRedisSlidingLogLimiter(
	client *redis.Client,
	limit int,
	window time.Duration,
) *RedisSlidingLogLimiter {
	return &RedisSlidingLogLimiter{
		client: client,
		limit:  limit,
		window: window,
		script: redis.NewScript(scripts.RedisSlidingLog),
	}
}

func (l *RedisSlidingLogLimiter) Allow(
	ctx context.Context,
	clientID string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	key := fmt.Sprintf(
		"terminus:ratelimit:sliding_log:{%s}",
		clientID,
	)

	now := time.Now()
	nowMs := now.UnixMilli()
	member := fmt.Sprintf("%d-%d", now.UnixNano(), l.reqCounter.Add(1))

	result, err := l.script.Run(
		ctx,
		l.client,
		[]string{key},
		l.window.Milliseconds(),
		l.limit,
		nowMs,
		member,
	).Int()

	if err != nil {
		return false, fmt.Errorf("redis sliding log: %w", err)
	}

	return result == 1, nil
}

