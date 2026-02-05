package api

import (
	"encoding/json"
	"errors"
	"media-crawler-go/internal/logger"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type pythonCrawlerStartRequest struct {
	Platform          string `json:"platform"`
	LoginType         string `json:"login_type"`
	CrawlerType       string `json:"crawler_type"`
	Keywords          string `json:"keywords"`
	SpecifiedIDs      string `json:"specified_ids"`
	CreatorIDs        string `json:"creator_ids"`
	StartPage         int    `json:"start_page"`
	EnableComments    *bool  `json:"enable_comments"`
	EnableSubComments *bool  `json:"enable_sub_comments"`
	SaveOption        string `json:"save_option"`
	Cookies           string `json:"cookies"`
	Headless          *bool  `json:"headless"`
}

func (s *Server) handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleAPIEnvCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Environment check ok",
		"output":  runtime.Version(),
	})
}

func (s *Server) handleAPIConfigPlatforms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"platforms": []map[string]any{
			{"value": "xhs", "label": "Xiaohongshu", "icon": "book-open"},
			{"value": "dy", "label": "Douyin", "icon": "music"},
			{"value": "ks", "label": "Kuaishou", "icon": "video"},
			{"value": "bili", "label": "Bilibili", "icon": "tv"},
			{"value": "wb", "label": "Weibo", "icon": "message-circle"},
			{"value": "tieba", "label": "Baidu Tieba", "icon": "messages-square"},
			{"value": "zhihu", "label": "Zhihu", "icon": "help-circle"},
		},
	})
}

func (s *Server) handleAPIConfigOptions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"login_types": []map[string]any{
			{"value": "qrcode", "label": "QR Code Login"},
			{"value": "phone", "label": "Phone Login"},
			{"value": "cookie", "label": "Cookie Login"},
		},
		"crawler_types": []map[string]any{
			{"value": "search", "label": "Search Mode"},
			{"value": "detail", "label": "Detail Mode"},
			{"value": "creator", "label": "Creator Mode"},
		},
		"save_options": []map[string]any{
			{"value": "json", "label": "JSON File"},
			{"value": "csv", "label": "CSV File"},
			{"value": "excel", "label": "Excel File"},
			{"value": "sqlite", "label": "SQLite Database"},
			{"value": "db", "label": "MySQL Database"},
			{"value": "mongodb", "label": "MongoDB Database"},
			{"value": "postgres", "label": "PostgreSQL Database"},
		},
	})
}

func (s *Server) handleAPICrawlerStart(w http.ResponseWriter, r *http.Request) {
	var req pythonCrawlerStartRequest
	dec := json.NewDecoder(r.Body)
	_ = dec.Decode(&req)

	runReq := RunRequest{
		Platform:    strings.TrimSpace(req.Platform),
		CrawlerType: strings.TrimSpace(req.CrawlerType),
		Keywords:    strings.TrimSpace(req.Keywords),
		LoginType:   strings.TrimSpace(req.LoginType),
		LoginPhone:  "",
		Cookies:     strings.TrimSpace(req.Cookies),
		Headless:    req.Headless,
	}
	if req.StartPage > 0 {
		v := req.StartPage
		runReq.StartPage = &v
	}
	runReq.EnableComments = req.EnableComments
	runReq.EnableSubComments = req.EnableSubComments

	specified := splitCSV(req.SpecifiedIDs)
	creators := splitCSV(req.CreatorIDs)

	platformKey := strings.ToLower(strings.TrimSpace(req.Platform))
	crawlerType := strings.ToLower(strings.TrimSpace(req.CrawlerType))

	switch platformKey {
	case "xhs":
		switch crawlerType {
		case "detail":
			runReq.XhsSpecifiedNoteUrls = specified
		case "creator":
			runReq.XhsCreatorIdList = creators
		}
	case "dy", "douyin":
		switch crawlerType {
		case "detail":
			runReq.DouyinSpecifiedNoteUrls = specified
		case "creator":
			runReq.DouyinCreatorIdList = creators
		}
	case "ks", "kuaishou":
		switch crawlerType {
		case "detail":
			runReq.KSSpecifiedNoteUrls = specified
		case "creator":
			runReq.KSCreatorUrlList = creators
		}
	case "bili", "bilibili":
		switch crawlerType {
		case "detail":
			runReq.BiliSpecifiedVideoUrls = specified
		case "creator":
			runReq.BiliCreatorIdList = creators
		}
	case "wb", "weibo":
		switch crawlerType {
		case "detail":
			runReq.WBSpecifiedNoteUrls = specified
		case "creator":
			runReq.WBCreatorIdList = creators
		}
	case "tieba":
		switch crawlerType {
		case "detail":
			runReq.TiebaSpecifiedNoteUrls = specified
		case "creator":
			runReq.TiebaCreatorUrlList = creators
		}
	case "zhihu":
		switch crawlerType {
		case "detail":
			runReq.ZhihuSpecifiedNoteUrls = specified
		case "creator":
			runReq.ZhihuCreatorUrlList = creators
		}
	}

	applyPythonSaveOption(&runReq, req.SaveOption)

	if err := s.manager.Run(runReq); err != nil {
		if errors.Is(err, ErrTaskRunning) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "Crawler is already running"})
			return
		}
		var ve ValidationError
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "message": "Crawler started successfully"})
}

