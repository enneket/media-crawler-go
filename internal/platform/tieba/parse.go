package tieba

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	reTiebaP   = regexp.MustCompile(`(?i)/p/(\d+)`)
	reTiebaKZ  = regexp.MustCompile(`(?i)[?&]kz=(\d+)`)
	reTiebaKZ2 = regexp.MustCompile(`(?i)^kz=(\d+)$`)
	reDigits   = regexp.MustCompile(`^\d+$`)
	reBadChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
)

func ParseThreadID(input string) (threadID string, noteID string, err error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", "", fmt.Errorf("empty input")
	}
	if reDigits.MatchString(s) {
		return s, sanitizeID(s), nil
	}
	if m := reTiebaP.FindStringSubmatch(s); len(m) == 2 {
		return m[1], sanitizeID(m[1]), nil
	}
	if m := reTiebaKZ.FindStringSubmatch(s); len(m) == 2 {
		return m[1], sanitizeID(m[1]), nil
	}
	if m := reTiebaKZ2.FindStringSubmatch(s); len(m) == 2 {
		return m[1], sanitizeID(m[1]), nil
	}
	if u, err2 := url.Parse(s); err2 == nil && u != nil {
		if v := u.Query().Get("kz"); reDigits.MatchString(v) {
			return v, sanitizeID(v), nil
		}
	}
	h := sha1.Sum([]byte(s))
	fallback := hex.EncodeToString(h[:])
	return "", sanitizeID(fallback), fmt.Errorf("cannot parse tieba thread id")
}

func ThreadURL(threadID string) string {
	id := strings.TrimSpace(threadID)
	if id == "" {
		return ""
	}
	return fmt.Sprintf("https://tieba.baidu.com/p/%s", url.PathEscape(id))
}

func sanitizeID(s string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return ""
	}
	return reBadChars.ReplaceAllString(v, "_")
}
