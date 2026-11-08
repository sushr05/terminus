package ratelimit

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sushr05/terminus/scripts"
)

type RedisTokenBucketLimiter struct {
	client     *redis.Client
	capacity   int
	refillRate float64
	script     *redis.Script
}

func NewRedisTokenBucketLimiter(
	client *redis.Client,
	capacity int,
	refillRate float64,
) *RedisTokenBucketLimiter {
	return &RedisTokenBucketLimiter{
		client:     client,
		capacity:   capacity,
		refillRate: refillRate,
		script:     redis.NewScript(scripts.RedisTokenBucket),
	}
}

func (l *RedisTokenBucketLimiter) Allow(
	ctx context.Context,
	clientID string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	key := fmt.Sprintf(
		"terminus:ratelimit:token_bucket:{%s}",
		clientID,
	)

	now := float64(time.Now().UnixNano()) / 1e9
	ttl := math.Ceil(float64(l.capacity)/l.refillRate)*2 + 60

	result, err := l.script.Run(
		ctx,
		l.client,
		[]string{key},
		l.capacity,
		l.refillRate,
		now,
		ttl,
	).Int()

	if err != nil {
		return false, fmt.Errorf("redis token bucket: %w", err)
	}

	return result == 1, nil
}

