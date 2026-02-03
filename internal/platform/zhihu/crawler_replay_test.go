package zhihu

import (
	"context"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestZhihuCrawlerSearchReplay(t *testing.T) {
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
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<a href="/question/123">q</a><a href="/question/123/answer/456">a</a>`))
	})
	mux.HandleFunc("/question/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body>ok</body></html>`))
	})
	mux.HandleFunc("/question/123/answer/456", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body>ok2</body></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	config.AppConfig = config.Config{
		Platform:       "zhihu",
		StoreBackend:   "file",
		SaveDataOption: "json",
		DataDir:        "data",
	}

	c := NewCrawler()
	req := crawler.Request{
		Platform:     "zhihu",
		Mode:         crawler.ModeSearch,
		Keywords:     []string{srv.URL + "/search"},
		MaxNotes:     10,
		Concurrency:  2,
		StartPage:    1,
	}
	res, err := c.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Succeeded != 2 {
		t.Fatalf("succeeded=%d", res.Succeeded)
	}
	if _, err := os.Stat(filepath.Join("data", "zhihu", "notes", "123", "note.json")); err != nil {
		t.Fatalf("note 123 not saved: %v", err)
	}
	if _, err := os.Stat(filepath.Join("data", "zhihu", "notes", "123_456", "note.json")); err != nil {
		t.Fatalf("note 123_456 not saved: %v", err)
	}
}

func TestZhihuCrawlerCreatorReplay(t *testing.T) {
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
	mux.HandleFunc("/people/u1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<a href="/question/123">q</a><a href="/question/123/answer/456">a</a>`))
	})
	mux.HandleFunc("/question/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body>ok</body></html>`))
	})
	mux.HandleFunc("/question/123/answer/456", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body>ok2</body></html>`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	config.AppConfig = config.Config{
		Platform:       "zhihu",
		StoreBackend:   "file",
		SaveDataOption: "json",
		DataDir:        "data",
	}

	c := NewCrawler()
	req := crawler.Request{
		Platform:    "zhihu",
		Mode:        crawler.ModeCreator,
		Inputs:      []string{srv.URL + "/people/u1"},
		MaxNotes:    10,
		Concurrency: 2,
	}
	res, err := c.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Succeeded != 2 {
		t.Fatalf("succeeded=%d", res.Succeeded)
	}
	if _, err := os.Stat(filepath.Join("data", "zhihu", "notes", "123", "note.json")); err != nil {
		t.Fatalf("note 123 not saved: %v", err)
	}
	if _, err := os.Stat(filepath.Join("data", "zhihu", "notes", "123_456", "note.json")); err != nil {
		t.Fatalf("note 123_456 not saved: %v", err)
	}
	ents, err := os.ReadDir(filepath.Join("data", "zhihu"))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	hasCreator := false
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), "creators_") && strings.HasSuffix(e.Name(), ".json") {
			hasCreator = true
			break
		}
	}
	if !hasCreator {
		t.Fatalf("expected creators_*.json in data/zhihu")
	}
}

