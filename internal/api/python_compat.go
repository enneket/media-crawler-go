package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"media-crawler-go/internal/browser"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/logger"
	"net/http"
	"os"
	"path/filepath"
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
	rep := envReportFromConfig()
	if rep.OK {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "MediaCrawler environment configured correctly",
			"output":  rep.Summary(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": false,
		"message": "Environment check failed",
		"error":   rep.Summary(),
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

type envReport struct {
	Generated      string `json:"generated"`
	GoVersion      string `json:"go_version"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	DataDir        string `json:"data_dir"`
	DataDirOK      bool   `json:"data_dir_ok"`
	DataDirError   string `json:"data_dir_error,omitempty"`
	ChromePath     string `json:"chrome_path,omitempty"`
	ChromeOK       bool   `json:"chrome_ok"`
	ChromeError    string `json:"chrome_error,omitempty"`
	CDPEndpoint    string `json:"cdp_endpoint,omitempty"`
	CDPReachable   bool   `json:"cdp_reachable"`
	CDPError       string `json:"cdp_error,omitempty"`
	Notes          []string
	OK             bool `json:"ok"`
}

func envReportFromConfig() envReport {
	rep := envReport{
		Generated: time.Now().UTC().Format(time.RFC3339Nano),
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		DataDir:   strings.TrimSpace(configDataDir()),
	}

	if rep.DataDir == "" {
		rep.DataDir = "data"
	}
	if err := ensureWritableDir(rep.DataDir); err != nil {
		rep.DataDirOK = false
		rep.DataDirError = err.Error()
	} else {
		rep.DataDirOK = true
	}

	chrome, err := browserDetect(rep)
	if err != nil {
		rep.ChromeOK = false
		rep.ChromeError = err.Error()
	} else {
		rep.ChromeOK = true
		rep.ChromePath = chrome
	}

	rep.CDPEndpoint = cdpEndpointFromConfig()
	if rep.CDPEndpoint != "" {
		ok, err := httpGetOK(rep.CDPEndpoint+"/json/version", 800*time.Millisecond)
		if err != nil {
			rep.CDPReachable = false
			rep.CDPError = err.Error()
		} else {
			rep.CDPReachable = ok
		}
	}

	rep.OK = rep.DataDirOK
	if !rep.ChromeOK {
		rep.Notes = append(rep.Notes, "Chrome/Chromium not found: set CUSTOM_BROWSER_PATH or CHROME_PATH (needed for CDP mode)")
	}
	if rep.CDPEndpoint != "" && !rep.CDPReachable {
		rep.Notes = append(rep.Notes, "CDP endpoint not reachable: start Chrome with --remote-debugging-port or disable ENABLE_CDP_MODE")
	}
	return rep
}

func (r envReport) Summary() string {
	parts := []string{
		fmt.Sprintf("go=%s %s/%s", r.GoVersion, r.OS, r.Arch),
		fmt.Sprintf("data_dir=%s ok=%v", r.DataDir, r.DataDirOK),
	}
	if r.ChromePath != "" {
		parts = append(parts, fmt.Sprintf("chrome=%s ok=%v", r.ChromePath, r.ChromeOK))
	} else {
		parts = append(parts, fmt.Sprintf("chrome ok=%v", r.ChromeOK))
	}
	if r.CDPEndpoint != "" {
		parts = append(parts, fmt.Sprintf("cdp=%s ok=%v", r.CDPEndpoint, r.CDPReachable))
	}
	if len(r.Notes) > 0 {
		parts = append(parts, "notes="+strings.Join(r.Notes, "; "))
	}
	if r.DataDirError != "" {
		parts = append(parts, "data_dir_error="+r.DataDirError)
	}
	if r.ChromeError != "" {
		parts = append(parts, "chrome_error="+r.ChromeError)
	}
	if r.CDPError != "" {
		parts = append(parts, "cdp_error="+r.CDPError)
	}
	return strings.Join(parts, "\n")
}

func ensureWritableDir(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return err
	}
	fp := filepath.Join(abs, ".write_test")
	if err := os.WriteFile(fp, []byte("ok"), 0644); err != nil {
		return err
	}
	_ = os.Remove(fp)
	return nil
}

func configDataDir() string {
	return strings.TrimSpace(config.AppConfig.DataDir)
}

func cdpEndpointFromConfig() string {
	if !config.AppConfig.EnableCDPMode || config.AppConfig.CDPDebugPort <= 0 {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d", config.AppConfig.CDPDebugPort)
}

func httpGetOK(url string, timeout time.Duration) (bool, error) {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}

func browserDetect(rep envReport) (string, error) {
	if rep.OS == "" {
		return "", fmt.Errorf("unknown runtime OS")
	}
	return browser.DetectBinary(config.AppConfig.CustomBrowserPath)
}
