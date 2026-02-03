package api

import (
	"context"
	"encoding/json"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebUIAndConfigEndpoints(t *testing.T) {
	config.AppConfig = config.Config{Platform: "xhs"}
	runFn := func(ctx context.Context) (crawler.Result, error) { return crawler.Result{}, nil }
	srv := NewServer(NewTaskManagerWithRunner(runFn))

	{
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("index code=%d body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(strings.ToLower(w.Header().Get("content-type")), "text/html") {
			t.Fatalf("unexpected content-type=%q", w.Header().Get("content-type"))
		}
		if !strings.Contains(w.Body.String(), "media-crawler-go") {
			t.Fatalf("index body missing title")
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/config/platforms", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("platforms code=%d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Platforms []platformInfo `json:"platforms"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode platforms: %v body=%s", err, w.Body.String())
		}
		if len(resp.Platforms) == 0 {
			t.Fatalf("empty platforms")
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/config/options", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("options code=%d body=%s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode options: %v body=%s", err, w.Body.String())
		}
		if _, ok := resp["defaults"]; !ok {
			t.Fatalf("missing defaults")
		}
		if v, ok := resp["store_backends"].([]any); ok {
			found := false
			for _, it := range v {
				if s, _ := it.(string); s == "mongodb" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("store_backends missing mongodb: %v", v)
			}
		} else {
			t.Fatalf("store_backends missing or invalid")
		}
		if d, ok := resp["defaults"].(map[string]any); ok {
			if _, ok := d["mongo_db"]; !ok {
				t.Fatalf("defaults missing mongo_db")
			}
		} else {
			t.Fatalf("defaults invalid")
		}
	}

	{
		r := httptest.NewRequest(http.MethodGet, "/env/check", nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("env code=%d body=%s", w.Code, w.Body.String())
		}
		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode env: %v body=%s", err, w.Body.String())
		}
		if ok, _ := resp["ok"].(bool); !ok {
			t.Fatalf("expected ok=true, got=%v", resp["ok"])
		}
	}
}
