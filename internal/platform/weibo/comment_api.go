package weibo

import (
	"context"
	"fmt"
	"html"
	"media-crawler-go/internal/crawler"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type hotflowResp struct {
	Ok   int         `json:"ok"`
	Data hotflowData `json:"data"`
}

type hotflowData struct {
	MaxID     int64            `json:"max_id"`
	MaxIDType int              `json:"max_id_type"`
	Data      []hotflowComment `json:"data"`
}

type hotflowComment struct {
	ID          any             `json:"id"`
	RootID      any             `json:"rootid"`
	Text        string          `json:"text"`
	CreatedAt   string          `json:"created_at"`
	LikeCount   int64           `json:"like_count"`
	TotalNumber int64           `json:"total_number"`
	Source      string          `json:"source"`
	User        hotflowUser     `json:"user"`
	Comments    []hotflowComment `json:"comments"`
}

type hotflowUser struct {
	ID         any    `json:"id"`
	ScreenName string `json:"screen_name"`
}

func (c *Client) GetNoteComments(ctx context.Context, noteID string, maxID int64, maxIDType int) (hotflowData, error) {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return hotflowData{}, fmt.Errorf("empty note id")
	}
	params := map[string]string{
		"id":          noteID,
		"mid":         noteID,
		"max_id_type": strconv.Itoa(maxIDType),
	}
	if maxID > 0 {
		params["max_id"] = strconv.FormatInt(maxID, 10)
	}
	referer := "https://m.weibo.cn/detail/" + noteID
	var out hotflowResp
	r, err := c.httpClient.R().
		SetContext(ctx).
		SetHeaders(map[string]string{
			"referer": referer,
		}).
		SetQueryParams(params).
		SetResult(&out).
		Get("/comments/hotflow")
	if err != nil {
		return hotflowData{}, err
	}
	if r.StatusCode() != http.StatusOK {
		return hotflowData{}, crawler.NewHTTPStatusError("weibo", "/comments/hotflow", r.StatusCode(), r.String())
	}
	if out.Ok != 1 {
		return hotflowData{}, fmt.Errorf("weibo api not ok: ok=%d", out.Ok)
	}
	return out.Data, nil
}

type commentClient interface {
	GetNoteComments(context.Context, string, int64, int) (hotflowData, error)
}

func fetchAllNoteComments(ctx context.Context, client commentClient, noteID string, max int, sleepSec int, enableSub bool) ([]Comment, error) {
	noteID = strings.TrimSpace(noteID)
	if client == nil || noteID == "" || max == 0 {
		return nil, nil
	}
	if max < 0 {
		max = 5000
	}
	if sleepSec < 0 {
		sleepSec = 0
	}

	out := make([]Comment, 0, 128)
	maxID := int64(-1)
	maxIDType := 0
	for len(out) < max {
		data, err := client.GetNoteComments(ctx, noteID, maxID, maxIDType)
		if err != nil {
			return out, err
		}
		maxID = data.MaxID
		maxIDType = data.MaxIDType
		isEnd := maxID == 0
		list := data.Data
		if len(list) == 0 {
			break
		}
		for _, it := range list {
			out = append(out, toComment(noteID, it))
			if len(out) >= max {
				break
			}
			if enableSub && len(it.Comments) > 0 {
				for _, sub := range it.Comments {
					out = append(out, toComment(noteID, sub))
					if len(out) >= max {
						break
					}
				}
			}
			if len(out) >= max {
				break
			}
		}
		if isEnd {
			break
		}
		if sleepSec > 0 {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(time.Duration(sleepSec) * time.Second):
			}
		}
	}
	return out, nil
}

func toComment(noteID string, it hotflowComment) Comment {
	return Comment{
		NoteID:          noteID,
		CommentID:       strings.TrimSpace(fmt.Sprintf("%v", it.ID)),
		ParentCommentID: strings.TrimSpace(fmt.Sprintf("%v", it.RootID)),
		Content:         stripHTML(it.Text),
		CreateTime:      parseWeiboTime(it.CreatedAt),
		LikeCount:       it.LikeCount,
		UserID:          strings.TrimSpace(fmt.Sprintf("%v", it.User.ID)),
		UserNickname:    strings.TrimSpace(it.User.ScreenName),
	}
}

var reHTMLTags = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = reHTMLTags.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(s))
}

func parseWeiboTime(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if t, err := time.Parse(time.RubyDate, s); err == nil {
		return t.Unix()
	}
	if t, err := time.Parse(time.RFC1123Z, s); err == nil {
		return t.Unix()
	}
	if t, err := time.Parse(time.RFC1123, s); err == nil {
		return t.Unix()
	}
	return 0
}

