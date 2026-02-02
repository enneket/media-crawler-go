package kuaishou

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var (
	reShortVideo = regexp.MustCompile(`(?i)kuaishou\.com/short-video/([a-zA-Z0-9_-]+)`)
	rePhoto      = regexp.MustCompile(`(?i)kuaishou\.com/photo/([a-zA-Z0-9_-]+)`)
	reBad        = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
)

func ParseKSID(input string) (ksid string, noteID string, err error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", "", fmt.Errorf("empty input")
	}
	if m := reShortVideo.FindStringSubmatch(s); len(m) == 2 {
		return m[1], sanitizeID(m[1]), nil
	}
	if m := rePhoto.FindStringSubmatch(s); len(m) == 2 {
		return m[1], sanitizeID(m[1]), nil
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		h := sha1.Sum([]byte(s))
		fallback := hex.EncodeToString(h[:])
		return "", sanitizeID(fallback), fmt.Errorf("cannot parse ks id")
	}
	return s, sanitizeID(s), nil
}

func sanitizeID(s string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return ""
	}
	return reBad.ReplaceAllString(v, "_")
}
