package store

import (
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/config"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Store interface {
	Save(data interface{}, filename string) error
}

type JsonStore struct {
	Dir string
}

func NewJsonStore(dir string) *JsonStore {
	return &JsonStore{Dir: dir}
}

func (s *JsonStore) Save(data interface{}, filename string) error {
	if err := os.MkdirAll(s.Dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(s.Dir, filename)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(data)
}

type CsvStore struct {
	Dir string
	mu  sync.Mutex
}

func NewCsvStore(dir string) *CsvStore {
	return &CsvStore{Dir: dir}
}

// Simple CSV saver - assumes flat struct or needs conversion
// For complex nested JSON like XHS notes, CSV is hard.
// We will just implement JSON store for now as per config default.

func GetStore() Store {
	// Create data directory
	dataDir := "data"
	if config.AppConfig.SaveDataOption == "json" {
		return NewJsonStore(filepath.Join(dataDir, "xhs"))
	}
	return NewJsonStore(filepath.Join(dataDir, "xhs"))
}

func SaveNote(note interface{}) error {
	s := GetStore()
	date := time.Now().Format("2006-01-02")
	return s.Save(note, fmt.Sprintf("notes_%s.json", date))
}

func SaveComments(comments interface{}) error {
	s := GetStore()
	date := time.Now().Format("2006-01-02")
	return s.Save(comments, fmt.Sprintf("comments_%s.json", date))
}
