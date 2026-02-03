package api

import (
	"context"
	"encoding/json"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDataFilesPreviewDownload(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	dataDir := "data_custom"
	config.AppConfig = config.Config{DataDir: dataDir}

	if err := os.MkdirAll(filepath.Join(dataDir, "xhs", "notes", "n1"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "xhs", "notes", "n2"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	note1 := `{"note_id":"n1","title":"hello"}`
	if err := os.WriteFile(filepath.Join(dataDir, "xhs", "notes", "n1", "note.json"), []byte(note1), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	comments := "{\"comment_id\":\"c1\"}\n{\"comment_id\":\"c2\"}\n"
	if err := os.WriteFile(filepath.Join(dataDir, "xhs", "notes", "n1", "comments.jsonl"), []byte(comments), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	csvContent := "\uFEFFa,b\n1,2\n3,4\n"
	if err := os.WriteFile(filepath.Join(dataDir, "xhs", "notes", "n2", "comments.csv"), []byte(csvContent), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runFn := func(ctx context.Context) (crawler.Result, error) { return crawler.Result{}, nil }
	srv := NewServer(NewTaskManagerWithRunner(runFn))

	{
		r := httptest.NewRequest(http.MethodGet, "/data/files?platform=xhs", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("list code=%d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Files []dataFileInfo `json:"files"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v body=%s", err, w.Body.String())
		}
		if len(resp.Files) < 2 {
			t.Fatalf("expected >=2 files, got=%d", len(resp.Files))
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/data/files/xhs/notes/n1/note.json?preview=true", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("preview json code=%d body=%s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v body=%s", err, w.Body.String())
		}
		if resp["total"].(float64) != 1 {
			t.Fatalf("expected total=1, got=%v", resp["total"])
		}
		data, ok := resp["data"].(map[string]any)
		if !ok || data["note_id"] != "n1" {
			t.Fatalf("unexpected data=%v", resp["data"])
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/data/files/xhs/notes/n1/comments.jsonl?preview=true&limit=1", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("preview jsonl code=%d body=%s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v body=%s", err, w.Body.String())
		}
		if resp["total"].(float64) != 2 {
			t.Fatalf("expected total=2, got=%v", resp["total"])
		}
		data, ok := resp["data"].([]any)
		if !ok || len(data) != 1 {
			t.Fatalf("unexpected data=%v", resp["data"])
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/data/files/xhs/notes/n2/comments.csv?preview=true&limit=1", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("preview csv code=%d body=%s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v body=%s", err, w.Body.String())
		}
		if resp["total"].(float64) != 2 {
			t.Fatalf("expected total=2, got=%v", resp["total"])
		}
		cols, ok := resp["columns"].([]any)
		if !ok || len(cols) != 2 || cols[0].(string) != "a" {
			t.Fatalf("unexpected columns=%v", resp["columns"])
		}
		data, ok := resp["data"].([]any)
		if !ok || len(data) != 1 {
			t.Fatalf("unexpected data=%v", resp["data"])
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/data/download/xhs/notes/n1/note.json", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("download code=%d body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Header().Get("content-disposition"), "note.json") {
			t.Fatalf("missing content-disposition: %v", w.Header())
		}
		if strings.TrimSpace(w.Body.String()) != note1 {
			t.Fatalf("unexpected body=%q", w.Body.String())
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/data/files/../go.mod?preview=true", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusMovedPermanently {
			t.Fatalf("expected 301, got=%d body=%s", w.Code, w.Body.String())
		}
	}
}
