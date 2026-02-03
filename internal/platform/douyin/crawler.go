package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/browser"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/downloader"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/proxy"
	"media-crawler-go/internal/store"
	"os/exec"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type DouyinCrawler struct {
	pw      *playwright.Playwright
	browser playwright.BrowserContext
	page    playwright.Page
	client  *Client
	signer  *Signer

	cdpBrowser playwright.Browser
	cdpCmd     *exec.Cmd

	proxyPool *proxy.Pool
	cleanupUD func()
}

func NewCrawler() *DouyinCrawler {
	return &DouyinCrawler{}
}

func (c *DouyinCrawler) Run(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	logger.Info("douyin crawler started")

	if config.AppConfig.EnableIPProxy {
		provider, err := proxy.NewProvider(config.AppConfig.IPProxyProviderName)
		if err != nil {
			logger.Warn("proxy provider init failed", "err", err)
		} else {
			c.proxyPool = proxy.NewPool(provider, config.AppConfig.IPProxyPoolCount)
		}
	}

	if err := c.initBrowser(ctx); err != nil {
		return crawler.Result{}, err
	}
	defer c.close()

	signer, err := NewSigner()
	if err != nil {
		return crawler.Result{}, err
	}
	c.signer = signer

	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	c.client = NewClient(c.signer, userAgent)
	if c.proxyPool != nil {
		c.client.InitProxyPool(c.proxyPool)
	}

	if config.AppConfig.Headless && config.AppConfig.Cookies == "" {
		return crawler.Result{}, fmt.Errorf("HEADLESS=true requires COOKIES set for douyin")
	}

	if err := c.login(ctx); err != nil {
		return crawler.Result{}, err
	}

	req.Platform = "douyin"
	if req.Mode == "" {
		req.Mode = crawler.NormalizeMode(config.AppConfig.CrawlerType)
	}
	out := crawler.NewResult(req)

	msToken := c.getMsToken()

	var res crawler.Result
	var runErr error
	switch req.Mode {
	case crawler.ModeDetail:
		res, runErr = c.runDetailMode(ctx, req, msToken)
	case crawler.ModeCreator:
		res, runErr = c.runCreatorMode(ctx, req, msToken)
	case crawler.ModeSearch:
		res, runErr = c.runSearchMode(ctx, req, msToken)
	default:
		return crawler.Result{}, fmt.Errorf("douyin mode not implemented: %s (supported: search/detail/creator)", req.Mode)
	}
	res.StartedAt = out.StartedAt
	return res, runErr
}

func (c *DouyinCrawler) initBrowser(ctx context.Context) error {
	if err := playwright.Install(); err != nil {
		return fmt.Errorf("failed to install playwright: %v", err)
	}
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not launch playwright: %v", err)
	}
	c.pw = pw

	absDir, cleanup, err := browser.PrepareUserDataDir(config.AppConfig.UserDataDir, config.AppConfig.SaveLoginState, "douyin")
	if err != nil {
		return fmt.Errorf("prepare user data dir: %v", err)
	}
	c.cleanupUD = cleanup

	if config.AppConfig.EnableCDPMode {
		timeoutSec := config.AppConfig.BrowserLaunchTimeout
		if timeoutSec <= 0 {
			timeoutSec = 60
		}
		sess, err := browser.StartOrConnectCDP(ctx, pw, browser.CDPOptions{
			DebugPort:         config.AppConfig.CDPDebugPort,
			CustomBrowserPath: config.AppConfig.CustomBrowserPath,
			UserDataDir:       absDir,
			Headless:          config.AppConfig.CDPHeadless,
			LaunchTimeout:     time.Duration(timeoutSec) * time.Second,
		})
		if err == nil {
			c.cdpCmd = sess.Cmd
			c.cdpBrowser = sess.Browser
			c.browser = sess.Context
			c.page = sess.Page
			c.page.AddInitScript(playwright.Script{Content: playwright.String("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")})
			return nil
		}
		logger.Warn("cdp mode init failed; falling back to persistent context", "err", err)
	}

	launchOpts := playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(config.AppConfig.Headless),
		Channel:  playwright.String("chrome"),
		Viewport: &playwright.Size{Width: 1920, Height: 1080},
	}
	browserCtx, err := pw.Chromium.LaunchPersistentContext(absDir, launchOpts)
	if err != nil {
		browserCtx, err = pw.Chromium.LaunchPersistentContext(absDir, playwright.BrowserTypeLaunchPersistentContextOptions{
			Headless: playwright.Bool(config.AppConfig.Headless),
			Viewport: &playwright.Size{Width: 1920, Height: 1080},
		})
		if err != nil {
			return fmt.Errorf("could not launch browser: %v", err)
		}
	}
	c.browser = browserCtx
	pages := browserCtx.Pages()
	if len(pages) > 0 {
		c.page = pages[0]
	} else {
		page, err := browserCtx.NewPage()
		if err != nil {
			return err
		}
		c.page = page
	}
	c.page.AddInitScript(playwright.Script{Content: playwright.String("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")})
	return nil
}

