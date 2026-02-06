package bilibili

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Dynamic struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	PubTime   string `json:"pub_time"`
	Content   string `json:"content"`
	Likes     int64  `json:"likes"`
	Comments  int64  `json:"comments"`
	Forwards  int64  `json:"forwards"`
	JumpURL   string `json:"jump_url"`
}

func (d *Dynamic) CSVHeader() []string {
	return []string{
		"Dynamic ID",
		"Type",
		"Publish Time",
		"Content",
		"Likes",
		"Comments",
		"Forwards",
		"URL",
	}
}

func (d *Dynamic) ToCSV() []string {
	return []string{
		d.ID,
		d.Type,
		d.PubTime,
		d.Content,
		strconv.FormatInt(d.Likes, 10),
		strconv.FormatInt(d.Comments, 10),
		strconv.FormatInt(d.Forwards, 10),
		d.JumpURL,
	}
}

func extractDynamics(data any) ([]*Dynamic, string) {
	m, ok := data.(map[string]any)
	if !ok {
		return nil, ""
	}
	items, _ := m["items"].([]any)
	offset, _ := m["offset"].(string)

	out := make([]*Dynamic, 0, len(items))
	for _, it := range items {
		im, ok := it.(map[string]any)
		if !ok {
			continue
		}
		id := fmt.Sprintf("%v", im["id_str"])
		dtype := fmt.Sprintf("%v", im["type"])
		
		modules, _ := im["modules"].(map[string]any)
		
		// Author
		author, _ := modules["module_author"].(map[string]any)
		pubTs := toInt64(author["pub_ts"])
		pubTime := time.Unix(pubTs, 0).Format("2006-01-02 15:04:05")
		jumpUrl, _ := author["jump_url"].(string)
		if jumpUrl == "" {
			jumpUrl = fmt.Sprintf("https://t.bilibili.com/%s", id)
		}

		// Stat
		stat, _ := modules["module_stat"].(map[string]any)
		like, _ := stat["like"].(map[string]any)
		comment, _ := stat["comment"].(map[string]any)
		forward, _ := stat["forward"].(map[string]any)
		
		// Content
		content := ""
		dynamic, _ := modules["module_dynamic"].(map[string]any)
		
		// Try desc first (common text)
		if desc, ok := dynamic["desc"].(map[string]any); ok {
			content = fmt.Sprintf("%v", desc["text"])
		}
		
		// Try major (opus/archive)
		major, _ := dynamic["major"].(map[string]any)
		if opus, ok := major["opus"].(map[string]any); ok {
			if summary, ok := opus["summary"].(map[string]any); ok {
				t := fmt.Sprintf("%v", summary["text"])
				if content != "" {
					content += "\n" + t
				} else {
					content = t
				}
			}
		}
		if archive, ok := major["archive"].(map[string]any); ok {
			t := fmt.Sprintf("Video: %v", archive["title"])
			if content != "" {
				content += "\n" + t
			} else {
				content = t
			}
			// If it's a video, jump_url might be better
			if url, ok := archive["jump_url"].(string); ok {
				jumpUrl = url
			}
		}

		out = append(out, &Dynamic{
			ID:       id,
			Type:     dtype,
			PubTime:  pubTime,
			Content:  strings.TrimSpace(content),
			Likes:    toInt64(like["count"]),
			Comments: toInt64(comment["count"]),
			Forwards: toInt64(forward["count"]),
			JumpURL:  jumpUrl,
		})
	}
	return out, offset
}
