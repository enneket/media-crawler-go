package xhs

import (
	"context"
	"fmt"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/downloader"
	"media-crawler-go/internal/store"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type XhsCrawler struct {
	pw          *playwright.Playwright
	browser     playwright.BrowserContext
	page        playwright.Page
	client      *Client
	signer      *Signer
	downloader  *downloader.Downloader
}

func NewCrawler() *XhsCrawler {
	return &XhsCrawler{}
}

func (c *XhsCrawler) Start(ctx context.Context) error {
	fmt.Println("XhsCrawler started...")

	if err := c.initBrowser(); err != nil {
		return err
	}
	defer c.close()

	c.signer = NewSigner(c.page)
	c.client = NewClient(c.signer)
	c.downloader = downloader.NewDownloader("data/xhs/media")

	// Inject cookies from config if present
	if config.AppConfig.Cookies != "" {
		cookies := make([]playwright.OptionalCookie, 0)
		for _, cookieStr := range strings.Split(config.AppConfig.Cookies, ";") {
			parts := strings.SplitN(strings.TrimSpace(cookieStr), "=", 2)
			if len(parts) == 2 {
				cookies = append(cookies, playwright.OptionalCookie{
					Name:   parts[0],
					Value:  parts[1],
					Domain: playwright.String(".xiaohongshu.com"),
					Path:   playwright.String("/"),
				})
			}
		}
		if len(cookies) > 0 {
			if err := c.browser.AddCookies(cookies); err != nil {
				fmt.Printf("Warning: failed to add cookies: %v\n", err)
			}
		}
	}

	// Go to homepage
	if _, err := c.page.Goto("https://www.xiaohongshu.com"); err != nil {
		return fmt.Errorf("failed to goto homepage: %v", err)
	}

	// Update cookies
	if err := c.client.UpdateCookies(c.browser); err != nil {
		return fmt.Errorf("failed to update cookies: %v", err)
	}

	// Check login
	if !c.client.Pong() {
		fmt.Println("Not logged in. Please log in manually in the browser window.")
		time.Sleep(5 * time.Second)
		if err := c.client.UpdateCookies(c.browser); err != nil {
			return err
		}
		if !c.client.Pong() {
			return fmt.Errorf("login failed or timed out")
		}
	}
	fmt.Println("Login successful!")

	switch config.AppConfig.CrawlerType {
	case "search":
		return c.runSearchMode()
	case "detail":
		return c.runDetailMode()
	default:
		return fmt.Errorf("unknown crawler type: %s", config.AppConfig.CrawlerType)
	}
}

func (c *XhsCrawler) runSearchMode() error {
	keywords := config.GetKeywords()
	for _, keyword := range keywords {
		fmt.Printf("Searching for keyword: %s\n", keyword)
		res, err := c.client.GetNoteByKeyword(keyword, 1)
		if err != nil {
			fmt.Printf("Search failed: %v\n", err)
			continue
		}

		fmt.Printf("Found %d items\n", len(res.Items))
		for _, item := range res.Items {
			noteId := item.Id
			if noteId == "" {
				noteId = item.NoteCard.NoteId
			}
			fmt.Printf("- [%s] %s (ID: %s)\n", item.NoteCard.User.Nickname, item.NoteCard.Title, noteId)
			c.processNote(noteId, item.XsecSource, item.XsecToken)
			
			// Random sleep between notes
			time.Sleep(time.Duration(1+time.Now().Unix()%2) * time.Second)
		}
		time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
	}
	return nil
}

func (c *XhsCrawler) runDetailMode() error {
	urls := config.AppConfig.XhsSpecifiedNoteUrls
	fmt.Printf("Running detail mode with %d urls\n", len(urls))
	for _, url := range urls {
		noteId := extractNoteId(url)
		if noteId == "" {
			fmt.Printf("Invalid URL: %s\n", url)
			continue
		}
		fmt.Printf("Processing note ID: %s\n", noteId)
		c.processNote(noteId, "", "")
		time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
	}
	return nil
}

func (c *XhsCrawler) processNote(noteId, xsecSource, xsecToken string) {
	fmt.Printf("  Fetching detail for note %s...\n", noteId)
	noteDetail, err := c.client.GetNoteById(noteId, xsecSource, xsecToken)
	if err != nil {
		fmt.Printf("  Failed to get note detail: %v\n", err)
		return
	}

	// Save Note
	if err := store.SaveNote(noteDetail); err != nil {
		fmt.Printf("  Failed to save note: %v\n", err)
	} else {
		fmt.Printf("  Note saved.\n")
	}

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
			fmt.Printf("  Downloading %d media files...\n", len(urls))
			c.downloader.BatchDownload(urls, filenames)
		}
	}

	// Get Comments
	if config.AppConfig.EnableGetComments {
		// If we don't have token (e.g. detail mode), GetNoteById response contains token?
		// Note struct has XsecToken field.
		token := xsecToken
		if token == "" {
			token = noteDetail.XsecToken
		}
		
		fmt.Printf("  Fetching comments for note %s...\n", noteId)
		commentsRes, err := c.client.GetNoteComments(noteId, token, "")
		if err != nil {
			fmt.Printf("  Failed to get comments: %v\n", err)
		} else {
			fmt.Printf("  Found %d comments\n", len(commentsRes.Comments))
			if config.AppConfig.SaveDataOption == "csv" {
				for _, comment := range commentsRes.Comments {
					comment.NoteId = noteId
					if err := store.SaveComments(&comment); err != nil {
						fmt.Printf("  Failed to save comment CSV: %v\n", err)
					}
				}
			} else {
				data := map[string]interface{}{
					"note_id":  noteId,
					"comments": commentsRes.Comments,
				}
				if err := store.SaveComments(data); err != nil {
					fmt.Printf("  Failed to save comments: %v\n", err)
				}
			}
		}
	}
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

	browser, err := pw.Chromium.LaunchPersistentContext(userDataDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(config.AppConfig.Headless),
		Channel:  playwright.String("chrome"),
		Viewport: &playwright.Size{Width: 1920, Height: 1080},
	})
	if err != nil {
		browser, err = pw.Chromium.LaunchPersistentContext(userDataDir, playwright.BrowserTypeLaunchPersistentContextOptions{
			Headless: playwright.Bool(config.AppConfig.Headless),
			Viewport: &playwright.Size{Width: 1920, Height: 1080},
		})
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
	if c.pw != nil {
		c.pw.Stop()
	}
}
