package bilibili

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

func (f fakeClientWithMedia) GetView(ctx context.Context, bvid string, aid int64) (ViewResponse, error) {
	payload := map[string]any{
		"aid":  170001,
		"cid":  333,
		"bvid": bvid,
		"pic":  f.base + "/cover.jpg",
	}
	b, _ := json.Marshal(payload)
	return ViewResponse{Code: 0, Data: b}, nil
}

func (f fakeClientWithMedia) SearchVideo(ctx context.Context, keyword string, page int, searchType string) (SearchResponse, error) {
	b, _ := json.Marshal(map[string]any{})
	return SearchResponse{Code: 0, Data: b}, nil
}

func (f fakeClientWithMedia) GetUpInfo(ctx context.Context, mid string) (UpInfoResponse, error) {
	b, _ := json.Marshal(map[string]any{})
	return UpInfoResponse{Code: 0, Data: b}, nil
}

func (f fakeClientWithMedia) ListUpVideos(ctx context.Context, mid string, page int, pageSize int) (UpVideosResponse, error) {
	b, _ := json.Marshal(map[string]any{})
	return UpVideosResponse{Code: 0, Data: b}, nil
}

func (f fakeClientWithMedia) GetPlayURL(ctx context.Context, aid int64, cid int64, qn int) (PlayURLResponse, error) {
	payload := map[string]any{
		"durl": []any{
			map[string]any{"url": f.base + "/v.mp4"},
		},
	}
	b, _ := json.Marshal(payload)
	return PlayURLResponse{Code: 0, Data: b}, nil
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
	mux.HandleFunc("/cover.jpg", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("c")) })
	mux.HandleFunc("/v.mp4", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("v")) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	config.AppConfig = config.Config{
		Platform:          "bilibili",
		StoreBackend:      "file",
		SaveDataOption:    "json",
		DataDir:           "data",
		EnableGetComments: false,
		EnableGetMedias:   true,
	}

	c := NewCrawlerWithClient(fakeClientWithMedia{base: srv.URL})
	req := crawler.Request{Platform: "bilibili", Mode: crawler.ModeDetail, Inputs: []string{"BV1Q5411W7bH"}, Concurrency: 1}
	if _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}

	mediaDir := filepath.Join("data", "bilibili", "notes", "BV1Q5411W7BH", "media")
	for _, f := range []string{"BV1Q5411W7BH_cover_0.jpg", "BV1Q5411W7BH_video.mp4"} {
		if _, err := os.Stat(filepath.Join(mediaDir, f)); err != nil {
			t.Fatalf("expected media saved: %s: %v", f, err)
		}
	}
}

