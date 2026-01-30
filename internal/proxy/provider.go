package proxy

import "context"

type Provider interface {
	Name() ProviderName
	GetProxies(ctx context.Context, num int) ([]Proxy, error)
}
