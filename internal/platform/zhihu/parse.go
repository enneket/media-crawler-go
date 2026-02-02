package zhihu

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var (
	reAnswer = regexp.MustCompile(`(?i)zhihu\.com/question/(\d+)(?:/answer/(\d+))?`)
	reQID    = regexp.MustCompile(`(?i)question/(\d+)`)
	reAID    = regexp.MustCompile(`(?i)answer/(\d+)`)
	reDigits = regexp.MustCompile(`^\d+$`)
	reBad    = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
)

func ParseZhihuID(input string) (qid, aid, noteID string, err error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", "", "", fmt.Errorf("empty input")
	}
	if reDigits.MatchString(s) {
		return s, "", sanitizeID(s), nil
	}
	if m := reAnswer.FindStringSubmatch(s); len(m) == 3 {
		qid = m[1]
		aid = m[2]
		if aid != "" {
			return qid, aid, sanitizeID(qid + "_" + aid), nil
		}
		return qid, "", sanitizeID(qid), nil
	}
	if m := reQID.FindStringSubmatch(s); len(m) == 2 {
		qid = m[1]
		if a := reAID.FindStringSubmatch(s); len(a) == 2 {
			aid = a[1]
		}
		if aid != "" {
			return qid, aid, sanitizeID(qid + "_" + aid), nil
		}
		return qid, "", sanitizeID(qid), nil
	}
	h := sha1.Sum([]byte(s))
	fallback := hex.EncodeToString(h[:])
	return "", "", sanitizeID(fallback), fmt.Errorf("cannot parse zhihu id")
}

func sanitizeID(s string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return ""
	}
	return reBad.ReplaceAllString(v, "_")
}
