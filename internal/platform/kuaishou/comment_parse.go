package kuaishou

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reNextData  = regexp.MustCompile(`(?is)<script[^>]+id=["']__NEXT_DATA__["'][^>]*>(.*?)</script>`)
	reHTMLTags  = regexp.MustCompile(`<[^>]+>`)
	markersKS   = []string{"__NEXT_DATA__", "__APOLLO_STATE__", "__INITIAL_STATE__", "commentList", "comments"}
)

func parseCommentsFromHTML(pageContent string, noteID string, max int, enableSub bool) []Comment {
	if strings.TrimSpace(pageContent) == "" || strings.TrimSpace(noteID) == "" {
		return nil
	}
	if max == 0 {
		return nil
	}
	if max < 0 {
		max = 5000
	}

	var roots []any
	if js := extractNextDataJSON(pageContent); js != "" {
		var v any
		if json.Unmarshal([]byte(js), &v) == nil {
			roots = append(roots, v)
		}
	}

	for _, mk := range markersKS {
		if mk == "__NEXT_DATA__" {
			continue
		}
		if v := extractJSONObjectByMarker(pageContent, mk); v != nil {
			roots = append(roots, v)
		}
	}
	if len(roots) == 0 {
		return nil
	}

	out := make([]Comment, 0, 32)
	seen := map[string]struct{}{}
	for _, root := range roots {
		extractCommentsFromAny(root, noteID, enableSub, max, &out, seen, 0)
		if len(out) >= max {
			break
		}
	}
	return out
}

func extractNextDataJSON(pageContent string) string {
	m := reNextData.FindStringSubmatch(pageContent)
	if len(m) != 2 {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(m[1]))
}

func extractJSONObjectByMarker(pageContent string, marker string) any {
	idx := strings.Index(pageContent, marker)
	if idx < 0 {
		return nil
	}
	if marker == "commentList" || marker == "comments" {
		start := idx
		for start > 0 && pageContent[start] != '{' && pageContent[start] != '[' {
			start--
		}
		if start < 0 {
			return nil
		}
		js := extractBalancedJSON(pageContent[start:])
		if js == "" {
			return nil
		}
		var v any
		if json.Unmarshal([]byte(js), &v) != nil {
			return nil
		}
		return v
	}

	eq := strings.Index(pageContent[idx:], "=")
	if eq < 0 {
		return nil
	}
	s := pageContent[idx+eq+1:]
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "window.") {
		if j := strings.Index(s, "="); j >= 0 {
			s = strings.TrimSpace(s[j+1:])
		}
	}
	js := extractBalancedJSON(s)
	if js == "" {
		return nil
	}
	var v any
	if json.Unmarshal([]byte(js), &v) != nil {
		return nil
	}
	return v
}

func extractBalancedJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var open, close byte
	switch s[0] {
	case '{':
		open, close = '{', '}'
	case '[':
		open, close = '[', ']'
	default:
		return ""
	}
	depth := 0
	inStr := false
	esc := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			if ch == '\\' {
				esc = true
				continue
			}
			if ch == '"' {
				inStr = false
			}
			continue
		}
		if ch == '"' {
			inStr = true
			continue
		}
		if ch == open {
			depth++
		} else if ch == close {
			depth--
			if depth == 0 {
				return s[:i+1]
			}
		}
	}
	return ""
}

func extractCommentsFromAny(v any, noteID string, enableSub bool, max int, out *[]Comment, seen map[string]struct{}, depth int) {
	if v == nil || len(*out) >= max || depth > 8 {
		return
	}
	switch vv := v.(type) {
	case map[string]any:
		if c, ok := mapToComment(vv, noteID); ok {
			if c.CommentID != "" {
				if _, exist := seen[c.CommentID]; !exist {
					if enableSub || c.ParentCommentID == "" {
						seen[c.CommentID] = struct{}{}
						*out = append(*out, c)
					}
				}
			}
			if len(*out) >= max {
				return
			}
		}
		for _, x := range vv {
			if len(*out) >= max {
				break
			}
			extractCommentsFromAny(x, noteID, enableSub, max, out, seen, depth+1)
		}
	case []any:
		for _, x := range vv {
			if len(*out) >= max {
				break
			}
			extractCommentsFromAny(x, noteID, enableSub, max, out, seen, depth+1)
		}
	}
}

func mapToComment(m map[string]any, noteID string) (Comment, bool) {
	if m == nil {
		return Comment{}, false
	}
	id := firstString(m, "commentId", "id")
	content := firstString(m, "content", "text", "message")
	if id == "" || content == "" {
		return Comment{}, false
	}
	parent := firstString(m, "replyToCommentId", "parentCommentId", "rootCommentId")
	createTime := firstInt64(m, "timestamp", "createTime", "createdAt", "createdTime")
	likeCount := firstInt64(m, "likeCount", "likedCount", "diggCount")

	userID := ""
	userName := ""
	if u, ok := m["author"].(map[string]any); ok {
		userID = firstString(u, "id", "userId", "uid")
		userName = firstString(u, "name", "nickname", "userName")
	} else if u, ok := m["user"].(map[string]any); ok {
		userID = firstString(u, "id", "userId", "uid")
		userName = firstString(u, "name", "nickname", "userName")
	}

	return Comment{
		NoteID:          noteID,
		CommentID:       id,
		ParentCommentID: parent,
		Content:         stripHTML(content),
		CreateTime:      createTime,
		LikeCount:       likeCount,
		UserID:          userID,
		UserNickname:    userName,
	}, true
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func firstInt64(m map[string]any, keys ...string) int64 {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			if n := toInt64(v); n != 0 {
				return n
			}
		}
	}
	return 0
}

func toInt64(v any) int64 {
	switch vv := v.(type) {
	case int64:
		return vv
	case int:
		return int64(vv)
	case float64:
		return int64(vv)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(vv), 10, 64)
		return n
	default:
		return 0
	}
}

func stripHTML(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = reHTMLTags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

