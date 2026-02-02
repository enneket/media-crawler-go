package bilibili

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
		return fmt.Errorf("bilibili only supports CRAWLER_TYPE=detail for now")
	}
}

func (c *Crawler) runDetail(ctx context.Context) error {
	inputs := config.AppConfig.BiliSpecifiedVideoUrls
	if len(inputs) == 0 {
		return fmt.Errorf("empty BILI_SPECIFIED_VIDEO_URL_LIST")
	}
	logger.Info("bilibili detail start", "inputs", len(inputs))
	for _, input := range inputs {
		bvid, aid, noteID, err := ParseVideoID(input)
		if err != nil {
			logger.Warn("skip invalid bilibili input", "value", input, "err", err)
			continue
		}
		res, err := c.client.GetView(ctx, bvid, aid)
		if err != nil {
			logger.Error("fetch view failed", "note_id", noteID, "err", err)
			continue
		}
		var data any
		if err := json.Unmarshal(res.Data, &data); err != nil {
			logger.Error("decode view data failed", "note_id", noteID, "err", err)
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
