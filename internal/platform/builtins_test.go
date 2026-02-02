package platform_test

import (
	"testing"

	"media-crawler-go/internal/platform"
	_ "media-crawler-go/internal/platform/bilibili"
	_ "media-crawler-go/internal/platform/douyin"
	_ "media-crawler-go/internal/platform/xhs"
)

func TestBuiltinsRegistered(t *testing.T) {
	if _, err := platform.New("xhs"); err != nil {
		t.Fatalf("New(xhs) err: %v", err)
	}
	if _, err := platform.New("douyin"); err != nil {
		t.Fatalf("New(douyin) err: %v", err)
	}
	if _, err := platform.New("dy"); err != nil {
		t.Fatalf("New(dy) err: %v", err)
	}
	if _, err := platform.New("bilibili"); err != nil {
		t.Fatalf("New(bilibili) err: %v", err)
	}
	if _, err := platform.New("bili"); err != nil {
		t.Fatalf("New(bili) err: %v", err)
	}
}
