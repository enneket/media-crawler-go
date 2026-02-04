package tieba

import (
	"context"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/proxy"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	httpClient *resty.Client
	switcher   *proxy.Switcher
	proxyPool  *proxy.Pool
}

func NewClient() *Client {
	retryCount := config.AppConfig.HttpRetryCount
	if retryCount <= 0 {
		retryCount = 3
	}
	baseDelay := time.Duration(config.AppConfig.HttpRetryBaseDelayMs) * time.Millisecond
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}
	maxDelay := time.Duration(config.AppConfig.HttpRetryMaxDelayMs) * time.Millisecond
	if maxDelay <= 0 {
		maxDelay = 4 * time.Second
	}
	timeout := time.Duration(config.AppConfig.HttpTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	switcher := proxy.NewSwitcher()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = switcher.ProxyFunc
	hc := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	httpClient := resty.NewWithClient(hc)
	httpClient.SetHeaders(map[string]string{
		"accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"accept-language": "zh-CN,zh;q=0.9,en;q=0.8",
		"user-agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	})
	httpClient.SetRetryCount(retryCount)
	httpClient.SetRetryWaitTime(baseDelay)
	httpClient.SetRetryMaxWaitTime(maxDelay)
	out := &Client{httpClient: httpClient, switcher: switcher}
	httpClient.AddRetryCondition(func(r *resty.Response, err error) bool {
		if err != nil {
			return crawler.ShouldRetryError(err)
		}
		if r == nil {
			return true
		}
		code := r.StatusCode()
		if out.proxyPool != nil && crawler.ShouldInvalidateProxyStatus(code) {
			out.proxyPool.InvalidateCurrent()
		}
		if code == http.StatusForbidden && out.proxyPool != nil {
			return true
		}
		return crawler.ShouldRetryStatus(code)
	})

	return out
}

type FetchResult struct {
	URL         string `json:"url"`
	StatusCode  int    `json:"status_code"`
	ContentType string `json:"content_type,omitempty"`
	Body        string `json:"body,omitempty"`
	OriginalLen int    `json:"original_len,omitempty"`
	Truncated   bool   `json:"truncated,omitempty"`
	FetchedAt   int64  `json:"fetched_at,omitempty"`
}

func (c *Client) InitProxyPool(pool *proxy.Pool) {
	c.proxyPool = pool
}

func (c *Client) ensureProxy(ctx context.Context) error {
	if c.proxyPool == nil || c.switcher == nil {
		return nil
	}
	p, err := c.proxyPool.GetOrRefresh(ctx)
	if err != nil {
		return err
	}
	u, err := p.HTTPURL()
	if err != nil {
		return err
	}
	return c.switcher.Set(u)
}

func (c *Client) FetchHTML(ctx context.Context, url string) (FetchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.ensureProxy(ctx); err != nil {
		return FetchResult{}, err
	}
	r, err := c.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return FetchResult{}, err
	}
	if r.IsError() {
		return FetchResult{}, crawler.NewHTTPStatusError("tieba", url, r.StatusCode(), r.String())
	}
	body := r.Body()
	out := FetchResult{
		URL:         url,
		StatusCode:  r.StatusCode(),
		ContentType: r.Header().Get("content-type"),
		OriginalLen: len(body),
		FetchedAt:   time.Now().Unix(),
	}
	const maxBody = 2_000_000
	if len(body) > maxBody {
		body = body[:maxBody]
		out.Truncated = true
	}
	out.Body = string(body)
	return out, nil
}
