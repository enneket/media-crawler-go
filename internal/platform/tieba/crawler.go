package tieba

import (
	"context"
	"fmt"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/store"
	"time"
)

type Crawler struct {
	client *Client
}

func NewCrawler() *Crawler {
	return &Crawler{client: NewClient()}
}

func (c *Crawler) Run(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req.Platform = "tieba"
	if req.Mode == "" {
		req.Mode = crawler.ModeDetail
	}
	if req.Mode != crawler.ModeDetail {
		return crawler.Result{}, fmt.Errorf("tieba only supports mode=detail for now")
	}
	if len(req.Inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs for tieba detail")
	}

	out := crawler.NewResult(req)
	limit := req.Concurrency
	if limit <= 0 {
		limit = 1
	}

	r := crawler.ForEachLimit(ctx, req.Inputs, limit, func(ctx context.Context, input string) error {
		threadID, noteID, err := ParseThreadID(input)
		if err != nil {
			logger.Warn("tieba parse thread failed", "input", input, "err", err)
		}
		url := ThreadURL(threadID)
		if url == "" {
			url = input
		}
		logger.Info("tieba fetch html", "url", url, "note_id", noteID)
		res, err := c.client.FetchHTML(ctx, url)
		if err != nil {
			logger.Error("tieba fetch failed", "url", url, "err", err)
			return err
		}
		record := map[string]any{
			"url":            res.URL,
			"status_code":    res.StatusCode,
			"content_type":   res.ContentType,
			"body":           res.Body,
			"original_len":   res.OriginalLen,
			"truncated":      res.Truncated,
			"fetched_at":     res.FetchedAt,
			"thread_id":      threadID,
			"parsed_note_id": noteID,
		}
		if noteID == "" {
			noteID = threadID
		}
		if noteID == "" {
			noteID = fmt.Sprintf("tieba_%d", time.Now().UnixNano())
		}
		if err := store.SaveNoteDetail(noteID, record); err != nil {
			logger.Error("tieba save note failed", "note_id", noteID, "err", err)
			return err
		}
		logger.Info("tieba note saved", "note_id", noteID)
		return nil
	})

	out.Processed = r.Processed
	out.Succeeded = r.Succeeded
	out.Failed = r.Failed
	out.FinishedAt = time.Now().Unix()
	return out, nil
}
