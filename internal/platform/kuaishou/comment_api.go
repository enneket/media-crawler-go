package kuaishou

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type jsonPostClient interface {
	PostJSON(context.Context, string, any) (FetchResult, error)
}

func fetchCommentsPreferAPI(ctx context.Context, client any, html string, noteID string, photoID string, max int, enableSub bool) []Comment {
	if strings.TrimSpace(noteID) == "" || max == 0 {
		return nil
	}
	if max < 0 {
		max = 5000
	}
	if strings.TrimSpace(photoID) != "" {
		if pc, ok := client.(jsonPostClient); ok {
			comments, err := fetchAllPhotoCommentsAPI(ctx, pc, noteID, photoID, max, enableSub)
			if err == nil && len(comments) > 0 {
				return comments
			}
		}
	}
	return parseCommentsFromHTML(html, noteID, max, enableSub)
}

type ksGraphQLResponse struct {
	Data map[string]any `json:"data"`
}

func fetchAllPhotoCommentsAPI(ctx context.Context, c jsonPostClient, noteID string, photoID string, max int, enableSub bool) ([]Comment, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	noteID = strings.TrimSpace(noteID)
	photoID = strings.TrimSpace(photoID)
	if noteID == "" || photoID == "" || max == 0 {
		return nil, nil
	}
	if max < 0 {
		max = 5000
	}

	endpoints := []string{
		"https://live.kuaishou.com/m_graphql",
		"https://www.kuaishou.com/graphql",
	}

	var lastErr error
	for _, ep := range endpoints {
		out, err := fetchAllPhotoCommentsAPIOnce(ctx, c, ep, noteID, photoID, max, enableSub)
		if err == nil && len(out) > 0 {
			return out, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	return nil, lastErr
}

func fetchAllPhotoCommentsAPIOnce(ctx context.Context, c jsonPostClient, endpoint string, noteID string, photoID string, max int, enableSub bool) ([]Comment, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if max < 0 {
		max = 5000
	}
	limit := 20
	pcursor := ""

	out := make([]Comment, 0, 64)
	seen := make(map[string]struct{}, 128)

	for len(out) < max {
		query := fmt.Sprintf(`query {
  shortVideoCommentList(photoId: "%s", page: 1, pcursor: "%s", count: %d) {
    pcursor
    commentList {
      commentId
      authorId
      authorName
      content
      timestamp
      likedCount
      replyToCommentId
      subCommentCount
      subCommentsPcursor
      subComments {
        commentId
        authorId
        authorName
        content
        timestamp
        likedCount
        replyToCommentId
        replyToUserName
      }
    }
  }
}`, escapeGQLString(photoID), escapeGQLString(pcursor), limit)

		res, err := c.PostJSON(ctx, endpoint, map[string]any{"query": query})
		if err != nil {
			return out, err
		}
		var resp ksGraphQLResponse
		if err := json.Unmarshal([]byte(res.Body), &resp); err != nil {
			return out, err
		}
		sv, _ := resp.Data["shortVideoCommentList"].(map[string]any)
		if sv == nil {
			return out, fmt.Errorf("missing shortVideoCommentList")
		}

		nextPcursor := strings.TrimSpace(fmt.Sprintf("%v", sv["pcursor"]))
		rawList, _ := sv["commentList"].([]any)
		if len(rawList) == 0 {
			break
		}

		for _, it := range rawList {
			if len(out) >= max {
				break
			}
			m, _ := it.(map[string]any)
			if m == nil {
				continue
			}
			root, ok := mapToAPIComment(m, noteID, "")
			if !ok || root.CommentID == "" {
				continue
			}
			if _, ok := seen[root.CommentID]; ok {
				continue
			}
			seen[root.CommentID] = struct{}{}
			out = append(out, root)

			if enableSub {
				subs := parseInlineSubComments(m, noteID, root.CommentID)
				for i := range subs {
					if len(out) >= max {
						break
					}
					if subs[i].CommentID == "" {
						continue
					}
					if _, ok := seen[subs[i].CommentID]; ok {
						continue
					}
					seen[subs[i].CommentID] = struct{}{}
					out = append(out, subs[i])
				}

				subCount := toInt64(m["subCommentCount"])
				if subCount > int64(len(subs)) && len(out) < max {
					more, _ := fetchAllSubCommentsAPI(ctx, c, endpoint, noteID, photoID, root.CommentID, max-len(out))
					for i := range more {
						if len(out) >= max {
							break
						}
						if more[i].CommentID == "" {
							continue
						}
						if _, ok := seen[more[i].CommentID]; ok {
							continue
						}
						seen[more[i].CommentID] = struct{}{}
						out = append(out, more[i])
					}
				}
			}
		}

		if nextPcursor == "" || nextPcursor == pcursor {
			break
		}
		pcursor = nextPcursor
	}
	return out, nil
}

func fetchAllSubCommentsAPI(ctx context.Context, c jsonPostClient, endpoint string, noteID string, photoID string, rootCommentID string, max int) ([]Comment, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	noteID = strings.TrimSpace(noteID)
	photoID = strings.TrimSpace(photoID)
	rootCommentID = strings.TrimSpace(rootCommentID)
	if noteID == "" || photoID == "" || rootCommentID == "" || max == 0 {
		return nil, nil
	}
	if max < 0 {
		max = 5000
	}

	limit := 20
	pcursor := ""
	out := make([]Comment, 0, 32)
	seen := make(map[string]struct{}, 64)

	for len(out) < max {
		query := fmt.Sprintf(`query {
  subCommentList(photoId: "%s", rootCommentId: "%s", pcursor: "%s", count: %d) {
    pcursor
    subCommentsList {
      commentId
      authorId
      authorName
      content
      timestamp
      likedCount
      replyToCommentId
      replyToUserName
    }
  }
}`, escapeGQLString(photoID), escapeGQLString(rootCommentID), escapeGQLString(pcursor), limit)

		res, err := c.PostJSON(ctx, endpoint, map[string]any{"query": query})
		if err != nil {
			return out, err
		}
		var resp ksGraphQLResponse
		if err := json.Unmarshal([]byte(res.Body), &resp); err != nil {
			return out, err
		}
		sl, _ := resp.Data["subCommentList"].(map[string]any)
		if sl == nil {
			break
		}
		nextPcursor := strings.TrimSpace(fmt.Sprintf("%v", sl["pcursor"]))
		rawList, _ := sl["subCommentsList"].([]any)
		if len(rawList) == 0 {
			break
		}
		for _, it := range rawList {
			if len(out) >= max {
				break
			}
			m, _ := it.(map[string]any)
			if m == nil {
				continue
			}
			cmt, ok := mapToAPIComment(m, noteID, rootCommentID)
			if !ok || cmt.CommentID == "" {
				continue
			}
			if _, ok := seen[cmt.CommentID]; ok {
				continue
			}
			seen[cmt.CommentID] = struct{}{}
			out = append(out, cmt)
		}
		if nextPcursor == "" || nextPcursor == pcursor {
			break
		}
		pcursor = nextPcursor
	}
	return out, nil
}

func parseInlineSubComments(root map[string]any, noteID string, rootCommentID string) []Comment {
	raw, _ := root["subComments"].([]any)
	if len(raw) == 0 {
		return nil
	}
	out := make([]Comment, 0, len(raw))
	for _, it := range raw {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		cmt, ok := mapToAPIComment(m, noteID, rootCommentID)
		if !ok || cmt.CommentID == "" {
			continue
		}
		out = append(out, cmt)
	}
	return out
}

func mapToAPIComment(m map[string]any, noteID string, parentID string) (Comment, bool) {
	if m == nil || strings.TrimSpace(noteID) == "" {
		return Comment{}, false
	}
	id := strings.TrimSpace(fmt.Sprintf("%v", m["commentId"]))
	if id == "" || id == "<nil>" {
		id = strings.TrimSpace(fmt.Sprintf("%v", m["id"]))
	}
	if id == "" || id == "<nil>" {
		return Comment{}, false
	}

	content := strings.TrimSpace(fmt.Sprintf("%v", m["content"]))
	if content == "" || content == "<nil>" {
		content = strings.TrimSpace(fmt.Sprintf("%v", m["text"]))
	}
	if content == "" || content == "<nil>" {
		return Comment{}, false
	}

	ts := toInt64(m["timestamp"])
	if ts == 0 {
		ts = toInt64(m["createTime"])
	}
	if ts > 1_000_000_000_000 {
		ts = ts / 1000
	}
	like := toInt64(m["likedCount"])
	if like == 0 {
		like = toInt64(m["likeCount"])
	}

	userID := strings.TrimSpace(fmt.Sprintf("%v", m["authorId"]))
	if userID == "<nil>" {
		userID = ""
	}
	userName := strings.TrimSpace(fmt.Sprintf("%v", m["authorName"]))
	if userName == "<nil>" {
		userName = ""
	}

	if strings.TrimSpace(parentID) == "" {
		parentID = strings.TrimSpace(fmt.Sprintf("%v", m["replyToCommentId"]))
		if parentID == "<nil>" {
			parentID = ""
		}
	}

	return Comment{
		NoteID:          noteID,
		CommentID:       id,
		ParentCommentID: parentID,
		Content:         stripHTML(content),
		CreateTime:      ts,
		LikeCount:       like,
		UserID:          userID,
		UserNickname:    userName,
	}, true
}

func escapeGQLString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
