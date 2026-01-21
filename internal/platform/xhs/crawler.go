package xhs

import (
	"context"
	"fmt"
	"media-crawler-go/internal/config"
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
		// Wait for login? 
		// For now, we just wait a bit or prompt.
		// Since we can't easily prompt in this environment, we might fail or wait loop.
		// But let's assume we might have cookies or we wait.
		time.Sleep(5 * time.Second)
		if err := c.client.UpdateCookies(c.browser); err != nil {
			return err
		}
		if !c.client.Pong() {
			return fmt.Errorf("login failed or timed out")
		}
	}
	fmt.Println("Login successful!")

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
			fmt.Printf("- [%s] %s (ID: %s)\n", item.NoteCard.User.Nickname, item.NoteCard.Title, item.Id)
			
			// 1. Get Note Detail
			noteId := item.Id
			if noteId == "" {
				noteId = item.NoteCard.NoteId
			}
			
			// We need to use the token from the search result to get details?
			// Python code uses: note_detail = await self.xhs_client.get_note_by_id(note_id, xsec_source, xsec_token)
			// item has XsecToken and XsecSource
			
			fmt.Printf("  Fetching detail for note %s...\n", noteId)
			noteDetail, err := c.client.GetNoteById(noteId, item.XsecSource, item.XsecToken)
			if err != nil {
				fmt.Printf("  Failed to get note detail: %v\n", err)
				// Fallback to HTML parsing if needed? Not implemented yet.
			} else {
				// Save Note
				if err := store.SaveNote(noteDetail); err != nil {
					fmt.Printf("  Failed to save note: %v\n", err)
				} else {
					fmt.Printf("  Note saved.\n")
				}
			}

			// 2. Get Comments
			if config.AppConfig.EnableGetComments {
				fmt.Printf("  Fetching comments for note %s...\n", noteId)
				// For comments, we need xsec_token. If GetNoteById returned detail, it might have refreshed token?
				// But usually we use the one from list or detail.
				// Python: await self.batch_get_note_comments(note_ids, xsec_tokens)
				
				commentsRes, err := c.client.GetNoteComments(noteId, item.XsecToken, "")
				if err != nil {
					fmt.Printf("  Failed to get comments: %v\n", err)
				} else {
					fmt.Printf("  Found %d comments\n", len(commentsRes.Comments))
					// Save Comments
					// We might want to wrap them with noteId
					data := map[string]interface{}{
						"note_id": noteId,
						"comments": commentsRes.Comments,
					}
					if err := store.SaveComments(data); err != nil {
						fmt.Printf("  Failed to save comments: %v\n", err)
					}
				}
			}
			
			// Random sleep between notes
			time.Sleep(time.Duration(1 + time.Now().Unix()%2) * time.Second)
		}
		
		time.Sleep(time.Duration(config.AppConfig.CrawlerMaxSleepSec) * time.Second)
	}
	
	return nil
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

	// Ensure directory exists (Playwright might create it, but good to be safe)
	if _, err := os.Stat(userDataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(userDataDir, 0755); err != nil {
			return fmt.Errorf("could not create user data dir: %v", err)
		}
	}

	browser, err := pw.Chromium.LaunchPersistentContext(userDataDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(config.AppConfig.Headless),
		Channel:  playwright.String("chrome"), // Try to use installed chrome if available, or default
		Viewport: &playwright.Size{Width: 1920, Height: 1080},
	})
	if err != nil {
		// Fallback to bundled chromium if chrome not found
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
	
	// Inject stealth?
	// Playwright-go doesn't have stealth built-in.
	// We can inject scripts.
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
