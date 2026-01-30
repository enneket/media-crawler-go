package proxy

import (
	"context"
	"testing"
	"time"
)

type mockProvider struct {
	name    ProviderName
	proxies []Proxy
}

func (m *mockProvider) Name() ProviderName { return m.name }
func (m *mockProvider) GetProxies(ctx context.Context, num int) ([]Proxy, error) {
	if num <= 0 {
		return nil, nil
	}
	if len(m.proxies) > num {
		return m.proxies[:num], nil
	}
	return m.proxies, nil
}

func TestPoolGetOrRefresh(t *testing.T) {
	p1 := Proxy{IP: "1.1.1.1", Port: 8080, ExpiredAt: time.Now().Add(2 * time.Second)}
	p2 := Proxy{IP: "2.2.2.2", Port: 8080, ExpiredAt: time.Now().Add(2 * time.Second)}

	pool := NewPool(&mockProvider{name: ProviderKuaiDaiLi, proxies: []Proxy{p1, p2}}, 2)
	pool.SetExpiryBuffer(200 * time.Millisecond)

	got1, err := pool.GetOrRefresh(context.Background())
	if err != nil {
		t.Fatalf("GetOrRefresh err: %v", err)
	}
	got2, err := pool.GetOrRefresh(context.Background())
	if err != nil {
		t.Fatalf("GetOrRefresh err: %v", err)
	}
	if got1.IP != got2.IP {
		t.Fatalf("expected same current proxy before expiry; got %v then %v", got1, got2)
	}

	time.Sleep(2200 * time.Millisecond)

	got3, err := pool.GetOrRefresh(context.Background())
	if err != nil {
		t.Fatalf("GetOrRefresh err: %v", err)
	}
	if got3.IP == got1.IP {
		t.Fatalf("expected proxy refresh after expiry; still got %v", got3)
	}
}
