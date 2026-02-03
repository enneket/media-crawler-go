package api

import (
	"encoding/json"
	"media-crawler-go/internal/logger"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/websocket"
)

func (s *Server) handleWSLogs(w http.ResponseWriter, r *http.Request) {
	websocket.Server{
		Handshake: func(cfg *websocket.Config, req *http.Request) error { return nil },
		Handler: func(conn *websocket.Conn) {
			conn.PayloadType = websocket.TextFrame
			ch, cancel := logger.Subscribe()
			defer cancel()

			for msg := range ch {
				if err := websocket.Message.Send(conn, string(msg)); err != nil {
					return
				}
			}
		},
	}.ServeHTTP(w, r)
}

func (s *Server) handleWSStatus(w http.ResponseWriter, r *http.Request) {
	interval := time.Second
	if v := stringsTrimSpace(r.URL.Query().Get("interval_ms")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < 100 {
				n = 100
			}
			if n > 5000 {
				n = 5000
			}
			interval = time.Duration(n) * time.Millisecond
		}
	}

	websocket.Server{
		Handshake: func(cfg *websocket.Config, req *http.Request) error { return nil },
		Handler: func(conn *websocket.Conn) {
			conn.PayloadType = websocket.TextFrame

			send := func() bool {
				st := s.manager.Status()
				b, err := json.Marshal(st)
				if err != nil {
					return false
				}
				b = append(b, '\n')
				return websocket.Message.Send(conn, string(b)) == nil
			}

			if !send() {
				return
			}
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for range ticker.C {
				if !send() {
					return
				}
			}
		},
	}.ServeHTTP(w, r)
}

func stringsTrimSpace(s string) string {
	i := 0
	j := len(s)
	for i < j {
		if s[i] != ' ' && s[i] != '\n' && s[i] != '\t' && s[i] != '\r' {
			break
		}
		i++
	}
	for j > i {
		c := s[j-1]
		if c != ' ' && c != '\n' && c != '\t' && c != '\r' {
			break
		}
		j--
	}
	return s[i:j]
}

