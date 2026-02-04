package bilibili

import (
	"context"
	"encoding/json"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"os"
	"path/filepath"
	"testing"
)

type fakeClient struct{}

func (f fakeClient) GetView(ctx context.Context, bvid string, aid int64) (ViewResponse, error) {
	payload := map[string]any{"bvid": bvid, "aid": aid}
	b, _ := json.Marshal(payload)
	return ViewResponse{Code: 0, Data: b}, nil
}

func (f fakeClient) SearchVideo(ctx context.Context, keyword string, page int, searchType string) (SearchResponse, error) {
	data := map[string]any{
		"result": []any{
			map[string]any{"bvid": "BV1Q5411W7bH", "aid": 170001},
		},
	}
	b, _ := json.Marshal(data)
	return SearchResponse{Code: 0, Data: b}, nil
}

func (f fakeClient) GetUpInfo(ctx context.Context, mid string) (UpInfoResponse, error) {
	b, _ := json.Marshal(map[string]any{"mid": mid})
	return UpInfoResponse{Code: 0, Data: b}, nil
}

func (f fakeClient) ListUpVideos(ctx context.Context, mid string, page int, pageSize int) (UpVideosResponse, error) {
	data := map[string]any{
		"list": map[string]any{
			"vlist": []any{
				map[string]any{"bvid": "BV1Q5411W7bH", "aid": 170001},
			},
		},
	}
	b, _ := json.Marshal(data)
	return UpVideosResponse{Code: 0, Data: b}, nil
}

type fakeClientWithComments struct{}

func (f fakeClientWithComments) GetView(ctx context.Context, bvid string, aid int64) (ViewResponse, error) {
	payload := map[string]any{"bvid": bvid, "aid": aid}
	b, _ := json.Marshal(payload)
	return ViewResponse{Code: 0, Data: b}, nil
}

func (f fakeClientWithComments) SearchVideo(ctx context.Context, keyword string, page int, searchType string) (SearchResponse, error) {
	data := map[string]any{
		"result": []any{
			map[string]any{"bvid": "BV1Q5411W7bH", "aid": 170001},
		},
	}
	b, _ := json.Marshal(data)
	return SearchResponse{Code: 0, Data: b}, nil
}

func (f fakeClientWithComments) GetUpInfo(ctx context.Context, mid string) (UpInfoResponse, error) {
	b, _ := json.Marshal(map[string]any{"mid": mid})
	return UpInfoResponse{Code: 0, Data: b}, nil
}

func (f fakeClientWithComments) ListUpVideos(ctx context.Context, mid string, page int, pageSize int) (UpVideosResponse, error) {
	data := map[string]any{
		"list": map[string]any{
			"vlist": []any{
				map[string]any{"bvid": "BV1Q5411W7bH", "aid": 170001},
			},
		},
	}
	b, _ := json.Marshal(data)
	return UpVideosResponse{Code: 0, Data: b}, nil
}

func (f fakeClientWithComments) GetVideoComments(ctx context.Context, oid int64, page int, pageSize int, sort int) (replyMainResp, error) {
	return replyMainResp{
		Code: 0,
		Data: &replyMainData{
			Cursor: struct {
				IsEnd bool `json:"is_end"`
				Next  int  `json:"next"`
			}{IsEnd: true, Next: 0},
			Replies: []replyItem{
				{
					RPID:   1,
					Parent: 0,
					CTime:  1700000000,
					Like:   3,
					Content: struct {
						Message string `json:"message"`
					}{Message: "hi"},
					Member: struct {
						Mid   string `json:"mid"`
						Uname string `json:"uname"`
					}{Mid: "42", Uname: "u"},
				},
			},
		},
	}, nil
}

func (f fakeClientWithComments) GetVideoSubComments(ctx context.Context, oid int64, root int64, page int, pageSize int) (replySubResp, error) {
	return replySubResp{Code: 0, Data: &replySubData{Replies: nil}}, nil
}

func TestCrawlerSearchAndCreator(t *testing.T) {
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
		Platform:        "bilibili",
		StoreBackend:    "file",
		SaveDataOption:  "json",
		DataDir:         "data",
		BiliSearchMode:  "video",
		CrawlerMaxSleepSec: 0,
	}

	c := NewCrawlerWithClient(fakeClient{})

	{
		req := crawler.Request{Platform: "bilibili", Mode: crawler.ModeSearch, Keywords: []string{"k"}, MaxNotes: 1, Concurrency: 1, StartPage: 1}
		if _, err := c.Run(context.Background(), req); err != nil {
			t.Fatalf("search run: %v", err)
		}
		path := filepath.Join("data", "bilibili", "notes", "BV1Q5411W7BH", "note.json")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected note saved at %s: %v", path, err)
		}
	}

	{
		req := crawler.Request{Platform: "bilibili", Mode: crawler.ModeCreator, Inputs: []string{"123456"}, MaxNotes: 1, Concurrency: 1}
		if _, err := c.Run(context.Background(), req); err != nil {
			t.Fatalf("creator run: %v", err)
		}
		profile := filepath.Join("data", "bilibili", "creators", "123456", "profile.json")
		if _, err := os.Stat(profile); err != nil {
			t.Fatalf("expected profile saved at %s: %v", profile, err)
		}
	}
}

func TestCrawlerCommentsSaved(t *testing.T) {
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
		Platform:            "bilibili",
		StoreBackend:        "file",
		SaveDataOption:      "json",
		DataDir:             "data",
		BiliSearchMode:      "video",
		EnableGetComments:   true,
		EnableGetSubComments: false,
		CrawlerMaxComments:  10,
		CrawlerMaxSleepSec:  0,
	}

	c := NewCrawlerWithClient(fakeClientWithComments{})
	req := crawler.Request{Platform: "bilibili", Mode: crawler.ModeSearch, Keywords: []string{"k"}, MaxNotes: 1, Concurrency: 1, StartPage: 1}
	if _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}

	noteComments := filepath.Join("data", "bilibili", "notes", "BV1Q5411W7BH", "comments.jsonl")
	if _, err := os.Stat(noteComments); err != nil {
		t.Fatalf("expected comments saved at %s: %v", noteComments, err)
	}
	globalComments := filepath.Join("data", "bilibili", "comments.jsonl")
	if _, err := os.Stat(globalComments); err != nil {
		t.Fatalf("expected global comments saved at %s: %v", globalComments, err)
	}
}
