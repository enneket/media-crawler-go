package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestServerRunStopStatus(t *testing.T) {
	done := make(chan struct{})
	runFn := func(ctx context.Context) error {
		close(done)
		<-ctx.Done()
		return nil
	}

	mgr := NewTaskManagerWithRunner(runFn)
	srv := NewServer(mgr)

	r1 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("healthz code=%d body=%s", w1.Code, w1.Body.String())
	}

	body, _ := json.Marshal(RunRequest{Platform: "xhs", CrawlerType: "search"})
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

func TestTaskManagerRunConflict(t *testing.T) {
	var started sync.Once
	block := make(chan struct{})
	runFn := func(ctx context.Context) error {
		started.Do(func() {})
		<-block
		return nil
	}

	m := NewTaskManagerWithRunner(runFn)
	if err := m.Run(RunRequest{}); err != nil {
		t.Fatalf("first run err: %v", err)
	}
	if err := m.Run(RunRequest{}); err == nil {
		t.Fatalf("expected conflict error")
	}
	close(block)
}
