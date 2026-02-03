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

	_ "github.com/go-sql-driver/mysql"
)

var (
	mysqlOnce sync.Once
	mysqlInst *sql.DB
	mysqlErr  error
)

func mysqlDSN() string {
	return strings.TrimSpace(config.AppConfig.MySQLDSN)
}

func mysqlDB() (*sql.DB, error) {
	if backendKind() != backendMySQL {
		return nil, errors.New("mysql backend disabled")
	}
	mysqlOnce.Do(func() {
		dsn := mysqlDSN()
		if dsn == "" {
			mysqlErr = errors.New("MYSQL_DSN is empty")
			return
		}
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			mysqlErr = err
			return
		}
		setDBPoolDefaults(db, 8)
		db.SetConnMaxIdleTime(2 * time.Minute)

		if err := initMySQLSchema(db); err != nil {
			_ = db.Close()
			mysqlErr = err
			return
		}
		mysqlInst = db
	})
	return mysqlInst, mysqlErr
}

func initMySQLSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS notes (
			platform VARCHAR(32) NOT NULL,
			note_id VARCHAR(191) NOT NULL,
			data_json LONGTEXT NOT NULL,
			updated_at BIGINT NOT NULL,
			PRIMARY KEY (platform, note_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		`CREATE TABLE IF NOT EXISTS creators (
			platform VARCHAR(32) NOT NULL,
			creator_id VARCHAR(191) NOT NULL,
			data_json LONGTEXT NOT NULL,
			updated_at BIGINT NOT NULL,
			PRIMARY KEY (platform, creator_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		`CREATE TABLE IF NOT EXISTS comments (
			platform VARCHAR(32) NOT NULL,
			comment_id VARCHAR(191) NOT NULL,
			note_id VARCHAR(191) NOT NULL,
			data_json LONGTEXT NOT NULL,
			created_at BIGINT NOT NULL,
			PRIMARY KEY (platform, comment_id),
			KEY idx_comments_note (platform, note_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("mysql init schema: %w", err)
		}
	}
	return nil
}

func mysqlUpsertNote(noteID string, note any) error {
	db, err := mysqlDB()
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
		`INSERT INTO notes(platform, note_id, data_json, updated_at) VALUES(?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE data_json=VALUES(data_json), updated_at=VALUES(updated_at);`,
		platform, noteID, string(b), now,
	)
	return err
}

func mysqlUpsertCreator(creatorID string, data any) error {
	db, err := mysqlDB()
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
		`INSERT INTO creators(platform, creator_id, data_json, updated_at) VALUES(?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE data_json=VALUES(data_json), updated_at=VALUES(updated_at);`,
		platform, creatorID, string(b), now,
	)
	return err
}

func mysqlInsertComments(noteID string, items []any, keyFn func(any) (string, error)) error {
	db, err := mysqlDB()
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

	stmt, err := tx.Prepare(`INSERT IGNORE INTO comments(platform, comment_id, note_id, data_json, created_at) VALUES(?, ?, ?, ?, ?);`)
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
