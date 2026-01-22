package douyin

import (
	"context"
	"fmt"
)

type DouyinCrawler struct {
}

func NewCrawler() *DouyinCrawler {
	return &DouyinCrawler{}
}

func (c *DouyinCrawler) Start(ctx context.Context) error {
	fmt.Println("DouyinCrawler started... (Not implemented yet)")
	return nil
}
