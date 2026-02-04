package weibo

import (
	"context"
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/downloader"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/proxy"
	"media-crawler-go/internal/store"
	"strconv"
	"strings"
	"time"
)

type Crawler struct {
	client apiClient
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

func NewCrawlerWithClient(client apiClient) *Crawler {
	if client == nil {
		client = NewClient()
	}
	return &Crawler{client: client}
}

type apiClient interface {
	Show(context.Context, string) (ShowResponse, error)
	SearchByKeyword(context.Context, string, int, string) (GetIndexResponse, error)
	CreatorInfo(context.Context, string) (GetIndexResponse, error)
	NotesByCreator(context.Context, string, string, string) (GetIndexResponse, error)
	GetNoteComments(context.Context, string, int64, int) (hotflowData, error)
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
			return crawler.Error{Kind: crawler.ErrorKindInvalidInput, Platform: req.Platform, Msg: "invalid weibo input", Err: err}
		}
		return c.fetchAndSaveStatus(ctx, id, noteID)
	})
	out := crawler.NewResult(req)
	out.Processed = itemRes.Processed
	out.Succeeded = itemRes.Succeeded
	out.Failed = itemRes.Failed
	out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, itemRes.FailureKinds)
	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *Crawler) fetchAndSaveStatus(ctx context.Context, id string, noteID string) error {
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

	if !config.AppConfig.EnableGetComments {
		if config.AppConfig.EnableGetMedias {
			c.downloadMedias(noteID, data)
		}
		return nil
	}
	comments, err := fetchAllNoteComments(
		ctx,
		c.client,
		noteID,
		config.AppConfig.CrawlerMaxComments,
		config.AppConfig.CrawlerMaxSleepSec,
		config.AppConfig.EnableGetSubComments,
	)
	if err != nil {
		logger.Error("fetch weibo comments failed", "note_id", noteID, "err", err)
		return nil
	}
	if len(comments) == 0 {
		return nil
	}

	switch config.AppConfig.SaveDataOption {
	case "csv":
		items := make([]any, 0, len(comments))
		globalItems := make([]any, 0, len(comments))
		for i := range comments {
			items = append(items, &comments[i])
			globalItems = append(globalItems, &store.UnifiedComment{
				Platform:        "weibo",
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
			logger.Error("save weibo comments csv failed", "note_id", noteID, "err", err)
		}
		if _, err := store.AppendUniqueGlobalCommentsCSV(
			globalItems,
			func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
			(&store.UnifiedComment{}).CSVHeader(),
			func(item any) ([]string, error) { return item.(*store.UnifiedComment).ToCSV(), nil },
		); err != nil {
			logger.Error("save weibo global comments csv failed", "note_id", noteID, "err", err)
		}
	case "xlsx":
		items := make([]any, 0, len(comments))
		globalItems := make([]any, 0, len(comments))
		for i := range comments {
			items = append(items, &comments[i])
			globalItems = append(globalItems, &store.UnifiedComment{
				Platform:        "weibo",
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
			logger.Error("save weibo comments xlsx failed", "note_id", noteID, "err", err)
		}
		if _, err := store.AppendUniqueGlobalCommentsXLSX(
			globalItems,
			func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
			(&store.UnifiedComment{}).CSVHeader(),
			func(item any) ([]string, error) { return item.(*store.UnifiedComment).ToCSV(), nil },
		); err != nil {
			logger.Error("save weibo global comments xlsx failed", "note_id", noteID, "err", err)
		}
	default:
		items := make([]any, 0, len(comments))
		globalItems := make([]any, 0, len(comments))
		for i := range comments {
			items = append(items, comments[i])
			globalItems = append(globalItems, &store.UnifiedComment{
				Platform:        "weibo",
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
			logger.Error("save weibo comments jsonl failed", "note_id", noteID, "err", err)
		}
		if _, err := store.AppendUniqueGlobalCommentsJSONL(
			globalItems,
			func(item any) (string, error) { return item.(*store.UnifiedComment).CommentID, nil },
		); err != nil {
			logger.Error("save weibo global comments jsonl failed", "note_id", noteID, "err", err)
		}
	}

	if config.AppConfig.EnableGetMedias {
		c.downloadMedias(noteID, data)
	}
	return nil
}

func (c *Crawler) downloadMedias(noteID string, data any) {
	urls, filenames := ExtractWeiboMediaURLs(noteID, data)
	if len(urls) == 0 {
		return
	}
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"Referer":    fmt.Sprintf("https://m.weibo.cn/detail/%s", noteID),
	}
	if ck := strings.TrimSpace(config.AppConfig.Cookies); ck != "" {
		headers["Cookie"] = ck
	}
	d := downloader.NewDownloader(store.NoteMediaDir(noteID))
	_ = d.BatchDownloadWithHeaders(urls, filenames, headers)
}

func (c *Crawler) runSearch(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	keywords := req.Keywords
	if len(keywords) == 0 {
		if v := strings.TrimSpace(config.AppConfig.Keywords); v != "" {
			keywords = strings.Split(v, ",")
		}
	}
	keywords = trimStrings(keywords)
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

	searchType := strings.TrimSpace(config.AppConfig.WBSearchType)
	if searchType == "" {
		searchType = "1"
	}

	seen := map[string]struct{}{}
	out := crawler.NewResult(req)

	for _, kw := range keywords {
		page := startPage
		for out.Succeeded+out.Failed < maxNotes {
			res, err := c.client.SearchByKeyword(ctx, kw, page, searchType)
			if err != nil {
				return out, err
			}
			var data map[string]any
			if err := json.Unmarshal(res.Data, &data); err != nil {
				return out, err
			}
			ids := extractNoteIDsFromIndex(data)
			ids = filterNewIDs(ids, seen, maxNotes-(out.Succeeded+out.Failed))
			if len(ids) == 0 {
				break
			}
			itemRes := crawler.ForEachLimit(ctx, ids, limit, func(ctx context.Context, id string) error {
				return c.fetchAndSaveStatus(ctx, id, id)
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
	inputs := req.Inputs
	if len(inputs) == 0 {
		inputs = config.AppConfig.WBCreatorIdList
	}
	inputs = trimStrings(inputs)
	if len(inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs (WB_CREATOR_ID_LIST)")
	}

	maxNotes := req.MaxNotes
	if maxNotes <= 0 {
		maxNotes = 50
	}
	limit := req.Concurrency
	if limit <= 0 {
		limit = 1
	}

	out := crawler.NewResult(req)
	logger.Info("weibo creator start", "creators", len(inputs))

	for _, creatorID := range inputs {
		info, err := c.client.CreatorInfo(ctx, creatorID)
		if err != nil {
			return out, err
		}
		var infoData map[string]any
		if err := json.Unmarshal(info.Data, &infoData); err != nil {
			return out, err
		}
		profile := any(infoData)
		if ui, ok := infoData["userInfo"]; ok {
			profile = ui
		}
		if err := store.SaveCreatorProfile(creatorID, profile); err != nil {
			return out, err
		}

		containerID := "107603" + creatorID
		sinceID := "0"
		seen := map[string]struct{}{}

		for out.Succeeded+out.Failed < maxNotes {
			res, err := c.client.NotesByCreator(ctx, creatorID, containerID, sinceID)
			if err != nil {
				return out, err
			}
			var data map[string]any
			if err := json.Unmarshal(res.Data, &data); err != nil {
				return out, err
			}
			sinceID = extractSinceID(data)

			ids := extractNoteIDsFromIndex(data)
			ids = filterNewIDs(ids, seen, maxNotes-(out.Succeeded+out.Failed))
			if len(ids) == 0 {
				break
			}
			itemRes := crawler.ForEachLimit(ctx, ids, limit, func(ctx context.Context, id string) error {
				return c.fetchAndSaveStatus(ctx, id, id)
			})
			out.Processed += itemRes.Processed
			out.Succeeded += itemRes.Succeeded
			out.Failed += itemRes.Failed
			out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, itemRes.FailureKinds)

			if sinceID == "" || sinceID == "0" {
				break
			}
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

func extractSinceID(data map[string]any) string {
	cardlistInfo, ok := data["cardlistInfo"].(map[string]any)
	if !ok {
		return ""
	}
	v, ok := cardlistInfo["since_id"]
	if !ok || v == nil {
		return ""
	}
	switch vv := v.(type) {
	case string:
		return strings.TrimSpace(vv)
	case float64:
		return strconv.FormatInt(int64(vv), 10)
	default:
		return fmt.Sprintf("%v", vv)
	}
}

func extractNoteIDsFromIndex(data map[string]any) []string {
	cards, ok := data["cards"].([]any)
	if !ok || len(cards) == 0 {
		return nil
	}
	out := make([]string, 0, 32)
	for _, it := range cards {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, extractNoteIDsFromCard(m)...)
		if cg, ok := m["card_group"].([]any); ok {
			for _, git := range cg {
				gm, ok := git.(map[string]any)
				if !ok {
					continue
				}
				out = append(out, extractNoteIDsFromCard(gm)...)
			}
		}
	}
	return trimStrings(out)
}

func extractNoteIDsFromCard(card map[string]any) []string {
	ct, ok := card["card_type"]
	if !ok {
		return nil
	}
	if toInt(ct) != 9 {
		return nil
	}
	mblog, ok := card["mblog"].(map[string]any)
	if !ok {
		return nil
	}
	id, ok := mblog["id"]
	if !ok || id == nil {
		return nil
	}
	return []string{fmt.Sprintf("%v", id)}
}

func toInt(v any) int {
	switch vv := v.(type) {
	case int:
		return vv
	case int64:
		return int(vv)
	case float64:
		return int(vv)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(vv))
		return n
	default:
		return 0
	}
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
