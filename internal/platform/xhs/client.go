package xhs

import (
	"context"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/proxy"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/playwright-community/playwright-go"
)

type Client struct {
	HttpClient *resty.Client
	Signer     *Signer
	Cookies    map[string]string
	UserAgent  string

	ProxyPool     *proxy.Pool
	ProxySwitcher *proxy.Switcher
}

var randSeedOnce sync.Once

func NewClient(signer *Signer) *Client {
	randSeedOnce.Do(func() { rand.Seed(time.Now().UnixNano()) })

	switcher := proxy.NewSwitcher()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = switcher.ProxyFunc

	timeoutSec := config.AppConfig.HttpTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeoutSec) * time.Second,
	}

	client := resty.NewWithClient(httpClient)
	client.SetBaseURL("https://edith.xiaohongshu.com")

	// Default headers
	client.SetHeaders(map[string]string{
		"accept":          "application/json, text/plain, */*",
		"accept-language": "zh-CN,zh;q=0.9",
		"cache-control":   "no-cache",
		"content-type":    "application/json;charset=UTF-8",
		"origin":          "https://www.xiaohongshu.com",
		"pragma":          "no-cache",
		"referer":         "https://www.xiaohongshu.com/",
	})

	return &Client{
		HttpClient:    client,
		Signer:        signer,
		Cookies:       make(map[string]string),
		ProxySwitcher: switcher,
	}
}

func (c *Client) InitProxyPool(pool *proxy.Pool) {
	c.ProxyPool = pool
}

