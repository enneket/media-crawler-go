package store

import (
	"media-crawler-go/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendUniqueGlobalCommentsCSV(t *testing.T) {
	dataDir := t.TempDir()
	oldCfg := config.AppConfig
	config.AppConfig.DataDir = dataDir
	config.AppConfig.Platform = "test"
	t.Cleanup(func() { config.AppConfig = oldCfg })
	dir := filepath.Join(dataDir, "test")

	items := []any{
		&UnifiedComment{Platform: "xhs", NoteID: "n1", CommentID: "c1", Content: "a"},
		&UnifiedComment{Platform: "xhs", NoteID: "n1", CommentID: "c2", Content: "b"},
	}
	n, err := AppendUniqueGlobalCommentsCSV(
		items,
		func(item any) (string, error) { return item.(*UnifiedComment).CommentID, nil },
		(&UnifiedComment{}).CSVHeader(),
		func(item any) ([]string, error) { return item.(*UnifiedComment).ToCSV(), nil },
	)
	if err != nil {
		t.Fatalf("append csv: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2, got %d", n)
	}
	n, err = AppendUniqueGlobalCommentsCSV(
		items,
		func(item any) (string, error) { return item.(*UnifiedComment).CommentID, nil },
		(&UnifiedComment{}).CSVHeader(),
		func(item any) ([]string, error) { return item.(*UnifiedComment).ToCSV(), nil },
	)
	if err != nil {
		t.Fatalf("append csv dup: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}

	if _, err := os.Stat(filepath.Join(dir, "comments.csv")); err != nil {
		t.Fatalf("comments.csv missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "comments.global.idx")); err != nil {
		t.Fatalf("comments.global.idx missing: %v", err)
	}
}
