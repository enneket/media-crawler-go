package store

import (
	"encoding/csv"
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

type CSVer interface {
	ToCSV() []string
	CSVHeader() []string
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

func (s *CsvStore) Save(data interface{}, filename string) error {
	item, ok := data.(CSVer)
	if !ok {
		return fmt.Errorf("data does not implement CSVer interface")
	}

	if err := os.MkdirAll(s.Dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(s.Dir, filename)

	s.mu.Lock()
	defer s.mu.Unlock()

	fileExists := false
	if _, err := os.Stat(path); err == nil {
		fileExists = true
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write BOM for Excel compatibility
	if !fileExists {
		file.WriteString("\xEF\xBB\xBF")
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if !fileExists {
		if err := writer.Write(item.CSVHeader()); err != nil {
			return err
		}
	}

	return writer.Write(item.ToCSV())
}

func GetStore() Store {
	path := PlatformDir()

	if config.AppConfig.SaveDataOption == "csv" {
		return NewCsvStore(path)
	}
	if config.AppConfig.SaveDataOption == "xlsx" {
		return NewXlsxStore(path)
	}
	return NewJsonStore(path)
}

func SaveNote(note interface{}) error {
	s := GetStore()
	date := time.Now().Format("2006-01-02")
	ext := "json"
	if config.AppConfig.SaveDataOption == "csv" {
		ext = "csv"
	}
	if config.AppConfig.SaveDataOption == "xlsx" {
		ext = "xlsx"
	}
	return s.Save(note, fmt.Sprintf("notes_%s.%s", date, ext))
}

func SaveComments(comments interface{}) error {
	// Comments is usually a list or map wrapper.
	// If it's CSV, we need to handle it carefully.
	// For now, let's assume comments are passed one by one or wrapped.
	// But current crawler passes `map[string]interface{}` for JSON.
	// We need to fix crawler to pass `Comment` object if we want CSV support for comments.

	s := GetStore()
	date := time.Now().Format("2006-01-02")
	ext := "json"
	if config.AppConfig.SaveDataOption == "csv" {
		ext = "csv"
	}
	if config.AppConfig.SaveDataOption == "xlsx" {
		ext = "xlsx"
	}
	return s.Save(comments, fmt.Sprintf("comments_%s.%s", date, ext))
}

func SaveCreator(userID string, creator interface{}) error {
	if err := sqliteUpsertCreator(userID, creator); err != nil {
		return err
	}
	s := GetStore()
	date := time.Now().Format("2006-01-02")
	ext := "json"
	if config.AppConfig.SaveDataOption == "csv" {
		ext = "csv"
	}
	if config.AppConfig.SaveDataOption == "xlsx" {
		ext = "xlsx"
	}
	filename := fmt.Sprintf("creators_%s.%s", date, ext)
	return s.Save(creator, filename)
}
