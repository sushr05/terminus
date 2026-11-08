package ratelimit

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sushr05/terminus/scripts"
)

type RedisLeakyBucketLimiter struct {
	client   *redis.Client
	capacity int
	leakRate float64
	script   *redis.Script
}

func NewRedisLeakyBucketLimiter(
	client *redis.Client,
	capacity int,
	leakRate float64,
) *RedisLeakyBucketLimiter {
	return &RedisLeakyBucketLimiter{
		client:   client,
		capacity: capacity,
		leakRate: leakRate,
		script:   redis.NewScript(scripts.RedisLeakyBucket),
	}
}

func (l *RedisLeakyBucketLimiter) Allow(
	ctx context.Context,
	clientID string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	key := fmt.Sprintf(
		"terminus:ratelimit:leaky_bucket:{%s}",
		clientID,
	)

	now := float64(time.Now().UnixNano()) / 1e9
	ttl := math.Ceil(float64(l.capacity)/l.leakRate)*2 + 60

	result, err := l.script.Run(
		ctx,
		l.client,
		[]string{key},
		l.capacity,
		l.leakRate,
		now,
		ttl,
	).Int()

	if err != nil {
		return false, fmt.Errorf("redis leaky bucket: %w", err)
	}

	return result == 1, nil
}

