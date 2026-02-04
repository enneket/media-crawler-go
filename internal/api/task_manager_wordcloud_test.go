package api

import (
	"context"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTaskManagerAutoWordcloud(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	config.AppConfig = config.Config{
		Platform:           "xhs",
		CrawlerType:        "search",
		Keywords:           "k",
		DataDir:            "data",
		StoreBackend:       "file",
		SaveDataOption:     "json",
		EnableGetComments:  true,
		EnableGetWordcloud: true,
	}

	dir := filepath.Join("data", "xhs", "notes", "n1")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	commentsPath := filepath.Join(dir, "comments.jsonl")
	if err := os.WriteFile(commentsPath, []byte("{\"content\":\"golang\"}\n{\"content\":\"golang\"}\n"), 0644); err != nil {
		t.Fatalf("write comments: %v", err)
	}

	m := NewTaskManagerWithRunner(func(ctx context.Context) (crawler.Result, error) {
		return crawler.Result{}, nil
	})
	if err := m.Run(RunRequest{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		matches, _ := filepath.Glob(filepath.Join("data", "xhs", "wordcloud_comments_*.svg"))
		if len(matches) > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected auto wordcloud svg generated")
}

