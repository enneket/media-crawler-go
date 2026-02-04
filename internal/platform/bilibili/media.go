package bilibili

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
)

func ExtractBilibiliMediaURLs(noteID string, viewData any) (urls []string, filenames []string) {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" || viewData == nil {
		return nil, nil
	}
	m, ok := viewData.(map[string]any)
	if !ok {
		return nil, nil
	}
	pic := strings.TrimSpace(fmt.Sprintf("%v", m["pic"]))
	if strings.HasPrefix(pic, "//") {
		pic = "https:" + pic
	}
	if strings.HasPrefix(pic, "http") {
		ext := extFromURL(pic, "jpg")
		return []string{pic}, []string{fmt.Sprintf("%s_cover_0.%s", noteID, ext)}
	}
	return nil, nil
}

func ExtractCIDFromViewData(viewData any) int64 {
	m, ok := viewData.(map[string]any)
	if !ok || m == nil {
		return 0
	}
	if n := toInt64(m["cid"]); n > 0 {
		return n
	}
	if pages, ok := m["pages"].([]any); ok && len(pages) > 0 {
		if p0, ok := pages[0].(map[string]any); ok {
			return toInt64(p0["cid"])
		}
	}
	return 0
}

func ExtractBilibiliPlayURLs(noteID string, playData json.RawMessage) (urls []string, filenames []string) {
	noteID = strings.TrimSpace(noteID)
	if noteID == "" || len(playData) == 0 {
		return nil, nil
	}
	var v map[string]any
	if err := json.Unmarshal(playData, &v); err != nil {
		return nil, nil
	}

	if durl, ok := v["durl"].([]any); ok && len(durl) > 0 {
		if m0, ok := durl[0].(map[string]any); ok {
			if u := strings.TrimSpace(fmt.Sprintf("%v", m0["url"])); strings.HasPrefix(u, "http") {
				return []string{u}, []string{fmt.Sprintf("%s_video.mp4", noteID)}
			}
		}
	}

	var outU []string
	var outF []string
	if dash, ok := v["dash"].(map[string]any); ok && dash != nil {
		if vu := firstDashURL(dash, "video"); vu != "" {
			outU = append(outU, vu)
			outF = append(outF, fmt.Sprintf("%s_video.mp4", noteID))
		}
		if au := firstDashURL(dash, "audio"); au != "" {
			outU = append(outU, au)
			outF = append(outF, fmt.Sprintf("%s_audio.m4a", noteID))
		}
	}
	return outU, outF
}

func firstDashURL(dash map[string]any, key string) string {
	arr, ok := dash[key].([]any)
	if !ok || len(arr) == 0 {
		return ""
	}
	m0, ok := arr[0].(map[string]any)
	if !ok || m0 == nil {
		return ""
	}
	for _, k := range []string{"baseUrl", "base_url", "url"} {
		u := strings.TrimSpace(fmt.Sprintf("%v", m0[k]))
		if strings.HasPrefix(u, "http") {
			return u
		}
	}
	return ""
}

func BilibiliReferer(bvid string, aid int64, noteID string) string {
	bvid = strings.TrimSpace(bvid)
	if bvid != "" {
		return "https://www.bilibili.com/video/" + bvid
	}
	if aid > 0 {
		return "https://www.bilibili.com/video/av" + strconv.FormatInt(aid, 10)
	}
	if strings.HasPrefix(strings.ToLower(noteID), "av") {
		return "https://www.bilibili.com/video/" + noteID
	}
	if strings.HasPrefix(strings.ToUpper(noteID), "BV") {
		return "https://www.bilibili.com/video/" + noteID
	}
	return "https://www.bilibili.com/"
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
	case "png", "gif", "webp", "mp4", "m4a", "flv":
		return ext
	default:
		return fallback
	}
}

