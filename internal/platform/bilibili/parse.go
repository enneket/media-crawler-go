package bilibili

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	reBVID = regexp.MustCompile(`(?i)\bBV[0-9A-Za-z]{10}\b`)
	reAID  = regexp.MustCompile(`(?i)\bav(\d+)\b`)
)

func ParseVideoID(input string) (bvid string, aid int64, noteID string, err error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", 0, "", fmt.Errorf("empty input")
	}
	if strings.HasPrefix(s, "BV") || strings.HasPrefix(s, "bv") {
		m := reBVID.FindString(s)
		if m == "" {
			return "", 0, "", fmt.Errorf("invalid bvid: %s", s)
		}
		m = strings.ToUpper(m)
		return m, 0, m, nil
	}
	if strings.HasPrefix(s, "av") || strings.HasPrefix(s, "AV") {
		m := reAID.FindStringSubmatch(s)
		if len(m) != 2 {
			return "", 0, "", fmt.Errorf("invalid aid: %s", s)
		}
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || n <= 0 {
			return "", 0, "", fmt.Errorf("invalid aid: %s", s)
		}
		return "", n, fmt.Sprintf("av%d", n), nil
	}

	u, err := url.Parse(s)
	if err == nil && u != nil && u.Host != "" {
		if m := reBVID.FindString(u.Path); m != "" {
			m = strings.ToUpper(m)
			return m, 0, m, nil
		}
		if m := reAID.FindStringSubmatch(u.Path); len(m) == 2 {
			n, err := strconv.ParseInt(m[1], 10, 64)
			if err == nil && n > 0 {
				return "", n, fmt.Sprintf("av%d", n), nil
			}
		}
		if bv := strings.TrimSpace(u.Query().Get("bvid")); bv != "" {
			if m := reBVID.FindString(bv); m != "" {
				m = strings.ToUpper(m)
				return m, 0, m, nil
			}
		}
		if av := strings.TrimSpace(u.Query().Get("aid")); av != "" {
			n, err := strconv.ParseInt(av, 10, 64)
			if err == nil && n > 0 {
				return "", n, fmt.Sprintf("av%d", n), nil
			}
		}
	}

	if m := reBVID.FindString(s); m != "" {
		m = strings.ToUpper(m)
		return m, 0, m, nil
	}
	if m := reAID.FindStringSubmatch(s); len(m) == 2 {
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err == nil && n > 0 {
			return "", n, fmt.Sprintf("av%d", n), nil
		}
	}

	return "", 0, "", fmt.Errorf("cannot parse bilibili video id from: %s", input)
}
