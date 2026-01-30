package douyin

import (
	"context"
	"encoding/json"
	"fmt"
	"media-crawler-go/internal/browser"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/downloader"
	"media-crawler-go/internal/proxy"
	"media-crawler-go/internal/store"
	"os"
	"path/filepath"
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

	proxyPool *proxy.Pool
}

func NewCrawler() *DouyinCrawler {
	return &DouyinCrawler{}
}

func (c *DouyinCrawler) Start(ctx context.Context) error {
	fmt.Println("DouyinCrawler started...")

	if config.AppConfig.EnableIPProxy {
		provider, err := proxy.NewProvider(config.AppConfig.IPProxyProviderName)
		if err != nil {
			fmt.Printf("Warning: proxy provider init failed: %v\n", err)
		} else {
			c.proxyPool = proxy.NewPool(provider, config.AppConfig.IPProxyPoolCount)
		}
	}

	if err := c.initBrowser(ctx); err != nil {
		return err
	}
	defer c.close()

	signer, err := NewSigner()
	if err != nil {
		return err
	}
	c.signer = signer

	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	c.client = NewClient(c.signer, userAgent)
	if c.proxyPool != nil {
		c.client.InitProxyPool(c.proxyPool)
	}

	if config.AppConfig.Headless && config.AppConfig.Cookies == "" {
		return fmt.Errorf("HEADLESS=true requires COOKIES set for douyin")
	}

	if err := c.login(ctx); err != nil {
		return err
	}

	switch config.AppConfig.CrawlerType {
	case "detail":
		return c.runDetailMode(ctx)
	default:
		return fmt.Errorf("douyin crawler type not implemented: %s (supported: detail)", config.AppConfig.CrawlerType)
	}
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

	userDataDir := config.AppConfig.UserDataDir
	if userDataDir == "" {
		userDataDir = "browser_data/douyin"
	}
	absDir, err := filepath.Abs(userDataDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return err
	}

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
			c.cdpBrowser = sess.Browser
			c.browser = sess.Context
			c.page = sess.Page
			c.page.AddInitScript(playwright.Script{Content: playwright.String("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")})
			return nil
		}
		fmt.Printf("Warning: CDP mode init failed, falling back to persistent context: %v\n", err)
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
	if c.pw != nil {
		_ = c.pw.Stop()
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
	fmt.Println("Not logged in. Please log in manually in the browser window.")
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

func (c *DouyinCrawler) runDetailMode(ctx context.Context) error {
	inputs := config.AppConfig.DouyinSpecifiedNoteUrls
	fmt.Printf("Running detail mode with %d inputs\n", len(inputs))
	if len(inputs) == 0 {
		return fmt.Errorf("DY_SPECIFIED_NOTE_URL_LIST is empty")
	}

	msToken := ""
	v, err := c.page.Evaluate("() => window.localStorage && window.localStorage.getItem('xmst')")
	if err == nil {
		if s, ok := v.(string); ok {
			msToken = s
		}
	}

	for _, input := range inputs {
		awemeID := ExtractAwemeID(input)
		if awemeID == "" && strings.Contains(input, "v.douyin.com") {
			finalURL, err := ResolveShortURL(input)
			if err == nil {
				awemeID = ExtractAwemeID(finalURL)
			}
		}
		if awemeID == "" {
			fmt.Printf("Skip invalid douyin url/id: %s\n", input)
			continue
		}

		fmt.Printf("Fetching aweme_id: %s\n", awemeID)
		detail, err := c.client.GetVideoByID(ctx, awemeID, msToken, "")
		if err != nil {
			fmt.Printf("Failed to fetch detail %s: %v\n", awemeID, err)
			continue
		}

		if config.AppConfig.SaveDataOption == "csv" {
			var rec VideoDetail
			b, _ := json.Marshal(detail)
			_ = json.Unmarshal(b, &rec)
			if err := store.SaveNoteDetail(awemeID, &rec); err != nil {
				fmt.Printf("Failed to save CSV %s: %v\n", awemeID, err)
			}
		} else {
			if err := store.SaveNoteDetail(awemeID, detail); err != nil {
				fmt.Printf("Failed to save %s: %v\n", awemeID, err)
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
				fmt.Printf("Failed to fetch comments %s: %v\n", awemeID, err)
			} else {
				if config.AppConfig.SaveDataOption == "csv" {
					items := make([]any, 0, len(comments))
					for i := range comments {
						items = append(items, &comments[i])
					}
					_, err := store.AppendUniqueCSV(
						store.NoteDir(awemeID),
						"comments.csv",
						"comments.idx",
						items,
						func(item any) (string, error) { return item.(*Comment).CID, nil },
						(&Comment{}).CSVHeader(),
						func(item any) ([]string, error) { return item.(*Comment).ToCSV(), nil },
					)
					if err != nil {
						fmt.Printf("Failed to save comments csv %s: %v\n", awemeID, err)
					}
				} else {
					items := make([]any, 0, len(comments))
					for i := range comments {
						items = append(items, comments[i])
					}
					_, err := store.AppendUniqueJSONL(
						store.NoteDir(awemeID),
						"comments.jsonl",
						"comments.idx",
						items,
						func(item any) (string, error) { return item.(Comment).CID, nil },
					)
					if err != nil {
						fmt.Printf("Failed to save comments %s: %v\n", awemeID, err)
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

		time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
	}
	return nil
}
