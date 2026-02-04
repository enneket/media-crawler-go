package weibo

import (
	"context"
	"encoding/json"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type fakeClientWithMedia struct {
	base string
}

func (f fakeClientWithMedia) Show(ctx context.Context, id string) (ShowResponse, error) {
	payload := map[string]any{
		"id": id,
		"pics": []any{
			map[string]any{"large": map[string]any{"url": f.base + "/img1.jpg"}},
			map[string]any{"large": map[string]any{"url": f.base + "/img2.png"}},
		},
		"page_info": map[string]any{
			"page_pic": f.base + "/cover.jpg",
			"media_info": map[string]any{
				"stream_url": f.base + "/v.mp4",
			},
		},
	}
	b, _ := json.Marshal(payload)
	return ShowResponse{Ok: 1, Data: b}, nil
}

func (f fakeClientWithMedia) SearchByKeyword(ctx context.Context, keyword string, page int, searchType string) (GetIndexResponse, error) {
	b, _ := json.Marshal(map[string]any{})
	return GetIndexResponse{Ok: 1, Data: b}, nil
}

func (f fakeClientWithMedia) CreatorInfo(ctx context.Context, creatorID string) (GetIndexResponse, error) {
	b, _ := json.Marshal(map[string]any{})
	return GetIndexResponse{Ok: 1, Data: b}, nil
}

func (f fakeClientWithMedia) NotesByCreator(ctx context.Context, creatorID string, containerID string, sinceID string) (GetIndexResponse, error) {
	b, _ := json.Marshal(map[string]any{})
	return GetIndexResponse{Ok: 1, Data: b}, nil
}

func (f fakeClientWithMedia) GetNoteComments(ctx context.Context, noteID string, maxID int64, maxIDType int) (hotflowData, error) {
	return hotflowData{MaxID: 0, MaxIDType: 0, Data: nil}, nil
}

func TestCrawlerMediasSaved(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	mux := http.NewServeMux()
	mux.HandleFunc("/img1.jpg", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("1")) })
	mux.HandleFunc("/img2.png", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("2")) })
	mux.HandleFunc("/cover.jpg", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("c")) })
	mux.HandleFunc("/v.mp4", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("v")) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	config.AppConfig = config.Config{
		Platform:          "weibo",
		StoreBackend:      "file",
		SaveDataOption:    "json",
		DataDir:           "data",
		EnableGetComments: false,
		EnableGetMedias:   true,
	}

	c := NewCrawlerWithClient(fakeClientWithMedia{base: srv.URL})
	req := crawler.Request{Platform: "weibo", Mode: crawler.ModeDetail, Inputs: []string{"4KjD8oZ4D"}, Concurrency: 1}
	if _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}

	mediaDir := filepath.Join("data", "weibo", "notes", "4KjD8oZ4D", "media")
	for _, f := range []string{"4KjD8oZ4D_0.jpg", "4KjD8oZ4D_1.png", "4KjD8oZ4D_cover_0.jpg", "4KjD8oZ4D_video.mp4"} {
		if _, err := os.Stat(filepath.Join(mediaDir, f)); err != nil {
			t.Fatalf("expected media saved: %s: %v", f, err)
		}
	}
}

