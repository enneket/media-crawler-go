package api

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type autoWordcloudOptions struct {
	DataDir      string
	Platform     string
	NoteID       string
	StoreBackend string
	SQLitePath   string

	MaxComments int
	MaxWords    int
	MinCount    int
	Width       int
	Height      int
}

func autoGenerateWordcloud(opts autoWordcloudOptions) (string, error) {
	opts.Platform = strings.TrimSpace(opts.Platform)
	if opts.Platform == "" {
		return "", nil
	}
	opts.DataDir = strings.TrimSpace(opts.DataDir)
	if opts.DataDir == "" {
		opts.DataDir = "data"
	}
	if opts.MaxComments <= 0 {
		opts.MaxComments = 5000
	}
	if opts.MaxWords <= 0 {
		opts.MaxWords = 200
	}
	if opts.MinCount <= 0 {
		opts.MinCount = 2
	}
	if opts.Width <= 0 {
		opts.Width = 1200
	}
	if opts.Height <= 0 {
		opts.Height = 800
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	texts, err := collectCommentTextsAuto(ctx, opts)
	if err != nil {
		return "", err
	}
	if len(texts) == 0 {
		return "", nil
	}

	counts := countWords(texts, opts.MaxWords, opts.MinCount)
	if len(counts) == 0 {
		return "", nil
	}

	seed := seedFor(opts.Platform + ":" + strings.TrimSpace(opts.NoteID))
	svg := renderWordcloudSVG(counts, opts.Width, opts.Height, seed)

	fn := "wordcloud_comments_" + time.Now().Format("20060102_150405") + ".svg"
	if strings.TrimSpace(opts.NoteID) != "" {
		fn = "wordcloud_" + sanitizeFilename(opts.NoteID) + "_" + time.Now().Format("20060102_150405") + ".svg"
	}
	outDir := filepath.Join(opts.DataDir, opts.Platform)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}
	full := filepath.Join(outDir, fn)
	if err := os.WriteFile(full, []byte(svg), 0644); err != nil {
		return "", err
	}
	if rel, err := filepath.Rel(opts.DataDir, full); err == nil {
		return filepath.ToSlash(rel), nil
	}
	return fmt.Sprintf("%s/%s", opts.Platform, fn), nil
}

func collectCommentTextsAuto(ctx context.Context, opts autoWordcloudOptions) ([]string, error) {
	if opts.MaxComments <= 0 {
		return nil, nil
	}
	if strings.EqualFold(strings.TrimSpace(opts.StoreBackend), "sqlite") {
		return collectCommentTextsFromSQLitePath(ctx, opts.Platform, opts.NoteID, opts.MaxComments, opts.SQLitePath)
	}
	return collectCommentTextsFromFiles(ctx, opts.DataDir, opts.Platform, opts.NoteID, opts.MaxComments)
}

func collectCommentTextsFromSQLitePath(ctx context.Context, platform, noteID string, max int, sqlitePath string) ([]string, error) {
	sqlitePath = strings.TrimSpace(sqlitePath)
	if sqlitePath == "" {
		sqlitePath = "data/media_crawler.db"
	}
	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

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
