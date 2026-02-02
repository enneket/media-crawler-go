package xhs

import (
	"context"
	"fmt"
	"media-crawler-go/internal/browser"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/downloader"
	"media-crawler-go/internal/logger"
	"media-crawler-go/internal/proxy"
	"media-crawler-go/internal/store"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type XhsCrawler struct {
	pw         *playwright.Playwright
	browser    playwright.BrowserContext
	page       playwright.Page
	client     *Client
	signer     *Signer
	cdpBrowser playwright.Browser
	cdpCmd     *exec.Cmd
	proxyPool  *proxy.Pool
	proxy      *proxy.Proxy
}

func NewCrawler() *XhsCrawler {
	return &XhsCrawler{}
}

func (c *XhsCrawler) Run(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	logger.Info("xhs crawler started")

	if config.AppConfig.EnableIPProxy {
		provider, err := proxy.NewProvider(config.AppConfig.IPProxyProviderName)
		if err != nil {
			logger.Warn("proxy provider init failed", "err", err)
		} else {
			pool := proxy.NewPool(provider, config.AppConfig.IPProxyPoolCount)
			p, err := pool.GetOrRefresh(ctx)
			if err != nil {
				logger.Warn("proxy pool fetch failed", "err", err)
			} else {
				c.proxyPool = pool
				c.proxy = &p
			}
		}
	}

	if err := c.initBrowser(); err != nil {
		return crawler.Result{}, err
	}
	defer c.close()

	c.signer = NewSigner(c.page)
	c.client = NewClient(c.signer)
	if c.proxyPool != nil {
		c.client.InitProxyPool(c.proxyPool)
	}

	if config.AppConfig.Headless && config.AppConfig.LoginType != "cookie" && config.AppConfig.Cookies == "" {
		return crawler.Result{}, fmt.Errorf("HEADLESS=true requires LOGIN_TYPE=cookie and COOKIES set")
	}

	if err := c.login(ctx); err != nil {
		return crawler.Result{}, err
	}

	logger.Info("login successful")

	req.Platform = "xhs"
	if req.Mode == "" {
		req.Mode = crawler.NormalizeMode(config.AppConfig.CrawlerType)
	}
	out := crawler.NewResult(req)

	var res crawler.Result
	var err error
	switch req.Mode {
	case crawler.ModeSearch:
		res, err = c.runSearchMode(ctx, req)
	case crawler.ModeDetail:
		res, err = c.runDetailMode(ctx, req)
	case crawler.ModeCreator:
		res, err = c.runCreatorMode(ctx, req)
	default:
		return crawler.Result{}, fmt.Errorf("unknown mode: %s", req.Mode)
	}
	res.StartedAt = out.StartedAt
	return res, err
}

func (c *XhsCrawler) login(ctx context.Context) error {
	loginType := strings.ToLower(strings.TrimSpace(config.AppConfig.LoginType))

	if config.AppConfig.Cookies != "" {
		cookies := buildCookies(config.AppConfig.Cookies)
		if len(cookies) > 0 {
			if err := c.browser.AddCookies(cookies); err != nil {
				logger.Warn("failed to add cookies", "err", err)
			}
		}
	}

	if _, err := c.page.Goto("https://www.xiaohongshu.com"); err != nil {
		return fmt.Errorf("failed to goto homepage: %v", err)
	}
	if err := c.client.UpdateCookies(c.browser); err != nil {
		return fmt.Errorf("failed to update cookies: %v", err)
	}

	if c.client.Pong() {
		return nil
	}

	if loginType == "cookie" {
		if config.AppConfig.Cookies == "" {
			return fmt.Errorf("LOGIN_TYPE=cookie requires COOKIES set")
		}
		return fmt.Errorf("cookie login failed (Pong check failed); refresh cookies and retry")
	}
	if loginType != "qrcode" && loginType != "phone" && loginType != "" {
		return fmt.Errorf("invalid LOGIN_TYPE: %s (supported: qrcode|phone|cookie)", loginType)
	}

	logger.Info("not logged in; complete login in browser window", "login_type", loginType)
	timeoutSec := config.AppConfig.LoginWaitTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)

	for time.Now().Before(deadline) {
		if err := c.client.UpdateCookies(c.browser); err == nil && c.client.Pong() {
			time.Sleep(5 * time.Second)
			return nil
		}
		content, err := c.page.Content()
		if err == nil && strings.Contains(content, "请通过验证") {
			logger.Warn("captcha detected; verify manually in browser window")
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("login timed out after %ds", timeoutSec)
}

func (c *XhsCrawler) runSearchMode(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	keywords := req.Keywords
	if len(keywords) == 0 {
		keywords = config.GetKeywords()
	}
	if len(keywords) == 0 {
		return crawler.Result{}, fmt.Errorf("empty keywords")
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

	type noteTask struct {
		NoteID     string
		XsecSource string
		XsecToken  string
		Nickname   string
		Title      string
	}

	out := crawler.NewResult(req)
	seen := make(map[string]struct{})

	for _, keyword := range keywords {
		logger.Info("searching keyword", "keyword", keyword)
		page := startPage
		for {
			select {
			case <-ctx.Done():
				out.FinishedAt = time.Now().Unix()
				return out, ctx.Err()
			default:
			}

			res, err := c.client.GetNoteByKeyword(keyword, page)
			if err != nil {
				logger.Error("search failed", "page", page, "err", err)
				break
			}
			if len(res.Items) == 0 {
				logger.Info("no items found", "page", page)
				break
			}
			logger.Info("search page items", "page", page, "items", len(res.Items))

			tasks := make([]noteTask, 0, len(res.Items))
			for _, item := range res.Items {
				if maxNotes > 0 && out.Processed >= maxNotes {
					break
				}
				noteID := item.Id
				if noteID == "" {
					noteID = item.NoteCard.NoteId
				}
				if noteID == "" {
					continue
				}
				if _, ok := seen[noteID]; ok {
					continue
				}
				seen[noteID] = struct{}{}
				tasks = append(tasks, noteTask{
					NoteID:     noteID,
					XsecSource: item.XsecSource,
					XsecToken:  item.XsecToken,
					Nickname:   item.NoteCard.User.Nickname,
					Title:      item.NoteCard.Title,
				})
			}

			r := crawler.ForEachLimit(ctx, tasks, concurrency, func(ctx context.Context, t noteTask) error {
				logger.Info("note", "nickname", t.Nickname, "title", t.Title, "note_id", t.NoteID)
				return c.processNote(t.NoteID, t.XsecSource, t.XsecToken)
			})
			out.Succeeded += r.Succeeded
			out.Failed += r.Failed
			out.Processed += r.Processed
			out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)

			if maxNotes > 0 && out.Processed >= maxNotes {
				break
			}
			if !res.HasMore {
				break
			}

			page++
			if config.AppConfig.CrawlerMaxSleepSec > 0 {
				time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
			}
		}
	}
	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *XhsCrawler) runDetailMode(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	inputs := req.Inputs
	if len(inputs) == 0 {
		inputs = config.AppConfig.XhsSpecifiedNoteUrls
	}
	logger.Info("running detail mode", "urls", len(inputs))
	if len(inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs (XHS_SPECIFIED_NOTE_URL_LIST)")
	}
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = config.AppConfig.MaxConcurrencyNum
	}
	if concurrency < 1 {
		concurrency = 1
	}

	out := crawler.NewResult(req)
	r := crawler.ForEachLimit(ctx, inputs, concurrency, func(ctx context.Context, input string) error {
		noteID := extractNoteId(input)
		if noteID == "" {
			logger.Warn("invalid url", "url", input)
			return crawler.Error{Kind: crawler.ErrorKindInvalidInput, Platform: req.Platform, URL: input, Msg: "invalid xhs url"}
		}
		logger.Info("processing note", "note_id", noteID)
		return c.processNote(noteID, "", "")
	})
	out.Processed = r.Processed
	out.Succeeded = r.Succeeded
	out.Failed = r.Failed
	out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)
	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *XhsCrawler) runCreatorMode(ctx context.Context, req crawler.Request) (crawler.Result, error) {
	inputs := req.Inputs
	if len(inputs) == 0 {
		inputs = config.AppConfig.XhsCreatorIdList
	}
	logger.Info("running creator mode", "creators", len(inputs))
	if len(inputs) == 0 {
		return crawler.Result{}, fmt.Errorf("empty inputs (XHS_CREATOR_ID_LIST)")
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
	for _, userID := range inputs {
		select {
		case <-ctx.Done():
			out.FinishedAt = time.Now().Unix()
			return out, ctx.Err()
		default:
		}
		creatorID := ExtractCreatorID(userID)
		if creatorID == "" {
			logger.Warn("invalid creator id/url", "value", userID)
			continue
		}
		logger.Info("processing creator", "creator_id", creatorID)

		if err := c.fetchAndSaveCreator(creatorID); err != nil {
			logger.Error("fetch creator info failed", "creator_id", creatorID, "err", err)
		}

		processed := 0
		seen := make(map[string]struct{})
		cursor := ""
		for {
			select {
			case <-ctx.Done():
				out.FinishedAt = time.Now().Unix()
				return out, ctx.Err()
			default:
			}
			res, err := c.client.GetNotesByCreator(creatorID, cursor)
			if err != nil {
				logger.Error("get notes for creator failed", "creator_id", creatorID, "err", err)
				break
			}

			logger.Info("creator notes", "creator_id", creatorID, "notes", len(res.Notes))
			tasks := make([]Note, 0, len(res.Notes))
			for _, note := range res.Notes {
				if maxNotes > 0 && processed >= maxNotes {
					break
				}
				if note.NoteId == "" {
					continue
				}
				if _, ok := seen[note.NoteId]; ok {
					continue
				}
				seen[note.NoteId] = struct{}{}
				processed++
				tasks = append(tasks, note)
			}

			r := crawler.ForEachLimit(ctx, tasks, concurrency, func(ctx context.Context, note Note) error {
				logger.Info("note", "nickname", note.User.Nickname, "title", note.Title, "note_id", note.NoteId)
				return c.processNote(note.NoteId, note.XsecSource, note.XsecToken)
			})
			out.Succeeded += r.Succeeded
			out.Failed += r.Failed
			out.Processed += r.Processed
			out.FailureKinds = crawler.MergeFailureKinds(out.FailureKinds, r.FailureKinds)

			if maxNotes > 0 && processed >= maxNotes {
				break
			}
			if !res.HasMore || res.Cursor == "" {
				break
			}
			cursor = res.Cursor
			if config.AppConfig.CrawlerMaxSleepSec > 0 {
				time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
			}
		}
	}

	out.FinishedAt = time.Now().Unix()
	return out, nil
}

func (c *XhsCrawler) fetchAndSaveCreator(userID string) error {
	if c.browser == nil {
		return fmt.Errorf("browser context not initialized")
	}

	page, err := c.browser.NewPage()
	if err != nil {
		return err
	}
	defer page.Close()

	page.AddInitScript(playwright.Script{Content: playwright.String("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")})

	url := fmt.Sprintf("https://www.xiaohongshu.com/user/profile/%s", userID)
	if _, err := page.Goto(url); err != nil {
		return err
	}
	html, err := page.Content()
	if err != nil {
		return err
	}
	userPageData, err := ExtractCreatorUserPageData(html)
	if err != nil {
		return err
	}
	record, err := BuildCreatorRecord(userID, userPageData)
	if err != nil {
		return err
	}
	return store.SaveCreator(userID, record)
}

func (c *XhsCrawler) processNote(noteId, xsecSource, xsecToken string) error {
	logger.Info("fetching note detail", "note_id", noteId)
	noteDetail, err := c.client.GetNoteById(noteId, xsecSource, xsecToken)
	if err != nil {
		logger.Error("get note detail failed", "note_id", noteId, "err", err)
		return err
	}

	if err := store.SaveNoteDetail(noteId, &noteDetail); err != nil {
		logger.Error("save note failed", "note_id", noteId, "err", err)
		return err
	}
	logger.Info("note saved", "note_id", noteId)

	// Download Medias
	if config.AppConfig.EnableGetMedias {
		var urls []string
		var filenames []string

		// Images
		for i, img := range noteDetail.ImageList {
			url := img.UrlDefault
			if url == "" {
				url = img.Url
			}
			if url != "" {
				urls = append(urls, url)
				ext := "jpg"
				if strings.Contains(url, ".webp") {
					ext = "webp"
				} else if strings.Contains(url, ".png") {
					ext = "png"
				}
				filenames = append(filenames, fmt.Sprintf("%s_%d.%s", noteDetail.NoteId, i, ext))
			}
		}

		// Video
		if noteDetail.Type == "video" && noteDetail.Video.Media.Stream != nil {
			if streams, ok := noteDetail.Video.Media.Stream["h264"]; ok && len(streams) > 0 {
				url := streams[0].MasterUrl
				if url != "" {
					urls = append(urls, url)
					filenames = append(filenames, fmt.Sprintf("%s_video.mp4", noteDetail.NoteId))
				}
			}
		}

		if len(urls) > 0 {
			logger.Info("downloading media files", "note_id", noteId, "count", len(urls))
			noteDownloader := downloader.NewDownloader(store.NoteMediaDir(noteId))
			noteDownloader.BatchDownload(urls, filenames)
		}
	}

	// Get Comments
	if config.AppConfig.EnableGetComments {
		token := xsecToken
		if token == "" {
			token = noteDetail.XsecToken
		}

		logger.Info("fetching comments", "note_id", noteId)
		comments, err := c.fetchAllComments(noteId, token)
		if err != nil {
			logger.Error("get comments failed", "note_id", noteId, "err", err)
		} else {
			logger.Info("comments fetched", "note_id", noteId, "comments", len(comments))
			if config.AppConfig.SaveDataOption == "csv" {
				items := make([]any, 0, len(comments))
				for i := range comments {
					comment := comments[i]
					comment.NoteId = noteId
					items = append(items, &comment)
				}
				_, err := store.AppendUniqueCommentsCSV(
					noteId,
					items,
					func(item any) (string, error) { return item.(*Comment).Id, nil },
					(&Comment{}).CSVHeader(),
					func(item any) ([]string, error) { return item.(*Comment).ToCSV(), nil },
				)
				if err != nil {
					logger.Error("save comments csv failed", "note_id", noteId, "err", err)
				}
			} else {
				items := make([]any, 0, len(comments))
				for i := range comments {
					comments[i].NoteId = noteId
					items = append(items, comments[i])
				}
				_, err := store.AppendUniqueCommentsJSONL(
					noteId,
					items,
					func(item any) (string, error) { return item.(Comment).Id, nil },
				)
				if err != nil {
					logger.Error("save comments failed", "note_id", noteId, "err", err)
				}
			}
		}
	}
	return nil
}

func (c *XhsCrawler) fetchAllComments(noteId, xsecToken string) ([]Comment, error) {
	maxCount := config.AppConfig.CrawlerMaxComments
	if maxCount == 0 {
		maxCount = -1
	}

	var all []Comment
	cursor := ""
	hasMore := true
	for hasMore && (maxCount < 0 || len(all) < maxCount) {
		res, err := c.client.GetNoteComments(noteId, xsecToken, cursor)
		if err != nil {
			return all, err
		}

		hasMore = res.HasMore
		cursor = res.Cursor

		pageComments := res.Comments
		if maxCount >= 0 && len(all)+len(pageComments) > maxCount {
			pageComments = pageComments[:maxCount-len(all)]
		}
		all = append(all, pageComments...)

		if config.AppConfig.EnableGetSubComments && (maxCount < 0 || len(all) < maxCount) {
			remaining := -1
			if maxCount >= 0 {
				remaining = maxCount - len(all)
			}
			sub := c.fetchSubComments(noteId, xsecToken, pageComments, remaining)
			if remaining >= 0 && len(sub) > remaining {
				sub = sub[:remaining]
			}
			all = append(all, sub...)
		}

		if hasMore && cursor != "" && (maxCount < 0 || len(all) < maxCount) {
			time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
		}
	}

	return all, nil
}

func (c *XhsCrawler) fetchSubComments(noteId, xsecToken string, comments []Comment, remaining int) []Comment {
	if !config.AppConfig.EnableGetSubComments {
		return nil
	}

	var out []Comment
	for _, root := range comments {
		if remaining >= 0 && len(out) >= remaining {
			break
		}

		if len(root.SubComments) > 0 {
			for _, sub := range root.SubComments {
				out = append(out, sub)
				if remaining >= 0 && len(out) >= remaining {
					break
				}
			}
		}

		if !root.SubCommentHasMore {
			continue
		}
		if root.Id == "" {
			continue
		}

		cursor := root.SubCommentCursor
		hasMore := true
		for hasMore {
			if remaining >= 0 && len(out) >= remaining {
				return out
			}

			res, err := c.client.GetNoteSubComments(noteId, root.Id, xsecToken, cursor, 10)
			if err != nil {
				break
			}
			hasMore = res.HasMore
			cursor = res.Cursor
			if len(res.Comments) == 0 {
				break
			}
			for _, sub := range res.Comments {
				out = append(out, sub)
				if remaining >= 0 && len(out) >= remaining {
					return out
				}
			}

			if hasMore && cursor != "" {
				time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
			} else {
				break
			}
		}
	}
	return out
}

func extractNoteId(url string) string {
	// Simple regex or split
	// https://www.xiaohongshu.com/explore/64a...
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		// Remove query params
		if idx := strings.Index(last, "?"); idx != -1 {
			last = last[:idx]
		}
		return last
	}
	return ""
}

