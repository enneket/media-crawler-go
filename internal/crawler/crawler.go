package crawler

import "context"

type Crawler interface {
	Start(ctx context.Context) error
}

type AbstractCrawler struct {
	// Common fields if needed
}