func (c *DouyinCrawler) close() {
	if c.browser != nil {
		_ = c.browser.Close()
	}
	if c.cdpBrowser != nil {
		_ = c.cdpBrowser.Close()
	}
	if c.cdpCmd != nil && c.cdpCmd.Process != nil && config.AppConfig.AutoCloseBrowser {
		_ = c.cdpCmd.Process.Kill()
	}
	if c.pw != nil {
		_ = c.pw.Stop()
	}
	if c.cleanupUD != nil {
		c.cleanupUD()
	}
}

func (c *DouyinCrawler) login(ctx context.Context) error {
	if config.AppConfig.Cookies != "" {
		cookies := buildCookiesForDouyin(config.AppConfig.Cookies)
		if len(cookies) > 0 {
			_ = c.browser.AddCookies(cookies)
		}
	}
	if _, err := c.page.Goto("https://www.douyin.com"); err != nil {
		return err
	}
	if err := c.client.UpdateCookies(c.browser); err != nil {
		return err
	}
	if ok := c.isLoggedIn(); ok {
		return nil
	}
	logger.Info("not logged in; log in manually in browser window")
	timeoutSec := config.AppConfig.LoginWaitTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		_ = c.client.UpdateCookies(c.browser)
		if c.isLoggedIn() {
			time.Sleep(3 * time.Second)
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("login timed out after %ds", timeoutSec)
}

func (c *DouyinCrawler) isLoggedIn() bool {
	val, err := c.page.Evaluate("() => window.localStorage && window.localStorage.getItem('HasUserLogin')")
	if err == nil {
		if s, ok := val.(string); ok && s == "1" {
			return true
		}
	}
	cookies, err := c.browser.Cookies()
	if err != nil {
		return false
	}
	for _, ck := range cookies {
		if ck.Name == "LOGIN_STATUS" && ck.Value == "1" {
			return true
		}
	}
	return false
}

func buildCookiesForDouyin(cookieStr string) []playwright.OptionalCookie {
	if strings.TrimSpace(cookieStr) == "" {
		return nil
	}
	var out []playwright.OptionalCookie
	for _, cookieItem := range strings.Split(cookieStr, ";") {
		parts := strings.SplitN(strings.TrimSpace(cookieItem), "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := parts[1]
		if name == "" || value == "" {
			continue
		}
		out = append(out, playwright.OptionalCookie{
			Name:   name,
			Value:  value,
			Domain: playwright.String(".douyin.com"),
			Path:   playwright.String("/"),
		})
	}
	return out
}

func (c *DouyinCrawler) getMsToken() string {
	if c.page == nil {
		return ""
	}
	v, err := c.page.Evaluate("() => window.localStorage && window.localStorage.getItem('xmst')")
	if err != nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func (c *DouyinCrawler) runDetailMode(ctx context.Context, req crawler.Request, msToken string) (crawler.Result, error) {
	inputs := req.Inputs
	if len(inputs) == 0 {
		inputs = config.AppConfig.DouyinSpecifiedNoteUrls
	}
	logger.Info("running detail mode", "inputs", len(inputs))
	if len(inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs (DY_SPECIFIED_NOTE_URL_LIST)")
	}

	var ids []string
	for _, input := range inputs {
		awemeID := resolveAwemeID(input)
		if awemeID == "" {
			logger.Warn("skip invalid douyin url/id", "value", input)
			continue
		}
		ids = append(ids, awemeID)
	}

	out := crawler.NewResult(req)
	r := c.processAwemeIDs(ctx, ids, msToken, req.Concurrency)
	out.Processed = r.Processed
	out.Succeeded = r.Succeeded
	out.Failed = r.Failed
	out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)
	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *DouyinCrawler) runCreatorMode(ctx context.Context, req crawler.Request, msToken string) (crawler.Result, error) {
	inputs := req.Inputs
	if len(inputs) == 0 {
		inputs = config.AppConfig.DouyinCreatorIdList
	}
	logger.Info("running creator mode", "inputs", len(inputs))
	if len(inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs (DY_CREATOR_ID_LIST)")
	}

	limit := req.MaxNotes
	if limit == 0 {
		limit = config.AppConfig.CrawlerMaxNotesCount
	}
	out := crawler.NewResult(req)

	for _, input := range inputs {
		secUserID := ExtractSecUserID(input)
		if secUserID == "" {
			logger.Warn("skip invalid creator id/url", "value", input)
			continue
		}
		logger.Info("fetching creator profile", "creator_id", secUserID)
		profile, err := c.client.GetUserInfo(ctx, secUserID, msToken)
		if err == nil {
			_ = store.SaveCreatorProfile(secUserID, profile)
		} else {
			logger.Error("fetch creator profile failed", "creator_id", secUserID, "err", err)
		}

		maxCursor := ""
		hasMore := 1
		processed := 0
		for hasMore == 1 && (limit <= 0 || processed < limit) {
			resp, err := c.client.GetUserAwemePosts(ctx, secUserID, maxCursor, msToken)
			if err != nil {
				return crawler.Result{}, err
			}
			hasMore = resp.HasMore
			maxCursor = resp.MaxCursor

			var ids []string
			for _, aweme := range resp.AwemeList {
				id, _ := aweme["aweme_id"].(string)
				if id == "" {
					continue
				}
				ids = append(ids, id)
				processed++
				if limit > 0 && processed >= limit {
					break
				}
			}
			r := c.processAwemeIDs(ctx, ids, msToken, req.Concurrency)
			out.Succeeded += r.Succeeded
			out.Failed += r.Failed
			out.Processed += r.Processed
			out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)
			if config.AppConfig.CrawlerMaxSleepSec > 0 {
				time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
			}
		}
	}
	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *DouyinCrawler) runSearchMode(ctx context.Context, req crawler.Request, msToken string) (crawler.Result, error) {
	keywords := req.Keywords
	if len(keywords) == 0 {
		keywords = config.GetKeywords()
	}
	logger.Info("running search mode", "keywords", len(keywords))
	if len(keywords) == 0 {
		return crawler.Result{}, fmt.Errorf("empty keywords")
	}

	limitCount := 10
	maxNotes := req.MaxNotes
	if maxNotes == 0 {
		maxNotes = config.AppConfig.CrawlerMaxNotesCount
	}
	startPage := req.StartPage
	if startPage <= 0 {
		startPage = config.AppConfig.StartPage
	}
	if startPage <= 0 {
		startPage = 1
	}
	if maxNotes > 0 && maxNotes < limitCount {
		limitCount = maxNotes
	}

	out := crawler.NewResult(req)
	for _, keyword := range keywords {
		page := 0
		searchID := ""
		for maxNotes <= 0 || (page-startPage+1)*limitCount <= maxNotes {
			if page < startPage {
				page++
				continue
			}
			offset := page*limitCount - limitCount
			resp, err := c.client.SearchInfoByKeyword(ctx, keyword, offset, limitCount, searchID, msToken)
			if err != nil {
				return crawler.Result{}, err
			}
			if len(resp.Data) == 0 {
				break
			}
			searchID = resp.Extra.LogID

			var ids []string
			for _, item := range resp.Data {
				aweme := pickAwemeInfoFromSearchItem(item)
				if aweme == nil {
					continue
				}
				id, _ := aweme["aweme_id"].(string)
				if id == "" {
					continue
				}
				ids = append(ids, id)
			}
			r := c.processAwemeIDs(ctx, ids, msToken, req.Concurrency)
			out.Succeeded += r.Succeeded
			out.Failed += r.Failed
			out.Processed += r.Processed
			out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)
			page++
			if config.AppConfig.CrawlerMaxSleepSec > 0 {
				time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
			}
		}
	}
	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func resolveAwemeID(input string) string {
	awemeID := ExtractAwemeID(input)
	if awemeID != "" {
		return awemeID
	}
	if strings.Contains(input, "v.douyin.com") {
		finalURL, err := ResolveShortURL(input)
		if err == nil {
			return ExtractAwemeID(finalURL)
		}
	}
	return ""
}

func (c *DouyinCrawler) processOneAweme(ctx context.Context, awemeID string, msToken string) error {
	logger.Info("fetching aweme", "aweme_id", awemeID)
	detail, err := c.client.GetVideoByID(ctx, awemeID, msToken, "")
	if err != nil {
		return err
	}

	if config.AppConfig.SaveDataOption == "csv" {
		var rec VideoDetail
		b, _ := json.Marshal(detail)
		_ = json.Unmarshal(b, &rec)
		if err := store.SaveNoteDetail(awemeID, &rec); err != nil {
			return err
		}
	} else {
		if err := store.SaveNoteDetail(awemeID, detail); err != nil {
			return err
		}
	}

	if config.AppConfig.EnableGetComments {
		comments, err := fetchAllAwemeComments(
			ctx,
			c.client,
			awemeID,
			config.AppConfig.CrawlerMaxComments,
			config.AppConfig.CrawlerMaxSleepSec,
			msToken,
			config.AppConfig.EnableGetSubComments,
		)
		if err != nil {
			logger.Error("fetch comments failed", "aweme_id", awemeID, "err", err)
		} else {
			if config.AppConfig.SaveDataOption == "csv" {
				items := make([]any, 0, len(comments))
				for i := range comments {
					items = append(items, &comments[i])
				}
				_, err := store.AppendUniqueCommentsCSV(
					awemeID,
					items,
					func(item any) (string, error) { return item.(*Comment).CID, nil },
					(&Comment{}).CSVHeader(),
					func(item any) ([]string, error) { return item.(*Comment).ToCSV(), nil },
				)
				if err != nil {
					logger.Error("save comments csv failed", "aweme_id", awemeID, "err", err)
				}
			} else if config.AppConfig.SaveDataOption == "xlsx" {
				items := make([]any, 0, len(comments))
				for i := range comments {
					items = append(items, &comments[i])
				}
				_, err := store.AppendUniqueCommentsXLSX(
					awemeID,
					items,
					func(item any) (string, error) { return item.(*Comment).CID, nil },
					(&Comment{}).CSVHeader(),
					func(item any) ([]string, error) { return item.(*Comment).ToCSV(), nil },
				)
				if err != nil {
					logger.Error("save comments xlsx failed", "aweme_id", awemeID, "err", err)
				}
			} else {
				items := make([]any, 0, len(comments))
				for i := range comments {
					items = append(items, comments[i])
				}
				_, err := store.AppendUniqueCommentsJSONL(
					awemeID,
					items,
					func(item any) (string, error) { return item.(Comment).CID, nil },
				)
				if err != nil {
					logger.Error("save comments failed", "aweme_id", awemeID, "err", err)
				}
			}
		}
	}

	if config.AppConfig.EnableGetMedias {
		headers := map[string]string{
			"User-Agent": c.client.UserAgent(),
			"Referer":    fmt.Sprintf("https://www.douyin.com/video/%s", awemeID),
		}
		if ck := c.client.CookieHeader(); ck != "" {
			headers["Cookie"] = ck
		}

		var rec VideoDetail
		b, _ := json.Marshal(detail)
		_ = json.Unmarshal(b, &rec)

		var urls []string
		var filenames []string

		if len(rec.Video.PlayAddr.URLList) > 0 {
			urls = append(urls, rec.Video.PlayAddr.URLList[0])
			filenames = append(filenames, fmt.Sprintf("%s_video.mp4", awemeID))
		}

		coverList := rec.Video.OriginCover.URLList
		if len(coverList) == 0 {
			coverList = rec.Video.Cover.URLList
		}
		for i, u := range coverList {
			if u == "" {
				continue
			}
			urls = append(urls, u)
			filenames = append(filenames, fmt.Sprintf("%s_cover_%d.jpg", awemeID, i))
			if i >= 2 {
				break
			}
		}

		if len(urls) > 0 {
			noteDownloader := downloader.NewDownloader(store.NoteMediaDir(awemeID))
			noteDownloader.BatchDownloadWithHeaders(urls, filenames, headers)
		}
	}

	if config.AppConfig.CrawlerMaxSleepSec > 0 {
		time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
	}
	return nil
}

func (c *DouyinCrawler) processAwemeIDs(ctx context.Context, ids []string, msToken string, concurrency int) crawler.ItemResult {
	if len(ids) == 0 {
		return crawler.ItemResult{}
	}
	seen := make(map[string]struct{}, len(ids))
	uniq := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	n := concurrency
	if n <= 0 {
		n = config.AppConfig.MaxConcurrencyNum
	}
	if n < 1 {
		n = 1
	}
	return crawler.ForEachLimit(ctx, uniq, n, func(ctx context.Context, id string) error {
		if err := c.processOneAweme(ctx, id, msToken); err != nil {
			logger.Error("process failed", "id", id, "err", err)
			return err
		}
		return nil
	})
}
