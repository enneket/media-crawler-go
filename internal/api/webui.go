package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed webui/*
var webuiFS embed.FS

func (s *Server) handleWebUIIndex(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimSpace(r.URL.Path)
	if p != "" && p != "/" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	sub, err := fs.Sub(webuiFS, "webui")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	b, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	w.Header().Set("content-type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}

func (s *Server) webUIAssetsHandler() http.Handler {
	sub, err := fs.Sub(webuiFS, "webui")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		})
	}
	return http.FileServer(http.FS(sub))
}
