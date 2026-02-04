package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type JiSuHTTP struct {
	Key    string
	Crypto string
	Client *http.Client
}

func NewJiSuHTTPFromEnv() *JiSuHTTP {
	key := os.Getenv("JISU_KEY")
	if key == "" {
		key = os.Getenv("jisu_key")
	}
	crypto := os.Getenv("JISU_CRYPTO")
	if crypto == "" {
		crypto = os.Getenv("jisu_crypto")
	}
	return &JiSuHTTP{
		Key:    key,
		Crypto: crypto,
		Client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (p *JiSuHTTP) Name() ProviderName {
	return ProviderJiSuHTTP
}

func (p *JiSuHTTP) GetProxies(ctx context.Context, num int) ([]Proxy, error) {
	if num <= 0 {
		num = 1
	}
	if num > 100 {
		num = 100
	}
	if p.Key == "" || p.Crypto == "" {
		return nil, fmt.Errorf("jisuhttp credentials missing: set JISU_KEY and JISU_CRYPTO")
	}

	endpoint, _ := url.Parse("https://api.jisuhttp.com/fetchips")
	q := endpoint.Query()
	q.Set("key", p.Key)
	q.Set("crypto", p.Crypto)
	q.Set("time", "30")
	q.Set("type", "json")
	q.Set("port", "2")
	q.Set("pw", "1")
	q.Set("se", "1")
	q.Set("num", strconv.Itoa(num))
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MediaCrawler-Go")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jisuhttp http status: %s", resp.Status)
	}

	type item struct {
		IP     string `json:"ip"`
		Port   int    `json:"port"`
		User   string `json:"user"`
		Pass   string `json:"pass"`
		Expire string `json:"expire"`
	}
	type apiResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data []item `json:"data"`
	}

	var r apiResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.Code != 0 {
		return nil, fmt.Errorf("jisuhttp api error: %s (code: %d)", r.Msg, r.Code)
	}

	out := make([]Proxy, 0, len(r.Data))
	for _, it := range r.Data {
		expiredAt := parseJiSuExpireTime(it.Expire)
		out = append(out, Proxy{
			IP:        it.IP,
			Port:      it.Port,
			User:      it.User,
			Password:  it.Pass,
			Protocol:  "http",
			ExpiredAt: expiredAt,
		})
	}
	return out, nil
}

func parseJiSuExpireTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05.000", s, time.Local); err == nil {
		return t
	}
	return time.Time{}
}
