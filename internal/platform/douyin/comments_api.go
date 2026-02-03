package douyin

import (
	"context"
	"fmt"
	"media-crawler-go/internal/crawler"
	"net/url"
	"time"
)

type commentListResp struct {
	Comments []Comment `json:"comments"`
	HasMore  int       `json:"has_more"`
	Cursor   int64     `json:"cursor"`
}

func (c *Client) GetAwemeComments(ctx context.Context, awemeID string, cursor int64, msToken string, referer string) (commentListResp, error) {
	if err := c.ensureProxy(ctx); err != nil {
		return commentListResp{}, err
	}
	params := defaultParams(msToken, "")
	params.Set("aweme_id", awemeID)
	params.Set("cursor", fmt.Sprintf("%d", cursor))
	params.Set("count", "20")
	params.Set("item_type", "0")

	aBogus, err := c.signer.SignDetail(params, c.userAgent)
	if err != nil {
		return commentListResp{}, err
	}
	params.Set("a_bogus", aBogus)

	var out commentListResp
	r, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Referer", referer).
		SetQueryString(params.Encode()).
		SetResult(&out).
		Get("/aweme/v1/web/comment/list/")
	if err != nil {
		return commentListResp{}, err
	}
	if r.IsError() {
		return commentListResp{}, crawler.NewHTTPStatusError("douyin", "/aweme/v1/web/comment/list/", r.StatusCode(), r.String())
	}
	return out, nil
}

func (c *Client) GetAwemeSubComments(ctx context.Context, awemeID string, rootCommentID string, cursor int64, msToken string, referer string) (commentListResp, error) {
	if err := c.ensureProxy(ctx); err != nil {
		return commentListResp{}, err
	}
	params := defaultParams(msToken, "")
	params.Set("comment_id", rootCommentID)
	params.Set("cursor", fmt.Sprintf("%d", cursor))
	params.Set("count", "20")
	params.Set("item_type", "0")
	params.Set("item_id", awemeID)

	aBogus, err := c.signer.SignReply(params, c.userAgent)
	if err != nil {
		return commentListResp{}, err
	}
	params.Set("a_bogus", aBogus)

	var out commentListResp
	r, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Referer", referer).
		SetQueryString(params.Encode()).
		SetResult(&out).
		Get("/aweme/v1/web/comment/list/reply/")
	if err != nil {
		return commentListResp{}, err
	}
	if r.IsError() {
		return commentListResp{}, crawler.NewHTTPStatusError("douyin", "/aweme/v1/web/comment/list/reply/", r.StatusCode(), r.String())
	}
	return out, nil
}

func fetchAllAwemeComments(ctx context.Context, client *Client, awemeID string, maxCount int, sleepSec int, msToken string, fetchSub bool) ([]Comment, error) {
	if maxCount == 0 {
		maxCount = -1
	}
	var all []Comment
	cursor := int64(0)
	hasMore := 1
	referer := fmt.Sprintf("https://www.douyin.com/video/%s", url.PathEscape(awemeID))

	for hasMore == 1 && (maxCount < 0 || len(all) < maxCount) {
		resp, err := client.GetAwemeComments(ctx, awemeID, cursor, msToken, referer)
		if err != nil {
			return all, err
		}
		hasMore = resp.HasMore
		cursor = resp.Cursor
		for i := range resp.Comments {
			resp.Comments[i].NoteID = awemeID
			resp.Comments[i].ParentCommentID = ""
		}
		all = append(all, resp.Comments...)

		if fetchSub {
			for _, root := range resp.Comments {
				subCursor := int64(0)
				subHasMore := 1
				for subHasMore == 1 && (maxCount < 0 || len(all) < maxCount) {
					subResp, err := client.GetAwemeSubComments(ctx, awemeID, root.CID, subCursor, msToken, referer)
					if err != nil {
						break
					}
					subHasMore = subResp.HasMore
					subCursor = subResp.Cursor
					for i := range subResp.Comments {
						subResp.Comments[i].NoteID = awemeID
						subResp.Comments[i].ParentCommentID = root.CID
					}
					all = append(all, subResp.Comments...)
					if sleepSec > 0 {
						time.Sleep(time.Duration(sleepSec) * time.Second)
					}
				}
			}
		}

		if sleepSec > 0 {
			time.Sleep(time.Duration(sleepSec) * time.Second)
		}
	}

	if maxCount >= 0 && len(all) > maxCount {
		all = all[:maxCount]
	}
	return all, nil
}
