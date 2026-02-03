package crawler

import (
	"fmt"
	"strings"
)

func NewHTTPStatusError(platform, url string, statusCode int, body string) error {
	kind := ErrorKindHTTP
	switch statusCode {
	case 401, 403:
		kind = ErrorKindForbidden
	case 429:
		kind = ErrorKindRateLimited
	}
	msg := fmt.Sprintf("http status=%d", statusCode)

	snippet := strings.TrimSpace(body)
	const maxSnippet = 1024
	if len(snippet) > maxSnippet {
		snippet = snippet[:maxSnippet]
	}
	if snippet != "" {
		msg = msg + " body=" + snippet
	}

	return Error{
		Kind:     kind,
		Platform: platform,
		URL:      url,
		Msg:      msg,
	}
}
