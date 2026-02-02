package bilibili

import (
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/platform"
)

func init() {
	platform.Register("bilibili", []string{"bili", "b站", "b"}, func() crawler.Crawler { return NewCrawler() })
}
