package proxy

import (
	"bufio"
	"context"
	"errors"
	"media-crawler-go/internal/config"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type StaticProvider struct {
	List string
	File string
}

func NewStaticFromConfigOrEnv() *StaticProvider {
	list := strings.TrimSpace(os.Getenv("IP_PROXY_LIST"))
	if list == "" {
		list = strings.TrimSpace(os.Getenv("PROXY_LIST"))
	}
	if list == "" {
		list = strings.TrimSpace(config.AppConfig.IPProxyList)
	}

	file := strings.TrimSpace(os.Getenv("IP_PROXY_FILE"))
	if file == "" {
		file = strings.TrimSpace(os.Getenv("PROXY_FILE"))
	}
	if file == "" {
		file = strings.TrimSpace(config.AppConfig.IPProxyFile)
	}

	return &StaticProvider{List: list, File: file}
}

func (p *StaticProvider) Name() ProviderName {
	return ProviderStatic
}

func (p *StaticProvider) GetProxies(ctx context.Context, num int) ([]Proxy, error) {
	if num <= 0 {
		num = 1
	}
	raw, err := p.loadEntries()
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, errors.New("static proxy list is empty: set IP_PROXY_LIST or IP_PROXY_FILE")
	}

	out := make([]Proxy, 0, min(num, len(raw)))
	for _, s := range raw {
		pr, ok := parseProxyEntry(s)
		if !ok {
			continue
		}
		out = append(out, pr)
		if len(out) >= num {
			break
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no valid proxy entries parsed from static list")
	}
	return out, nil
}

func (p *StaticProvider) loadEntries() ([]string, error) {
	if strings.TrimSpace(p.List) != "" {
		return splitProxyList(p.List), nil
	}
	if strings.TrimSpace(p.File) == "" {
		return nil, nil
	}
	f, err := os.Open(p.File)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, splitProxyList(line)...)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func splitProxyList(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, ";", ",")
	s = strings.ReplaceAll(s, "\n", ",")
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func parseProxyEntry(s string) (Proxy, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Proxy{}, false
	}

	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		u, err = url.Parse("http://" + s)
		if err != nil || u.Host == "" {
			return Proxy{}, false
		}
	}

	host := u.Host
	if strings.Contains(host, "@") {
		if v, err := url.Parse("http://" + host); err == nil && v.Host != "" {
			u.User = v.User
			host = v.Host
		}
	}

	h, p, err := net.SplitHostPort(host)
	if err != nil {
		return Proxy{}, false
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return Proxy{}, false
	}

	pr := Proxy{
		IP:       h,
		Port:     port,
		Protocol: u.Scheme,
	}
	if pr.Protocol == "" {
		pr.Protocol = "http"
	}
	if u.User != nil {
		pr.User = u.User.Username()
		if pw, ok := u.User.Password(); ok {
			pr.Password = pw
		}
	}
	return pr, true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
