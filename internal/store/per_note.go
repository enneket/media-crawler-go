package store

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"media-crawler-go/internal/config"
)

func PlatformDir() string {
	dataDir := strings.TrimSpace(config.AppConfig.DataDir)
	if dataDir == "" {
		dataDir = "data"
	}
	platform := strings.TrimSpace(config.AppConfig.Platform)
	if platform == "" {
		platform = "xhs"
	}
	return filepath.Join(dataDir, platform)
}

func NotesDir() string {
	return filepath.Join(PlatformDir(), "notes")
}

func NoteDir(noteID string) string {
	return filepath.Join(NotesDir(), noteID)
}

func NoteMediaDir(noteID string) string {
	return filepath.Join(NoteDir(noteID), "media")
}

func SaveNoteDetail(noteID string, note interface{}) error {
	if strings.TrimSpace(noteID) == "" {
		return errors.New("note_id is empty")
	}
	if err := sqlUpsertNote(noteID, note); err != nil {
		return err
	}
	dir := NoteDir(noteID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if config.AppConfig.SaveDataOption == "csv" {
		item, ok := note.(CSVer)
		if !ok {
			return fmt.Errorf("note does not implement CSVer interface")
		}
		path := filepath.Join(dir, "note.csv")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := f.WriteString("\xEF\xBB\xBF"); err != nil {
			return err
		}
		w := csv.NewWriter(f)
		if err := w.Write(item.CSVHeader()); err != nil {
			return err
		}
		if err := w.Write(item.ToCSV()); err != nil {
			return err
		}
		w.Flush()
		return w.Error()
	}
	if config.AppConfig.SaveDataOption == "xlsx" {
		wb := NewXlsxStore(dir)
		return wb.Save(note, "note.xlsx")
	}

	path := filepath.Join(dir, "note.json")
	b, err := json.Marshal(note)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0644); err != nil {
		return err
	}
	_ = pythonCompatAppendJSON("contents", note)
	return nil
}

func AppendUniqueJSONL(dir, dataFilename, indexFilename string, items []any, keyFn func(any) (string, error)) (int, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, err
	}

	indexPath := filepath.Join(dir, indexFilename)
	seen, err := loadIndex(indexPath)
	if err != nil {
		return 0, err
	}

	filtered := make([]any, 0, len(items))
	newKeys := make([]string, 0, len(items))
	for _, item := range items {
		k, err := keyFn(item)
		if err != nil {
			return 0, err
		}
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		filtered = append(filtered, item)
		newKeys = append(newKeys, k)
	}

	if len(filtered) == 0 {
		return 0, nil
	}

	dataPath := filepath.Join(dir, dataFilename)
	f, err := os.OpenFile(dataPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, item := range filtered {
		if err := enc.Encode(item); err != nil {
			return 0, err
		}
	}

	if err := appendIndex(indexPath, newKeys); err != nil {
		return 0, err
	}
	return len(filtered), nil
}

func AppendUniqueCSV(dir, dataFilename, indexFilename string, items []any, keyFn func(any) (string, error), header []string, rowFn func(any) ([]string, error)) (int, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, err
	}

	indexPath := filepath.Join(dir, indexFilename)
	seen, err := loadIndex(indexPath)
	if err != nil {
		return 0, err
	}

	rows := make([][]string, 0, len(items))
	newKeys := make([]string, 0, len(items))
	for _, item := range items {
		k, err := keyFn(item)
		if err != nil {
			return 0, err
		}
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		r, err := rowFn(item)
		if err != nil {
			return 0, err
		}
		seen[k] = struct{}{}
		rows = append(rows, r)
		newKeys = append(newKeys, k)
	}

	if len(rows) == 0 {
		return 0, nil
	}

	dataPath := filepath.Join(dir, dataFilename)
	fileExists := false
	if _, err := os.Stat(dataPath); err == nil {
		fileExists = true
	}
	f, err := os.OpenFile(dataPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	if !fileExists {
		if _, err := f.WriteString("\xEF\xBB\xBF"); err != nil {
			return 0, err
		}
	}

	w := csv.NewWriter(f)
	if !fileExists {
		if err := w.Write(header); err != nil {
			return 0, err
		}
	}
	for _, r := range rows {
		if err := w.Write(r); err != nil {
			return 0, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return 0, err
	}

	if err := appendIndex(indexPath, newKeys); err != nil {
		return 0, err
	}
	return len(rows), nil
}

func loadIndex(path string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		k := strings.TrimSpace(scanner.Text())
		if k == "" {
			continue
		}
		out[k] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func appendIndex(path string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, k := range keys {
		if _, err := f.WriteString(k + "\n"); err != nil {
			return err
		}
	}
	return nil
}
