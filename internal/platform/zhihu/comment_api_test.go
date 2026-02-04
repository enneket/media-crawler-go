package zhihu

import (
	"context"
	"testing"
)

type fakeJSONClient struct {
	body string
}

func (f fakeJSONClient) FetchJSON(ctx context.Context, u string) (FetchResult, error) {
	return FetchResult{URL: u, StatusCode: 200, Body: f.body, FetchedAt: 1}, nil
}

func TestParseZhihuAPIComment_Basic(t *testing.T) {
	m := map[string]any{
		"id":           float64(123),
		"content":      "<p>Hello</p>",
		"created_time": float64(1700000000),
		"vote_count":   float64(7),
		"author": map[string]any{
			"id":   "u1",
			"name": "n1",
		},
	}
	c, ok := parseZhihuAPIComment(m, "n")
	if !ok {
		t.Fatalf("expected ok")
	}
	if c.CommentID != "123" {
		t.Fatalf("CommentID=%q", c.CommentID)
	}
	if c.Content != "Hello" {
		t.Fatalf("Content=%q", c.Content)
	}
	if c.LikeCount != 7 {
		t.Fatalf("LikeCount=%d", c.LikeCount)
	}
}

func TestFetchPagedCommentsAPI_UsesPagingNext(t *testing.T) {
	f := fakeJSONClient{
		body: `{"data":[{"id":"1","content":"a","created_time":1,"vote_count":0,"author":{"id":"u","name":"n"}}],"paging":{"is_end":true,"next":""}}`,
	}
	out, err := fetchPagedCommentsAPI(context.Background(), f, "n", "https://www.zhihu.com/api/v4/answers/x/comments?limit=20&offset=0", 10, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 1 || out[0].CommentID != "1" {
		t.Fatalf("unexpected out: %+v", out)
	}
}
