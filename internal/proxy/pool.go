package proxy

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

var ErrNoProxyAvailable = errors.New("no proxy available")

type Pool struct {
	provider Provider
	count    int
	buffer   time.Duration

	mu      sync.Mutex
	proxies []Proxy
	current *Proxy
}

func NewPool(provider Provider, count int) *Pool {
	if count <= 0 {
		count = 2
	}
	return &Pool{
		provider: provider,
		count:    count,
		buffer:   30 * time.Second,
	}
}

func (p *Pool) SetExpiryBuffer(buffer time.Duration) {
	if buffer <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.buffer = buffer
}

func (p *Pool) GetOrRefresh(ctx context.Context) (Proxy, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.current != nil && !p.current.IsExpired(p.buffer) {
		return *p.current, nil
	}

	if len(p.proxies) == 0 {
		proxies, err := p.provider.GetProxies(ctx, p.count)
		if err != nil {
			return Proxy{}, err
		}
		p.proxies = append(p.proxies[:0], proxies...)
	}

	if len(p.proxies) == 0 {
		return Proxy{}, ErrNoProxyAvailable
	}

	idx := rand.Intn(len(p.proxies))
	next := p.proxies[idx]
	p.proxies[idx] = p.proxies[len(p.proxies)-1]
	p.proxies = p.proxies[:len(p.proxies)-1]

	p.current = &next
	return next, nil
}

func (p *Pool) Current() (Proxy, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current == nil {
		return Proxy{}, false
	}
	return *p.current, true
}

func (p *Pool) InvalidateCurrent() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = nil
}
