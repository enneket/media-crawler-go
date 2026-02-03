package proxy

import (
	"fmt"
)

func NewProvider(name string) (Provider, error) {
	switch ProviderName(name) {
	case ProviderKuaiDaiLi:
		return NewKuaiDaiLiFromEnv(), nil
	case ProviderWanDouHTTP:
		return NewWanDouHTTPFromEnv(), nil
	case ProviderStatic:
		return NewStaticFromConfigOrEnv(), nil
	default:
		return nil, fmt.Errorf("unknown proxy provider: %s", name)
	}
}
