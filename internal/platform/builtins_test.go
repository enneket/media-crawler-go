package platform_test

import (
	"testing"

	"media-crawler-go/internal/platform"
	_ "media-crawler-go/internal/platform/bilibili"
	_ "media-crawler-go/internal/platform/douyin"
	_ "media-crawler-go/internal/platform/kuaishou"
	_ "media-crawler-go/internal/platform/tieba"
	_ "media-crawler-go/internal/platform/weibo"
	_ "media-crawler-go/internal/platform/xhs"
	_ "media-crawler-go/internal/platform/zhihu"
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
	if _, err := platform.New("weibo"); err != nil {
		t.Fatalf("New(weibo) err: %v", err)
	}
	if _, err := platform.New("wb"); err != nil {
		t.Fatalf("New(wb) err: %v", err)
	}
	if _, err := platform.New("tieba"); err != nil {
		t.Fatalf("New(tieba) err: %v", err)
	}
	if _, err := platform.New("tb"); err != nil {
		t.Fatalf("New(tb) err: %v", err)
	}
	if _, err := platform.New("zhihu"); err != nil {
		t.Fatalf("New(zhihu) err: %v", err)
	}
	if _, err := platform.New("zh"); err != nil {
		t.Fatalf("New(zh) err: %v", err)
	}
	if _, err := platform.New("kuaishou"); err != nil {
		t.Fatalf("New(kuaishou) err: %v", err)
	}
	if _, err := platform.New("ks"); err != nil {
		t.Fatalf("New(ks) err: %v", err)
	}
}
