package store

import (
	"database/sql"
	"errors"
	"fmt"
	"media-crawler-go/internal/config"
	"strings"
)

type sqlBackendKind string

const (
	backendFile     sqlBackendKind = "file"
	backendSQLite   sqlBackendKind = "sqlite"
	backendMySQL    sqlBackendKind = "mysql"
	backendPostgres sqlBackendKind = "postgres"
	backendMongoDB  sqlBackendKind = "mongodb"
)

func backendKind() sqlBackendKind {
	v := strings.ToLower(strings.TrimSpace(config.AppConfig.StoreBackend))
	switch v {
	case "sqlite":
		return backendSQLite
	case "mysql":
		return backendMySQL
	case "postgres", "postgresql":
		return backendPostgres
	case "mongodb", "mongo":
		return backendMongoDB
	default:
		return backendFile
	}
}

func sqlEnabled() bool {
	k := backendKind()
	return k == backendSQLite || k == backendMySQL || k == backendPostgres || k == backendMongoDB
}

func requireSQLBackend() error {
	if !sqlEnabled() {
		return errors.New("sql backend disabled")
	}
	return nil
}

func placeholder(k sqlBackendKind, idx int) string {
	if k == backendPostgres {
		return fmt.Sprintf("$%d", idx)
	}
	return "?"
}

func isDriverDisabled(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "unknown driver")
}

func setDBPoolDefaults(db *sql.DB, maxOpen int) {
	if db == nil {
		return
	}
	if maxOpen <= 0 {
		maxOpen = 4
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	db.SetConnMaxLifetime(0)
}
