package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/store"
	"strconv"
	"strings"
	"time"
)

type apiClient interface {
	GetView(context.Context, string, int64) (ViewResponse, error)
	SearchVideo(context.Context, string, int, string) (SearchResponse, error)
	GetUpInfo(context.Context, string) (UpInfoResponse, error)
	ListUpVideos(context.Context, string, int, int) (UpVideosResponse, error)
}

type Crawler struct {
	client apiClient
}

func NewCrawler() *Crawler {
	return &Crawler{client: NewClient()}
}

func NewCrawlerWithClient(client apiClient) *Crawler {
	if client == nil {
		client = NewClient()
	}
	return &Crawler{client: client}
}

func (c *Crawler) Run(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req.Platform = "bilibili"
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
		inputs = config.AppConfig.BiliSpecifiedVideoUrls
	}
	if len(inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs (BILI_SPECIFIED_VIDEO_URL_LIST)")
	}
	logger.Info("bilibili detail start", "inputs", len(inputs))
	limit := req.Concurrency
	if limit <= 0 {
		limit = 1
	}
	itemRes := crawler.ForEachLimit(ctx, inputs, limit, func(ctx context.Context, input string) error {
		bvid, aid, noteID, err := ParseVideoID(input)
		if err != nil {
			logger.Warn("skip invalid bilibili input", "value", input, "err", err)
			return crawler.Error{Kind: crawler.ErrorKindInvalidInput, Platform: req.Platform, Msg: "invalid bilibili input", Err: err}
		}
		res, err := c.client.GetView(ctx, bvid, aid)
		if err != nil {
			logger.Error("fetch view failed", "note_id", noteID, "err", err)
			return err
		}
		var data any
		if err := json.Unmarshal(res.Data, &data); err != nil {
			logger.Error("decode view data failed", "note_id", noteID, "err", err)
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
	out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, itemRes.FailureKinds)
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

	searchType := strings.TrimSpace(config.AppConfig.BiliSearchMode)
	if searchType == "" {
		searchType = "video"
	}

	out := crawler.NewResult(req)
	seen := map[string]struct{}{}
	for _, kw := range keywords {
		page := startPage
		for out.Succeeded+out.Failed < maxNotes {
			res, err := c.client.SearchVideo(ctx, kw, page, searchType)
			if err != nil {
				return out, err
			}
			var data map[string]any
			if err := json.Unmarshal(res.Data, &data); err != nil {
				return out, err
			}
			videos := extractSearchVideos(data)
			videos = filterNewVideos(videos, seen, maxNotes-(out.Succeeded+out.Failed))
			if len(videos) == 0 {
				break
			}
			itemRes := crawler.ForEachLimit(ctx, videos, limit, func(ctx context.Context, v videoRef) error {
				view, err := c.client.GetView(ctx, v.BVID, v.AID)
				if err != nil {
					return err
				}
				var payload any
				if err := json.Unmarshal(view.Data, &payload); err != nil {
					return err
				}
				return store.SaveNoteDetail(v.NoteID, payload)
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
		inputs = trimStrings(config.AppConfig.BiliCreatorIdList)
	}
	if len(inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs (BILI_CREATOR_ID_LIST)")
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
	for _, in := range inputs {
		mid, err := ParseCreatorID(in)
		if err != nil {
			return out, crawler.Error{Kind: crawler.ErrorKindInvalidInput, Platform: req.Platform, Msg: "invalid bilibili creator input", Err: err}
		}
		info, err := c.client.GetUpInfo(ctx, mid)
		if err != nil {
			return out, err
		}
		var infoData any
		if err := json.Unmarshal(info.Data, &infoData); err != nil {
			return out, err
		}
		if err := store.SaveCreatorProfile(mid, infoData); err != nil {
			return out, err
		}

		page := 1
		pageSize := 30
		seen := map[string]struct{}{}
		for out.Succeeded+out.Failed < maxNotes {
			res, err := c.client.ListUpVideos(ctx, mid, page, pageSize)
			if err != nil {
				return out, err
			}
			var data map[string]any
			if err := json.Unmarshal(res.Data, &data); err != nil {
				return out, err
			}
			videos := extractUpVideos(data)
			videos = filterNewVideos(videos, seen, maxNotes-(out.Succeeded+out.Failed))
			if len(videos) == 0 {
				break
			}
			itemRes := crawler.ForEachLimit(ctx, videos, limit, func(ctx context.Context, v videoRef) error {
				view, err := c.client.GetView(ctx, v.BVID, v.AID)
				if err != nil {
					return err
				}
				var payload any
				if err := json.Unmarshal(view.Data, &payload); err != nil {
					return err
				}
				return store.SaveNoteDetail(v.NoteID, payload)
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

type videoRef struct {
	BVID   string
	AID    int64
	NoteID string
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

func filterNewVideos(videos []videoRef, seen map[string]struct{}, limit int) []videoRef {
	out := make([]videoRef, 0, len(videos))
	for _, v := range videos {
		key := v.NoteID
		if key == "" {
			if v.BVID != "" {
				key = v.BVID
			} else if v.AID > 0 {
				key = "av" + strconv.FormatInt(v.AID, 10)
			}
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func extractSearchVideos(data map[string]any) []videoRef {
	raw, ok := data["result"].([]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	out := make([]videoRef, 0, len(raw))
	for _, it := range raw {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		bvid := strings.TrimSpace(fmt.Sprintf("%v", m["bvid"]))
		if strings.HasPrefix(strings.ToLower(bvid), "bv") {
			bvid = strings.ToUpper(bvid)
		}
		aid := toInt64(m["aid"])
		noteID := bvid
		if noteID == "" && aid > 0 {
			noteID = "av" + strconv.FormatInt(aid, 10)
		}
		if noteID == "" {
			continue
		}
		out = append(out, videoRef{BVID: bvid, AID: aid, NoteID: noteID})
	}
	return out
}

func extractUpVideos(data map[string]any) []videoRef {
	list, _ := dataGet(data, "list", "vlist").([]any)
	if len(list) == 0 {
		return nil
	}
	out := make([]videoRef, 0, len(list))
	for _, it := range list {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		bvid := strings.TrimSpace(fmt.Sprintf("%v", m["bvid"]))
		if strings.HasPrefix(strings.ToLower(bvid), "bv") {
			bvid = strings.ToUpper(bvid)
		}
		aid := toInt64(m["aid"])
		noteID := bvid
		if noteID == "" && aid > 0 {
			noteID = "av" + strconv.FormatInt(aid, 10)
		}
		if noteID == "" {
			continue
		}
		out = append(out, videoRef{BVID: bvid, AID: aid, NoteID: noteID})
	}
	return out
}

func dataGet(m map[string]any, keys ...string) any {
	var cur any = m
	for _, k := range keys {
		next, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = next[k]
	}
	return cur
}

func toInt64(v any) int64 {
	switch vv := v.(type) {
	case int64:
		return vv
	case int:
		return int64(vv)
	case float64:
		return int64(vv)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(vv), 10, 64)
		return n
	default:
		return 0
	}
}
