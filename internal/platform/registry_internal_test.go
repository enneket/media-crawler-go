package platform

import (
	"context"
	"media-crawler-go/internal/crawler"
	"testing"
)

type mockCrawler struct{}

func (m *mockCrawler) Start(ctx context.Context) error { return nil }

func TestRegisterAndNew(t *testing.T) {
	mu.Lock()
	orig := factories
	factories = map[string]Factory{}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		factories = orig
		mu.Unlock()
	})

	Register("foo", []string{"bar", "Baz"}, func() crawler.Crawler { return &mockCrawler{} })

	if !Exists("foo") || !Exists("bar") || !Exists("baz") {
		t.Fatalf("expected Exists to be true for registered names")
	}
	if Exists("unknown") {
		t.Fatalf("expected Exists to be false for unknown")
	}

	if _, err := New("foo"); err != nil {
		t.Fatalf("New(foo) err: %v", err)
	}
	if _, err := New("bar"); err != nil {
		t.Fatalf("New(bar) err: %v", err)
	}
	if _, err := New("baz"); err != nil {
		t.Fatalf("New(baz) err: %v", err)
	}
	if _, err := New("unknown"); err == nil {
		t.Fatalf("expected error for unknown platform")
	}
}
