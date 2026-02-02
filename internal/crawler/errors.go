package crawler

import "fmt"

type ErrorKind string

const (
	ErrorKindUnknown      ErrorKind = "unknown"
	ErrorKindRiskHint     ErrorKind = "risk_hint"
	ErrorKindHTTP         ErrorKind = "http"
	ErrorKindInvalidInput ErrorKind = "invalid_input"
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

func NewRiskHintError(platform, url, hint string) error {
	return Error{
		Kind:     ErrorKindRiskHint,
		Platform: platform,
		URL:      url,
		Msg:      fmt.Sprintf("risk hint detected: %s", hint),
	}
}
