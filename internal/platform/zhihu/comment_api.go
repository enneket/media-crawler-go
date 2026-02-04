package zhihu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type jsonFetchClient interface {
	FetchJSON(context.Context, string) (FetchResult, error)
}

type zhihuAPIListResponse struct {
	Data   []map[string]any `json:"data"`
	Paging struct {
		IsEnd bool   `json:"is_end"`
		Next  string `json:"next"`
	} `json:"paging"`
}

func fetchCommentsPreferAPI(ctx context.Context, client any, html string, noteID string, answerID string, max int, enableSub bool) []Comment {
	if strings.TrimSpace(noteID) == "" || max == 0 {
		return nil
	}
	if max < 0 {
		max = 5000
	}

	if strings.TrimSpace(answerID) != "" {
		if jf, ok := client.(jsonFetchClient); ok {
			comments, err := fetchAllAnswerCommentsAPI(ctx, jf, noteID, answerID, max, enableSub)
			if err == nil && len(comments) > 0 {
				return comments
			}
		}
	}
	return parseCommentsFromHTML(html, noteID, max, enableSub)
}

func fetchAllAnswerCommentsAPI(ctx context.Context, c jsonFetchClient, noteID string, answerID string, max int, enableSub bool) ([]Comment, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	noteID = strings.TrimSpace(noteID)
	answerID = strings.TrimSpace(answerID)
	if noteID == "" || answerID == "" || max == 0 {
		return nil, nil
	}
	if max < 0 {
		max = 5000
	}

	limit := 20
	endpoints := []string{
		fmt.Sprintf("https://www.zhihu.com/api/v4/answers/%s/root_comments?order_by=score&limit=%d&offset=%d&status=open", url.PathEscape(answerID), limit, 0),
		fmt.Sprintf("https://www.zhihu.com/api/v4/answers/%s/comments?order_by=score&limit=%d&offset=%d&status=open", url.PathEscape(answerID), limit, 0),
	}
	var lastErr error
	for _, firstURL := range endpoints {
		out, err := fetchPagedCommentsAPI(ctx, c, noteID, firstURL, max, enableSub)
		if err == nil && len(out) > 0 {
			return out, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	return nil, lastErr
}

func fetchPagedCommentsAPI(ctx context.Context, c jsonFetchClient, noteID string, firstURL string, max int, enableSub bool) ([]Comment, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	noteID = strings.TrimSpace(noteID)
	firstURL = strings.TrimSpace(firstURL)
	if noteID == "" || firstURL == "" || max == 0 {
		return nil, nil
	}
	if max < 0 {
		max = 5000
	}

	out := make([]Comment, 0, 64)
	seen := make(map[string]struct{}, 128)

	nextURL := firstURL
	offset := 0
	limit := 20

	for len(out) < max && strings.TrimSpace(nextURL) != "" {
		res, err := c.FetchJSON(ctx, nextURL)
		if err != nil {
			return out, err
		}
		var resp zhihuAPIListResponse
		if err := json.Unmarshal([]byte(res.Body), &resp); err != nil {
			return out, err
		}
		for _, m := range resp.Data {
			if len(out) >= max {
				break
			}
			cmt, ok := parseZhihuAPIComment(m, noteID)
			if !ok {
				continue
			}
			if !enableSub && strings.TrimSpace(cmt.ParentCommentID) != "" {
				continue
			}
			if _, ok := seen[cmt.CommentID]; ok {
				continue
			}
			seen[cmt.CommentID] = struct{}{}
			out = append(out, cmt)

			if enableSub {
				childCount := toInt64(pickAny(m, "child_comment_count", "childCommentCount", "child_comment_count_int"))
				if childCount > 0 && len(out) < max {
					children, _ := fetchAllChildCommentsAPI(ctx, c, noteID, cmt.CommentID, max-len(out))
					for i := range children {
						if len(out) >= max {
							break
						}
						if children[i].ParentCommentID == "" {
							children[i].ParentCommentID = cmt.CommentID
						}
						if _, ok := seen[children[i].CommentID]; ok {
							continue
						}
						seen[children[i].CommentID] = struct{}{}
						out = append(out, children[i])
					}
				}
			}
		}

		if resp.Paging.IsEnd {
			break
		}
		if strings.TrimSpace(resp.Paging.Next) != "" {
			nextURL = resp.Paging.Next
			continue
		}
		offset += limit
		nextURL = updateQueryOffset(nextURL, offset, limit)
	}
	return out, nil
}

func fetchAllChildCommentsAPI(ctx context.Context, c jsonFetchClient, noteID string, parentCommentID string, max int) ([]Comment, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	noteID = strings.TrimSpace(noteID)
	parentCommentID = strings.TrimSpace(parentCommentID)
	if noteID == "" || parentCommentID == "" || max == 0 {
		return nil, nil
	}
	if max < 0 {
		max = 5000
	}

	limit := 20
	endpoints := []string{
		fmt.Sprintf("https://www.zhihu.com/api/v4/comments/%s/child_comments?limit=%d&offset=%d", url.PathEscape(parentCommentID), limit, 0),
		fmt.Sprintf("https://www.zhihu.com/api/v4/comment_v5/comments/%s/child_comment?limit=%d&offset=%d", url.PathEscape(parentCommentID), limit, 0),
	}
	var lastErr error
	for _, firstURL := range endpoints {
		out, err := fetchPagedCommentsAPI(ctx, c, noteID, firstURL, max, true)
		if err == nil && len(out) > 0 {
			for i := range out {
				if out[i].ParentCommentID == "" {
					out[i].ParentCommentID = parentCommentID
				}
			}
			return out, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	return nil, lastErr
}

func parseZhihuAPIComment(m map[string]any, noteID string) (Comment, bool) {
	if m == nil || strings.TrimSpace(noteID) == "" {
		return Comment{}, false
	}
	id := toStringID(pickAny(m, "id", "comment_id", "commentId"))
	if id == "" {
		return Comment{}, false
	}
	parentID := ""
	if rt, ok := pickAny(m, "reply_to", "replyTo").(map[string]any); ok && rt != nil {
		parentID = toStringID(pickAny(rt, "id"))
	}
	if parentID == "" {
		parentID = toStringID(pickAny(m, "reply_to_comment_id", "replyToCommentId", "replyToCommentID"))
	}

	content := stripHTML(fmt.Sprintf("%v", pickAny(m, "content", "text", "body")))
	createTime := toInt64(pickAny(m, "created_time", "createdTime", "created"))
	likeCount := toInt64(pickAny(m, "like_count", "likeCount", "vote_count", "voteCount"))

	userID := ""
	userName := ""
	if author, ok := pickAny(m, "author", "user").(map[string]any); ok && author != nil {
		userID = toStringID(pickAny(author, "id", "member_id", "url_token", "urlToken"))
		userName = strings.TrimSpace(fmt.Sprintf("%v", pickAny(author, "name", "nickname")))
		if userName == "<nil>" {
			userName = ""
		}
	}

	return Comment{
		NoteID:          noteID,
		CommentID:       id,
		ParentCommentID: parentID,
		Content:         content,
		CreateTime:      createTime,
		LikeCount:       likeCount,
		UserID:          userID,
		UserNickname:    userName,
	}, true
}

func pickAny(m map[string]any, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

func toStringID(v any) string {
	switch vv := v.(type) {
	case string:
		s := strings.TrimSpace(vv)
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	case float64:
		if vv == 0 {
			return ""
		}
		return strconv.FormatInt(int64(vv), 10)
	case int:
		if vv == 0 {
			return ""
		}
		return strconv.FormatInt(int64(vv), 10)
	case int64:
		if vv == 0 {
			return ""
		}
		return strconv.FormatInt(vv, 10)
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	}
}

func updateQueryOffset(raw string, offset int, limit int) string {
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return raw
	}
	q := u.Query()
	q.Set("offset", strconv.Itoa(offset))
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	u.RawQuery = q.Encode()
	return u.String()
}
