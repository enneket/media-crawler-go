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
	State          string         `json:"state"`
	Platform       string         `json:"platform,omitempty"`
	Crawler        string         `json:"crawler_type,omitempty"`
	StartedAt      int64          `json:"started_at,omitempty"`
	FinishedAt     int64          `json:"finished_at,omitempty"`
	Processed      int            `json:"processed,omitempty"`
	Succeeded      int            `json:"succeeded,omitempty"`
	Failed         int            `json:"failed,omitempty"`
	FailureKinds   map[string]int `json:"failure_kinds,omitempty"`
	LastError      string         `json:"last_error,omitempty"`
	LastErrorKind  string         `json:"last_error_kind,omitempty"`
	LastRiskHint   string         `json:"last_risk_hint,omitempty"`
	LastErrorURL   string         `json:"last_error_url,omitempty"`
	LastHTTPStatus int            `json:"last_http_status,omitempty"`
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
	BiliCreatorIdList      []string `json:"bili_creator_id_list,omitempty"`
	WBSpecifiedNoteUrls    []string `json:"wb_specified_note_url_list,omitempty"`
	WBCreatorIdList        []string `json:"wb_creator_id_list,omitempty"`

	TiebaSpecifiedNoteUrls []string `json:"tieba_specified_note_url_list,omitempty"`
	TiebaCreatorUrlList    []string `json:"tieba_creator_url_list,omitempty"`
	ZhihuSpecifiedNoteUrls []string `json:"zhihu_specified_note_url_list,omitempty"`
	KSSpecifiedNoteUrls    []string `json:"ks_specified_note_url_list,omitempty"`

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
		m.status.FailureKinds = res.FailureKinds
		if err != nil {
			m.status.LastError = err.Error()
			m.status.LastErrorKind = string(crawler.KindOf(err))
			var ce crawler.Error
			if errors.As(err, &ce) {
				m.status.LastErrorURL = ce.URL
				m.status.LastHTTPStatus = ce.StatusCode
				if ce.Kind == crawler.ErrorKindRiskHint {
					m.status.LastRiskHint = ce.Hint
					if m.status.LastRiskHint == "" {
						m.status.LastRiskHint = ce.Msg
					}
				}
			}
		} else {
			m.status.LastError = ""
			m.status.LastErrorKind = ""
			m.status.LastRiskHint = ""
			m.status.LastErrorURL = ""
			m.status.LastHTTPStatus = 0
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
	if len(req.BiliCreatorIdList) > 0 {
		cfg.BiliCreatorIdList = req.BiliCreatorIdList
	}
	if len(req.WBSpecifiedNoteUrls) > 0 {
		cfg.WBSpecifiedNoteUrls = req.WBSpecifiedNoteUrls
	}
	if len(req.WBCreatorIdList) > 0 {
		cfg.WBCreatorIdList = req.WBCreatorIdList
	}
	if len(req.TiebaSpecifiedNoteUrls) > 0 {
		cfg.TiebaSpecifiedNoteUrls = req.TiebaSpecifiedNoteUrls
	}
	if len(req.TiebaCreatorUrlList) > 0 {
		cfg.TiebaCreatorUrlList = req.TiebaCreatorUrlList
	}
	if len(req.ZhihuSpecifiedNoteUrls) > 0 {
		cfg.ZhihuSpecifiedNoteUrls = req.ZhihuSpecifiedNoteUrls
	}
	if len(req.KSSpecifiedNoteUrls) > 0 {
		cfg.KuaishouSpecifiedNoteUrls = req.KSSpecifiedNoteUrls
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
		switch crawlerType {
		case "search":
			if strings.TrimSpace(cfg.Keywords) == "" {
				return ValidationError{Msg: "keywords is required for search"}
			}
		case "detail":
			if len(cfg.BiliSpecifiedVideoUrls) == 0 {
				return ValidationError{Msg: "bili_specified_video_url_list is required for detail"}
			}
		case "creator":
			if len(cfg.BiliCreatorIdList) == 0 {
				return ValidationError{Msg: "bili_creator_id_list is required for creator"}
			}
		default:
			return ValidationError{Msg: fmt.Sprintf("unsupported crawler_type for bilibili: %s", crawlerType)}
		}
	case "weibo", "wb", "微博":
		switch crawlerType {
		case "search":
			if strings.TrimSpace(cfg.Keywords) == "" {
				return ValidationError{Msg: "keywords is required for search"}
			}
		case "detail":
			if len(cfg.WBSpecifiedNoteUrls) == 0 {
				return ValidationError{Msg: "wb_specified_note_url_list is required for detail"}
			}
		case "creator":
			if len(cfg.WBCreatorIdList) == 0 {
				return ValidationError{Msg: "wb_creator_id_list is required for creator"}
			}
		default:
			return ValidationError{Msg: fmt.Sprintf("unsupported crawler_type for weibo: %s", crawlerType)}
		}
	case "tieba", "tb", "贴吧":
		switch crawlerType {
		case "search":
			if strings.TrimSpace(cfg.Keywords) == "" {
				return ValidationError{Msg: "keywords is required for search"}
			}
		case "detail":
			if len(cfg.TiebaSpecifiedNoteUrls) == 0 {
				return ValidationError{Msg: "tieba_specified_note_url_list is required for detail"}
			}
		case "creator":
			if len(cfg.TiebaCreatorUrlList) == 0 {
				return ValidationError{Msg: "tieba_creator_url_list is required for creator"}
			}
		default:
			return ValidationError{Msg: fmt.Sprintf("unsupported crawler_type for tieba: %s", crawlerType)}
		}
	case "zhihu", "zh", "知乎":
		if crawlerType != "detail" {
			return ValidationError{Msg: "zhihu only supports crawler_type=detail"}
		}
		if len(cfg.ZhihuSpecifiedNoteUrls) == 0 {
			return ValidationError{Msg: "zhihu_specified_note_url_list is required for detail"}
		}
	case "kuaishou", "ks", "快手":
		if crawlerType != "detail" {
			return ValidationError{Msg: "kuaishou only supports crawler_type=detail"}
		}
		if len(cfg.KuaishouSpecifiedNoteUrls) == 0 {
			return ValidationError{Msg: "ks_specified_note_url_list is required for detail"}
		}
	default:
		if crawlerType == "search" && strings.TrimSpace(cfg.Keywords) == "" {
			return ValidationError{Msg: "keywords is required for search"}
		}
	}
	return nil
}
