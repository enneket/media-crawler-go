package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"media-crawler-go/internal/config"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	pgOnce sync.Once
	pgInst *sql.DB
	pgErr  error
)

func postgresDSN() string {
	return strings.TrimSpace(config.AppConfig.PostgresDSN)
}

func postgresDB() (*sql.DB, error) {
	if backendKind() != backendPostgres {
		return nil, errors.New("postgres backend disabled")
	}
	pgOnce.Do(func() {
		dsn := postgresDSN()
		if dsn == "" {
			pgErr = errors.New("POSTGRES_DSN is empty")
			return
		}
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			pgErr = err
			return
		}
		setDBPoolDefaults(db, 8)
		db.SetConnMaxIdleTime(2 * time.Minute)

		if err := initPostgresSchema(db); err != nil {
			_ = db.Close()
			pgErr = err
			return
		}
		pgInst = db
	})
	return pgInst, pgErr
}

func initPostgresSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS notes (
			platform TEXT NOT NULL,
			note_id TEXT NOT NULL,
			data_json TEXT NOT NULL,
			updated_at BIGINT NOT NULL,
			PRIMARY KEY (platform, note_id)
		);`,
		`CREATE TABLE IF NOT EXISTS creators (
			platform TEXT NOT NULL,
			creator_id TEXT NOT NULL,
			data_json TEXT NOT NULL,
			updated_at BIGINT NOT NULL,
			PRIMARY KEY (platform, creator_id)
		);`,
		`CREATE TABLE IF NOT EXISTS comments (
			platform TEXT NOT NULL,
			comment_id TEXT NOT NULL,
			note_id TEXT NOT NULL,
			data_json TEXT NOT NULL,
			created_at BIGINT NOT NULL,
			PRIMARY KEY (platform, comment_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_note ON comments(platform, note_id);`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("postgres init schema: %w", err)
		}
	}
	return nil
}

func postgresUpsertNote(noteID string, note any) error {
	db, err := postgresDB()
	if err != nil {
		return err
	}
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
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
		 VALUES($1, $2, $3, $4)
		 ON CONFLICT (platform, note_id)
		 DO UPDATE SET data_json=EXCLUDED.data_json, updated_at=EXCLUDED.updated_at;`,
		platform, noteID, string(b), now,
	)
	return err
}

func postgresUpsertCreator(creatorID string, data any) error {
	db, err := postgresDB()
	if err != nil {
		return err
	}
	creatorID = strings.TrimSpace(creatorID)
	if creatorID == "" {
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
		 VALUES($1, $2, $3, $4)
		 ON CONFLICT (platform, creator_id)
		 DO UPDATE SET data_json=EXCLUDED.data_json, updated_at=EXCLUDED.updated_at;`,
		platform, creatorID, string(b), now,
	)
	return err
}

func postgresInsertComments(noteID string, items []any, keyFn func(any) (string, error)) error {
	db, err := postgresDB()
	if err != nil {
		return err
	}
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return errors.New("note_id is empty")
	}
	if len(items) == 0 {
		return nil
	}
	platform := strings.TrimSpace(config.AppConfig.Platform)
	if platform == "" {
		platform = "xhs"
	}
	now := time.Now().Unix()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`INSERT INTO comments(platform, comment_id, note_id, data_json, created_at) VALUES($1, $2, $3, $4, $5) ON CONFLICT (platform, comment_id) DO NOTHING;`)
	if err != nil {
		return err
	}
	defer stmt.Close()

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
