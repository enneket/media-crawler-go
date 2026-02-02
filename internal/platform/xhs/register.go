package xhs

import (
	"media-crawler-go/internal/crawler"
	"media-crawler-go/internal/platform"
)

func init() {
	platform.Register("xhs", nil, func() crawler.Crawler { return NewCrawler() })
}
