package weibo

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

func ExtractWeiboMediaURLs(noteID string, data any) (urls []string, filenames []string) {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" || data == nil {
		return nil, nil
	}
	m, ok := data.(map[string]any)
	if !ok {
		return nil, nil
	}

	outURLs := make([]string, 0, 8)
	outNames := make([]string, 0, 8)
	seen := map[string]struct{}{}

	add := func(u string, name string) {
		u = strings.TrimSpace(u)
		name = strings.TrimSpace(name)
		if u == "" || name == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		outURLs = append(outURLs, u)
		outNames = append(outNames, name)
	}

	imgs := extractWeiboImages(m)
	for i, u := range imgs {
		ext := extFromURL(u, "jpg")
		add(u, fmt.Sprintf("%s_%d.%s", noteID, i, ext))
	}

	videoURL, coverURL := extractWeiboVideoAndCover(m)
	if coverURL != "" {
		ext := extFromURL(coverURL, "jpg")
		add(coverURL, fmt.Sprintf("%s_cover_0.%s", noteID, ext))
	}
	if videoURL != "" {
		add(videoURL, fmt.Sprintf("%s_video.mp4", noteID))
	}

	return outURLs, outNames
}

func extractWeiboImages(m map[string]any) []string {
	pics, ok := m["pics"].([]any)
	if !ok || len(pics) == 0 {
		return nil
	}
	out := make([]string, 0, len(pics))
	for _, it := range pics {
		pm, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if large, ok := pm["large"].(map[string]any); ok {
			if u := asString(large["url"]); strings.HasPrefix(u, "http") {
				out = append(out, u)
				continue
			}
		}
		if u := asString(pm["url"]); strings.HasPrefix(u, "http") {
			out = append(out, u)
			continue
		}
		if u := asString(pm["pic_big"]); strings.HasPrefix(u, "http") {
			out = append(out, u)
			continue
		}
		if u := asString(pm["pic_large"]); strings.HasPrefix(u, "http") {
			out = append(out, u)
			continue
		}
	}
	return out
}

func extractWeiboVideoAndCover(m map[string]any) (videoURL string, coverURL string) {
	pi, ok := m["page_info"].(map[string]any)
	if !ok || pi == nil {
		return "", ""
	}

	if u := asString(pi["page_pic"]); strings.HasPrefix(u, "http") {
		coverURL = u
	} else if u := asString(pi["pic"]); strings.HasPrefix(u, "http") {
		coverURL = u
	} else if u := asString(pi["page_pic_small"]); strings.HasPrefix(u, "http") {
		coverURL = u
	}

	if mediaInfo, ok := pi["media_info"].(map[string]any); ok && mediaInfo != nil {
		if u := asString(mediaInfo["stream_url_hd"]); strings.HasPrefix(u, "http") {
			videoURL = u
			return videoURL, coverURL
		}
		if u := asString(mediaInfo["stream_url"]); strings.HasPrefix(u, "http") {
			videoURL = u
			return videoURL, coverURL
		}
		if u := asString(mediaInfo["h5_url"]); strings.HasPrefix(u, "http") && videoURL == "" {
			videoURL = u
		}
	}

	if urlsMap, ok := pi["urls"].(map[string]any); ok && urlsMap != nil {
		for _, k := range []string{"mp4_1080p_mp4", "mp4_720p_mp4", "mp4_hd_mp4", "mp4_sd_mp4", "mp4_ld_mp4"} {
			if u := asString(urlsMap[k]); strings.HasPrefix(u, "http") {
				videoURL = u
				return videoURL, coverURL
			}
		}
	}

	return videoURL, coverURL
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func extFromURL(u string, fallback string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return fallback
	}
	pu, err := url.Parse(u)
	if err != nil || pu == nil {
		return fallback
	}
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(pu.Path), "."))
	switch ext {
	case "jpg", "jpeg":
		return "jpg"
	case "png", "gif", "webp", "mp4", "m4a":
		return ext
	default:
		return fallback
	}
}

