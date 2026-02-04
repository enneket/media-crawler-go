package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"media-crawler-go/internal/config"
)

var pyCompatMu sync.Mutex

func pythonCompatEnabled() bool {
	if !config.AppConfig.PythonCompatOutput {
		return false
	}
	if strings.ToLower(strings.TrimSpace(config.AppConfig.StoreBackend)) != "file" {
		return false
	}
	return true
}

func pythonCompatAppendJSON(itemType string, item any) error {
	if !pythonCompatEnabled() {
		return nil
	}
	if strings.ToLower(strings.TrimSpace(config.AppConfig.SaveDataOption)) != "json" {
		return nil
	}
	itemType = strings.TrimSpace(itemType)
	if itemType == "" {
		return nil
	}

	dataDir := strings.TrimSpace(config.AppConfig.DataDir)
	if dataDir == "" {
		dataDir = "data"
	}
	platform := strings.TrimSpace(config.AppConfig.Platform)
	if platform == "" {
		platform = "xhs"
	}
	crawlerType := strings.TrimSpace(config.AppConfig.CrawlerType)
	if crawlerType == "" {
		crawlerType = "search"
	}
	date := time.Now().Format("2006-01-02")

	dir := filepath.Join(dataDir, platform, "json")
	filename := fmt.Sprintf("%s_%s_%s.json", crawlerType, itemType, date)
	path := filepath.Join(dir, filename)

	pyCompatMu.Lock()
	defer pyCompatMu.Unlock()

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var arr []any
	if b, err := os.ReadFile(path); err == nil && len(bytesTrimSpace(b)) > 0 {
		if err := json.Unmarshal(b, &arr); err != nil {
			arr = nil
		}
	}
	arr = append(arr, item)

	b, err := json.MarshalIndent(arr, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0644)
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

