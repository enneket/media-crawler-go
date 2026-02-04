package douyin

import (
	"context"
	"media-crawler-go/internal/proxy"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	reVideoID = regexp.MustCompile(`/video/(\d+)`)
	reUserID  = regexp.MustCompile(`/user/([^/?]+)`)
)

func ExtractAwemeID(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	if isDigits(s) {
		return s
	}
	u, err := url.Parse(s)
	if err == nil {
		if id := u.Query().Get("modal_id"); id != "" && isDigits(id) {
			return id
		}
		if m := reVideoID.FindStringSubmatch(u.Path); len(m) == 2 {
			return m[1]
		}
		if strings.Contains(u.Host, "v.douyin.com") {
			return ""
		}
	}
	return ""
}

func ExtractSecUserID(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "MS4wLjABAAAA") && !strings.Contains(s, "/") {
		return s
	}
	u, err := url.Parse(s)
	if err == nil {
		if m := reUserID.FindStringSubmatch(u.Path); len(m) == 2 {
			return m[1]
		}
	}
	if !strings.HasPrefix(s, "http") && !strings.Contains(s, "douyin.com") && !strings.Contains(s, "/") {
		return s
	}
	return ""
}

func ResolveShortURL(raw string) (string, error) {
	return ResolveShortURLWithProxy(context.Background(), raw, nil)
}

func ResolveShortURLWithProxy(ctx context.Context, raw string, pool *proxy.Pool) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if pool != nil {
		p, err := pool.GetOrRefresh(ctx)
		if err != nil {
			return "", err
		}
		u, err := p.HTTPURL()
		if err != nil {
			return "", err
		}
		uu, err := url.Parse(u)
		if err != nil {
			return "", err
		}
		transport.Proxy = http.ProxyURL(uu)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	finalURL := resp.Request.URL.String()
	return finalURL, nil
}

func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}
