package downloader

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadWithHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "1" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer ts.Close()

	dir := t.TempDir()
	d := NewDownloader(dir)
	err := d.DownloadWithHeaders(ts.URL, "a.txt", map[string]string{"X-Test": "1"})
	if err != nil {
		t.Fatalf("DownloadWithHeaders err: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatalf("read file err: %v", err)
	}
	if string(b) != "ok" {
		t.Fatalf("unexpected body: %q", string(b))
	}
	err = d.DownloadWithHeaders(ts.URL, "b.txt", map[string]string{})
	if err == nil {
		t.Fatalf("expected error without header")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.part-*"))
	if len(matches) != 0 {
		t.Fatalf("unexpected tmp files: %v", matches)
	}
}

func TestDownloadRetryOnServerError(t *testing.T) {
	var n int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = io.WriteString(w, "ok2")
	}))
	defer ts.Close()

	dir := t.TempDir()
	d := NewDownloader(dir)
	err := d.Download(ts.URL, "c.txt")
	if err != nil {
		t.Fatalf("Download err: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "c.txt"))
	if err != nil {
		t.Fatalf("read file err: %v", err)
	}
	if string(b) != "ok2" {
		t.Fatalf("unexpected body: %q", string(b))
	}
}
