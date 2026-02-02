package crawler

import (
	"context"
	"strings"
	"time"
)

type Mode string

const (
	ModeSearch  Mode = "search"
	ModeDetail  Mode = "detail"
	ModeCreator Mode = "creator"
)

func NormalizeMode(s string) Mode {
	v := strings.ToLower(strings.TrimSpace(s))
	switch v {
	case "detail":
		return ModeDetail
	case "creator":
		return ModeCreator
	default:
		return ModeSearch
	}
}

type Request struct {
	Platform string
	Mode     Mode

	Keywords []string
	Inputs   []string

	StartPage   int
	MaxNotes    int
	Concurrency int
}

type Result struct {
	Platform     string         `json:"platform,omitempty"`
	Mode         string         `json:"mode,omitempty"`
	StartedAt    int64          `json:"started_at,omitempty"`
	FinishedAt   int64          `json:"finished_at,omitempty"`
	Processed    int            `json:"processed,omitempty"`
	Succeeded    int            `json:"succeeded,omitempty"`
	Failed       int            `json:"failed,omitempty"`
	FailureKinds map[string]int `json:"failure_kinds,omitempty"`
}

func NewResult(req Request) Result {
	return Result{
		Platform:  req.Platform,
		Mode:      string(req.Mode),
		StartedAt: time.Now().Unix(),
	}
}

type Runner interface {
	Run(ctx context.Context, req Request) (Result, error)
}
