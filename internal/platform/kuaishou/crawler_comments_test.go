package kuaishou

import (
	"context"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"os"
	"path/filepath"
	"testing"
)

type fakeFetchClientWithComments struct{}

func (f fakeFetchClientWithComments) FetchHTML(ctx context.Context, u string) (FetchResult, error) {
	body := `<html><body><script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"commentList":[{"commentId":"c1","content":"<b>hi</b>","timestamp":1700000000,"likeCount":1,"author":{"id":"u1","name":"n"}},{"commentId":"c2","content":"sub","replyToCommentId":"c1","timestamp":1700000001,"likeCount":0,"author":{"id":"u2","name":"n2"}}]}}}</script></body></html>`
	return FetchResult{URL: u, StatusCode: 200, Body: body, FetchedAt: 1}, nil
}

func TestKuaishouCrawlerCommentsSaved(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	config.AppConfig = config.Config{
		Platform:             "kuaishou",
		StoreBackend:         "file",
		SaveDataOption:       "json",
		DataDir:              "data",
		EnableGetComments:    true,
		EnableGetSubComments: true,
		CrawlerMaxComments:   10,
		CrawlerMaxSleepSec:   0,
	}

	c := NewCrawlerWithClient(fakeFetchClientWithComments{})
	req := crawler.Request{Platform: "kuaishou", Mode: crawler.ModeDetail, Inputs: []string{"https://www.kuaishou.com/short-video/abc123"}, Concurrency: 1}
	if _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}

	noteComments := filepath.Join("data", "kuaishou", "notes", "abc123", "comments.jsonl")
	if _, err := os.Stat(noteComments); err != nil {
		t.Fatalf("expected comments saved at %s: %v", noteComments, err)
	}
	globalComments := filepath.Join("data", "kuaishou", "comments.jsonl")
	if _, err := os.Stat(globalComments); err != nil {
		t.Fatalf("expected global comments saved at %s: %v", globalComments, err)
	}
}

