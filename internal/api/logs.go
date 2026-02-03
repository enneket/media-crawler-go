package api

import (
	"media-crawler-go/internal/logger"
	"net/http"
	"strconv"
)

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit < 0 {
		limit = 0
	}
	if limit > 2000 {
		limit = 2000
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logger.Recent(limit)})
}

