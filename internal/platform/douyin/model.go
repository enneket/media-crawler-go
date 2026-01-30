package douyin

import "fmt"

type VideoDetail struct {
	AwemeID    string `json:"aweme_id"`
	Desc       string `json:"desc"`
	CreateTime int64  `json:"create_time"`
	Author     struct {
		SecUID   string `json:"sec_uid"`
		Nickname string `json:"nickname"`
		UID      string `json:"uid"`
	} `json:"author"`
	Statistics struct {
		CommentCount int64 `json:"comment_count"`
		DiggCount    int64 `json:"digg_count"`
		CollectCount int64 `json:"collect_count"`
		ShareCount   int64 `json:"share_count"`
		PlayCount    int64 `json:"play_count"`
	} `json:"statistics"`
	Video struct {
		PlayAddr struct {
			URLList []string `json:"url_list"`
		} `json:"play_addr"`
		Cover struct {
			URLList []string `json:"url_list"`
		} `json:"cover"`
		OriginCover struct {
			URLList []string `json:"url_list"`
		} `json:"origin_cover"`
	} `json:"video"`
}

func (v *VideoDetail) CSVHeader() []string {
	return []string{
		"aweme_id",
		"desc",
		"create_time",
		"author_uid",
		"author_sec_uid",
		"author_nickname",
		"digg_count",
		"comment_count",
		"collect_count",
		"share_count",
		"play_count",
	}
}

func (v *VideoDetail) ToCSV() []string {
	return []string{
		v.AwemeID,
		v.Desc,
		fmt.Sprintf("%d", v.CreateTime),
		v.Author.UID,
		v.Author.SecUID,
		v.Author.Nickname,
		fmt.Sprintf("%d", v.Statistics.DiggCount),
		fmt.Sprintf("%d", v.Statistics.CommentCount),
		fmt.Sprintf("%d", v.Statistics.CollectCount),
		fmt.Sprintf("%d", v.Statistics.ShareCount),
		fmt.Sprintf("%d", v.Statistics.PlayCount),
	}
}
