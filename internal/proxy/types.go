package proxy

import (
	"fmt"
	"net/url"
	"time"
)

type ProviderName string

const (
	ProviderKuaiDaiLi  ProviderName = "kuaidaili"
	ProviderWanDouHTTP ProviderName = "wandouhttp"
	ProviderJiSuHTTP   ProviderName = "jisuhttp"
	ProviderStatic     ProviderName = "static"
)

type Proxy struct {
	IP        string
	Port      int
	User      string
	Password  string
	Protocol  string
	ExpiredAt time.Time
}

func (p Proxy) IsExpired(buffer time.Duration) bool {
	if p.ExpiredAt.IsZero() {
		return false
	}
	return time.Now().After(p.ExpiredAt.Add(-buffer))
}

func (p Proxy) HTTPURL() (string, error) {
	u := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", p.IP, p.Port),
	}
	if p.User != "" || p.Password != "" {
		u.User = url.UserPassword(p.User, p.Password)
	}
	return u.String(), nil
}

func (p Proxy) ChromeProxyServer() string {
	scheme := p.Protocol
	if scheme == "" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, p.IP, p.Port)
}
