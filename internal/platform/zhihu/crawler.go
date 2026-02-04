package zhihu

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/proxy"
	"media-crawler-go/internal/store"
	"net/url"
	"strings"
	"time"
)

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

type fetchClient interface {
	FetchHTML(context.Context, string) (FetchResult, error)
}

func (c *Crawler) Run(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req.Platform = "zhihu"
	if req.Mode == "" {
		req.Mode = crawler.ModeDetail
	}
	switch req.Mode {
	case crawler.ModeDetail:
		return c.runDetail(ctx, req)
	case crawler.ModeSearch:
		return c.runSearch(ctx, req)
	case crawler.ModeCreator:
		return c.runCreator(ctx, req)
	default:
		return crawler.Result{}, fmt.Errorf("unsupported zhihu mode: %s", req.Mode)
	}
}

func (c *Crawler) runDetail(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	if len(req.Inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs for zhihu detail")
	}
	out := crawler.NewResult(req)
	limit := req.Concurrency
	if limit <= 0 {
		limit = 1
	}
	r := crawler.ForEachLimit(ctx, req.Inputs, limit, func(ctx context.Context, input string) error {
		u := normalizeZhihuDetailURL(input)
		return c.fetchAndSaveDetail(ctx, req.Platform, u)
	})
	out.Processed = r.Processed
	out.Succeeded = r.Succeeded
	out.Failed = r.Failed
	out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)
	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *Crawler) runSearch(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	keywords := req.Keywords
	if len(keywords) == 0 {
		keywords = config.GetKeywords()
	}
	if len(keywords) == 0 {
		return crawler.Result{}, fmt.Errorf("empty keywords for zhihu search")
	}

	startPage := req.StartPage
	if startPage <= 0 {
		startPage = config.AppConfig.StartPage
	}
	if startPage < 1 {
		startPage = 1
	}
	maxNotes := req.MaxNotes
	if maxNotes == 0 {
		maxNotes = config.AppConfig.CrawlerMaxNotesCount
	}
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = config.AppConfig.MaxConcurrencyNum
	}
	if concurrency < 1 {
		concurrency = 1
	}

	out := crawler.NewResult(req)
	seen := make(map[string]struct{}, 256)

	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		logger.Info("zhihu searching keyword", "keyword", keyword)
		page := startPage
		for {
			if maxNotes > 0 && out.Processed >= maxNotes {
				break
			}
			select {
			case <-ctx.Done():
				out.FinishedAt = time.Now().Unix()
				return out, ctx.Err()
			default:
			}

			searchURL := ""
			singlePage := false
			if strings.HasPrefix(keyword, "http://") || strings.HasPrefix(keyword, "https://") {
				searchURL = keyword
				singlePage = true
			} else {
				searchURL = fmt.Sprintf("https://www.zhihu.com/search?type=content&q=%s&page=%d", url.QueryEscape(keyword), page)
			}
			res, err := c.client.FetchHTML(ctx, searchURL)
			if err != nil {
				logger.Error("zhihu search fetch failed", "url", searchURL, "err", err)
				break
			}
			baseURL := "https://www.zhihu.com"
			if pu, err := url.Parse(res.URL); err == nil && pu.Scheme != "" && pu.Host != "" {
				baseURL = pu.Scheme + "://" + pu.Host
			}
			candidates := ExtractDetailURLsFromHTML(res.Body, baseURL, 500)
			if len(candidates) == 0 {
				break
			}

			tasks := make([]string, 0, len(candidates))
			for _, u := range candidates {
				if maxNotes > 0 && out.Processed+len(tasks) >= maxNotes {
					break
				}
				if _, ok := seen[u]; ok {
					continue
				}
				seen[u] = struct{}{}
				tasks = append(tasks, u)
			}
			if len(tasks) == 0 {
				break
			}

			r := crawler.ForEachLimit(ctx, tasks, concurrency, func(ctx context.Context, u string) error {
				return c.fetchAndSaveDetail(ctx, req.Platform, u)
			})
			out.Succeeded += r.Succeeded
			out.Failed += r.Failed
			out.Processed += r.Processed
			out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)

			page++
			if config.AppConfig.CrawlerMaxSleepSec > 0 {
				crawler.Sleep(ctx, time.Duration(config.AppConfig.CrawlerMaxSleepSec)*time.Second)
			}
			if singlePage {
				break
			}
		}
	}

	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *Crawler) runCreator(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	if len(req.Inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs for zhihu creator")
	}
	maxNotes := req.MaxNotes
	if maxNotes == 0 {
		maxNotes = config.AppConfig.CrawlerMaxNotesCount
	}
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = config.AppConfig.MaxConcurrencyNum
	}
	if concurrency < 1 {
		concurrency = 1
	}

	out := crawler.NewResult(req)
	seen := make(map[string]struct{}, 256)

	for _, input := range req.Inputs {
		select {
		case <-ctx.Done():
			out.FinishedAt = time.Now().Unix()
			return out, ctx.Err()
		default:
		}

		creatorURL := normalizeZhihuCreatorURL(input)
		creatorID := stableID("zhihu_creator", input)
		logger.Info("zhihu fetch creator", "creator_id", creatorID, "url", creatorURL)

		res, err := c.client.FetchHTML(ctx, creatorURL)
		if err != nil {
			logger.Error("zhihu creator fetch failed", "url", creatorURL, "err", err)
			out.Failed++
			out.Processed++
			kind := crawler.KindOf(err)
			if kind == "" {
				kind = crawler.ErrorKindUnknown
			}
			out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, map[string]int{string(kind): 1})
			continue
		}
		riskHint := crawler.DetectRiskHint(res.Body)
		if err := store.SaveCreator(creatorID, map[string]any{
			"url":          res.URL,
			"status_code":  res.StatusCode,
			"content_type": res.ContentType,
			"body":         res.Body,
			"original_len": res.OriginalLen,
			"truncated":    res.Truncated,
			"fetched_at":   res.FetchedAt,
			"risk_hint":    riskHint,
		}); err != nil {
			logger.Error("zhihu save creator failed", "creator_id", creatorID, "err", err)
		}

		baseURL := "https://www.zhihu.com"
		if pu, err := url.Parse(res.URL); err == nil && pu.Scheme != "" && pu.Host != "" {
			baseURL = pu.Scheme + "://" + pu.Host
		}
		candidates := ExtractDetailURLsFromHTML(res.Body, baseURL, 500)
		tasks := make([]string, 0, len(candidates))
		for _, u := range candidates {
			if maxNotes > 0 && out.Processed+len(tasks) >= maxNotes {
				break
			}
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			tasks = append(tasks, u)
		}
		if len(tasks) == 0 {
			continue
		}

		r := crawler.ForEachLimit(ctx, tasks, concurrency, func(ctx context.Context, u string) error {
			return c.fetchAndSaveDetail(ctx, req.Platform, u)
		})
		out.Succeeded += r.Succeeded
		out.Failed += r.Failed
		out.Processed += r.Processed
		out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)
	}

	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *Crawler) fetchAndSaveDetail(ctx context.Context, platform string, url string) error {
	qid, aid, noteID, _ := ParseZhihuID(url)
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
		noteID = stableID("zhihu", url)
	}
	if err := store.SaveNoteDetail(noteID, record); err != nil {
		logger.Error("zhihu save note failed", "note_id", noteID, "err", err)
		return err
	}
	logger.Info("zhihu note saved", "note_id", noteID)
	if riskHint != "" {
		return crawler.NewRiskHintError(platform, res.URL, riskHint)
	}

	if config.AppConfig.EnableGetComments {
		comments := parseCommentsFromHTML(res.Body, noteID, config.AppConfig.CrawlerMaxComments, config.AppConfig.EnableGetSubComments)
		if len(comments) > 0 {
			switch config.AppConfig.SaveDataOption {
			case "csv":
				items := make([]any, 0, len(comments))
				globalItems := make([]any, 0, len(comments))
				for i := range comments {
					items = append(items, &comments[i])
					globalItems = append(globalItems, &store.UnifiedComment{
						Platform:        "zhihu",
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
					logger.Error("zhihu save comments csv failed", "note_id", noteID, "err", err)
				}
				if _, err := store.AppendUniqueGlobalCommentsCSV(
					globalItems,
					func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
					(&store.UnifiedComment{}).CSVHeader(),
					func(item any) ([]string, error) { return item.(*store.UnifiedComment).ToCSV(), nil },
				); err != nil {
					logger.Error("zhihu save global comments csv failed", "note_id", noteID, "err", err)
				}
			case "xlsx":
				items := make([]any, 0, len(comments))
				globalItems := make([]any, 0, len(comments))
				for i := range comments {
					items = append(items, &comments[i])
					globalItems = append(globalItems, &store.UnifiedComment{
						Platform:        "zhihu",
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
					logger.Error("zhihu save comments xlsx failed", "note_id", noteID, "err", err)
				}
				if _, err := store.AppendUniqueGlobalCommentsXLSX(
					globalItems,
					func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
					(&store.UnifiedComment{}).CSVHeader(),
					func(item any) ([]string, error) { return item.(*store.UnifiedComment).ToCSV(), nil },
				); err != nil {
					logger.Error("zhihu save global comments xlsx failed", "note_id", noteID, "err", err)
				}
			default:
				items := make([]any, 0, len(comments))
				globalItems := make([]any, 0, len(comments))
				for i := range comments {
					items = append(items, comments[i])
					globalItems = append(globalItems, &store.UnifiedComment{
						Platform:        "zhihu",
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
					logger.Error("zhihu save comments jsonl failed", "note_id", noteID, "err", err)
				}
				if _, err := store.AppendUniqueGlobalCommentsJSONL(
					globalItems,
					func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
				); err != nil {
					logger.Error("zhihu save global comments jsonl failed", "note_id", noteID, "err", err)
				}
			}
		}
	}
	return nil
}

func normalizeZhihuDetailURL(input string) string {
	qid, aid, _, err := ParseZhihuID(input)
	if err == nil && qid != "" && aid != "" {
		return fmt.Sprintf("https://www.zhihu.com/question/%s/answer/%s", qid, aid)
	}
	if err == nil && qid != "" && (len(strings.TrimSpace(input)) < 40) {
		return fmt.Sprintf("https://www.zhihu.com/question/%s", qid)
	}
	return strings.TrimSpace(input)
}

func normalizeZhihuCreatorURL(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return s
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}
	return "https://www.zhihu.com/people/" + url.PathEscape(s)
}

func stableID(prefix string, raw string) string {
	h := sha1.Sum([]byte(prefix + ":" + strings.TrimSpace(raw)))
	return prefix + "_" + hex.EncodeToString(h[:])
}
