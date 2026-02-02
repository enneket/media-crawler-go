package weibo

import (
	"context"
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/config"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	httpClient *resty.Client
}

func NewClient() *Client {
	timeoutSec := config.AppConfig.HttpTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	hc := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	rc := resty.NewWithClient(hc)
	rc.SetBaseURL("https://m.weibo.cn")
	rc.SetHeaders(map[string]string{
		"accept":          "application/json, text/plain, */*",
		"accept-language": "zh-CN,zh;q=0.9",
		"referer":         "https://m.weibo.cn/",
		"user-agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	})
	if ck := strings.TrimSpace(config.AppConfig.Cookies); ck != "" {
		rc.SetHeader("cookie", ck)
	}

	retryCount := config.AppConfig.HttpRetryCount
	if retryCount <= 0 {
		retryCount = 3
	}
	baseMs := config.AppConfig.HttpRetryBaseDelayMs
	if baseMs <= 0 {
		baseMs = 500
	}
	maxMs := config.AppConfig.HttpRetryMaxDelayMs
	if maxMs <= 0 {
		maxMs = 4000
	}
	rc.SetRetryCount(retryCount)
	rc.SetRetryWaitTime(time.Duration(baseMs) * time.Millisecond)
	rc.SetRetryMaxWaitTime(time.Duration(maxMs) * time.Millisecond)
	rc.AddRetryCondition(func(r *resty.Response, err error) bool {
		if err != nil {
			return true
		}
		if r == nil {
			return false
		}
		code := r.StatusCode()
		return code == http.StatusTooManyRequests || (code >= 500 && code <= 599)
	})

	return &Client{httpClient: rc}
}

type ShowResponse struct {
	Ok   int             `json:"ok"`
	Data json.RawMessage `json:"data"`
}

func (c *Client) Show(ctx context.Context, id string) (ShowResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ShowResponse{}, fmt.Errorf("empty weibo id")
	}
	var out ShowResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam("id", id).
		SetResult(&out).
		Get("/statuses/show")
	if err != nil {
		return ShowResponse{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return ShowResponse{}, fmt.Errorf("bad status: %s", resp.Status())
	}
	if out.Ok != 1 {
		return ShowResponse{}, fmt.Errorf("weibo api not ok: ok=%d", out.Ok)
	}
	return out, nil
}
