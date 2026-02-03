package cache

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	prefix string
}

type RedisOptions struct {
	Addr     string
	Password string
	DB       int
	Prefix   string
}

func NewRedisCache(opts RedisOptions) (*RedisCache, error) {
	prefix := strings.TrimSpace(opts.Prefix)
	if prefix == "" {
		prefix = "media_crawler:"
	}
	c := redis.NewClient(&redis.Options{
		Addr:     strings.TrimSpace(opts.Addr),
		Password: opts.Password,
		DB:       opts.DB,
	})
	return &RedisCache{client: c, prefix: prefix}, nil
}

func (c *RedisCache) key(k string) string {
	return c.prefix + k
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}
	return v, true, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, c.key(key), value, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.key(key)).Err()
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

