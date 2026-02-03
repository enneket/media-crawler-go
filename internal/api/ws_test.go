package api

import (
	"context"
	"encoding/json"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/logger"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

func TestWebSocketLogsAndStatus(t *testing.T) {
	config.AppConfig = config.Config{LogLevel: "debug", LogFormat: "json"}
	logger.InitFromConfig()

	runFn := func(ctx context.Context) (crawler.Result, error) { return crawler.Result{}, nil }
	srv := NewServer(NewTaskManagerWithRunner(runFn))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	wsBase := "ws" + strings.TrimPrefix(ts.URL, "http")

	{
		conn, err := websocket.Dial(wsBase+"/ws/logs", "", ts.URL)
		if err != nil {
			t.Fatalf("dial logs: %v", err)
		}
		defer conn.Close()

		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		want := "ws_test_log_123"
		logger.Info(want, "k", "v")

		var msg string
		if err := websocket.Message.Receive(conn, &msg); err != nil {
			t.Fatalf("recv logs: %v", err)
		}
		if !strings.Contains(msg, want) {
			t.Fatalf("unexpected log msg=%q", msg)
		}
	}

	{
		conn, err := websocket.Dial(wsBase+"/ws/status?interval_ms=100", "", ts.URL)
		if err != nil {
			t.Fatalf("dial status: %v", err)
		}
		defer conn.Close()

		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		var msg string
		if err := websocket.Message.Receive(conn, &msg); err != nil {
			t.Fatalf("recv status: %v", err)
		}
		var st Status
		if err := json.Unmarshal([]byte(strings.TrimSpace(msg)), &st); err != nil {
			t.Fatalf("unmarshal status: %v msg=%q", err, msg)
		}
		if st.State == "" {
			t.Fatalf("missing state: %+v", st)
		}
	}
}

