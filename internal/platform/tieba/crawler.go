package tieba

import (
	"context"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/proxy"
	"media-crawler-go/internal/store"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type fetchClient interface {
	FetchHTML(context.Context, string) (FetchResult, error)
}

type Crawler struct {
	client fetchClient
}

func NewCrawler() *Crawler {
	cli := NewClient()
	if config.AppConfig.EnableIPProxy {
		provider, err := proxy.NewProvider(config.AppConfig.IPProxyProviderName)
		if err != nil {
			logger.Warn("proxy provider init failed", "err", err)
		} else {
			cli.InitProxyPool(proxy.NewPool(provider, config.AppConfig.IPProxyPoolCount))
		}
	}
	return &Crawler{client: cli}
}

func NewCrawlerWithClient(client fetchClient) *Crawler {
	if client == nil {
		client = NewClient()
	}
	return &Crawler{client: client}
}

func (c *Crawler) Run(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req.Platform = "tieba"
	if req.Mode == "" {
		req.Mode = crawler.ModeDetail
	}
	out := crawler.NewResult(req)
	var res crawler.Result
	var err error
	switch req.Mode {
	case crawler.ModeSearch:
		res, err = c.runSearch(ctx, req)
	case crawler.ModeCreator:
		res, err = c.runCreator(ctx, req)
	default:
		res, err = c.runDetail(ctx, req)
	}
	res.StartedAt = out.StartedAt
	return res, err
}

func (c *Crawler) runDetail(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	if len(req.Inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs for tieba detail")
	}
	limit := req.Concurrency
	if limit <= 0 {
		limit = 1
	}
	out := crawler.NewResult(req)
	r := crawler.ForEachLimit(ctx, req.Inputs, limit, func(ctx context.Context, input string) error {
		return c.fetchAndSaveThread(ctx, req.Platform, input)
	})

	out.Processed = r.Processed
	out.Succeeded = r.Succeeded
	out.Failed = r.Failed
	out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)
	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *Crawler) runSearch(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	keywords := trimStrings(req.Keywords)
	if len(keywords) == 0 {
		keywords = trimStrings(strings.Split(config.AppConfig.Keywords, ","))
	}
	if len(keywords) == 0 {
		return crawler.Result{}, fmt.Errorf("empty keywords")
	}
	startPage := req.StartPage
	if startPage <= 0 {
		startPage = 1
	}
	maxNotes := req.MaxNotes
	if maxNotes <= 0 {
		maxNotes = 20
	}
	limit := req.Concurrency
	if limit <= 0 {
		limit = 1
	}

	out := crawler.NewResult(req)
	seen := map[string]struct{}{}
	for _, kw := range keywords {
		page := startPage
		for out.Succeeded+out.Failed < maxNotes {
			searchURL := buildSearchURL(kw, page)
			res, err := c.client.FetchHTML(ctx, searchURL)
			if err != nil {
				return out, err
			}
			riskHint := crawler.DetectRiskHint(res.Body)
			if riskHint != "" {
				return out, crawler.NewRiskHintError(req.Platform, res.URL, riskHint)
			}
			ids := extractThreadIDsFromHTML(res.Body)
			ids = filterNewIDs(ids, seen, maxNotes-(out.Succeeded+out.Failed))
			if len(ids) == 0 {
				break
			}
			itemRes := crawler.ForEachLimit(ctx, ids, limit, func(ctx context.Context, id string) error {
				return c.fetchAndSaveThread(ctx, req.Platform, id)
			})
			out.Processed += itemRes.Processed
			out.Succeeded += itemRes.Succeeded
			out.Failed += itemRes.Failed
			out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, itemRes.FailureKinds)
			page++
			sleepSec := config.AppConfig.CrawlerMaxSleepSec
			if sleepSec > 0 {
				select {
				case <-ctx.Done():
					return out, ctx.Err()
				case <-time.After(time.Duration(sleepSec) * time.Second):
				}
			}
		}
	}
	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *Crawler) runCreator(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	inputs := trimStrings(req.Inputs)
	if len(inputs) == 0 {
		inputs = trimStrings(config.AppConfig.TiebaCreatorUrlList)
	}
	if len(inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs (TIEBA_CREATOR_URL_LIST)")
	}
	maxNotes := req.MaxNotes
	if maxNotes <= 0 {
		maxNotes = 20
	}
	limit := req.Concurrency
	if limit <= 0 {
		limit = 1
	}

	out := crawler.NewResult(req)
	for _, in := range inputs {
		creatorURL := normalizeCreatorURL(in)
		creatorID := creatorKey(in)
		res, err := c.client.FetchHTML(ctx, creatorURL)
		if err != nil {
			return out, err
		}
		riskHint := crawler.DetectRiskHint(res.Body)
		record := map[string]any{
			"url":          res.URL,
			"status_code":  res.StatusCode,
			"content_type": res.ContentType,
			"body":         res.Body,
			"original_len": res.OriginalLen,
			"truncated":    res.Truncated,
			"fetched_at":   res.FetchedAt,
			"creator_id":   creatorID,
			"risk_hint":    riskHint,
		}
		if err := store.SaveCreatorProfile(creatorID, record); err != nil {
			return out, err
		}
		if riskHint != "" {
			return out, crawler.NewRiskHintError(req.Platform, res.URL, riskHint)
		}
		ids := extractThreadIDsFromCreatorHTML(res.Body)
		seen := map[string]struct{}{}
		ids = filterNewIDs(ids, seen, maxNotes-(out.Succeeded+out.Failed))
		if len(ids) == 0 {
			continue
		}
		itemRes := crawler.ForEachLimit(ctx, ids, limit, func(ctx context.Context, id string) error {
			return c.fetchAndSaveThread(ctx, req.Platform, id)
		})
		out.Processed += itemRes.Processed
		out.Succeeded += itemRes.Succeeded
		out.Failed += itemRes.Failed
		out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, itemRes.FailureKinds)
		sleepSec := config.AppConfig.CrawlerMaxSleepSec
		if sleepSec > 0 {
			select {
			case <-ctx.Done():
				return out, ctx.Err()
			case <-time.After(time.Duration(sleepSec) * time.Second):
			}
		}
	}
	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *Crawler) fetchAndSaveThread(ctx context.Context, platform string, input string) error {
	threadID, noteID, err := ParseThreadID(input)
	if err != nil {
		logger.Warn("tieba parse thread failed", "input", input, "err", err)
	}
	pageURL := ThreadURL(threadID)
	if pageURL == "" {
		pageURL = input
	}
	logger.Info("tieba fetch html", "url", pageURL, "note_id", noteID)
	res, err := c.client.FetchHTML(ctx, pageURL)
	if err != nil {
		logger.Error("tieba fetch failed", "url", pageURL, "err", err)
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
		"thread_id":      threadID,
		"parsed_note_id": noteID,
		"risk_hint":      riskHint,
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
	if riskHint != "" {
		return crawler.NewRiskHintError(platform, res.URL, riskHint)
	}

	if config.AppConfig.EnableGetComments && strings.TrimSpace(threadID) != "" {
		comments, err := fetchAllThreadComments(
			ctx,
			c.client,
			threadID,
			noteID,
			config.AppConfig.CrawlerMaxComments,
			config.AppConfig.CrawlerMaxSleepSec,
			config.AppConfig.EnableGetSubComments,
		)
		if err != nil {
			logger.Error("tieba fetch comments failed", "note_id", noteID, "thread_id", threadID, "err", err)
		} else if len(comments) > 0 {
			switch config.AppConfig.SaveDataOption {
			case "csv":
				items := make([]any, 0, len(comments))
				globalItems := make([]any, 0, len(comments))
				for i := range comments {
					items = append(items, &comments[i])
					globalItems = append(globalItems, &store.UnifiedComment{
						Platform:        "tieba",
						NoteID:          noteID,
						CommentID:       comments[i].CommentID,
						ParentCommentID: comments[i].ParentCommentID,
						Content:         comments[i].Content,
						CreateTime:      comments[i].CreateTime,
						LikeCount:       comments[i].LikeCount,
						UserID:          comments[i].UserID,
						UserNickname:    comments[i].UserNickname,
					})
				}
				if _, err := store.AppendUniqueCommentsCSV(
					noteID,
					items,
					func(item any) (string, error) { return item.(*Comment).CommentID, nil },
					(Comment{}).CSVHeader(),
					func(item any) ([]string, error) { return item.(*Comment).ToCSV(), nil },
				); err != nil {
					logger.Error("tieba save comments csv failed", "note_id", noteID, "err", err)
				}
				if _, err := store.AppendUniqueGlobalCommentsCSV(
					globalItems,
					func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
					(&store.UnifiedComment{}).CSVHeader(),
					func(item any) ([]string, error) { return item.(*store.UnifiedComment).ToCSV(), nil },
				); err != nil {
					logger.Error("tieba save global comments csv failed", "note_id", noteID, "err", err)
				}
			case "xlsx_book":
				globalItems := make([]any, 0, len(comments))
				for i := range comments {
					globalItems = append(globalItems, &store.UnifiedComment{
						Platform:        "tieba",
						NoteID:          noteID,
						CommentID:       comments[i].CommentID,
						ParentCommentID: comments[i].ParentCommentID,
						Content:         comments[i].Content,
						CreateTime:      comments[i].CreateTime,
						LikeCount:       comments[i].LikeCount,
						UserID:          comments[i].UserID,
						UserNickname:    comments[i].UserNickname,
					})
				}
				if _, err := store.AppendUniqueGlobalCommentsBook(
					globalItems,
					func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
					(&store.UnifiedComment{}).CSVHeader(),
					func(item any) ([]string, error) { return item.(*store.UnifiedComment).ToCSV(), nil },
				); err != nil {
					logger.Error("tieba save global comments xlsx_book failed", "note_id", noteID, "err", err)
				}
				if _, err := store.AppendUniqueGlobalCommentsJSONL(
					globalItems,
					func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
				); err != nil {
					logger.Error("tieba save global comments jsonl failed", "note_id", noteID, "err", err)
				}
			case "xlsx":
				items := make([]any, 0, len(comments))
				globalItems := make([]any, 0, len(comments))
				for i := range comments {
					items = append(items, &comments[i])
					globalItems = append(globalItems, &store.UnifiedComment{
						Platform:        "tieba",
						NoteID:          noteID,
						CommentID:       comments[i].CommentID,
						ParentCommentID: comments[i].ParentCommentID,
						Content:         comments[i].Content,
						CreateTime:      comments[i].CreateTime,
						LikeCount:       comments[i].LikeCount,
						UserID:          comments[i].UserID,
						UserNickname:    comments[i].UserNickname,
					})
				}
				if _, err := store.AppendUniqueCommentsXLSX(
					noteID,
					items,
					func(item any) (string, error) { return item.(*Comment).CommentID, nil },
					(Comment{}).CSVHeader(),
					func(item any) ([]string, error) { return item.(*Comment).ToCSV(), nil },
				); err != nil {
					logger.Error("tieba save comments xlsx failed", "note_id", noteID, "err", err)
				}
				if _, err := store.AppendUniqueGlobalCommentsXLSX(
					globalItems,
					func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
					(&store.UnifiedComment{}).CSVHeader(),
					func(item any) ([]string, error) { return item.(*store.UnifiedComment).ToCSV(), nil },
				); err != nil {
					logger.Error("tieba save global comments xlsx failed", "note_id", noteID, "err", err)
				}
			default:
				items := make([]any, 0, len(comments))
				globalItems := make([]any, 0, len(comments))
				for i := range comments {
					items = append(items, comments[i])
					globalItems = append(globalItems, &store.UnifiedComment{
						Platform:        "tieba",
						NoteID:          noteID,
						CommentID:       comments[i].CommentID,
						ParentCommentID: comments[i].ParentCommentID,
						Content:         comments[i].Content,
						CreateTime:      comments[i].CreateTime,
						LikeCount:       comments[i].LikeCount,
						UserID:          comments[i].UserID,
						UserNickname:    comments[i].UserNickname,
					})
				}
				if _, err := store.AppendUniqueCommentsJSONL(
					noteID,
					items,
					func(item any) (string, error) { return item.(Comment).CommentID, nil },
				); err != nil {
					logger.Error("tieba save comments jsonl failed", "note_id", noteID, "err", err)
				}
				if _, err := store.AppendUniqueGlobalCommentsJSONL(
					globalItems,
					func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
				); err != nil {
					logger.Error("tieba save global comments jsonl failed", "note_id", noteID, "err", err)
				}
			}
		}
	}
	return nil
}

func trimStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func buildSearchURL(keyword string, page int) string {
	if page <= 0 {
		page = 1
	}
	pn := (page - 1) * 10
	return "https://tieba.baidu.com/f/search/res?ie=utf-8&qw=" + url.QueryEscape(strings.TrimSpace(keyword)) + "&pn=" + strconv.Itoa(pn)
}

var (
	reThreadHref = regexp.MustCompile(`(?i)href\s*=\s*"[^"]*/p/(\d+)[^"]*"`)
	reDataTid    = regexp.MustCompile(`(?i)data-tid\s*=\s*"(\d+)"`)
)

func extractThreadIDsFromHTML(html string) []string {
	out := make([]string, 0, 32)
	for _, m := range reDataTid.FindAllStringSubmatch(html, -1) {
		if len(m) == 2 {
			out = append(out, m[1])
		}
	}
	for _, m := range reThreadHref.FindAllStringSubmatch(html, -1) {
		if len(m) == 2 {
			out = append(out, m[1])
		}
	}
	return trimStrings(out)
}

func extractThreadIDsFromCreatorHTML(html string) []string {
	return extractThreadIDsFromHTML(html)
}

func filterNewIDs(ids []string, seen map[string]struct{}, limit int) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func normalizeCreatorURL(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "://") {
		return s
	}
	if strings.HasPrefix(s, "id=") || strings.HasPrefix(s, "un=") {
		return "https://tieba.baidu.com/home/main?" + s
	}
	return "https://tieba.baidu.com/home/main?un=" + url.QueryEscape(s)
}

func creatorKey(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return fmt.Sprintf("tieba_creator_%d", time.Now().UnixNano())
	}
	key, err := ParseCreatorKey(s)
	if err != nil || strings.TrimSpace(key) == "" {
		return fmt.Sprintf("tieba_creator_%d", time.Now().UnixNano())
	}
	return key
}
