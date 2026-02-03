package downloader

import (
	"fmt"
	"io"
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/logger"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Downloader struct {
	Client *http.Client
	Dir    string
}

func NewDownloader(dir string) *Downloader {
	return &Downloader{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		Dir: dir,
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

	if err := os.MkdirAll(d.Dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(d.Dir, filename)

	// Check if file exists
	if _, err := os.Stat(path); err == nil {
		return nil // File already exists
	}

	const maxRetry = 3
	var lastErr error
	for attempt := 0; attempt < maxRetry; attempt++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
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
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			continue
		}

		func() {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				lastErr = crawler.NewHTTPStatusError("downloader", url, resp.StatusCode, "")
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
		if resp != nil && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) {
			time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			continue
		}
		return lastErr
	}
	return lastErr
}

// BatchDownload downloads multiple files concurrently
func (d *Downloader) BatchDownload(urls []string, filenames []string) []error {
	if len(urls) != len(filenames) {
		return []error{fmt.Errorf("urls and filenames length mismatch")}
	}

	var wg sync.WaitGroup
	errors := make([]error, len(urls))

	for i, url := range urls {
		wg.Add(1)
		go func(i int, u, f string) {
			defer wg.Done()
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

	var wg sync.WaitGroup
	errors := make([]error, len(urls))

	for i, url := range urls {
		wg.Add(1)
		go func(i int, u, f string) {
			defer wg.Done()
			if err := d.DownloadWithHeaders(u, f, headers); err != nil {
				errors[i] = err
				logger.Error("download failed", "url", u, "err", err)
			}
		}(i, url, filenames[i])
	}

	wg.Wait()
	return errors
}
