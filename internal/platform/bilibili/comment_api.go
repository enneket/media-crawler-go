package bilibili

import (
	"context"
	"fmt"
	"media-crawler-go/internal/crawler"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type replyItem struct {
	RPID    int64 `json:"rpid"`
	Parent  int64 `json:"parent"`
	CTime   int64 `json:"ctime"`
	Like    int   `json:"like"`
	Content struct {
		Message string `json:"message"`
	} `json:"content"`
	Member struct {
		Mid   string `json:"mid"`
		Uname string `json:"uname"`
	} `json:"member"`
}

type replyMainData struct {
	Cursor struct {
		IsEnd bool `json:"is_end"`
		Next  int  `json:"next"`
	} `json:"cursor"`
	Replies []replyItem `json:"replies"`
}

type replyMainResp struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    *replyMainData `json:"data"`
}

type replySubData struct {
	Replies []replyItem `json:"replies"`
}

type replySubResp struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    *replySubData `json:"data"`
}

func (c *Client) GetVideoComments(ctx context.Context, oid int64, page int, pageSize int, sort int) (replyMainResp, error) {
	if oid <= 0 {
		return replyMainResp{}, fmt.Errorf("invalid oid")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if sort < 0 {
		sort = 0
	}
	var out replyMainResp
	r, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"type": "1",
			"oid":  strconv.FormatInt(oid, 10),
			"pn":   strconv.Itoa(page),
			"ps":   strconv.Itoa(pageSize),
			"sort": strconv.Itoa(sort),
		}).
		SetResult(&out).
		Get("/x/v2/reply/main")
	if err != nil {
		return replyMainResp{}, err
	}
	if r.StatusCode() != http.StatusOK {
		return replyMainResp{}, crawler.NewHTTPStatusError("bilibili", "/x/v2/reply/main", r.StatusCode(), r.String())
	}
	if out.Code != 0 {
		return replyMainResp{}, fmt.Errorf("bilibili api error: code=%d message=%s", out.Code, strings.TrimSpace(out.Message))
	}
	return out, nil
}

func (c *Client) GetVideoSubComments(ctx context.Context, oid int64, root int64, page int, pageSize int) (replySubResp, error) {
	if oid <= 0 {
		return replySubResp{}, fmt.Errorf("invalid oid")
	}
	if root <= 0 {
		return replySubResp{}, fmt.Errorf("invalid root")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var out replySubResp
	r, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"type": "1",
			"oid":  strconv.FormatInt(oid, 10),
			"root": strconv.FormatInt(root, 10),
			"pn":   strconv.Itoa(page),
			"ps":   strconv.Itoa(pageSize),
		}).
		SetResult(&out).
		Get("/x/v2/reply/reply")
	if err != nil {
		return replySubResp{}, err
	}
	if r.StatusCode() != http.StatusOK {
		return replySubResp{}, crawler.NewHTTPStatusError("bilibili", "/x/v2/reply/reply", r.StatusCode(), r.String())
	}
	if out.Code != 0 {
		return replySubResp{}, fmt.Errorf("bilibili api error: code=%d message=%s", out.Code, strings.TrimSpace(out.Message))
	}
	return out, nil
}

type commentClient interface {
	GetVideoComments(context.Context, int64, int, int, int) (replyMainResp, error)
	GetVideoSubComments(context.Context, int64, int64, int, int) (replySubResp, error)
}

func fetchAllVideoComments(ctx context.Context, client commentClient, oid int64, max int, sleepSec int, enableSub bool) ([]Comment, error) {
	if client == nil || oid <= 0 || max == 0 {
		return nil, nil
	}
	if max < 0 {
		max = 5000
	}
	page := 1
	pageSize := 20
	sort := 0
	out := make([]Comment, 0, 256)
	for len(out) < max {
		resp, err := client.GetVideoComments(ctx, oid, page, pageSize, sort)
		if err != nil {
			return out, err
		}
		if resp.Data == nil || len(resp.Data.Replies) == 0 {
			break
		}
		for _, r := range resp.Data.Replies {
			out = append(out, Comment{
				CommentID:       strconv.FormatInt(r.RPID, 10),
				ParentCommentID: "",
				Content:         r.Content.Message,
				CreateTime:      r.CTime,
				LikeCount:       r.Like,
				UserID:          strings.TrimSpace(r.Member.Mid),
				UserNickname:    strings.TrimSpace(r.Member.Uname),
			})
			if len(out) >= max {
				break
			}
			if enableSub {
				sub, err := fetchAllVideoSubComments(ctx, client, oid, r.RPID, max-len(out), sleepSec)
				if err != nil {
					return out, err
				}
				out = append(out, sub...)
				if len(out) >= max {
					break
				}
			}
		}
		if resp.Data.Cursor.IsEnd {
			break
		}
		page = resp.Data.Cursor.Next
		if page <= 0 {
			page++
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

func fetchAllVideoSubComments(ctx context.Context, client commentClient, oid int64, root int64, max int, sleepSec int) ([]Comment, error) {
	if client == nil || oid <= 0 || root <= 0 || max == 0 {
		return nil, nil
	}
	if max < 0 {
		max = 5000
	}
	page := 1
	pageSize := 20
	out := make([]Comment, 0, 64)
	rootID := strconv.FormatInt(root, 10)
	for len(out) < max {
		resp, err := client.GetVideoSubComments(ctx, oid, root, page, pageSize)
		if err != nil {
			return out, err
		}
		if resp.Data == nil || len(resp.Data.Replies) == 0 {
			break
		}
		for _, r := range resp.Data.Replies {
			out = append(out, Comment{
				CommentID:       strconv.FormatInt(r.RPID, 10),
				ParentCommentID: rootID,
				Content:         r.Content.Message,
				CreateTime:      r.CTime,
				LikeCount:       r.Like,
				UserID:          strings.TrimSpace(r.Member.Mid),
				UserNickname:    strings.TrimSpace(r.Member.Uname),
			})
			if len(out) >= max {
				break
			}
		}
		page++
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

