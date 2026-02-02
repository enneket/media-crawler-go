package douyin

import (
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/platform"
)

func init() {
	platform.Register("douyin", []string{"dy"}, func() crawler.Crawler { return NewCrawler() })
}
