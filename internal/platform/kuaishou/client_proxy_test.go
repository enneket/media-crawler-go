package kuaishou

import (
	"context"
	"media-crawler-go/internal/proxy"
	"net/http"
	"testing"
)

type mockProvider struct {
	proxies []proxy.Proxy
}

func (m *mockProvider) Name() proxy.ProviderName { return proxy.ProviderStatic }

func (m *mockProvider) GetProxies(ctx context.Context, num int) ([]proxy.Proxy, error) {
	if len(m.proxies) == 0 {
		return nil, nil
	}
	return m.proxies, nil
}

func TestClientEnsureProxySetsSwitcher(t *testing.T) {
	c := NewClient()
	pool := proxy.NewPool(&mockProvider{proxies: []proxy.Proxy{{IP: "127.0.0.1", Port: 8080}}}, 1)
	c.InitProxyPool(pool)
	if err := c.ensureProxy(context.Background()); err != nil {
		t.Fatalf("ensureProxy err: %v", err)
	}
	u, err := c.switcher.ProxyFunc(&http.Request{})
	if err != nil {
		t.Fatalf("ProxyFunc err: %v", err)
	}
	if u == nil || u.Host != "127.0.0.1:8080" {
		t.Fatalf("unexpected proxy url: %v", u)
	}
}
