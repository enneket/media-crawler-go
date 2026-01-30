package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"
)

const kuaidailiDeltaExpiredSeconds = 5

type KuaiDaiLi struct {
	SecretID  string
	Signature string
	UserName  string
	UserPwd   string

	Client *http.Client
}

func NewKuaiDaiLiFromEnv() *KuaiDaiLi {
	secretID := os.Getenv("KDL_SECERT_ID")
	if secretID == "" {
		secretID = os.Getenv("kdl_secret_id")
	}
	signature := os.Getenv("KDL_SIGNATURE")
	if signature == "" {
		signature = os.Getenv("kdl_signature")
	}
	userName := os.Getenv("KDL_USER_NAME")
	if userName == "" {
		userName = os.Getenv("kdl_user_name")
	}
	userPwd := os.Getenv("KDL_USER_PWD")
	if userPwd == "" {
		userPwd = os.Getenv("kdl_user_pwd")
	}

	return &KuaiDaiLi{
		SecretID:  secretID,
		Signature: signature,
		UserName:  userName,
		UserPwd:   userPwd,
		Client:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (p *KuaiDaiLi) Name() ProviderName {
	return ProviderKuaiDaiLi
}

func (p *KuaiDaiLi) GetProxies(ctx context.Context, num int) ([]Proxy, error) {
	if num <= 0 {
		num = 1
	}
	if p.SecretID == "" || p.Signature == "" {
		return nil, fmt.Errorf("kuaidaili credentials missing: set KDL_SECERT_ID and KDL_SIGNATURE")
	}

	endpoint, _ := url.Parse("https://dps.kdlapi.com/api/getdps/")
	q := endpoint.Query()
	q.Set("secret_id", p.SecretID)
	q.Set("signature", p.Signature)
	q.Set("pt", "1")
	q.Set("format", "json")
	q.Set("sep", "1")
	q.Set("f_et", "1")
	q.Set("num", strconv.Itoa(num))
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kuaidaili http status: %s", resp.Status)
	}

	type apiResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ProxyList []string `json:"proxy_list"`
		} `json:"data"`
	}

	var r apiResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.Code != 0 {
		return nil, fmt.Errorf("kuaidaili api error: %s", r.Msg)
	}

	re := regexp.MustCompile(`(\d{1,3}(?:\.\d{1,3}){3}):(\d{1,5}),(\d+)`)
	now := time.Now()

	out := make([]Proxy, 0, len(r.Data.ProxyList))
	for _, item := range r.Data.ProxyList {
		m := re.FindStringSubmatch(item)
		if len(m) != 4 {
			continue
		}
		port, _ := strconv.Atoi(m[2])
		expireSeconds, _ := strconv.Atoi(m[3])
		expiredAt := now.Add(time.Duration(expireSeconds-kuaidailiDeltaExpiredSeconds) * time.Second)

		out = append(out, Proxy{
			IP:        m[1],
			Port:      port,
			User:      p.UserName,
			Password:  p.UserPwd,
			Protocol:  "http",
			ExpiredAt: expiredAt,
		})
	}
	return out, nil
}
