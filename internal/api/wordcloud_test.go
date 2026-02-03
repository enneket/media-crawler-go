package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
)

func TestWordcloudFromJSONL(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	if err := os.MkdirAll(filepath.Join("data", "xhs", "notes", "n1"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("data", "xhs", "notes", "n1", "comments.jsonl"), []byte("{\"content\":\"你好 世界 hello hello\"}\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	config.AppConfig = config.Config{
		Platform:       "xhs",
		StoreBackend:   "file",
		SaveDataOption: "json",
		DataDir:        "data",
	}

	srv := NewServer(NewTaskManagerWithRunner(func(ctx context.Context) (crawler.Result, error) {
		return crawler.Result{}, nil
	}))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/data/wordcloud?platform=xhs&note_id=n1&save=false&min_count=1&max_words=50")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(b))
	}
	ct := resp.Header.Get("content-type")
	if !strings.HasPrefix(strings.ToLower(ct), "image/svg+xml") {
		t.Fatalf("content-type=%s", ct)
	}
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), "<svg") {
		t.Fatalf("not svg: %s", string(b))
	}
}
