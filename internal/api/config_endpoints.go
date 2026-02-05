package api

import (
	"media-crawler-go/internal/config"
	"net/http"
)

type platformInfo struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Modes []string `json:"modes"`
}

func (s *Server) handleConfigPlatforms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"platforms": []platformInfo{
			{Key: "xhs", Label: "小红书", Modes: []string{"search", "detail", "creator"}},
			{Key: "douyin", Label: "抖音", Modes: []string{"search", "detail", "creator"}},
			{Key: "bilibili", Label: "Bilibili", Modes: []string{"search", "detail", "creator"}},
			{Key: "weibo", Label: "微博", Modes: []string{"search", "detail", "creator"}},
			{Key: "tieba", Label: "贴吧", Modes: []string{"search", "detail", "creator"}},
			{Key: "zhihu", Label: "知乎", Modes: []string{"search", "detail", "creator"}},
			{Key: "kuaishou", Label: "快手", Modes: []string{"search", "detail", "creator"}},
		},
	})
}

func (s *Server) handleConfigOptions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"crawler_types":    []string{"search", "detail", "creator"},
		"login_types":      []string{"qrcode", "phone", "cookie"},
		"store_backends":   []string{"file", "sqlite", "mysql", "postgres", "mongodb"},
		"save_data_option": []string{"json", "csv", "xlsx", "xlsx_book", "excel"},
		"descriptions": map[string]any{
			"store_backend": map[string]string{
				"file":    "不写入数据库，仅文件落盘",
				"sqlite":  "写入 SQLite（同时文件落盘）",
				"mysql":   "写入 MySQL（同时文件落盘）",
				"postgres": "写入 Postgres（同时文件落盘）",
				"mongodb": "写入 MongoDB（同时文件落盘）",
			},
			"save_data_option": map[string]string{
				"json":      "文件落盘为 json/jsonl（默认）",
				"csv":       "文件落盘为 csv",
				"xlsx":      "文件落盘为 xlsx（按 note 分文件）",
				"xlsx_book": "文件落盘为单个 workbook（Contents/Comments/Creators 分 Sheet）",
				"excel":     "兼容 Python：会被规范化为 xlsx_book",
			},
		},
		"proxy_providers":  []string{"kuaidaili", "wandouhttp", "jisuhttp", "jishuhttp", "jishu_http", "static"},
		"cache_backends":   []string{"memory", "redis", "none"},
		"bili_search_mode": []string{"video"},
		"wb_search_type":   []string{"1", "61", "60", "64"},
		"defaults": map[string]any{
			"platform":          config.AppConfig.Platform,
			"crawler_type":      config.AppConfig.CrawlerType,
			"keywords":          config.AppConfig.Keywords,
			"login_type":        config.AppConfig.LoginType,
			"login_phone":       config.AppConfig.LoginPhone,
			"store_backend":     config.AppConfig.StoreBackend,
			"sqlite_path":       config.AppConfig.SQLitePath,
			"mysql_dsn":         "",
			"postgres_dsn":      "",
			"mongo_uri":         "",
			"mongo_db":          config.AppConfig.MongoDB,
			"cache_backend":     config.AppConfig.CacheBackend,
			"cache_ttl_sec":     config.AppConfig.CacheDefaultTTLSec,
			"redis_addr":        config.AppConfig.RedisAddr,
			"redis_db":          config.AppConfig.RedisDB,
			"redis_key_prefix":  config.AppConfig.RedisKeyPrefix,
			"save_data_option":  config.AppConfig.SaveDataOption,
			"enable_ip_proxy":   config.AppConfig.EnableIPProxy,
			"ip_proxy_provider": config.AppConfig.IPProxyProviderName,
			"max_concurrency":   config.AppConfig.MaxConcurrencyNum,
			"enable_comments":   config.AppConfig.EnableGetComments,
			"enable_subcomment": config.AppConfig.EnableGetSubComments,
			"enable_medias":     config.AppConfig.EnableGetMedias,
			"bili_search_mode":  config.AppConfig.BiliSearchMode,
			"wb_search_type":    config.AppConfig.WBSearchType,
		},
	})
}

func (s *Server) handleEnvCheck(w http.ResponseWriter, r *http.Request) {
	rep := envReportFromConfig()
	writeJSON(w, http.StatusOK, rep)
}
