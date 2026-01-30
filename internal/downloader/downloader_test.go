package downloader

import (
	"io"
	"net/http"
	"net/http/httptest"
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
	err = d.DownloadWithHeaders(ts.URL, "b.txt", map[string]string{})
	if err == nil {
		t.Fatalf("expected error without header")
	}

	_ = filepath.Join(dir, "a.txt")
}
