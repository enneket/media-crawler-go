package kuaishou

import (
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/platform"
)

func init() {
	platform.Register("kuaishou", []string{"ks", "快手"}, func() crawler.Runner { return NewCrawler() })
}
