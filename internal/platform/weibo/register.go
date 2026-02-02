package weibo

import (
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/platform"
)

func init() {
	platform.Register("weibo", []string{"wb", "微博"}, func() crawler.Runner { return NewCrawler() })
}
