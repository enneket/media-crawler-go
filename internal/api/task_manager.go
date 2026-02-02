package api

import (
	"context"
	"errors"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/platform"
	"strings"
	"sync"
	"time"
)

type Status struct {
	State      string `json:"state"`
	Platform   string `json:"platform,omitempty"`
	Crawler    string `json:"crawler_type,omitempty"`
	StartedAt  int64  `json:"started_at,omitempty"`
	FinishedAt int64  `json:"finished_at,omitempty"`
	Processed  int    `json:"processed,omitempty"`
	Succeeded  int    `json:"succeeded,omitempty"`
	Failed     int    `json:"failed,omitempty"`
	LastError  string `json:"last_error,omitempty"`
}

type RunRequest struct {
	Platform    string `json:"platform,omitempty"`
	CrawlerType string `json:"crawler_type,omitempty"`
	Keywords    string `json:"keywords,omitempty"`

	XhsSpecifiedNoteUrls []string `json:"xhs_specified_note_url_list,omitempty"`
	XhsCreatorIdList     []string `json:"xhs_creator_id_list,omitempty"`

	DouyinSpecifiedNoteUrls []string `json:"dy_specified_note_url_list,omitempty"`
	DouyinCreatorIdList     []string `json:"dy_creator_id_list,omitempty"`

	BiliSpecifiedVideoUrls []string `json:"bili_specified_video_url_list,omitempty"`
	WBSpecifiedNoteUrls    []string `json:"wb_specified_note_url_list,omitempty"`

	StoreBackend   string `json:"store_backend,omitempty"`
	SQLitePath     string `json:"sqlite_path,omitempty"`
	SaveDataOption string `json:"save_data_option,omitempty"`
}

type TaskManager struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	status Status
	runFn  func(context.Context) (crawler.Result, error)
}

var ErrTaskRunning = errors.New("task is running")

type ValidationError struct {
	Msg string
}

func (e ValidationError) Error() string {
	return e.Msg
}

func NewTaskManager() *TaskManager {
	return NewTaskManagerWithRunner(runCrawler)
}

func NewTaskManagerWithRunner(runFn func(context.Context) (crawler.Result, error)) *TaskManager {
	if runFn == nil {
		runFn = runCrawler
	}
	return &TaskManager{status: Status{State: "idle"}, runFn: runFn}
}

func (m *TaskManager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *TaskManager) Run(req RunRequest) error {
	m.mu.Lock()
	if m.cancel != nil {
		m.mu.Unlock()
		return ErrTaskRunning
	}

	nextCfg := config.AppConfig
	applyRunRequestToConfig(&nextCfg, req)
	if err := validateRunConfig(nextCfg); err != nil {
		m.mu.Unlock()
		return err
	}
	config.AppConfig = nextCfg

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.status = Status{
		State:     "running",
		Platform:  config.AppConfig.Platform,
		Crawler:   config.AppConfig.CrawlerType,
		StartedAt: time.Now().Unix(),
	}
	m.mu.Unlock()

	go func() {
		res, err := m.runFn(ctx)
		m.mu.Lock()
		defer m.mu.Unlock()
		m.cancel = nil
		m.status.State = "idle"
		m.status.FinishedAt = time.Now().Unix()
		m.status.Processed = res.Processed
		m.status.Succeeded = res.Succeeded
		m.status.Failed = res.Failed
		if err != nil {
			m.status.LastError = err.Error()
		} else {
			m.status.LastError = ""
		}
	}()
	return nil
}

func (m *TaskManager) Stop() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel == nil {
		return false
	}
	m.cancel()
	m.status.State = "stopping"
	return true
}

func runCrawler(ctx context.Context) (crawler.Result, error) {
	r, err := platform.New(config.AppConfig.Platform)
	if err != nil {
		return crawler.Result{}, err
	}
	req := crawler.RequestFromConfig(config.AppConfig)
	return r.Run(ctx, req)
}

