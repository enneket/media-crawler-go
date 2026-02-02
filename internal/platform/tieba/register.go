package tieba

import (
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/platform"
)

func init() {
	platform.Register("tieba", []string{"tb", "贴吧"}, func() crawler.Runner { return NewCrawler() })
}
