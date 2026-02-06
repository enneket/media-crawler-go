package bilibili

import (
	"time"
)

func filterByDate(videos []videoRef, minTime, maxTime int64) []videoRef {
	if minTime <= 0 && maxTime <= 0 {
		return videos
	}
	out := make([]videoRef, 0, len(videos))
	for _, v := range videos {
		if v.PubTime == 0 {
			// If time is unknown, we generally keep it or drop it?
			// Let's keep it to be safe unless strict mode is needed.
			out = append(out, v)
			continue
		}
		if minTime > 0 && v.PubTime < minTime {
			continue
		}
		if maxTime > 0 && v.PubTime >= maxTime {
			continue
		}
		out = append(out, v)
	}
	return out
}

func filterByDailyLimit(videos []videoRef, maxPerDay int, dayCounts map[string]int) []videoRef {
	if maxPerDay <= 0 {
		return videos
	}
	out := make([]videoRef, 0, len(videos))
	for _, v := range videos {
		if v.PubTime == 0 {
			out = append(out, v)
			continue
		}
		day := time.Unix(v.PubTime, 0).Format("2006-01-02")
		if dayCounts[day] >= maxPerDay {
			continue
		}
		dayCounts[day]++
		out = append(out, v)
	}
	return out
}
