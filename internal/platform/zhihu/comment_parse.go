package zhihu

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	reInitialData = regexp.MustCompile(`(?is)<script[^>]+id=["']js-initialData["'][^>]*>(.*?)</script>`)
	reHTMLTags    = regexp.MustCompile(`<[^>]+>`)
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

	js := extractInitialDataJSON(pageContent)
	if js == "" {
		return nil
	}

	var root map[string]any
	if err := json.Unmarshal([]byte(js), &root); err != nil {
		return nil
	}
	initState, _ := root["initialState"].(map[string]any)
	if initState == nil {
		return nil
	}
	entities, _ := initState["entities"].(map[string]any)
	if entities == nil {
		return nil
	}
	commentsMap, _ := entities["comments"].(map[string]any)
	if commentsMap == nil || len(commentsMap) == 0 {
		return nil
	}

	out := make([]Comment, 0, 32)
	seen := map[string]struct{}{}
	for _, v := range commentsMap {
		if len(out) >= max {
			break
		}
		m, _ := v.(map[string]any)
		if m == nil {
			continue
		}
		id := strings.TrimSpace(fmt.Sprintf("%v", m["id"]))
		if id == "" || id == "<nil>" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		parentID := strings.TrimSpace(fmt.Sprintf("%v", m["replyToCommentId"]))
		if parentID == "<nil>" {
			parentID = ""
		}
		if !enableSub && parentID != "" {
			continue
		}

		content := stripHTML(fmt.Sprintf("%v", m["content"]))
		createTime := toInt64(m["createdTime"])
		likeCount := toInt64(m["likeCount"])

		userID := ""
		userName := ""
		if author, ok := m["author"].(map[string]any); ok {
			userID = strings.TrimSpace(fmt.Sprintf("%v", author["id"]))
			if userID == "<nil>" {
				userID = ""
			}
			userName = strings.TrimSpace(fmt.Sprintf("%v", author["name"]))
			if userName == "<nil>" {
				userName = ""
			}
		}

		out = append(out, Comment{
			NoteID:          noteID,
			CommentID:       id,
			ParentCommentID: parentID,
			Content:         content,
			CreateTime:      createTime,
			LikeCount:       likeCount,
			UserID:          userID,
			UserNickname:    userName,
		})
	}
	return out
}

func extractInitialDataJSON(pageContent string) string {
	m := reInitialData.FindStringSubmatch(pageContent)
	if len(m) != 2 {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(m[1]))
}

func stripHTML(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = reHTMLTags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
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
