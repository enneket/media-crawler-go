package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/playwright-community/playwright-go"
)

type CDPOptions struct {
	DebugPort         int
	CustomBrowserPath string
	UserDataDir       string
	Headless          bool
	ProxyServer       string
	LaunchTimeout     time.Duration
}

type CDPSession struct {
	Cmd     *exec.Cmd
	Browser playwright.Browser
	Context playwright.BrowserContext
	Page    playwright.Page
}

func StartOrConnectCDP(ctx context.Context, pw *playwright.Playwright, opts CDPOptions) (*CDPSession, error) {
	if pw == nil {
		return nil, fmt.Errorf("playwright is nil")
	}
	if opts.DebugPort <= 0 {
		opts.DebugPort = 9222
	}
	if opts.LaunchTimeout <= 0 {
		opts.LaunchTimeout = 60 * time.Second
	}
	if opts.UserDataDir == "" {
		opts.UserDataDir = "browser_data"
	}

	userDataDir, err := filepath.Abs(opts.UserDataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve user data dir: %w", err)
	}
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return nil, fmt.Errorf("create user data dir: %w", err)
	}

	endpoint := fmt.Sprintf("http://127.0.0.1:%d", opts.DebugPort)
	var cmd *exec.Cmd

	if err := waitCDPReady(ctx, endpoint, 800*time.Millisecond); err != nil {
		bin, err := detectBrowserBinary(opts.CustomBrowserPath)
		if err != nil {
			return nil, err
		}

		cmd = exec.CommandContext(ctx, bin, buildChromeArgs(opts.DebugPort, userDataDir, opts.Headless, opts.ProxyServer)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("start browser: %w", err)
		}

		waitCtx, cancel := context.WithTimeout(ctx, opts.LaunchTimeout)
		defer cancel()
		if err := waitCDPReady(waitCtx, endpoint, 250*time.Millisecond); err != nil {
			_ = cmd.Process.Kill()
			return nil, fmt.Errorf("cdp not ready: %w", err)
		}
	}

	browser, err := pw.Chromium.ConnectOverCDP(endpoint)
	if err != nil {
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil, fmt.Errorf("connect over cdp: %w", err)
	}

	contexts := browser.Contexts()
	var browserCtx playwright.BrowserContext
	if len(contexts) > 0 {
		browserCtx = contexts[0]
	} else {
		browserCtx, err = browser.NewContext()
		if err != nil {
			_ = browser.Close()
			if cmd != nil && cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			return nil, fmt.Errorf("new context: %w", err)
		}
	}

	pages := browserCtx.Pages()
	var page playwright.Page
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = browserCtx.NewPage()
		if err != nil {
			_ = browserCtx.Close()
			_ = browser.Close()
			if cmd != nil && cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			return nil, fmt.Errorf("new page: %w", err)
		}
	}

	return &CDPSession{
		Cmd:     cmd,
		Browser: browser,
		Context: browserCtx,
		Page:    page,
	}, nil
}

func waitCDPReady(ctx context.Context, endpoint string, interval time.Duration) error {
	url := endpoint + "/json/version"
	client := &http.Client{Timeout: 2 * time.Second}

	type versionResp struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil && resp != nil {
			var body versionResp
			_ = json.NewDecoder(resp.Body).Decode(&body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK && body.WebSocketDebuggerURL != "" {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func buildChromeArgs(port int, userDataDir string, headless bool, proxyServer string) []string {
	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", userDataDir),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-blink-features=AutomationControlled",
		"--disable-dev-shm-usage",
		"--lang=zh-CN",
		"--window-size=1920,1080",
	}
	if proxyServer != "" {
		args = append(args, fmt.Sprintf("--proxy-server=%s", proxyServer))
	}

	if headless {
		args = append(args, "--headless=new")
		args = append(args, "--disable-gpu")
	}

	if runtime.GOOS == "linux" {
		args = append(args, "--password-store=basic")
		args = append(args, "--use-mock-keychain")
	}

	return args
}
