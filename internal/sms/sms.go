package sms

import (
	"context"
	"media-crawler-go/internal/cache"
	"media-crawler-go/internal/config"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	cacheOnce sync.Once
	cacheInst cache.Cache
	codeRe    = regexp.MustCompile(`\b(\d{6})\b`)
)

func ExtractCode(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	m := codeRe.FindStringSubmatch(message)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func Store(platform string, phone string, code string, ttl time.Duration) error {
	platform = strings.ToLower(strings.TrimSpace(platform))
	phone = strings.TrimSpace(phone)
	code = strings.TrimSpace(code)
	if platform == "" || phone == "" || code == "" {
		return nil
	}
	c := getCache()
	if c == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = 3 * time.Minute
	}
	return c.Set(context.Background(), key(platform, phone), []byte(code), ttl)
}

func Pop(platform string, phone string) (string, bool) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	phone = strings.TrimSpace(phone)
	if platform == "" || phone == "" {
		return "", false
	}
	c := getCache()
	if c == nil {
		return "", false
	}
	k := key(platform, phone)
	b, ok, err := c.Get(context.Background(), k)
	if err != nil || !ok || len(b) == 0 {
		return "", false
	}
	_ = c.Delete(context.Background(), k)
	return string(b), true
}

func getCache() cache.Cache {
	cacheOnce.Do(func() {
		cacheInst = cache.NewFromConfig(config.AppConfig)
		if cacheInst == nil {
			cacheInst = cache.NewMemoryCache()
		}
	})
	return cacheInst
}

func key(platform string, phone string) string {
	return "sms:" + platform + ":" + phone
}

