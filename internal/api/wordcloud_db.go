package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func collectCommentTextsFromMySQLDSN(ctx context.Context, platform, noteID string, max int, dsn string) ([]string, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("MYSQL_DSN is empty")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)

	var rows *sql.Rows
	if strings.TrimSpace(noteID) != "" {
		rows, err = db.QueryContext(ctx, `SELECT data_json FROM comments WHERE platform=? AND note_id=? ORDER BY created_at DESC LIMIT ?`, platform, noteID, max)
	} else {
		rows, err = db.QueryContext(ctx, `SELECT data_json FROM comments WHERE platform=? ORDER BY created_at DESC LIMIT ?`, platform, max)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]string, 0, 256)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		text := extractCommentTextJSON([]byte(raw))
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, text)
		if len(out) >= max {
			break
		}
	}
	return out, nil
}

func collectCommentTextsFromPostgresDSN(ctx context.Context, platform, noteID string, max int, dsn string) ([]string, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("POSTGRES_DSN is empty")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)

	var rows *sql.Rows
	if strings.TrimSpace(noteID) != "" {
		rows, err = db.QueryContext(ctx, `SELECT data_json FROM comments WHERE platform=$1 AND note_id=$2 ORDER BY created_at DESC LIMIT $3`, platform, noteID, max)
	} else {
		rows, err = db.QueryContext(ctx, `SELECT data_json FROM comments WHERE platform=$1 ORDER BY created_at DESC LIMIT $2`, platform, max)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]string, 0, 256)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		text := extractCommentTextJSON([]byte(raw))
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, text)
		if len(out) >= max {
			break
		}
	}
	return out, nil
}

func collectCommentTextsFromMongo(ctx context.Context, platform, noteID string, max int, uri string, dbName string) ([]string, error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return nil, errors.New("MONGO_URI is empty")
	}
	dbName = strings.TrimSpace(dbName)
	if dbName == "" {
		dbName = "media_crawler"
	}

	cliCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cli, err := mongo.Connect(cliCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cli.Disconnect(context.Background()) }()
	if err := cli.Ping(cliCtx, readpref.Primary()); err != nil {
		return nil, err
	}

	filter := bson.M{"platform": platform}
	if strings.TrimSpace(noteID) != "" {
		filter["note_id"] = noteID
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(max)).
		SetProjection(bson.M{"data_json": 1})

	cur, err := cli.Database(dbName).Collection("comments").Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	type doc struct {
		DataJSON string `bson:"data_json"`
	}
	out := make([]string, 0, 256)
	for cur.Next(ctx) {
		var d doc
		if err := cur.Decode(&d); err != nil {
			continue
		}
		if strings.TrimSpace(d.DataJSON) == "" {
			continue
		}
		text := extractCommentTextJSON([]byte(d.DataJSON))
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, text)
		if len(out) >= max {
			break
		}
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("mongo cursor: %w", err)
	}
	return out, nil
}
