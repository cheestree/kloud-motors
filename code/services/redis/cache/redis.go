package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("cache miss")

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(host, port string, ttl time.Duration) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr: host + ":" + port,
	})

	return &RedisCache{
		client: client,
		ttl:    ttl,
	}
}

// Get fetches the JSON value from Redis and unmarshals it into dest.
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			slog.Info("Cache MISS", "key", key)
			return ErrCacheMiss
		}
		slog.Error("Cache GET error", "key", key, "error", err)
		return err
	}
	slog.Info("Cache HIT", "key", key)
	return json.Unmarshal([]byte(val), dest)
}

// Set marshals the value to JSON and stores it in Redis with the configured TTL.
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = c.client.Set(ctx, key, data, c.ttl).Err()
	if err == nil {
		slog.Info("Cache SET", "key", key, "ttl", c.ttl)
	} else {
		slog.Error("Cache SET error", "key", key, "error", err)
	}
	return err
}

// Delete removes a key from Redis.
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
