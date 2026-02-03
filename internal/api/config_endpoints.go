package api

import (
	"media-crawler-go/internal/config"
	"net/http"
	"runtime"
	"time"
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
		"store_backends":   []string{"file", "sqlite"},
		"save_data_option": []string{"json", "csv", "xlsx"},
		"cache_backends":   []string{"memory", "redis", "none"},
		"bili_search_mode": []string{"video"},
		"wb_search_type":   []string{"1", "61", "60", "64"},
		"defaults": map[string]any{
			"platform":          config.AppConfig.Platform,
			"crawler_type":      config.AppConfig.CrawlerType,
			"keywords":          config.AppConfig.Keywords,
			"login_type":        config.AppConfig.LoginType,
			"store_backend":     config.AppConfig.StoreBackend,
			"sqlite_path":       config.AppConfig.SQLitePath,
			"cache_backend":     config.AppConfig.CacheBackend,
			"cache_ttl_sec":     config.AppConfig.CacheDefaultTTLSec,
			"redis_addr":        config.AppConfig.RedisAddr,
			"redis_db":          config.AppConfig.RedisDB,
			"redis_key_prefix":  config.AppConfig.RedisKeyPrefix,
			"save_data_option":  config.AppConfig.SaveDataOption,
			"enable_ip_proxy":   config.AppConfig.EnableIPProxy,
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
	now := time.Now().UTC()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"generated":  now.Format(time.RFC3339Nano),
		"go_version": runtime.Version(),
	})
}
