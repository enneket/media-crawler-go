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
	reShortVideoPath = regexp.MustCompile(`(?i)/short-video/([a-zA-Z0-9_-]+)`)
	rePhoto      = regexp.MustCompile(`(?i)kuaishou\.com/photo/([a-zA-Z0-9_-]+)`)
	rePhotoPath  = regexp.MustCompile(`(?i)/photo/([a-zA-Z0-9_-]+)`)
	reProfile    = regexp.MustCompile(`(?i)kuaishou\.com/profile/([a-zA-Z0-9_-]+)`)
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

func ExtractDetailURLsFromHTML(html string, max int) []string {
	if max <= 0 {
		max = 200
	}
	seen := make(map[string]struct{}, 32)
	out := make([]string, 0, 32)
	add := func(kind string, id string) {
		if id == "" {
			return
		}
		u := "https://www.kuaishou.com/" + kind + "/" + id
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}

	for _, m := range reShortVideo.FindAllStringSubmatch(html, -1) {
		if len(m) == 2 {
			add("short-video", m[1])
		}
		if len(out) >= max {
			return out[:max]
		}
	}
	for _, m := range rePhoto.FindAllStringSubmatch(html, -1) {
		if len(m) == 2 {
			add("photo", m[1])
		}
		if len(out) >= max {
			return out[:max]
		}
	}
	for _, m := range reShortVideoPath.FindAllStringSubmatch(html, -1) {
		if len(m) == 2 {
			add("short-video", m[1])
		}
		if len(out) >= max {
			return out[:max]
		}
	}
	for _, m := range rePhotoPath.FindAllStringSubmatch(html, -1) {
		if len(m) == 2 {
			add("photo", m[1])
		}
		if len(out) >= max {
			return out[:max]
		}
	}
	return out
}

func ParseKSCreatorID(input string) (creatorID string, err error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", fmt.Errorf("empty input")
	}
	if m := reProfile.FindStringSubmatch(s); len(m) == 2 {
		return sanitizeID(m[1]), nil
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		h := sha1.Sum([]byte(s))
		return sanitizeID(hex.EncodeToString(h[:])), fmt.Errorf("cannot parse kuaishou creator id")
	}
	return sanitizeID(s), nil
}
