package tieba

import (
	"context"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type fakeFetchClient struct{}

func (f fakeFetchClient) FetchHTML(ctx context.Context, u string) (FetchResult, error) {
	body := "<html></html>"
	if pu, err := url.Parse(u); err == nil && pu != nil {
		switch pu.Path {
		case "/f/search/res":
			body = `<div class="s_post"><span class="p_title"><a data-tid="123" href="/p/123">t</a></span></div>`
		case "/home/main":
			body = `<ul class="new_list clearfix"><div class="thread_name"><a href="/p/456">x</a></div></ul>`
		}
	}
	return FetchResult{
		URL:        u,
		StatusCode: 200,
		Body:       body,
		FetchedAt:  1,
	}, nil
}

type fakeFetchClientWithComments struct{}

func (f fakeFetchClientWithComments) FetchHTML(ctx context.Context, u string) (FetchResult, error) {
	body := "<html></html>"
	if pu, err := url.Parse(u); err == nil && pu != nil {
		switch pu.Path {
		case "/p/123":
			body = `<div class="l_post l_post_bright j_l_post clearfix  " data-field='{"author":{"user_id":"u1","user_name":"alice"},"content":{"post_id":111,"forum_id":999,"comment_num":1,"content":"<div>hi<br/>there</div>"}}'></div>`
		case "/p/comment":
			body = `<li class="lzl_single_post j_lzl_s_p first_no_border" data-field='{"spid":222,"showname":"bob"}'><span class="lzl_content_main">sub <em>x</em></span></li>`
		}
	}
	return FetchResult{
		URL:        u,
		StatusCode: 200,
		Body:       body,
		FetchedAt:  1,
	}, nil
}

func TestTiebaCrawlerSearchAndCreator(t *testing.T) {
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
		Platform:           "tieba",
		StoreBackend:       "file",
		SaveDataOption:     "json",
		DataDir:            "data",
		CrawlerMaxSleepSec: 0,
	}

	c := NewCrawlerWithClient(fakeFetchClient{})

	{
		req := crawler.Request{Platform: "tieba", Mode: crawler.ModeSearch, Keywords: []string{"k"}, MaxNotes: 1, Concurrency: 1, StartPage: 1}
		if _, err := c.Run(context.Background(), req); err != nil {
			t.Fatalf("search run: %v", err)
		}
		path := filepath.Join("data", "tieba", "notes", "123", "note.json")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected note saved at %s: %v", path, err)
		}
	}

	{
		req := crawler.Request{Platform: "tieba", Mode: crawler.ModeCreator, Inputs: []string{"un=test"}, MaxNotes: 1, Concurrency: 1}
		if _, err := c.Run(context.Background(), req); err != nil {
			t.Fatalf("creator run: %v", err)
		}
		profile := filepath.Join("data", "tieba", "creators", "test", "profile.json")
		if _, err := os.Stat(profile); err != nil {
			t.Fatalf("expected profile saved at %s: %v", profile, err)
		}
		note := filepath.Join("data", "tieba", "notes", "456", "note.json")
		if _, err := os.Stat(note); err != nil {
			t.Fatalf("expected note saved at %s: %v", note, err)
		}
	}
}

func TestTiebaCrawlerCommentsSaved(t *testing.T) {
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
		Platform:             "tieba",
		StoreBackend:         "file",
		SaveDataOption:       "json",
		DataDir:              "data",
		EnableGetComments:    true,
		EnableGetSubComments: true,
		CrawlerMaxComments:   10,
		CrawlerMaxSleepSec:   0,
	}

	c := NewCrawlerWithClient(fakeFetchClientWithComments{})
	req := crawler.Request{Platform: "tieba", Mode: crawler.ModeDetail, Inputs: []string{"123"}, Concurrency: 1}
	if _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("detail run: %v", err)
	}
	noteComments := filepath.Join("data", "tieba", "notes", "123", "comments.jsonl")
	if _, err := os.Stat(noteComments); err != nil {
		t.Fatalf("expected comments saved at %s: %v", noteComments, err)
	}
	globalComments := filepath.Join("data", "tieba", "comments.jsonl")
	if _, err := os.Stat(globalComments); err != nil {
		t.Fatalf("expected global comments saved at %s: %v", globalComments, err)
	}
}
