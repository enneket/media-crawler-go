package douyin

import (
	"context"
	"fmt"
	"media-crawler-go/internal/crawler"
	"net/url"
)

type searchResp struct {
	Data  []map[string]any `json:"data"`
	Extra struct {
		LogID string `json:"logid"`
	} `json:"extra"`
}

func (c *Client) SearchInfoByKeyword(ctx context.Context, keyword string, offset int, count int, searchID string, msToken string) (searchResp, error) {
	if err := c.ensureProxy(ctx); err != nil {
		return searchResp{}, err
	}
	if count <= 0 {
		count = 15
	}

	params := defaultParams(msToken, "")
	params.Set("search_channel", "general")
	params.Set("enable_history", "1")
	params.Set("keyword", keyword)
	params.Set("search_source", "tab_search")
	params.Set("query_correct_type", "1")
	params.Set("is_filter_search", "0")
	params.Set("from_group_id", "7378810571505847586")
	params.Set("offset", fmt.Sprintf("%d", offset))
	params.Set("count", fmt.Sprintf("%d", count))
	params.Set("need_filter_settings", "1")
	params.Set("list_type", "multi")
	if searchID != "" {
		params.Set("search_id", searchID)
	}

	aBogus, err := c.signer.SignDetail(params, c.userAgent)
	if err != nil {
		return searchResp{}, err
	}
	params.Set("a_bogus", aBogus)

	var out searchResp
	r, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Referer", buildSearchReferer(keyword)).
		SetQueryString(params.Encode()).
		SetResult(&out).
		Get("/aweme/v1/web/general/search/single/")
	if err != nil {
		return out, err
	}
	if r.IsError() {
		return out, crawler.NewHTTPStatusError("douyin", "/aweme/v1/web/general/search/single/", r.StatusCode(), r.String())
	}
	return out, nil
}

func buildSearchReferer(keyword string) string {
	return fmt.Sprintf("https://www.douyin.com/search/%s?type=general", url.PathEscape(keyword))
}

func pickAwemeInfoFromSearchItem(item map[string]any) map[string]any {
	if v, ok := item["aweme_info"].(map[string]any); ok && v != nil {
		return v
	}
	mix, ok := item["aweme_mix_info"].(map[string]any)
	if !ok || mix == nil {
		return nil
	}
	mixItems, ok := mix["mix_items"].([]any)
	if !ok || len(mixItems) == 0 {
		return nil
	}
	first, ok := mixItems[0].(map[string]any)
	if !ok {
		return nil
	}
	return first
}
