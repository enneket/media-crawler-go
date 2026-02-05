package api

import (
	"bytes"
	"context"
	"encoding/json"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/sms"
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

func TestServerLogsEndpoint(t *testing.T) {
	config.AppConfig = config.Config{LogLevel: "info", LogFormat: "json"}
	logger.InitFromConfig()
	logger.Info("unit-test-logs-endpoint", "k", "v")

	srv := NewServer(NewTaskManagerWithRunner(func(ctx context.Context) (crawler.Result, error) {
		return crawler.Result{}, nil
	}))

	for _, path := range []string{"/logs?limit=2000", "/crawler/logs?limit=2000"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("logs code=%d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Logs []map[string]any `json:"logs"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal logs err: %v body=%s", err, w.Body.String())
		}
		found := false
		for _, it := range resp.Logs {
			if it["msg"] == "unit-test-logs-endpoint" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected log not found for path=%s, got=%d logs", path, len(resp.Logs))
		}
	}
}

func TestPythonCompatAPIEndpoints(t *testing.T) {
	config.AppConfig = config.Config{LogLevel: "info", LogFormat: "json"}
	logger.InitFromConfig()
	logger.Info("unit-test-python-compat-logs", "k", "v")

	done := make(chan struct{})
	runFn := func(ctx context.Context) (crawler.Result, error) {
		close(done)
		<-ctx.Done()
		return crawler.Result{}, nil
	}
	srv := NewServer(NewTaskManagerWithRunner(runFn))

	{
		r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("api health code=%d body=%s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal api health err: %v", err)
		}
		if resp["status"] != "ok" {
			t.Fatalf("unexpected api health resp: %v", resp)
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/api/crawler/logs?limit=2000", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("api crawler logs code=%d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Logs []map[string]any `json:"logs"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal api crawler logs err: %v body=%s", err, w.Body.String())
		}
		found := false
		for _, it := range resp.Logs {
			if it["message"] == "unit-test-python-compat-logs" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected python compat log not found, got=%d logs", len(resp.Logs))
		}
	}

	{
		body, _ := json.Marshal(map[string]any{
			"platform":     "xhs",
			"crawler_type": "search",
			"keywords":     "golang",
		})
		r := httptest.NewRequest(http.MethodPost, "/api/crawler/start", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("api crawler start code=%d body=%s", w.Code, w.Body.String())
		}
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("python compat runner did not start")
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/api/crawler/status", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("api crawler status code=%d body=%s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal api crawler status err: %v", err)
		}
		if resp["status"] != "running" {
			t.Fatalf("expected status=running, got=%v", resp)
		}
	}

	{
		r := httptest.NewRequest(http.MethodPost, "/api/crawler/stop", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("api crawler stop code=%d body=%s", w.Code, w.Body.String())
		}
	}
}

func TestEnvCheckEndpoints(t *testing.T) {
	config.AppConfig = config.Config{}
	srv := NewServer(NewTaskManagerWithRunner(func(ctx context.Context) (crawler.Result, error) {
		return crawler.Result{}, nil
	}))

	{
		r := httptest.NewRequest(http.MethodGet, "/env/check", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("env check code=%d body=%s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal env check err: %v body=%s", err, w.Body.String())
		}
		if _, ok := resp["data_dir"]; !ok {
			t.Fatalf("env check missing data_dir: %v", resp)
		}
		if _, ok := resp["data_dir_ok"]; !ok {
			t.Fatalf("env check missing data_dir_ok: %v", resp)
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/api/env/check", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("api env check code=%d body=%s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal api env check err: %v body=%s", err, w.Body.String())
		}
		if resp["success"] != true {
			t.Fatalf("expected success=true, got: %v", resp)
		}
	}
}

func TestSMSEndpointStoresCode(t *testing.T) {
	config.AppConfig = config.Config{CacheBackend: "memory"}
	srv := NewServer(NewTaskManagerWithRunner(func(ctx context.Context) (crawler.Result, error) {
		return crawler.Result{}, nil
	}))

	body, _ := json.Marshal(map[string]any{
		"platform":        "xhs",
		"current_number":  "13152442222",
		"sms_content":     "【小红书】您的验证码是: 171959， 3分钟内有效。",
		"timestamp":       "0",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/sms", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("sms code=%d body=%s", w.Code, w.Body.String())
	}

	code, ok := sms.Pop("xhs", "13152442222")
	if !ok || code != "171959" {
		t.Fatalf("expected code 171959, got ok=%v code=%q", ok, code)
	}
}
