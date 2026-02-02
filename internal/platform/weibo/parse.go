package weibo

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var reWeiboID = regexp.MustCompile(`(?i)\b([0-9A-Za-z]{6,})\b`)

func ParseStatusID(input string) (id string, noteID string, err error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", "", fmt.Errorf("empty input")
	}
	if looksLikeURL(s) {
		u, err := url.Parse(s)
		if err == nil && u != nil {
			if v := strings.TrimSpace(u.Query().Get("id")); v != "" {
				return v, v, nil
			}
			path := strings.Trim(u.Path, "/")
			if path != "" {
				parts := strings.Split(path, "/")
				last := parts[len(parts)-1]
				last = strings.TrimSpace(last)
				if m := reWeiboID.FindString(last); m != "" {
					return m, m, nil
				}
			}
		}
	}

	if m := reWeiboID.FindString(s); m != "" {
		return m, m, nil
	}
	return "", "", fmt.Errorf("cannot parse weibo status id from: %s", input)
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
