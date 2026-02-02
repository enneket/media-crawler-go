package crawler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

type ErrorKind string

const (
	ErrorKindUnknown      ErrorKind = "unknown"
	ErrorKindRiskHint     ErrorKind = "risk_hint"
	ErrorKindHTTP         ErrorKind = "http"
	ErrorKindInvalidInput ErrorKind = "invalid_input"
	ErrorKindCanceled     ErrorKind = "canceled"
	ErrorKindTimeout      ErrorKind = "timeout"
)

type Error struct {
	Kind     ErrorKind
	Platform string
	URL      string
	Msg      string
	Err      error
}

func (e Error) Error() string {
	base := e.Msg
	if base == "" && e.Err != nil {
		base = e.Err.Error()
	}
	if base == "" {
		base = string(e.Kind)
	}
	if e.Platform != "" && e.URL != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Platform, base, e.URL)
	}
	if e.Platform != "" {
		return fmt.Sprintf("%s: %s", e.Platform, base)
	}
	return base
}

func (e Error) Unwrap() error { return e.Err }

func KindOf(err error) ErrorKind {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return ErrorKindCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorKindTimeout
	}
	var ce Error
	if errors.As(err, &ce) && ce.Kind != "" {
		return ce.Kind
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return ErrorKindTimeout
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "http status=") {
		return ErrorKindHTTP
	}
	return ErrorKindUnknown
}

func MergeFailureKinds(dst map[string]int, src map[string]int) map[string]int {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]int, len(src))
	}
	for k, v := range src {
		dst[k] += v
	}
	return dst
}

func NewRiskHintError(platform, url, hint string) error {
	return Error{
		Kind:     ErrorKindRiskHint,
		Platform: platform,
		URL:      url,
		Msg:      fmt.Sprintf("risk hint detected: %s", hint),
	}
}
