package downloader

import (
	"fmt"
	"io"
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

	resp, err := d.Client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
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
				fmt.Printf("Failed to download %s: %v\n", u, err)
			}
		}(i, url, filenames[i])
	}

	wg.Wait()
	return errors
}
