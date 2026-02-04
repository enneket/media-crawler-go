package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	httpClient *resty.Client
}

func NewClient() *Client {
	timeoutSec := config.AppConfig.HttpTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 60
	}
	hc := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	rc := resty.NewWithClient(hc)
	rc.SetBaseURL("https://api.bilibili.com")
	rc.SetHeaders(map[string]string{
		"accept":          "application/json, text/plain, */*",
		"accept-language": "zh-CN,zh;q=0.9",
		"referer":         "https://www.bilibili.com/",
		"user-agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	})
	if ck := strings.TrimSpace(config.AppConfig.Cookies); ck != "" {
		rc.SetHeader("cookie", ck)
	}

	retryCount := config.AppConfig.HttpRetryCount
	if retryCount <= 0 {
		retryCount = 3
	}
	baseMs := config.AppConfig.HttpRetryBaseDelayMs
	if baseMs <= 0 {
		baseMs = 500
	}
	maxMs := config.AppConfig.HttpRetryMaxDelayMs
	if maxMs <= 0 {
		maxMs = 4000
	}
	rc.SetRetryCount(retryCount)
	rc.SetRetryWaitTime(time.Duration(baseMs) * time.Millisecond)
	rc.SetRetryMaxWaitTime(time.Duration(maxMs) * time.Millisecond)
	rc.AddRetryCondition(func(r *resty.Response, err error) bool {
		if err != nil {
			return crawler.ShouldRetryError(err)
		}
		if r == nil {
			return true
		}
		return crawler.ShouldRetryStatus(r.StatusCode())
	})

	return &Client{httpClient: rc}
}

type ViewResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) GetView(ctx context.Context, bvid string, aid int64) (ViewResponse, error) {
	req := c.httpClient.R().SetContext(ctx)
	if bvid != "" {
		req.SetQueryParam("bvid", bvid)
	} else if aid > 0 {
		req.SetQueryParam("aid", fmt.Sprintf("%d", aid))
	} else {
		return ViewResponse{}, fmt.Errorf("bvid/aid is empty")
	}
	var out ViewResponse
	resp, err := req.SetResult(&out).Get("/x/web-interface/view")
	if err != nil {
		return ViewResponse{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return ViewResponse{}, crawler.NewHTTPStatusError("bilibili", "/x/web-interface/view", resp.StatusCode(), resp.String())
	}
	if out.Code != 0 {
		return ViewResponse{}, fmt.Errorf("bilibili api error: code=%d message=%s", out.Code, out.Message)
	}
	return out, nil
}

type SearchResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) SearchVideo(ctx context.Context, keyword string, page int, searchType string) (SearchResponse, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return SearchResponse{}, fmt.Errorf("empty keyword")
	}
	if page <= 0 {
		page = 1
	}
	searchType = strings.ToLower(strings.TrimSpace(searchType))
	if searchType == "" {
		searchType = "video"
	}

	var out SearchResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"search_type": searchType,
			"keyword":     keyword,
			"page":        strconv.Itoa(page),
		}).
		SetResult(&out).
		Get("/x/web-interface/search/type")
	if err != nil {
		return SearchResponse{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return SearchResponse{}, crawler.NewHTTPStatusError("bilibili", "/x/web-interface/search/type", resp.StatusCode(), resp.String())
	}
	if out.Code != 0 {
		return SearchResponse{}, fmt.Errorf("bilibili api error: code=%d message=%s", out.Code, out.Message)
	}
	return out, nil
}

type UpInfoResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) GetUpInfo(ctx context.Context, mid string) (UpInfoResponse, error) {
	mid = strings.TrimSpace(mid)
	if mid == "" {
		return UpInfoResponse{}, fmt.Errorf("empty mid")
	}
	var out UpInfoResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParam("mid", mid).
		SetResult(&out).
		Get("/x/space/acc/info")
	if err != nil {
		return UpInfoResponse{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return UpInfoResponse{}, crawler.NewHTTPStatusError("bilibili", "/x/space/acc/info", resp.StatusCode(), resp.String())
	}
	if out.Code != 0 {
		return UpInfoResponse{}, fmt.Errorf("bilibili api error: code=%d message=%s", out.Code, out.Message)
	}
	return out, nil
}

type UpVideosResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) ListUpVideos(ctx context.Context, mid string, page int, pageSize int) (UpVideosResponse, error) {
	mid = strings.TrimSpace(mid)
	if mid == "" {
		return UpVideosResponse{}, fmt.Errorf("empty mid")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 30
	}

	var out UpVideosResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"mid": mid,
			"pn":  strconv.Itoa(page),
			"ps":  strconv.Itoa(pageSize),
		}).
		SetResult(&out).
		Get("/x/space/arc/search")
	if err != nil {
		return UpVideosResponse{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return UpVideosResponse{}, crawler.NewHTTPStatusError("bilibili", "/x/space/arc/search", resp.StatusCode(), resp.String())
	}
	if out.Code != 0 {
		return UpVideosResponse{}, fmt.Errorf("bilibili api error: code=%d message=%s", out.Code, out.Message)
	}
	return out, nil
}

type PlayURLResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) GetPlayURL(ctx context.Context, aid int64, cid int64, qn int) (PlayURLResponse, error) {
	if aid <= 0 || cid <= 0 {
		return PlayURLResponse{}, fmt.Errorf("invalid aid/cid")
	}
	if qn <= 0 {
		qn = 80
	}
	var out PlayURLResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"avid":     strconv.FormatInt(aid, 10),
			"cid":      strconv.FormatInt(cid, 10),
			"qn":       strconv.Itoa(qn),
			"fourk":    "1",
			"fnval":    "1",
			"platform": "pc",
		}).
		SetResult(&out).
		Get("/x/player/playurl")
	if err != nil {
		return PlayURLResponse{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return PlayURLResponse{}, crawler.NewHTTPStatusError("bilibili", "/x/player/playurl", resp.StatusCode(), resp.String())
	}
	if out.Code != 0 {
		return PlayURLResponse{}, fmt.Errorf("bilibili api error: code=%d message=%s", out.Code, out.Message)
	}
	return out, nil
}
