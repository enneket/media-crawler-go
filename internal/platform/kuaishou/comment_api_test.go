package kuaishou

import (
	"context"
	"strings"
	"testing"
)

type fakeGraphQLClient struct {
	page int
}

func (f *fakeGraphQLClient) PostJSON(ctx context.Context, url string, payload any) (FetchResult, error) {
	q := ""
	if m, ok := payload.(map[string]any); ok {
		if s, ok := m["query"].(string); ok {
			q = s
		}
	}
	if strings.Contains(q, "subCommentList(") {
		body := `{"data":{"subCommentList":{"pcursor":"","subCommentsList":[{"commentId":"s1","authorId":"u2","authorName":"n2","content":"sub","timestamp":1700000001,"likedCount":0}]}}}`
		return FetchResult{URL: url, StatusCode: 200, Body: body, FetchedAt: 1}, nil
	}

	if f.page == 0 {
		f.page++
		body := `{"data":{"shortVideoCommentList":{"pcursor":"next","commentList":[{"commentId":"c1","authorId":"u1","authorName":"n1","content":"hi","timestamp":1700000000,"likedCount":1,"subCommentCount":1}]}}}`
		return FetchResult{URL: url, StatusCode: 200, Body: body, FetchedAt: 1}, nil
	}
	body := `{"data":{"shortVideoCommentList":{"pcursor":"","commentList":[]}}}`
	return FetchResult{URL: url, StatusCode: 200, Body: body, FetchedAt: 1}, nil
}

func TestFetchAllPhotoCommentsAPIOnce_PaginatesAndSubcomments(t *testing.T) {
	f := &fakeGraphQLClient{}
	out, err := fetchAllPhotoCommentsAPIOnce(context.Background(), f, "https://live.kuaishou.com/m_graphql", "n", "pid", 10, true)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d out=%+v", len(out), out)
	}
	if out[0].CommentID != "c1" || out[1].CommentID != "s1" {
		t.Fatalf("unexpected ids: %+v", out)
	}
	if out[1].ParentCommentID != "c1" {
		t.Fatalf("sub ParentCommentID=%q", out[1].ParentCommentID)
	}
}
