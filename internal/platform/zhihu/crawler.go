package zhihu

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
	req.Platform = "zhihu"
	if req.Mode == "" {
		req.Mode = crawler.ModeDetail
	}
	if req.Mode != crawler.ModeDetail {
		return crawler.Result{}, fmt.Errorf("zhihu only supports mode=detail for now")
	}
	if len(req.Inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs for zhihu detail")
	}

	out := crawler.NewResult(req)
	limit := req.Concurrency
	if limit <= 0 {
		limit = 1
	}

	r := crawler.ForEachLimit(ctx, req.Inputs, limit, func(ctx context.Context, input string) error {
		qid, aid, noteID, err := ParseZhihuID(input)
		if err != nil {
			logger.Warn("zhihu parse failed", "input", input, "err", err)
		}
		url := input
		if qid != "" && aid != "" {
			url = fmt.Sprintf("https://www.zhihu.com/question/%s/answer/%s", qid, aid)
		} else if qid != "" && (len(input) < 40) {
			url = fmt.Sprintf("https://www.zhihu.com/question/%s", qid)
		}

		logger.Info("zhihu fetch html", "url", url, "note_id", noteID)
		res, err := c.client.FetchHTML(ctx, url)
		if err != nil {
			logger.Error("zhihu fetch failed", "url", url, "err", err)
			return err
		}
		riskHint := crawler.DetectRiskHint(res.Body)
		record := map[string]any{
			"url":            res.URL,
			"status_code":    res.StatusCode,
			"content_type":   res.ContentType,
			"body":           res.Body,
			"original_len":   res.OriginalLen,
			"truncated":      res.Truncated,
			"fetched_at":     res.FetchedAt,
			"question_id":    qid,
			"answer_id":      aid,
			"parsed_note_id": noteID,
			"risk_hint":      riskHint,
		}
		if noteID == "" {
			if qid != "" && aid != "" {
				noteID = qid + "_" + aid
			} else {
				noteID = qid
			}
		}
		if noteID == "" {
			noteID = fmt.Sprintf("zhihu_%d", time.Now().UnixNano())
		}
		if err := store.SaveNoteDetail(noteID, record); err != nil {
			logger.Error("zhihu save note failed", "note_id", noteID, "err", err)
			return err
		}
		logger.Info("zhihu note saved", "note_id", noteID)
		if riskHint != "" {
			return crawler.NewRiskHintError(req.Platform, res.URL, riskHint)
		}
		return nil
	})

	out.Processed = r.Processed
	out.Succeeded = r.Succeeded
	out.Failed = r.Failed
	out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)
	out.FinishedAt = time.Now().Unix()
	return out, nil
}
