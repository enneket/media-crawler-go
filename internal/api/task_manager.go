package api

import (
	"context"
	"errors"
	"media-crawler-go/internal/config"
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
	LastError  string `json:"last_error,omitempty"`
}

type RunRequest struct {
	Platform    string `json:"platform,omitempty"`
	CrawlerType string `json:"crawler_type,omitempty"`
	Keywords    string `json:"keywords,omitempty"`
}

type TaskManager struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	status Status
	runFn  func(context.Context) error
}

func NewTaskManager() *TaskManager {
	return NewTaskManagerWithRunner(runCrawler)
}

func NewTaskManagerWithRunner(runFn func(context.Context) error) *TaskManager {
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
		return errors.New("task is running")
	}
	applyRunRequest(req)
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
		err := m.runFn(ctx)
		m.mu.Lock()
		defer m.mu.Unlock()
		m.cancel = nil
		m.status.State = "idle"
		m.status.FinishedAt = time.Now().Unix()
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

func runCrawler(ctx context.Context) error {
	c, err := platform.New(config.AppConfig.Platform)
	if err != nil {
		return err
	}
	return c.Start(ctx)
}

func applyRunRequest(req RunRequest) {
	if v := strings.TrimSpace(req.Platform); v != "" {
		config.AppConfig.Platform = v
	}
	if v := strings.TrimSpace(req.CrawlerType); v != "" {
		config.AppConfig.CrawlerType = v
	}
	if v := strings.TrimSpace(req.Keywords); v != "" {
		config.AppConfig.Keywords = v
	}
}
