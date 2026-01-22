package xhs

import (
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/playwright-community/playwright-go"
)

type Client struct {
	HttpClient *resty.Client
	Signer     *Signer
	Cookies    map[string]string
	UserAgent  string
}

func NewClient(signer *Signer) *Client {
	client := resty.New()
	client.SetBaseURL("https://edith.xiaohongshu.com")
	
	// Default headers
	client.SetHeaders(map[string]string{
		"accept":          "application/json, text/plain, */*",
		"accept-language": "zh-CN,zh;q=0.9",
		"cache-control":   "no-cache",
		"content-type":    "application/json;charset=UTF-8",
		"origin":          "https://www.xiaohongshu.com",
		"pragma":          "no-cache",
		"referer":         "https://www.xiaohongshu.com/",
	})

	return &Client{
		HttpClient: client,
		Signer:     signer,
		Cookies:    make(map[string]string),
	}
}

func (c *Client) UpdateCookies(ctx playwright.BrowserContext) error {
	cookies, err := ctx.Cookies()
	if err != nil {
		return err
	}
	
	var cookieStrs []string
	for _, cookie := range cookies {
		c.Cookies[cookie.Name] = cookie.Value
		cookieStrs = append(cookieStrs, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}
	
	cookieHeader := strings.Join(cookieStrs, "; ")
	c.HttpClient.SetHeader("Cookie", cookieHeader)
	return nil
}

func (c *Client) SetUserAgent(ua string) {
	c.UserAgent = ua
	c.HttpClient.SetHeader("user-agent", ua)
}

func (c *Client) preHeaders(uri string, data interface{}, method string) (map[string]string, error) {
	a1 := c.Cookies["a1"]
	return c.Signer.Sign(uri, data, a1, method)
}

func (c *Client) Post(uri string, data interface{}, result interface{}) error {
	headers, err := c.preHeaders(uri, data, "POST")
	if err != nil {
		return err
	}

	resp, err := c.HttpClient.R().
		SetHeaders(headers).
		SetBody(data).
		SetResult(result).
		Post(uri)

	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("status: %d, body: %s", resp.StatusCode(), resp.String())
	}
	
	return nil
}

func (c *Client) Pong() bool {
	res, err := c.GetNoteByKeyword("Xiaohongshu", 1)
	if err != nil {
		return false
	}
	return len(res.Items) > 0
}

func (c *Client) GetNoteByKeyword(keyword string, page int) (*SearchResult, error) {
	uri := "/api/sns/web/v1/search/notes"
	data := map[string]interface{}{
		"keyword":   keyword,
		"page":      page,
		"page_size": 20,
		"search_id": GetSearchId(),
		"sort":      "general",
		"note_type": 0,
	}

	// Wrapper for response
	type Response struct {
		Success bool          `json:"success"`
		Code    int           `json:"code"`
		Msg     string        `json:"msg"`
		Data    SearchResult  `json:"data"`
	}

	var resp Response
	err := c.Post(uri, data, &resp)
	if err != nil {
		return nil, err
	}
	
	if !resp.Success {
		return nil, fmt.Errorf("api error: %s", resp.Msg)
	}

	return &resp.Data, nil
}

func (c *Client) GetNotesByCreator(userId, cursor string) (*CreatorNotesResult, error) {
	uri := "/api/sns/web/v1/user_posted"
	params := map[string]string{
		"user_id":       userId,
		"cursor":        cursor,
		"num":           "30",
		"image_formats": "jpg,webp,avif",
	}

	headers, err := c.preHeaders(uri, params, "GET")
	if err != nil {
		return nil, err
	}

	type Response struct {
		Success bool               `json:"success"`
		Code    int                `json:"code"`
		Msg     string             `json:"msg"`
		Data    CreatorNotesResult `json:"data"`
	}

	var resp Response
	r, err := c.HttpClient.R().
		SetHeaders(headers).
		SetQueryParams(params).
		SetResult(&resp).
		Get(uri)

	if err != nil {
		return nil, err
	}

	if r.IsError() {
		return nil, fmt.Errorf("status: %d, body: %s", r.StatusCode(), r.String())
	}

	if !resp.Success {
		return nil, fmt.Errorf("api error: %s", resp.Msg)
	}

	return &resp.Data, nil
}

func (c *Client) GetNoteById(noteId, xsecSource, xsecToken string) (*Note, error) {
	if xsecSource == "" {
		xsecSource = "pc_search"
	}

	uri := "/api/sns/web/v1/feed"
	data := map[string]interface{}{
		"source_note_id": noteId,
		"image_formats":  []string{"jpg", "webp", "avif"},
		"extra":          map[string]int{"need_body_topic": 1},
		"xsec_source":    xsecSource,
		"xsec_token":     xsecToken,
	}

	type Response struct {
		Success bool `json:"success"`
		Code    int  `json:"code"`
		Msg     string `json:"msg"`
		Data    struct {
			Items []struct {
				NoteCard Note `json:"note_card"`
			} `json:"items"`
		} `json:"data"`
	}

	var resp Response
	err := c.Post(uri, data, &resp)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("api error: %s", resp.Msg)
	}

	if len(resp.Data.Items) == 0 {
		return nil, fmt.Errorf("note not found")
	}

	note := resp.Data.Items[0].NoteCard
	return &note, nil
}

func (c *Client) GetNoteComments(noteId, xsecToken, cursor string) (*CommentResult, error) {
	uri := "/api/sns/web/v2/comment/page"
	params := map[string]string{
		"note_id":       noteId,
		"cursor":        cursor,
		"top_comment_id": "",
		"image_formats": "jpg,webp,avif",
		"xsec_token":    xsecToken,
	}

	// Sign for GET request
	headers, err := c.preHeaders(uri, params, "GET")
	if err != nil {
		return nil, err
	}

	type Response struct {
		Success bool          `json:"success"`
		Code    int           `json:"code"`
		Msg     string        `json:"msg"`
		Data    CommentResult `json:"data"`
	}

	var resp Response
	// Build query params
	// Resty handles params with SetQueryParams
	r, err := c.HttpClient.R().
		SetHeaders(headers).
		SetQueryParams(params).
		SetResult(&resp).
		Get(uri)
	
	if err != nil {
		return nil, err
	}

	if r.IsError() {
		return nil, fmt.Errorf("status: %d, body: %s", r.StatusCode(), r.String())
	}

	if !resp.Success {
		return nil, fmt.Errorf("api error: %s", resp.Msg)
	}

	return &resp.Data, nil
}