func (s *Server) handleAPICrawlerStop(w http.ResponseWriter, r *http.Request) {
	if !s.manager.Stop() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "No crawler is running"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "message": "Crawler stopped successfully"})
}

func (s *Server) handleAPICrawlerStatus(w http.ResponseWriter, r *http.Request) {
	st := s.manager.Status()
	state := strings.ToLower(strings.TrimSpace(st.State))
	status := "idle"
	switch state {
	case "running":
		status = "running"
	case "stopping":
		status = "stopping"
	default:
		if strings.TrimSpace(st.LastError) != "" {
			status = "error"
		}
	}
	started := ""
	if st.StartedAt > 0 {
		started = time.Unix(st.StartedAt, 0).UTC().Format(time.RFC3339)
	}
	resp := map[string]any{
		"status":       status,
		"platform":     st.Platform,
		"crawler_type": st.Crawler,
	}
	if started != "" {
		resp["started_at"] = started
	}
	if st.LastError != "" {
		resp["error_message"] = st.LastError
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAPICrawlerLogs(w http.ResponseWriter, r *http.Request) {
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

	evts := logger.Recent(limit)
	out := make([]map[string]any, 0, len(evts))
	for i, evt := range evts {
		level := strings.ToLower(evt.Level)
		if level == "warn" {
			level = "warning"
		}
		out = append(out, map[string]any{
			"id":        i + 1,
			"timestamp": evt.Time,
			"level":     level,
			"message":   evt.Msg,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": out})
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func applyPythonSaveOption(runReq *RunRequest, saveOption string) {
	if runReq == nil {
		return
	}
	v := strings.ToLower(strings.TrimSpace(saveOption))
	if v == "" {
		return
	}
	switch v {
	case "json", "csv", "excel", "xlsx", "xlsx_book":
		runReq.StoreBackend = "file"
		runReq.SaveDataOption = v
	case "sqlite":
		runReq.StoreBackend = "sqlite"
		if strings.TrimSpace(runReq.SaveDataOption) == "" {
			runReq.SaveDataOption = "json"
		}
	case "mongodb":
		runReq.StoreBackend = "mongodb"
		if strings.TrimSpace(runReq.SaveDataOption) == "" {
			runReq.SaveDataOption = "json"
		}
	case "db", "mysql":
		runReq.StoreBackend = "mysql"
		if strings.TrimSpace(runReq.SaveDataOption) == "" {
			runReq.SaveDataOption = "json"
		}
	case "postgres", "postgresql":
		runReq.StoreBackend = "postgres"
		if strings.TrimSpace(runReq.SaveDataOption) == "" {
			runReq.SaveDataOption = "json"
		}
	}
}

