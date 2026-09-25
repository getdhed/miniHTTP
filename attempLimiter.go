package main

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

func (r *RedisLoginAttemptLimiter) IsBlocked(
	ctx context.Context,
	key string,
	limit int64,
) (bool, time.Duration, error) {
	if limit <= 0 {
		return false, 0, errInvalidData
	}

	count, err := r.client.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	if count < limit {
		return false, 0, nil
	}

	ttl, err := r.client.PTTL(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	return true, ttl, nil
}

func (r *RedisLoginAttemptLimiter) RegisterFailure(
	ctx context.Context,
	key string,
	limit int64,
	window time.Duration,
) (bool, time.Duration, error) {
	windowSeconds := int64(window / time.Second)
	script := redis.NewScript(`
	local count = redis.call("INCR", KEYS[1])

	if count == 1 then
		redis.call("EXPIRE", KEYS[1], ARGV[1])
	end

	local ttl = redis.call("PTTL", KEYS[1])

	return {count, ttl}
	`)

	values, err := script.Run(ctx, r.client, []string{key}, windowSeconds).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	if len(values) != 2 {
		return false, 0, errUnexpectedResult
	}

	count := values[0]
	ttl := time.Duration(values[1]) * time.Millisecond

	if count >= limit {
		return true, ttl, nil
	}

	return false, ttl, nil
}