func (c *Client) UpdateCookies(ctx playwright.BrowserContext) error {
	cookies, err := ctx.Cookies()
	if err != nil {
		return err
	}

	var cookieStrs []string
	for _, cookie := range cookies {
		c.Cookies[cookie.Name] = cookie.Value
		cookieStrs = append(cookieStrs, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}

	cookieHeader := strings.Join(cookieStrs, "; ")
	c.HttpClient.SetHeader("Cookie", cookieHeader)
	return nil
}

func (c *Client) SetUserAgent(ua string) {
	c.UserAgent = ua
	c.HttpClient.SetHeader("user-agent", ua)
}

func (c *Client) preHeaders(uri string, data interface{}, method string) (map[string]string, error) {
	a1 := c.Cookies["a1"]
	return c.Signer.Sign(uri, data, a1, method)
}

func (c *Client) ensureProxy(ctx context.Context) error {
	if c.ProxyPool == nil || c.ProxySwitcher == nil {
		return nil
	}
	p, err := c.ProxyPool.GetOrRefresh(ctx)
	if err != nil {
		return err
	}
	proxyURL, err := p.HTTPURL()
	if err != nil {
		return err
	}
	return c.ProxySwitcher.Set(proxyURL)
}

func (c *Client) Post(ctx context.Context, uri string, data interface{}, result interface{}) error {
	if ctx == nil {
		ctx = context.Background()
	}
	retryCount, baseDelay, maxDelay := retryParams()
	var lastErr error

	for attempt := 0; attempt < retryCount; attempt++ {
		if err := c.ensureProxy(ctx); err != nil {
			lastErr = err
			if attempt < retryCount-1 {
				delay := backoffDelay(attempt, baseDelay, maxDelay)
				logger.Warn("xhs request retry (proxy)", "uri", uri, "attempt", attempt+1, "max_attempts", retryCount, "err", err, "sleep_ms", delay.Milliseconds())
				if !crawler.Sleep(ctx, delay) {
					return ctx.Err()
				}
				continue
			}
			return err
		}

		headers, err := c.preHeaders(uri, data, "POST")
		if err != nil {
			return err
		}

		resp, err := c.HttpClient.R().
			SetContext(ctx).
			SetHeaders(headers).
			SetBody(data).
			SetResult(result).
			Post(uri)

		if err == nil && !resp.IsError() {
			return nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = crawler.NewHTTPStatusError("xhs", uri, resp.StatusCode(), resp.String())
		}

		if shouldInvalidateProxy(resp) && c.ProxyPool != nil {
			c.ProxyPool.InvalidateCurrent()
		}
		if !shouldRetry(resp, err) {
			return lastErr
		}
		if attempt < retryCount-1 {
			delay := backoffDelay(attempt, baseDelay, maxDelay)
			logger.Warn("xhs request retry", "uri", uri, "attempt", attempt+1, "max_attempts", retryCount, "kind", string(crawler.KindOf(lastErr)), "err", lastErr, "sleep_ms", delay.Milliseconds())
			if !crawler.Sleep(ctx, delay) {
				return ctx.Err()
			}
		}
	}

	return lastErr
}

func (c *Client) Pong() bool {
	res, err := c.GetNoteByKeyword(context.Background(), "Xiaohongshu", 1)
	if err != nil {
		return false
	}
	return len(res.Items) > 0
}

func (c *Client) GetNoteByKeyword(ctx context.Context, keyword string, page int) (*SearchResult, error) {
	uri := "/api/sns/web/v1/search/notes"
	sort := config.AppConfig.SortType
	if sort == "" {
		sort = "general"
	}
	data := map[string]interface{}{
		"keyword":   keyword,
		"page":      page,
		"page_size": 20,
		"search_id": GetSearchId(),
		"sort":      sort,
		"note_type": 0,
	}

	// Wrapper for response
	type Response struct {
		Success bool         `json:"success"`
		Code    int          `json:"code"`
		Msg     string       `json:"msg"`
		Data    SearchResult `json:"data"`
	}

	var resp Response
	err := c.Post(ctx, uri, data, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("api error: %s", resp.Msg)
	}

	return &resp.Data, nil
}

func (c *Client) GetNotesByCreator(ctx context.Context, userId, cursor string) (*CreatorNotesResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	uri := "/api/sns/web/v1/user_posted"
	params := map[string]string{
		"user_id":       userId,
		"cursor":        cursor,
		"num":           "30",
		"image_formats": "jpg,webp,avif",
	}

	type Response struct {
		Success bool               `json:"success"`
		Code    int                `json:"code"`
		Msg     string             `json:"msg"`
		Data    CreatorNotesResult `json:"data"`
	}

	var resp Response
	retryCount, baseDelay, maxDelay := retryParams()
	var lastErr error

	for attempt := 0; attempt < retryCount; attempt++ {
		if err := c.ensureProxy(ctx); err != nil {
			lastErr = err
			if attempt < retryCount-1 {
				delay := backoffDelay(attempt, baseDelay, maxDelay)
				logger.Warn("xhs request retry (proxy)", "uri", uri, "attempt", attempt+1, "max_attempts", retryCount, "err", err, "sleep_ms", delay.Milliseconds())
				if !crawler.Sleep(ctx, delay) {
					return nil, ctx.Err()
				}
				continue
			}
			return nil, err
		}

		headers, err := c.preHeaders(uri, params, "GET")
		if err != nil {
			return nil, err
		}

		r, err := c.HttpClient.R().
			SetContext(ctx).
			SetHeaders(headers).
			SetQueryParams(params).
			SetResult(&resp).
			Get(uri)

		if err == nil && !r.IsError() && resp.Success {
			return &resp.Data, nil
		}

		if err != nil {
			lastErr = err
		} else if r.IsError() {
			lastErr = crawler.NewHTTPStatusError("xhs", uri, r.StatusCode(), r.String())
		} else {
			lastErr = fmt.Errorf("api error: %s", resp.Msg)
		}

		if shouldInvalidateProxy(r) && c.ProxyPool != nil {
			c.ProxyPool.InvalidateCurrent()
		}
		if !shouldRetry(r, err) {
			return nil, lastErr
		}
		if attempt < retryCount-1 {
			delay := backoffDelay(attempt, baseDelay, maxDelay)
			logger.Warn("xhs request retry", "uri", uri, "attempt", attempt+1, "max_attempts", retryCount, "kind", string(crawler.KindOf(lastErr)), "err", lastErr, "sleep_ms", delay.Milliseconds())
			if !crawler.Sleep(ctx, delay) {
				return nil, ctx.Err()
			}
		}
	}
	return nil, lastErr
}

func (c *Client) GetNoteById(ctx context.Context, noteId, xsecSource, xsecToken string) (*Note, error) {
	if xsecSource == "" {
		xsecSource = "pc_search"
	}

	uri := "/api/sns/web/v1/feed"
	data := map[string]interface{}{
		"source_note_id": noteId,
		"image_formats":  []string{"jpg", "webp", "avif"},
		"extra":          map[string]int{"need_body_topic": 1},
		"xsec_source":    xsecSource,
		"xsec_token":     xsecToken,
	}

	type Response struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Data    struct {
			Items []struct {
				NoteCard Note `json:"note_card"`
			} `json:"items"`
		} `json:"data"`
	}

	var resp Response
	err := c.Post(ctx, uri, data, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("api error: %s", resp.Msg)
	}

	if len(resp.Data.Items) == 0 {
		return nil, fmt.Errorf("note not found")
	}

	note := resp.Data.Items[0].NoteCard
	return &note, nil
}

