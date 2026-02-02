package zhihu

import (
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/platform"
)

func init() {
	platform.Register("zhihu", []string{"zh", "知乎"}, func() crawler.Runner { return NewCrawler() })
}
