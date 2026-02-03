package cache

import (
	"strings"

	"media-crawler-go/internal/config"
)

func NewFromConfig(cfg config.Config) Cache {
	backend := strings.ToLower(strings.TrimSpace(cfg.CacheBackend))
	switch backend {
	case "", "memory":
		return NewMemoryCache()
	case "redis":
		addr := strings.TrimSpace(cfg.RedisAddr)
		if addr == "" {
			return NewMemoryCache()
		}
		rc, err := NewRedisCache(RedisOptions{
			Addr:     addr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
			Prefix:   cfg.RedisKeyPrefix,
		})
		if err != nil {
			return NewMemoryCache()
		}
		return rc
	case "none", "disabled", "off":
		return nil
	default:
		return NewMemoryCache()
	}
}

