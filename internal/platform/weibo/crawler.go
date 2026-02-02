package weibo

import (
	"context"
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/config"
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
	req.Platform = "weibo"
	if req.Mode == "" {
		req.Mode = crawler.ModeDetail
	}
	out := crawler.NewResult(req)
	res, err := c.runDetail(ctx, req)
	res.StartedAt = out.StartedAt
	return res, err
}

func (c *Crawler) runDetail(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	if req.Mode != crawler.ModeDetail {
		return crawler.Result{}, fmt.Errorf("weibo only supports mode=detail for now")
	}
	inputs := req.Inputs
	if len(inputs) == 0 {
		inputs = config.AppConfig.WBSpecifiedNoteUrls
	}
	if len(inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs (WB_SPECIFIED_NOTE_URL_LIST)")
	}
	logger.Info("weibo detail start", "inputs", len(inputs))
	limit := req.Concurrency
	if limit <= 0 {
		limit = 1
	}
	itemRes := crawler.ForEachLimit(ctx, inputs, limit, func(ctx context.Context, input string) error {
		id, noteID, err := ParseStatusID(input)
		if err != nil {
			logger.Warn("skip invalid weibo input", "value", input, "err", err)
			return err
		}
		res, err := c.client.Show(ctx, id)
		if err != nil {
			logger.Error("fetch status failed", "note_id", noteID, "err", err)
			return err
		}
		var data any
		if err := json.Unmarshal(res.Data, &data); err != nil {
			logger.Error("decode status data failed", "note_id", noteID, "err", err)
			return err
		}
		if err := store.SaveNoteDetail(noteID, data); err != nil {
			logger.Error("save note failed", "note_id", noteID, "err", err)
			return err
		}
		logger.Info("note saved", "note_id", noteID)
		return nil
	})
	out := crawler.NewResult(req)
	out.Processed = itemRes.Processed
	out.Succeeded = itemRes.Succeeded
	out.Failed = itemRes.Failed
	out.FinishedAt = time.Now().Unix()
	return out, nil
}
