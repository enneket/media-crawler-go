package api

import (
	"bytes"
	"context"
	"encoding/json"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	_ "media-crawler-go/internal/platform/bilibili"
	_ "media-crawler-go/internal/platform/douyin"
	_ "media-crawler-go/internal/platform/kuaishou"
	_ "media-crawler-go/internal/platform/tieba"
	_ "media-crawler-go/internal/platform/weibo"
	_ "media-crawler-go/internal/platform/xhs"
	_ "media-crawler-go/internal/platform/zhihu"
)

func TestServerRunStopStatus(t *testing.T) {
	config.AppConfig = config.Config{}
	done := make(chan struct{})
	runFn := func(ctx context.Context) (crawler.Result, error) {
		close(done)
		<-ctx.Done()
		return crawler.Result{}, nil
	}

	mgr := NewTaskManagerWithRunner(runFn)
	srv := NewServer(mgr)

	r1 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("healthz code=%d body=%s", w1.Code, w1.Body.String())
	}

	body, _ := json.Marshal(RunRequest{Platform: "xhs", CrawlerType: "search", Keywords: "golang"})
	r2 := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(body))
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, r2)
	if w2.Code != http.StatusAccepted {
		t.Fatalf("run code=%d body=%s", w2.Code, w2.Body.String())
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("runner did not start")
	}

	r3 := httptest.NewRequest(http.MethodGet, "/status", nil)
	w3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w3, r3)
	if w3.Code != http.StatusOK {
		t.Fatalf("status code=%d body=%s", w3.Code, w3.Body.String())
	}

	r4 := httptest.NewRequest(http.MethodPost, "/stop", nil)
	w4 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w4, r4)
	if w4.Code != http.StatusAccepted {
		t.Fatalf("stop code=%d body=%s", w4.Code, w4.Body.String())
	}
}

func TestServerRunValidation(t *testing.T) {
	config.AppConfig = config.Config{}

	{
		config.AppConfig = config.Config{}
		runFn := func(ctx context.Context) (crawler.Result, error) { return crawler.Result{}, nil }
		srv := NewServer(NewTaskManagerWithRunner(runFn))
		body, _ := json.Marshal(RunRequest{Platform: "xhs", CrawlerType: "search"})
		r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got=%d body=%s", w.Code, w.Body.String())
		}
	}
	{
		config.AppConfig = config.Config{}
		runFn := func(ctx context.Context) (crawler.Result, error) { return crawler.Result{}, nil }
		srv := NewServer(NewTaskManagerWithRunner(runFn))
		body, _ := json.Marshal(RunRequest{Platform: "nope", CrawlerType: "detail"})
		r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got=%d body=%s", w.Code, w.Body.String())
		}
	}
	{
		config.AppConfig = config.Config{}
		runFn := func(ctx context.Context) (crawler.Result, error) { return crawler.Result{}, nil }
		srv := NewServer(NewTaskManagerWithRunner(runFn))
		body, _ := json.Marshal(RunRequest{Platform: "bilibili", CrawlerType: "detail"})
		r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got=%d body=%s", w.Code, w.Body.String())
		}
	}
	{
		config.AppConfig = config.Config{}
		runFn := func(ctx context.Context) (crawler.Result, error) { return crawler.Result{}, nil }
		srv := NewServer(NewTaskManagerWithRunner(runFn))
		body, _ := json.Marshal(RunRequest{
			Platform:               "bilibili",
			CrawlerType:            "detail",
			BiliSpecifiedVideoUrls: []string{"BV1Q5411W7bH"},
		})
		r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got=%d body=%s", w.Code, w.Body.String())
		}
	}
	{
		config.AppConfig = config.Config{}
		runFn := func(ctx context.Context) (crawler.Result, error) { return crawler.Result{}, nil }
		srv := NewServer(NewTaskManagerWithRunner(runFn))
		body, _ := json.Marshal(RunRequest{Platform: "tieba", CrawlerType: "detail"})
		r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got=%d body=%s", w.Code, w.Body.String())
		}
	}
	{
		config.AppConfig = config.Config{}
		runFn := func(ctx context.Context) (crawler.Result, error) { return crawler.Result{}, nil }
		srv := NewServer(NewTaskManagerWithRunner(runFn))
		body, _ := json.Marshal(RunRequest{Platform: "tieba", CrawlerType: "detail", TiebaSpecifiedNoteUrls: []string{"https://tieba.baidu.com/p/123"}})
		r := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got=%d body=%s", w.Code, w.Body.String())
		}
	}
}

func TestTaskManagerRunConflict(t *testing.T) {
	var started sync.Once
	block := make(chan struct{})
	runFn := func(ctx context.Context) (crawler.Result, error) {
		started.Do(func() {})
		<-block
		return crawler.Result{}, nil
	}

	m := NewTaskManagerWithRunner(runFn)
	config.AppConfig = config.Config{Platform: "xhs", CrawlerType: "search", Keywords: "golang"}
	if err := m.Run(RunRequest{}); err != nil {
		t.Fatalf("first run err: %v", err)
	}
	if err := m.Run(RunRequest{}); err == nil {
		t.Fatalf("expected conflict error")
	}
	close(block)
}