func (c *XhsCrawler) initBrowser() error {
	err := playwright.Install()
	if err != nil {
		return fmt.Errorf("failed to install playwright: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("could not launch playwright: %v", err)
	}
	c.pw = pw

	userDataDir, err := filepath.Abs(config.AppConfig.UserDataDir)
	if err != nil {
		return fmt.Errorf("could not resolve absolute path for user data dir: %v", err)
	}

	if _, err := os.Stat(userDataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(userDataDir, 0755); err != nil {
			return fmt.Errorf("could not create user data dir: %v", err)
		}
	}

	if config.AppConfig.EnableCDPMode {
		timeoutSec := config.AppConfig.BrowserLaunchTimeout
		if timeoutSec <= 0 {
			timeoutSec = 60
		}
		proxyServer := ""
		if c.proxy != nil {
			proxyServer = c.proxy.ChromeProxyServer()
		}
		sess, err := browser.StartOrConnectCDP(context.Background(), pw, browser.CDPOptions{
			DebugPort:         config.AppConfig.CDPDebugPort,
			CustomBrowserPath: config.AppConfig.CustomBrowserPath,
			UserDataDir:       userDataDir,
			Headless:          config.AppConfig.CDPHeadless,
			ProxyServer:       proxyServer,
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
	if c.proxy != nil {
		server := c.proxy.ChromeProxyServer()
		launchOpts.Proxy = &playwright.Proxy{
			Server: server,
		}
		if c.proxy.User != "" {
			launchOpts.Proxy.Username = playwright.String(c.proxy.User)
		}
		if c.proxy.Password != "" {
			launchOpts.Proxy.Password = playwright.String(c.proxy.Password)
		}
	}

	browser, err := pw.Chromium.LaunchPersistentContext(userDataDir, launchOpts)
	if err != nil {
		fallbackOpts := playwright.BrowserTypeLaunchPersistentContextOptions{
			Headless: playwright.Bool(config.AppConfig.Headless),
			Viewport: &playwright.Size{Width: 1920, Height: 1080},
		}
		if c.proxy != nil {
			server := c.proxy.ChromeProxyServer()
			fallbackOpts.Proxy = &playwright.Proxy{
				Server: server,
			}
			if c.proxy.User != "" {
				fallbackOpts.Proxy.Username = playwright.String(c.proxy.User)
			}
			if c.proxy.Password != "" {
				fallbackOpts.Proxy.Password = playwright.String(c.proxy.Password)
			}
		}
		browser, err = pw.Chromium.LaunchPersistentContext(userDataDir, fallbackOpts)
		if err != nil {
			return fmt.Errorf("could not launch browser: %v", err)
		}
	}
	c.browser = browser

	pages := browser.Pages()
	if len(pages) > 0 {
		c.page = pages[0]
	} else {
		page, err := browser.NewPage()
		if err != nil {
			return fmt.Errorf("could not create page: %v", err)
		}
		c.page = page
	}

	c.page.AddInitScript(playwright.Script{Content: playwright.String("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")})

	return nil
}

func (c *XhsCrawler) close() {
	if c.browser != nil {
		c.browser.Close()
	}
	if c.cdpBrowser != nil {
		c.cdpBrowser.Close()
	}
	if c.cdpCmd != nil && c.cdpCmd.Process != nil && config.AppConfig.AutoCloseBrowser {
		_ = c.cdpCmd.Process.Kill()
	}
	if c.pw != nil {
		c.pw.Stop()
	}
}
