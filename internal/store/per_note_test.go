package store

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendUniqueJSONL(t *testing.T) {
	dir := t.TempDir()
	items := []any{
		map[string]any{"id": "1", "v": "a"},
		map[string]any{"id": "2", "v": "b"},
	}
	n, err := AppendUniqueJSONL(dir, "comments.jsonl", "comments.idx", items, func(item any) (string, error) {
		return item.(map[string]any)["id"].(string), nil
	})
	if err != nil {
		t.Fatalf("AppendUniqueJSONL err: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 appended, got %d", n)
	}

	n, err = AppendUniqueJSONL(dir, "comments.jsonl", "comments.idx", items, func(item any) (string, error) {
		return item.(map[string]any)["id"].(string), nil
	})
	if err != nil {
		t.Fatalf("AppendUniqueJSONL err: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 appended on duplicates, got %d", n)
	}

	f, err := os.Open(filepath.Join(dir, "comments.jsonl"))
	if err != nil {
		t.Fatalf("open jsonl: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	lines := 0
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) != "" {
			lines++
		}
	}
	if lines != 2 {
		t.Fatalf("expected 2 lines in jsonl, got %d", lines)
	}
}

func TestAppendUniqueCSV(t *testing.T) {
	dir := t.TempDir()
	items := []any{"1", "2", "2", "3"}
	n, err := AppendUniqueCSV(
		dir,
		"comments.csv",
		"comments.idx",
		items,
		func(item any) (string, error) { return item.(string), nil },
		[]string{"id"},
		func(item any) ([]string, error) { return []string{item.(string)}, nil },
	)
	if err != nil {
		t.Fatalf("AppendUniqueCSV err: %v", err)
	}
	if n != 3 {
		t.Fatalf("expected 3 appended, got %d", n)
	}

	n, err = AppendUniqueCSV(
		dir,
		"comments.csv",
		"comments.idx",
		[]any{"3", "4"},
		func(item any) (string, error) { return item.(string), nil },
		[]string{"id"},
		func(item any) ([]string, error) { return []string{item.(string)}, nil },
	)
	if err != nil {
		t.Fatalf("AppendUniqueCSV err: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 appended, got %d", n)
	}
}