func (c *Client) GetNoteComments(ctx context.Context, noteId, xsecToken, cursor string) (*CommentResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	uri := "/api/sns/web/v2/comment/page"
	params := map[string]string{
		"note_id":        noteId,
		"cursor":         cursor,
		"top_comment_id": "",
		"image_formats":  "jpg,webp,avif",
		"xsec_token":     xsecToken,
	}

	type Response struct {
		Success bool          `json:"success"`
		Code    int           `json:"code"`
		Msg     string        `json:"msg"`
		Data    CommentResult `json:"data"`
	}

	var resp Response
	retryCount, baseDelay, maxDelay := retryParams()
	var lastErr error

	for attempt := 0; attempt < retryCount; attempt++ {
		if err := c.ensureProxy(ctx); err != nil {
			lastErr = err
			if attempt < retryCount-1 {
				delay := backoffDelay(attempt, baseDelay, maxDelay)
				logger.Warn("xhs request retry (proxy)", "uri", uri, "attempt", attempt+1, "max_attempts", retryCount, "err", err, "sleep_ms", delay.Milliseconds())
				if !crawler.Sleep(ctx, delay) {
					return nil, ctx.Err()
				}
				continue
			}
			return nil, err
		}

		headers, err := c.preHeaders(uri, params, "GET")
		if err != nil {
			return nil, err
		}

		r, err := c.HttpClient.R().
			SetContext(ctx).
			SetHeaders(headers).
			SetQueryParams(params).
			SetResult(&resp).
			Get(uri)

		if err == nil && !r.IsError() && resp.Success {
			return &resp.Data, nil
		}

		if err != nil {
			lastErr = err
		} else if r.IsError() {
			lastErr = crawler.NewHTTPStatusError("xhs", uri, r.StatusCode(), r.String())
		} else {
			lastErr = fmt.Errorf("api error: %s", resp.Msg)
		}

		if shouldInvalidateProxy(r) && c.ProxyPool != nil {
			c.ProxyPool.InvalidateCurrent()
		}
		if !shouldRetry(r, err) {
			return nil, lastErr
		}
		if attempt < retryCount-1 {
			delay := backoffDelay(attempt, baseDelay, maxDelay)
			logger.Warn("xhs request retry", "uri", uri, "attempt", attempt+1, "max_attempts", retryCount, "kind", string(crawler.KindOf(lastErr)), "err", lastErr, "sleep_ms", delay.Milliseconds())
			if !crawler.Sleep(ctx, delay) {
				return nil, ctx.Err()
			}
		}
	}
	return nil, lastErr
}

