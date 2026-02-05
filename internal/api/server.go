package api

import (
	"encoding/json"
	"errors"
	"io"
	"media-crawler-go/internal/cache"
	"media-crawler-go/internal/config"
	"net/http"
	"time"
)

type Server struct {
	manager *TaskManager
	mux     *http.ServeMux
	cache   cache.Cache
}

func NewServer(manager *TaskManager) *Server {
	if manager == nil {
		manager = NewTaskManager()
	}
	s := &Server{
		manager: manager,
		mux:     http.NewServeMux(),
		cache:   cache.NewFromConfig(config.AppConfig),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /api/health", s.handleAPIHealth)
	s.mux.HandleFunc("GET /status", s.handleStatus)
	s.mux.HandleFunc("POST /run", s.handleRun)
	s.mux.HandleFunc("POST /stop", s.handleStop)
	s.mux.HandleFunc("GET /logs", s.handleLogs)
	s.mux.HandleFunc("GET /crawler/logs", s.handleLogs)
	s.mux.HandleFunc("GET /api/logs", s.handleLogs)
	s.mux.HandleFunc("GET /api/crawler/logs", s.handleAPICrawlerLogs)
	s.mux.HandleFunc("POST /api/crawler/start", s.handleAPICrawlerStart)
	s.mux.HandleFunc("POST /api/crawler/stop", s.handleAPICrawlerStop)
	s.mux.HandleFunc("GET /api/crawler/status", s.handleAPICrawlerStatus)
	s.mux.HandleFunc("GET /config/platforms", s.handleConfigPlatforms)
	s.mux.HandleFunc("GET /config/options", s.handleConfigOptions)
	s.mux.HandleFunc("GET /env/check", s.handleEnvCheck)
	s.mux.HandleFunc("GET /api/env/check", s.handleAPIEnvCheck)
	s.mux.HandleFunc("GET /api/config/platforms", s.handleAPIConfigPlatforms)
	s.mux.HandleFunc("GET /api/config/options", s.handleAPIConfigOptions)
	s.mux.HandleFunc("GET /data/files", s.handleDataFilesList)
	s.mux.HandleFunc("GET /data/files/", s.handleDataFile)
	s.mux.HandleFunc("GET /data/download/", s.handleDataDownload)
	s.mux.HandleFunc("GET /data/stats", s.handleDataStats)
	s.mux.HandleFunc("GET /data/wordcloud", s.handleDataWordcloud)
	s.mux.HandleFunc("GET /ws/logs", s.handleWSLogs)
	s.mux.HandleFunc("GET /ws/status", s.handleWSStatus)
	s.mux.HandleFunc("GET /api/data/files", s.handleDataFilesList)
	s.mux.HandleFunc("GET /api/data/files/", s.handleDataFile)
	s.mux.HandleFunc("GET /api/data/download/", s.handleDataDownload)
	s.mux.HandleFunc("GET /api/data/stats", s.handleDataStats)
	s.mux.HandleFunc("GET /api/data/wordcloud", s.handleDataWordcloud)
	s.mux.HandleFunc("GET /api/ws/logs", s.handleWSLogs)
	s.mux.HandleFunc("GET /api/ws/status", s.handleWSStatus)
	s.mux.Handle("GET /assets/", http.StripPrefix("/assets/", s.webUIAssetsHandler()))
	s.mux.HandleFunc("GET /", s.handleWebUIIndex)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.manager.Status())
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	if err := s.manager.Run(req); err != nil {
		if errors.Is(err, ErrTaskRunning) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
			return
		}
		var ve ValidationError
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, s.manager.Status())
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	stopped := s.manager.Stop()
	writeJSON(w, http.StatusAccepted, map[string]any{"stopped": stopped})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func nowUnix() int64 {
	return time.Now().Unix()
}
