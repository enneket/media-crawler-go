package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"media-crawler-go/internal/config"
)

func resetSQLiteForTest(t *testing.T) {
	t.Helper()
	if sqliteInst != nil {
		_ = sqliteInst.Close()
	}
	sqliteInst = nil
	sqliteErr = nil
	sqliteOnce = sync.Once{}
}

func TestSQLiteUpsertNote(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	config.AppConfig.Platform = "xhs"
	config.AppConfig.StoreBackend = "sqlite"
	config.AppConfig.SQLitePath = filepath.Join(tmp, "data", "media_crawler.db")
	config.AppConfig.SaveDataOption = "json"

	resetSQLiteForTest(t)

	n1 := map[string]any{"id": "n1", "title": "a"}
	if err := SaveNoteDetail("n1", n1); err != nil {
		t.Fatalf("SaveNoteDetail err: %v", err)
	}
	n2 := map[string]any{"id": "n1", "title": "b"}
	if err := SaveNoteDetail("n1", n2); err != nil {
		t.Fatalf("SaveNoteDetail(upsert) err: %v", err)
	}

	db, err := sqliteDB()
	if err != nil {
		t.Fatalf("sqliteDB err: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notes WHERE platform=? AND note_id=?`, "xhs", "n1").Scan(&count); err != nil {
		t.Fatalf("query count err: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 note row, got %d", count)
	}
}

func TestSQLiteGlobalCommentDedupe(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	config.AppConfig.Platform = "xhs"
	config.AppConfig.StoreBackend = "sqlite"
	config.AppConfig.SQLitePath = filepath.Join(tmp, "data", "media_crawler.db")
	config.AppConfig.SaveDataOption = "json"

	resetSQLiteForTest(t)

	keyFn := func(item any) (string, error) {
		m := item.(map[string]any)
		return m["id"].(string), nil
	}

	_, err := AppendUniqueCommentsJSONL("note1", []any{map[string]any{"id": "c1", "text": "a"}}, keyFn)
	if err != nil {
		t.Fatalf("append note1 err: %v", err)
	}
	_, err = AppendUniqueCommentsJSONL("note2", []any{map[string]any{"id": "c1", "text": "a"}}, keyFn)
	if err != nil {
		t.Fatalf("append note2 err: %v", err)
	}

	db, err := sqliteDB()
	if err != nil {
		t.Fatalf("sqliteDB err: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM comments WHERE platform=? AND comment_id=?`, "xhs", "c1").Scan(&count); err != nil {
		t.Fatalf("query count err: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected global dedupe (1 row), got %d", count)
	}
}