func (c *Client) GetNoteSubComments(ctx context.Context, noteId, rootCommentId, xsecToken, cursor string, num int) (*CommentResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	uri := "/api/sns/web/v2/comment/sub/page"
	if num <= 0 {
		num = 10
	}
	params := map[string]string{
		"note_id":         noteId,
		"root_comment_id": rootCommentId,
		"num":             fmt.Sprintf("%d", num),
		"cursor":          cursor,
		"image_formats":   "jpg,webp,avif",
		"top_comment_id":  "",
		"xsec_token":      xsecToken,
	}

	type Response struct {
		Success bool          `json:"success"`
		Code    int           `json:"code"`
		Msg     string        `json:"msg"`
		Data    CommentResult `json:"data"`
	}

	var resp Response
	retryCount, baseDelay, maxDelay := retryParams()
	var lastErr error

	for attempt := 0; attempt < retryCount; attempt++ {
		if err := c.ensureProxy(ctx); err != nil {
			lastErr = err
			if attempt < retryCount-1 {
				delay := backoffDelay(attempt, baseDelay, maxDelay)
				logger.Warn("xhs request retry (proxy)", "uri", uri, "attempt", attempt+1, "max_attempts", retryCount, "err", err, "sleep_ms", delay.Milliseconds())
				if !crawler.Sleep(ctx, delay) {
					return nil, ctx.Err()
				}
				continue
			}
			return nil, err
		}

		headers, err := c.preHeaders(uri, params, "GET")
		if err != nil {
			return nil, err
		}

		r, err := c.HttpClient.R().
			SetContext(ctx).
			SetHeaders(headers).
			SetQueryParams(params).
			SetResult(&resp).
			Get(uri)

		if err == nil && !r.IsError() && resp.Success {
			return &resp.Data, nil
		}

		if err != nil {
			lastErr = err
		} else if r.IsError() {
			lastErr = crawler.NewHTTPStatusError("xhs", uri, r.StatusCode(), r.String())
		} else {
			lastErr = fmt.Errorf("api error: %s", resp.Msg)
		}

		if shouldInvalidateProxy(r) && c.ProxyPool != nil {
			c.ProxyPool.InvalidateCurrent()
		}
		if !shouldRetry(r, err) {
			return nil, lastErr
		}
		if attempt < retryCount-1 {
			delay := backoffDelay(attempt, baseDelay, maxDelay)
			logger.Warn("xhs request retry", "uri", uri, "attempt", attempt+1, "max_attempts", retryCount, "kind", string(crawler.KindOf(lastErr)), "err", lastErr, "sleep_ms", delay.Milliseconds())
			if !crawler.Sleep(ctx, delay) {
				return nil, ctx.Err()
			}
		}
	}
	return nil, lastErr
}

func retryParams() (int, time.Duration, time.Duration) {
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
	base := time.Duration(baseMs) * time.Millisecond
	max := time.Duration(maxMs) * time.Millisecond
	if max < base {
		max = base
	}
	return retryCount, base, max
}

func backoffDelay(attempt int, base time.Duration, max time.Duration) time.Duration {
	if attempt < 0 {
		return base
	}
	d := base * time.Duration(1<<attempt)
	if d > max {
		return max
	}
	jitter := 0.5 + rand.Float64()
	out := time.Duration(float64(d) * jitter)
	if out > max {
		return max
	}
	if out < base {
		return base
	}
	return out
}

func shouldRetry(resp *resty.Response, err error) bool {
	if err != nil {
		return crawler.ShouldRetryError(err)
	}
	if resp == nil {
		return true
	}
	return crawler.ShouldRetryStatus(resp.StatusCode())
}

func shouldInvalidateProxy(resp *resty.Response) bool {
	if resp == nil {
		return false
	}
	return crawler.ShouldInvalidateProxyStatus(resp.StatusCode())
}
