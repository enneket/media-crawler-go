package crawler

import (
	"media-crawler-go/internal/config"
	"strings"
)

func RequestFromConfig(cfg config.Config) Request {
	platform := strings.TrimSpace(cfg.Platform)
	mode := NormalizeMode(cfg.CrawlerType)

	out := Request{
		Platform:    platform,
		Mode:        mode,
		StartPage:   cfg.StartPage,
		MaxNotes:    cfg.CrawlerMaxNotesCount,
		Concurrency: cfg.MaxConcurrencyNum,
		Keywords:    splitCSV(cfg.Keywords),
	}

	switch strings.ToLower(platform) {
	case "xhs":
		switch mode {
		case ModeDetail:
			out.Inputs = cfg.XhsSpecifiedNoteUrls
		case ModeCreator:
			out.Inputs = cfg.XhsCreatorIdList
		}
	case "douyin", "dy":
		switch mode {
		case ModeDetail:
			out.Inputs = cfg.DouyinSpecifiedNoteUrls
		case ModeCreator:
			out.Inputs = cfg.DouyinCreatorIdList
		}
	case "bilibili", "bili", "b站", "b":
		switch mode {
		case ModeDetail:
			out.Inputs = cfg.BiliSpecifiedVideoUrls
		case ModeCreator:
			out.Inputs = cfg.BiliCreatorIdList
		}
	case "weibo", "wb", "微博":
		switch mode {
		case ModeDetail:
			out.Inputs = cfg.WBSpecifiedNoteUrls
		case ModeCreator:
			out.Inputs = cfg.WBCreatorIdList
		}
	case "tieba", "tb", "贴吧":
		switch mode {
		case ModeDetail:
			out.Inputs = cfg.TiebaSpecifiedNoteUrls
		case ModeCreator:
			out.Inputs = cfg.TiebaCreatorUrlList
		}
	case "zhihu", "zh", "知乎":
		switch mode {
		case ModeDetail:
			out.Inputs = cfg.ZhihuSpecifiedNoteUrls
		case ModeCreator:
			out.Inputs = cfg.ZhihuCreatorUrlList
		}
	case "kuaishou", "ks", "快手":
		switch mode {
		case ModeDetail:
			out.Inputs = cfg.KuaishouSpecifiedNoteUrls
		case ModeCreator:
			out.Inputs = cfg.KuaishouCreatorUrlList
		}
	}

	return out
}

func splitCSV(s string) []string {
	v := strings.TrimSpace(s)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
