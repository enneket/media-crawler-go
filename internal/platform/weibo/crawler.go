package weibo

import (
	"context"
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/store"
	"strings"
)

type Crawler struct {
	client *Client
}

func NewCrawler() *Crawler {
	return &Crawler{client: NewClient()}
}

func (c *Crawler) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	switch strings.ToLower(strings.TrimSpace(config.AppConfig.CrawlerType)) {
	case "", "detail":
		return c.runDetail(ctx)
	default:
		return fmt.Errorf("weibo only supports CRAWLER_TYPE=detail for now")
	}
}

func (c *Crawler) runDetail(ctx context.Context) error {
	inputs := config.AppConfig.WBSpecifiedNoteUrls
	if len(inputs) == 0 {
		return fmt.Errorf("empty WB_SPECIFIED_NOTE_URL_LIST")
	}
	logger.Info("weibo detail start", "inputs", len(inputs))
	for _, input := range inputs {
		id, noteID, err := ParseStatusID(input)
		if err != nil {
			logger.Warn("skip invalid weibo input", "value", input, "err", err)
			continue
		}
		res, err := c.client.Show(ctx, id)
		if err != nil {
			logger.Error("fetch status failed", "note_id", noteID, "err", err)
			continue
		}
		var data any
		if err := json.Unmarshal(res.Data, &data); err != nil {
			logger.Error("decode status data failed", "note_id", noteID, "err", err)
			continue
		}
		if err := store.SaveNoteDetail(noteID, data); err != nil {
			logger.Error("save note failed", "note_id", noteID, "err", err)
			continue
		}
		logger.Info("note saved", "note_id", noteID)
	}
	return nil
}
