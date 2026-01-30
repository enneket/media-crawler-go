package proxy

import (
	"net/http"
	"net/url"
	"sync/atomic"
)

type Switcher struct {
	current atomic.Value
}

func NewSwitcher() *Switcher {
	s := &Switcher{}
	s.current.Store((*url.URL)(nil))
	return s
}

func (s *Switcher) Set(raw string) error {
	if raw == "" {
		s.current.Store((*url.URL)(nil))
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	s.current.Store(u)
	return nil
}

func (s *Switcher) ProxyFunc(req *http.Request) (*url.URL, error) {
	if v := s.current.Load(); v != nil {
		if u, ok := v.(*url.URL); ok && u != nil {
			return u, nil
		}
	}
	return nil, nil
}
