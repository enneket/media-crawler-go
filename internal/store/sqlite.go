package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"media-crawler-go/internal/config"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	sqliteOnce sync.Once
	sqliteInst *sql.DB
	sqliteErr  error
)

func sqliteEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(config.AppConfig.StoreBackend), "sqlite")
}

func sqlitePath() string {
	p := strings.TrimSpace(config.AppConfig.SQLitePath)
	if p == "" {
		p = "data/media_crawler.db"
	}
	return p
}

func sqliteDB() (*sql.DB, error) {
	if !sqliteEnabled() {
		return nil, errors.New("sqlite backend disabled")
	}
	sqliteOnce.Do(func() {
		p := sqlitePath()
		if dir := filepath.Dir(p); dir != "" && dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}
		db, err := sql.Open("sqlite", p)
		if err != nil {
			sqliteErr = err
			return
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(0)

		if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
			_ = db.Close()
			sqliteErr = err
			return
		}
		if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
			_ = db.Close()
			sqliteErr = err
			return
		}

		stmts := []string{
			`CREATE TABLE IF NOT EXISTS notes (
				platform TEXT NOT NULL,
				note_id TEXT NOT NULL,
				data_json TEXT NOT NULL,
				updated_at INTEGER NOT NULL,
				PRIMARY KEY (platform, note_id)
			);`,
			`CREATE TABLE IF NOT EXISTS creators (
				platform TEXT NOT NULL,
				creator_id TEXT NOT NULL,
				data_json TEXT NOT NULL,
				updated_at INTEGER NOT NULL,
				PRIMARY KEY (platform, creator_id)
			);`,
			`CREATE TABLE IF NOT EXISTS comments (
				platform TEXT NOT NULL,
				comment_id TEXT NOT NULL,
				note_id TEXT NOT NULL,
				data_json TEXT NOT NULL,
				created_at INTEGER NOT NULL,
				PRIMARY KEY (platform, comment_id)
			);`,
			`CREATE INDEX IF NOT EXISTS idx_comments_note ON comments(platform, note_id);`,
		}
		for _, stmt := range stmts {
			if _, err := db.Exec(stmt); err != nil {
				_ = db.Close()
				sqliteErr = err
				return
			}
		}
		sqliteInst = db
	})
	return sqliteInst, sqliteErr
}

func sqliteUpsertNote(noteID string, note any) error {
	if !sqliteEnabled() {
		return nil
	}
	db, err := sqliteDB()
	if err != nil {
		return err
	}
	if strings.TrimSpace(noteID) == "" {
		return errors.New("note_id is empty")
	}
	b, err := json.Marshal(note)
	if err != nil {
		return err
	}
	platform := strings.TrimSpace(config.AppConfig.Platform)
	if platform == "" {
		platform = "xhs"
	}
	now := time.Now().Unix()
	_, err = db.Exec(
		`INSERT INTO notes(platform, note_id, data_json, updated_at)
		 VALUES(?, ?, ?, ?)
		 ON CONFLICT(platform, note_id)
		 DO UPDATE SET data_json=excluded.data_json, updated_at=excluded.updated_at;`,
		platform, noteID, string(b), now,
	)
	return err
}

func sqliteUpsertCreator(creatorID string, data any) error {
	if !sqliteEnabled() {
		return nil
	}
	db, err := sqliteDB()
	if err != nil {
		return err
	}
	if strings.TrimSpace(creatorID) == "" {
		return errors.New("creator_id is empty")
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	platform := strings.TrimSpace(config.AppConfig.Platform)
	if platform == "" {
		platform = "xhs"
	}
	now := time.Now().Unix()
	_, err = db.Exec(
		`INSERT INTO creators(platform, creator_id, data_json, updated_at)
		 VALUES(?, ?, ?, ?)
		 ON CONFLICT(platform, creator_id)
		 DO UPDATE SET data_json=excluded.data_json, updated_at=excluded.updated_at;`,
		platform, creatorID, string(b), now,
	)
	return err
}

func sqliteInsertComments(noteID string, items []any, keyFn func(any) (string, error)) error {
	if !sqliteEnabled() {
		return nil
	}
	db, err := sqliteDB()
	if err != nil {
		return err
	}
	if strings.TrimSpace(noteID) == "" {
		return errors.New("note_id is empty")
	}
	if len(items) == 0 {
		return nil
	}
	platform := strings.TrimSpace(config.AppConfig.Platform)
	if platform == "" {
		platform = "xhs"
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO comments(platform, comment_id, note_id, data_json, created_at) VALUES(?, ?, ?, ?, ?);`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, item := range items {
		id, err := keyFn(item)
		if err != nil {
			return err
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		b, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("marshal comment %s: %w", id, err)
		}
		if _, err := stmt.Exec(platform, id, noteID, string(b), now); err != nil {
			return err
		}
	}

	return tx.Commit()
}
