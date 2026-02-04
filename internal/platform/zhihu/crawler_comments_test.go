package zhihu

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
	body := `<html><head></head><body><script id="js-initialData" type="text/json">{"initialState":{"entities":{"comments":{"1":{"id":"1","content":"<p>hi</p>","createdTime":1700000000,"likeCount":2,"author":{"id":"u1","name":"n"}},"2":{"id":"2","content":"sub","replyToCommentId":"1","createdTime":1700000001,"likeCount":0,"author":{"id":"u2","name":"n2"}}}}}}</script></body></html>`
	return FetchResult{URL: u, StatusCode: 200, Body: body, FetchedAt: 1}, nil
}

func TestZhihuCrawlerCommentsSaved(t *testing.T) {
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
		Platform:             "zhihu",
		StoreBackend:         "file",
		SaveDataOption:       "json",
		DataDir:              "data",
		EnableGetComments:    true,
		EnableGetSubComments: true,
		CrawlerMaxComments:   10,
		CrawlerMaxSleepSec:   0,
	}

	c := NewCrawlerWithClient(fakeFetchClientWithComments{})
	req := crawler.Request{Platform: "zhihu", Mode: crawler.ModeDetail, Inputs: []string{"https://www.zhihu.com/question/123/answer/456"}, Concurrency: 1}
	if _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}

	noteComments := filepath.Join("data", "zhihu", "notes", "123_456", "comments.jsonl")
	if _, err := os.Stat(noteComments); err != nil {
		t.Fatalf("expected comments saved at %s: %v", noteComments, err)
	}
	globalComments := filepath.Join("data", "zhihu", "comments.jsonl")
	if _, err := os.Stat(globalComments); err != nil {
		t.Fatalf("expected global comments saved at %s: %v", globalComments, err)
	}
}