func applyRunRequestToConfig(cfg *config.Config, req RunRequest) {
	if cfg == nil {
		return
	}
	if v := strings.TrimSpace(req.Platform); v != "" {
		cfg.Platform = v
	}
	if v := strings.TrimSpace(req.CrawlerType); v != "" {
		cfg.CrawlerType = v
	}
	if v := strings.TrimSpace(req.Keywords); v != "" {
		cfg.Keywords = v
	}
	if len(req.XhsSpecifiedNoteUrls) > 0 {
		cfg.XhsSpecifiedNoteUrls = req.XhsSpecifiedNoteUrls
	}
	if len(req.XhsCreatorIdList) > 0 {
		cfg.XhsCreatorIdList = req.XhsCreatorIdList
	}
	if len(req.DouyinSpecifiedNoteUrls) > 0 {
		cfg.DouyinSpecifiedNoteUrls = req.DouyinSpecifiedNoteUrls
	}
	if len(req.DouyinCreatorIdList) > 0 {
		cfg.DouyinCreatorIdList = req.DouyinCreatorIdList
	}
	if len(req.BiliSpecifiedVideoUrls) > 0 {
		cfg.BiliSpecifiedVideoUrls = req.BiliSpecifiedVideoUrls
	}
	if len(req.WBSpecifiedNoteUrls) > 0 {
		cfg.WBSpecifiedNoteUrls = req.WBSpecifiedNoteUrls
	}
	if v := strings.TrimSpace(req.StoreBackend); v != "" {
		cfg.StoreBackend = v
	}
	if v := strings.TrimSpace(req.SQLitePath); v != "" {
		cfg.SQLitePath = v
	}
	if v := strings.TrimSpace(req.SaveDataOption); v != "" {
		cfg.SaveDataOption = v
	}
}

func validateRunConfig(cfg config.Config) error {
	platformName := strings.TrimSpace(cfg.Platform)
	if platformName == "" {
		return ValidationError{Msg: "platform is required"}
	}
	if !platform.Exists(platformName) {
		return ValidationError{Msg: fmt.Sprintf("unknown platform: %s", platformName)}
	}

	crawlerType := strings.ToLower(strings.TrimSpace(cfg.CrawlerType))
	if crawlerType == "" {
		crawlerType = "search"
	}

	if v := strings.ToLower(strings.TrimSpace(cfg.StoreBackend)); v != "" && v != "file" && v != "sqlite" {
		return ValidationError{Msg: fmt.Sprintf("invalid store_backend: %s", cfg.StoreBackend)}
	}
	if v := strings.ToLower(strings.TrimSpace(cfg.SaveDataOption)); v != "" && v != "json" && v != "csv" {
		return ValidationError{Msg: fmt.Sprintf("invalid save_data_option: %s", cfg.SaveDataOption)}
	}

	p := strings.ToLower(strings.TrimSpace(platformName))
	switch p {
	case "xhs":
		switch crawlerType {
		case "search":
			if strings.TrimSpace(cfg.Keywords) == "" {
				return ValidationError{Msg: "keywords is required for search"}
			}
		case "detail":
			if len(cfg.XhsSpecifiedNoteUrls) == 0 {
				return ValidationError{Msg: "xhs_specified_note_url_list is required for detail"}
			}
		case "creator":
			if len(cfg.XhsCreatorIdList) == 0 {
				return ValidationError{Msg: "xhs_creator_id_list is required for creator"}
			}
		default:
			return ValidationError{Msg: fmt.Sprintf("unsupported crawler_type for xhs: %s", crawlerType)}
		}
	case "douyin", "dy":
		switch crawlerType {
		case "search":
			if strings.TrimSpace(cfg.Keywords) == "" {
				return ValidationError{Msg: "keywords is required for search"}
			}
		case "detail":
			if len(cfg.DouyinSpecifiedNoteUrls) == 0 {
				return ValidationError{Msg: "dy_specified_note_url_list is required for detail"}
			}
		case "creator":
			if len(cfg.DouyinCreatorIdList) == 0 {
				return ValidationError{Msg: "dy_creator_id_list is required for creator"}
			}
		default:
			return ValidationError{Msg: fmt.Sprintf("unsupported crawler_type for douyin: %s", crawlerType)}
		}
	case "bilibili", "bili", "b站", "b":
		if crawlerType != "detail" {
			return ValidationError{Msg: "bilibili only supports crawler_type=detail"}
		}
		if len(cfg.BiliSpecifiedVideoUrls) == 0 {
			return ValidationError{Msg: "bili_specified_video_url_list is required for detail"}
		}
	case "weibo", "wb", "微博":
		if crawlerType != "detail" {
			return ValidationError{Msg: "weibo only supports crawler_type=detail"}
		}
		if len(cfg.WBSpecifiedNoteUrls) == 0 {
			return ValidationError{Msg: "wb_specified_note_url_list is required for detail"}
		}
	default:
		if crawlerType == "search" && strings.TrimSpace(cfg.Keywords) == "" {
			return ValidationError{Msg: "keywords is required for search"}
		}
	}
	return nil
}
