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
		out.Mode = ModeDetail
		out.Inputs = cfg.BiliSpecifiedVideoUrls
	case "weibo", "wb", "微博":
		switch mode {
		case ModeDetail:
			out.Inputs = cfg.WBSpecifiedNoteUrls
		case ModeCreator:
			out.Inputs = cfg.WBCreatorIdList
		}
	case "tieba", "tb", "贴吧":
		out.Mode = ModeDetail
		out.Inputs = cfg.TiebaSpecifiedNoteUrls
	case "zhihu", "zh", "知乎":
		out.Mode = ModeDetail
		out.Inputs = cfg.ZhihuSpecifiedNoteUrls
	case "kuaishou", "ks", "快手":
		out.Mode = ModeDetail
		out.Inputs = cfg.KuaishouSpecifiedNoteUrls
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
