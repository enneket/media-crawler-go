package cache

import (
	"context"
	"sync"
	"time"
)

type memoryEntry struct {
	value     []byte
	expiresAt time.Time
}

type MemoryCache struct {
	mu      sync.RWMutex
	items   map[string]memoryEntry
	closed  chan struct{}
	closeMu sync.Once
}

func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		items:  make(map[string]memoryEntry, 1024),
		closed: make(chan struct{}),
	}
	go c.janitor()
	return c
}

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}
	now := time.Now()
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false, nil
	}
	out := make([]byte, len(e.value))
	copy(out, e.value)
	return out, true, nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	v := make([]byte, len(value))
	copy(v, value)
	c.mu.Lock()
	c.items[key] = memoryEntry{value: v, expiresAt: exp}
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Close() error {
	c.closeMu.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *MemoryCache) janitor() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-c.closed:
			return
		case <-t.C:
			now := time.Now()
			c.mu.Lock()
			for k, e := range c.items {
				if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		}
	}
}

