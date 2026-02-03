package douyin

import (
	"context"
	"fmt"
	"media-crawler-go/internal/crawler"
	"net/url"
)

type userPostsResp struct {
	AwemeList []map[string]any `json:"aweme_list"`
	HasMore   int              `json:"has_more"`
	MaxCursor string           `json:"max_cursor"`
}

func (c *Client) GetUserInfo(ctx context.Context, secUserID string, msToken string) (map[string]any, error) {
	if err := c.ensureProxy(ctx); err != nil {
		return nil, err
	}
	params := defaultParams(msToken, "")
	params.Set("sec_user_id", secUserID)
	params.Set("publish_video_strategy_type", "2")
	params.Set("personal_center_strategy", "1")

	aBogus, err := c.signer.SignDetail(params, c.userAgent)
	if err != nil {
		return nil, err
	}
	params.Set("a_bogus", aBogus)

	var out map[string]any
	r, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Referer", fmt.Sprintf("https://www.douyin.com/user/%s", url.PathEscape(secUserID))).
		SetQueryString(params.Encode()).
		SetResult(&out).
		Get("/aweme/v1/web/user/profile/other/")
	if err != nil {
		return nil, err
	}
	if r.IsError() {
		return nil, crawler.NewHTTPStatusError("douyin", "/aweme/v1/web/user/profile/other/", r.StatusCode(), r.String())
	}
	return out, nil
}

func (c *Client) GetUserAwemePosts(ctx context.Context, secUserID string, maxCursor string, msToken string) (userPostsResp, error) {
	if err := c.ensureProxy(ctx); err != nil {
		return userPostsResp{}, err
	}
	params := defaultParams(msToken, "")
	params.Set("sec_user_id", secUserID)
	params.Set("count", "18")
	if maxCursor != "" {
		params.Set("max_cursor", maxCursor)
	}
	params.Set("locate_query", "false")
	params.Set("publish_video_strategy_type", "2")
	params.Set("verifyFp", "verify_ma3hrt8n_q2q2HyYA_uLyO_4N6D_BLvX_E2LgoGmkA1BU")
	params.Set("fp", "verify_ma3hrt8n_q2q2HyYA_uLyO_4N6D_BLvX_E2LgoGmkA1BU")

	aBogus, err := c.signer.SignDetail(params, c.userAgent)
	if err != nil {
		return userPostsResp{}, err
	}
	params.Set("a_bogus", aBogus)

	var out userPostsResp
	r, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Referer", fmt.Sprintf("https://www.douyin.com/user/%s", url.PathEscape(secUserID))).
		SetQueryString(params.Encode()).
		SetResult(&out).
		Get("/aweme/v1/web/aweme/post/")
	if err != nil {
		return out, err
	}
	if r.IsError() {
		return out, crawler.NewHTTPStatusError("douyin", "/aweme/v1/web/aweme/post/", r.StatusCode(), r.String())
	}
	return out, nil
}
