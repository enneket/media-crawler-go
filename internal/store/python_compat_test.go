package store

import (
	"encoding/json"
	"media-crawler-go/internal/config"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPythonCompatJSONArrayOutput(t *testing.T) {
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
		StoreBackend:       "file",
		SaveDataOption:     "json",
		DataDir:            "data",
		CrawlerType:        "search",
		PythonCompatOutput: true,
	}

	if err := SaveNoteDetail("n1", map[string]any{"id": "n1"}); err != nil {
		t.Fatalf("SaveNoteDetail: %v", err)
	}
	if err := SaveNoteDetail("n2", map[string]any{"id": "n2"}); err != nil {
		t.Fatalf("SaveNoteDetail: %v", err)
	}

	date := time.Now().Format("2006-01-02")
	outPath := filepath.Join("data", "xhs", "json", "search_contents_"+date+".json")
	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read python compat file: %v", err)
	}
	var arr []any
	if err := json.Unmarshal(b, &arr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(arr) != 2 {
		t.Fatalf("len=%d want=2", len(arr))
	}
}

