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

type WanDouHTTP struct {
	AppKey string
	Client *http.Client
}

func NewWanDouHTTPFromEnv() *WanDouHTTP {
	appKey := os.Getenv("WANDOU_APP_KEY")
	if appKey == "" {
		appKey = os.Getenv("wandou_app_key")
	}
	return &WanDouHTTP{
		AppKey: appKey,
		Client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (p *WanDouHTTP) Name() ProviderName {
	return ProviderWanDouHTTP
}

func (p *WanDouHTTP) GetProxies(ctx context.Context, num int) ([]Proxy, error) {
	if num <= 0 {
		num = 1
	}
	if num > 100 {
		num = 100
	}
	if p.AppKey == "" {
		return nil, fmt.Errorf("wandouhttp credentials missing: set WANDOU_APP_KEY")
	}

	endpoint, _ := url.Parse("https://api.wandouapp.com/")
	q := endpoint.Query()
	q.Set("app_key", p.AppKey)
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
		return nil, fmt.Errorf("wandouhttp http status: %s", resp.Status)
	}

	type item struct {
		IP         string `json:"ip"`
		Port       int    `json:"port"`
		ExpireTime string `json:"expire_time"`
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
	if r.Code != 200 {
		return nil, fmt.Errorf("wandouhttp api error: %s (code: %d)", r.Msg, r.Code)
	}

	out := make([]Proxy, 0, len(r.Data))
	for _, it := range r.Data {
		expiredAt := parseWanDouExpireTime(it.ExpireTime)
		out = append(out, Proxy{
			IP:        it.IP,
			Port:      it.Port,
			Protocol:  "http",
			ExpiredAt: expiredAt,
		})
	}
	return out, nil
}

func parseWanDouExpireTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t
	}
	return time.Time{}
}
