package douyin

import (
	"context"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/proxy"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/playwright-community/playwright-go"
)

type Client struct {
	httpClient *resty.Client
	signer     *Signer
	switcher   *proxy.Switcher
	proxyPool  *proxy.Pool

	userAgent string
	cookieStr string
}

func NewClient(signer *Signer, userAgent string) *Client {
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	}
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
	rc.SetBaseURL("https://www.douyin.com")
	rc.SetHeaders(map[string]string{
		"accept":          "application/json, text/plain, */*",
		"accept-language": "zh-CN,zh;q=0.9",
		"referer":         "https://www.douyin.com/",
		"user-agent":      userAgent,
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
	return &Client{
		httpClient: rc,
		signer:     signer,
		switcher:   switcher,
		userAgent:  userAgent,
	}
}

func (c *Client) InitProxyPool(pool *proxy.Pool) {
	c.proxyPool = pool
}

func (c *Client) UpdateCookies(ctx playwright.BrowserContext) error {
	cookies, err := ctx.Cookies()
	if err != nil {
		return err
	}
	var cookieStrs []string
	for _, ck := range cookies {
		cookieStrs = append(cookieStrs, fmt.Sprintf("%s=%s", ck.Name, ck.Value))
	}
	c.cookieStr = strings.Join(cookieStrs, "; ")
	c.httpClient.SetHeader("Cookie", c.cookieStr)
	return nil
}

func (c *Client) CookieHeader() string {
	return c.cookieStr
}

func (c *Client) UserAgent() string {
	return c.userAgent
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

func (c *Client) GetVideoByID(ctx context.Context, awemeID string, msToken string, webID string) (map[string]interface{}, error) {
	if err := c.ensureProxy(ctx); err != nil {
		return nil, err
	}
	if awemeID == "" {
		return nil, fmt.Errorf("aweme_id is empty")
	}

	params := defaultParams(msToken, webID)
	params.Set("aweme_id", awemeID)

	aBogus, err := c.signer.SignDetail(params, c.userAgent)
	if err != nil {
		return nil, err
	}
	params.Set("a_bogus", aBogus)

	var resp map[string]interface{}
	r, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryString(params.Encode()).
		SetResult(&resp).
		Get("/aweme/v1/web/aweme/detail/")
	if err != nil {
		return nil, err
	}
	if r.IsError() {
		return nil, fmt.Errorf("status: %d, body: %s", r.StatusCode(), r.String())
	}

	awemeAny := resp["aweme_detail"]
	aweme, ok := awemeAny.(map[string]interface{})
	if !ok || awemeAny == nil {
		return nil, fmt.Errorf("aweme_detail missing")
	}
	return aweme, nil
}

func defaultParams(msToken string, webID string) url.Values {
	if webID == "" {
		webID = strconv.FormatInt(time.Now().UnixNano(), 10)
		if len(webID) > 19 {
			webID = webID[len(webID)-19:]
		}
	}
	if msToken == "" {
		msToken = ""
	}

	v := url.Values{}
	v.Set("device_platform", "webapp")
	v.Set("aid", "6383")
	v.Set("channel", "channel_pc_web")
	v.Set("version_code", "190600")
	v.Set("version_name", "19.6.0")
	v.Set("update_version_code", "170400")
	v.Set("pc_client_type", "1")
	v.Set("cookie_enabled", "true")
	v.Set("browser_language", "zh-CN")
	v.Set("browser_platform", "MacIntel")
	v.Set("browser_name", "Chrome")
	v.Set("browser_version", "125.0.0.0")
	v.Set("browser_online", "true")
	v.Set("engine_name", "Blink")
	v.Set("os_name", "Mac OS")
	v.Set("os_version", "10.15.7")
	v.Set("cpu_core_num", "8")
	v.Set("device_memory", "8")
	v.Set("engine_version", "109.0")
	v.Set("platform", "PC")
	v.Set("screen_width", "2560")
	v.Set("screen_height", "1440")
	v.Set("effective_type", "4g")
	v.Set("round_trip_time", "50")
	v.Set("webid", webID)
	v.Set("msToken", msToken)
	return v
}
