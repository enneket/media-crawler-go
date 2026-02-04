package downloader

import (
	"fmt"
	"io"
	"math/rand"
	"media-crawler-go/internal/config"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/logger"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Downloader struct {
	Client         *http.Client
	Dir            string
	MaxConcurrency int
	RetryCount     int
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func NewDownloader(dir string) *Downloader {
	timeoutSec := config.AppConfig.HttpTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	maxConcurrency := config.AppConfig.MaxConcurrencyNum
	if maxConcurrency <= 0 {
		maxConcurrency = 4
	}
	if maxConcurrency > 8 {
		maxConcurrency = 8
	}
	retryCount := config.AppConfig.HttpRetryCount
	if retryCount <= 0 {
		retryCount = 3
	}
	baseDelay := time.Duration(config.AppConfig.HttpRetryBaseDelayMs) * time.Millisecond
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}
	maxDelay := time.Duration(config.AppConfig.HttpRetryMaxDelayMs) * time.Millisecond
	if maxDelay <= 0 {
		maxDelay = 4 * time.Second
	}
	return &Downloader{
		Client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
		Dir:            dir,
		MaxConcurrency: maxConcurrency,
		RetryCount:     retryCount,
		RetryBaseDelay: baseDelay,
		RetryMaxDelay:  maxDelay,
	}
}

func (d *Downloader) Download(url, filename string) error {
	return d.download(url, filename, nil)
}

func (d *Downloader) DownloadWithHeaders(url, filename string, headers map[string]string) error {
	return d.download(url, filename, headers)
}

func (d *Downloader) download(url, filename string, headers map[string]string) error {
	if url == "" {
		return fmt.Errorf("url is empty")
	}

	filename = sanitizeFilename(filename)
	if filename == "" {
		return fmt.Errorf("filename is empty")
	}

	if err := os.MkdirAll(d.Dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(d.Dir, filename)

	// Check if file exists
	if _, err := os.Stat(path); err == nil {
		return nil // File already exists
	}

	maxRetry := d.RetryCount
	if maxRetry <= 0 {
		maxRetry = 3
	}

	baseDelay := d.RetryBaseDelay
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}
	maxDelay := d.RetryMaxDelay
	if maxDelay <= 0 {
		maxDelay = 4 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt < maxRetry; attempt++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		if req.Header.Get("Accept") == "" {
			req.Header.Set("Accept", "*/*")
		}
		for k, v := range headers {
			if k == "" || v == "" {
				continue
			}
			req.Header.Set(k, v)
		}

		resp, err := d.Client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(backoffDelay(attempt, baseDelay, maxDelay))
			continue
		}

		func() {
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				lastErr = crawler.NewHTTPStatusError("downloader", url, resp.StatusCode, "")
				return
			}

			if isSuspiciousContentType(resp.Header.Get("Content-Type"), filename) {
				lastErr = fmt.Errorf("unexpected content-type for %s: %s", filename, resp.Header.Get("Content-Type"))
				return
			}

			tmp, err := os.CreateTemp(d.Dir, filename+".part-*")
			if err != nil {
				lastErr = err
				return
			}
			tmpPath := tmp.Name()
			defer func() {
				_ = tmp.Close()
				_ = os.Remove(tmpPath)
			}()

			if _, err := io.Copy(tmp, resp.Body); err != nil {
				lastErr = err
				return
			}
			if err := tmp.Close(); err != nil {
				lastErr = err
				return
			}

			if err := os.Rename(tmpPath, path); err != nil {
				lastErr = err
				return
			}
			lastErr = nil
		}()

		if lastErr == nil {
			return nil
		}
		if resp != nil && shouldRetryStatus(resp.StatusCode) {
			time.Sleep(backoffDelay(attempt, baseDelay, maxDelay))
			continue
		}
		return lastErr
	}
	return lastErr
}

func shouldRetryStatus(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusRequestTimeout || code >= 500
}

func backoffDelay(attempt int, base, max time.Duration) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	delay := base
	for i := 0; i < attempt; i++ {
		if delay >= max {
			delay = max
			break
		}
		delay *= 2
	}
	if delay > max {
		delay = max
	}
	jitter := time.Duration(rng.Int63n(int64(250 * time.Millisecond)))
	out := delay + jitter
	if out > max {
		return max
	}
	return out
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, string(os.PathSeparator), "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return name
}

func isSuspiciousContentType(contentType string, filename string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".m4a", ".jpg", ".jpeg", ".png", ".gif", ".webp", ".flv":
	default:
		return false
	}
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/json")
}

// BatchDownload downloads multiple files concurrently
func (d *Downloader) BatchDownload(urls []string, filenames []string) []error {
	if len(urls) != len(filenames) {
		return []error{fmt.Errorf("urls and filenames length mismatch")}
	}

	limit := d.MaxConcurrency
	if limit <= 0 {
		limit = 4
	}
	sem := make(chan struct{}, limit)

	var wg sync.WaitGroup
	errors := make([]error, len(urls))

	for i, url := range urls {
		wg.Add(1)
		go func(i int, u, f string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := d.Download(u, f); err != nil {
				errors[i] = err
				logger.Error("download failed", "url", u, "err", err)
			}
		}(i, url, filenames[i])
	}

	wg.Wait()
	return errors
}

func (d *Downloader) BatchDownloadWithHeaders(urls []string, filenames []string, headers map[string]string) []error {
	if len(urls) != len(filenames) {
		return []error{fmt.Errorf("urls and filenames length mismatch")}
	}

	limit := d.MaxConcurrency
	if limit <= 0 {
		limit = 4
	}
	sem := make(chan struct{}, limit)

	var wg sync.WaitGroup
	errors := make([]error, len(urls))

	for i, url := range urls {
		wg.Add(1)
		go func(i int, u, f string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := d.DownloadWithHeaders(u, f, headers); err != nil {
				errors[i] = err
				logger.Error("download failed", "url", u, "err", err)
			}
		}(i, url, filenames[i])
	}

	wg.Wait()
	return errors
}
