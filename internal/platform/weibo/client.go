package weibo

import (
	"context"
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/proxy"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	httpClient *resty.Client
	switcher   *proxy.Switcher
	proxyPool  *proxy.Pool
}

func NewClient() *Client {
	switcher := proxy.NewSwitcher()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = switcher.ProxyFunc

	timeoutSec := config.AppConfig.HttpTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	hc := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeoutSec) * time.Second,
	}
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
	out := &Client{httpClient: rc, switcher: switcher}
	rc.AddRetryCondition(func(r *resty.Response, err error) bool {
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

type ShowResponse struct {
	Ok   int             `json:"ok"`
	Data json.RawMessage `json:"data"`
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

func (c *Client) Show(ctx context.Context, id string) (ShowResponse, error) {
	if err := c.ensureProxy(ctx); err != nil {
		return ShowResponse{}, err
	}
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
		return ShowResponse{}, crawler.NewHTTPStatusError("weibo", fmt.Sprintf("/statuses/show?id=%s", id), resp.StatusCode(), resp.String())
	}
	if out.Ok != 1 {
		return ShowResponse{}, fmt.Errorf("weibo api not ok: ok=%d", out.Ok)
	}
	return out, nil
}

type GetIndexResponse struct {
	Ok   int             `json:"ok"`
	Data json.RawMessage `json:"data"`
}

func (c *Client) GetIndex(ctx context.Context, params map[string]string) (GetIndexResponse, error) {
	if err := c.ensureProxy(ctx); err != nil {
		return GetIndexResponse{}, err
	}
	var out GetIndexResponse
	req := c.httpClient.R().SetContext(ctx).SetResult(&out)
	for k, v := range params {
		req.SetQueryParam(k, v)
	}
	resp, err := req.Get("/api/container/getIndex")
	if err != nil {
		return GetIndexResponse{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return GetIndexResponse{}, crawler.NewHTTPStatusError("weibo", "/api/container/getIndex", resp.StatusCode(), resp.String())
	}
	if out.Ok != 1 {
		return GetIndexResponse{}, fmt.Errorf("weibo api not ok: ok=%d", out.Ok)
	}
	return out, nil
}

func (c *Client) SearchByKeyword(ctx context.Context, keyword string, page int, searchType string) (GetIndexResponse, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return GetIndexResponse{}, fmt.Errorf("empty keyword")
	}
	if page <= 0 {
		page = 1
	}
	searchType = strings.TrimSpace(searchType)
	if searchType == "" {
		searchType = "1"
	}
	containerid := fmt.Sprintf("100103type=%s&q=%s", searchType, keyword)
	return c.GetIndex(ctx, map[string]string{
		"containerid": containerid,
		"page_type":   "searchall",
		"page":        fmt.Sprintf("%d", page),
	})
}

func (c *Client) CreatorInfo(ctx context.Context, creatorID string) (GetIndexResponse, error) {
	creatorID = strings.TrimSpace(creatorID)
	if creatorID == "" {
		return GetIndexResponse{}, fmt.Errorf("empty creator id")
	}
	containerid := fmt.Sprintf("100505%s", creatorID)
	return c.GetIndex(ctx, map[string]string{
		"jumpfrom":    "weibocom",
		"type":        "uid",
		"value":       creatorID,
		"containerid": containerid,
	})
}

func (c *Client) NotesByCreator(ctx context.Context, creatorID string, containerID string, sinceID string) (GetIndexResponse, error) {
	creatorID = strings.TrimSpace(creatorID)
	if creatorID == "" {
		return GetIndexResponse{}, fmt.Errorf("empty creator id")
	}
	containerID = strings.TrimSpace(containerID)
	if containerID == "" {
		return GetIndexResponse{}, fmt.Errorf("empty container id")
	}
	if strings.TrimSpace(sinceID) == "" {
		sinceID = "0"
	}
	return c.GetIndex(ctx, map[string]string{
		"jumpfrom":    "weibocom",
		"type":        "uid",
		"value":       creatorID,
		"containerid": containerID,
		"since_id":    sinceID,
	})
}
