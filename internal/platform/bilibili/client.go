package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/config"
	"net/http"
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
	rc.SetBaseURL("https://api.bilibili.com")
	rc.SetHeaders(map[string]string{
		"accept":          "application/json, text/plain, */*",
		"accept-language": "zh-CN,zh;q=0.9",
		"referer":         "https://www.bilibili.com/",
		"user-agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	})

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

type ViewResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) GetView(ctx context.Context, bvid string, aid int64) (ViewResponse, error) {
	req := c.httpClient.R().SetContext(ctx)
	if bvid != "" {
		req.SetQueryParam("bvid", bvid)
	} else if aid > 0 {
		req.SetQueryParam("aid", fmt.Sprintf("%d", aid))
	} else {
		return ViewResponse{}, fmt.Errorf("bvid/aid is empty")
	}
	var out ViewResponse
	resp, err := req.SetResult(&out).Get("/x/web-interface/view")
	if err != nil {
		return ViewResponse{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return ViewResponse{}, fmt.Errorf("bad status: %s", resp.Status())
	}
	if out.Code != 0 {
		return ViewResponse{}, fmt.Errorf("bilibili api error: code=%d message=%s", out.Code, out.Message)
	}
	return out, nil
}
