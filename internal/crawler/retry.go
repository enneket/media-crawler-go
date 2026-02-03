package crawler

import (
	"context"
	"errors"
	"net/http"
)

func ShouldRetryError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

func ShouldRetryStatus(code int) bool {
	return code == http.StatusTooManyRequests || (code >= 500 && code <= 599)
}

func ShouldInvalidateProxyStatus(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusForbidden
}

