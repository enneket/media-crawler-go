package tieba

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

type parentCommentMeta struct {
	Comment   Comment
	ForumID   string
	SubCount  int
	PostNo    int
}

var (
	reDataFieldDiv = regexp.MustCompile(`data-field='([^']+)'`)
	reHTMLTags     = regexp.MustCompile(`<[^>]+>`)
)

func parseParentCommentsFromHTML(pageContent string, noteID string) []parentCommentMeta {
	if strings.TrimSpace(pageContent) == "" {
		return nil
	}
	matches := reDataFieldDiv.FindAllStringSubmatch(pageContent, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]parentCommentMeta, 0, 32)
	for _, m := range matches {
		if len(m) != 2 {
			continue
		}
		raw := html.UnescapeString(m[1])
		var v map[string]any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			continue
		}
		content, _ := v["content"].(map[string]any)
		if content == nil {
			continue
		}
		postID := strings.TrimSpace(fmt.Sprintf("%v", content["post_id"]))
		if postID == "" || postID == "<nil>" {
			continue
		}
		body := strings.TrimSpace(fmt.Sprintf("%v", content["content"]))
		if body == "" || body == "<nil>" {
			continue
		}
		forumID := strings.TrimSpace(fmt.Sprintf("%v", content["forum_id"]))
		commentNum := toInt(content["comment_num"])

		userID := ""
		userName := ""
		if author, ok := v["author"].(map[string]any); ok {
			userID = strings.TrimSpace(fmt.Sprintf("%v", author["user_id"]))
			if userID == "<nil>" {
				userID = ""
			}
			userName = strings.TrimSpace(fmt.Sprintf("%v", author["user_name"]))
			if userName == "<nil>" {
				userName = ""
			}
		}
		postNo := toInt(content["post_no"])

		out = append(out, parentCommentMeta{
			Comment: Comment{
				NoteID:          noteID,
				CommentID:       postID,
				ParentCommentID: "",
				Content:         stripHTML(body),
				CreateTime:      0,
				LikeCount:       0,
				UserID:          userID,
				UserNickname:    userName,
			},
			ForumID:  forumID,
			SubCount: commentNum,
			PostNo:   postNo,
		})
	}
	return out
}

type subCommentMeta struct {
	Comment Comment
}

func parseSubCommentsFromHTML(pageContent string, noteID string, parentCommentID string) []subCommentMeta {
	if strings.TrimSpace(pageContent) == "" {
		return nil
	}
	matches := reDataFieldDiv.FindAllStringSubmatch(pageContent, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]subCommentMeta, 0, 32)
	for _, m := range matches {
		if len(m) != 2 {
			continue
		}
		raw := html.UnescapeString(m[1])
		var v map[string]any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			continue
		}
		spid := strings.TrimSpace(fmt.Sprintf("%v", v["spid"]))
		if spid == "" || spid == "<nil>" {
			continue
		}
		showname := strings.TrimSpace(fmt.Sprintf("%v", v["showname"]))
		if showname == "<nil>" {
			showname = ""
		}

		content := extractFirstLzlContent(pageContent, spid)
		out = append(out, subCommentMeta{
			Comment: Comment{
				NoteID:          noteID,
				CommentID:       spid,
				ParentCommentID: parentCommentID,
				Content:         content,
				CreateTime:      0,
				LikeCount:       0,
				UserID:          "",
				UserNickname:    showname,
			},
		})
	}
	return out
}

func extractFirstLzlContent(pageContent string, spid string) string {
	if strings.TrimSpace(pageContent) == "" || strings.TrimSpace(spid) == "" {
		return ""
	}
	idx := strings.Index(pageContent, spid)
	if idx < 0 {
		return ""
	}
	start := idx - 3000
	if start < 0 {
		start = 0
	}
	end := idx + 3000
	if end > len(pageContent) {
		end = len(pageContent)
	}
	snippet := pageContent[start:end]
	if i := strings.Index(snippet, "lzl_content_main"); i >= 0 {
		snippet = snippet[i:]
	}
	if j := strings.Index(snippet, "</span>"); j >= 0 {
		snippet = snippet[:j]
	}
	return stripHTML(snippet)
}

func stripHTML(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = reHTMLTags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

func toInt(v any) int {
	switch vv := v.(type) {
	case int:
		return vv
	case int64:
		return int(vv)
	case float64:
		return int(vv)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(vv))
		return n
	default:
		return 0
	}
}

