package kuaishou

import (
	"context"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	httpClient *resty.Client
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

	httpClient := resty.NewWithClient(&http.Client{Timeout: timeout})
	headers := map[string]string{
		"accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"accept-language": "zh-CN,zh;q=0.9,en;q=0.8",
		"user-agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	}
	if ck := config.AppConfig.Cookies; ck != "" {
		headers["cookie"] = ck
	}
	httpClient.SetHeaders(headers)
	httpClient.SetRetryCount(retryCount)
	httpClient.SetRetryWaitTime(baseDelay)
	httpClient.SetRetryMaxWaitTime(maxDelay)
	httpClient.AddRetryCondition(func(r *resty.Response, err error) bool {
		if err != nil {
			return crawler.ShouldRetryError(err)
		}
		if r == nil {
			return true
		}
		return crawler.ShouldRetryStatus(r.StatusCode())
	})

	return &Client{httpClient: httpClient}
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

func (c *Client) FetchHTML(ctx context.Context, url string) (FetchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	r, err := c.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return FetchResult{}, err
	}
	if r.IsError() {
		return FetchResult{}, crawler.NewHTTPStatusError("kuaishou", url, r.StatusCode(), r.String())
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
