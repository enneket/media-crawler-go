package platform

import (
	"fmt"
	"media-crawler-go/internal/crawler"
	"sort"
	"strings"
	"sync"
)

type Factory func() crawler.Runner

var (
	mu        sync.RWMutex
	factories = map[string]Factory{}
)

func Register(name string, aliases []string, factory Factory) {
	if factory == nil {
		panic("platform: factory is nil")
	}
	keys := append([]string{name}, aliases...)
	mu.Lock()
	defer mu.Unlock()
	for _, k := range keys {
		n := normalize(k)
		if n == "" {
			continue
		}
		if _, exists := factories[n]; exists {
			panic(fmt.Sprintf("platform: duplicate register: %s", n))
		}
		factories[n] = factory
	}
}

func New(name string) (crawler.Runner, error) {
	n := normalize(name)
	mu.RLock()
	f := factories[n]
	mu.RUnlock()
	if f == nil {
		return nil, fmt.Errorf("unknown platform: %s (available: %s)", name, strings.Join(Names(), ", "))
	}
	return f(), nil
}

func Exists(name string) bool {
	n := normalize(name)
	mu.RLock()
	_, ok := factories[n]
	mu.RUnlock()
	return ok
}

func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	uniq := map[string]struct{}{}
	out := make([]string, 0, len(factories))
	for k := range factories {
		if _, ok := uniq[k]; ok {
			continue
		}
		uniq[k] = struct{}{}
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
