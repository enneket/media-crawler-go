package xhs

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type CreatorRecord struct {
	UserID       string `json:"user_id"`
	Nickname     string `json:"nickname"`
	Gender       string `json:"gender,omitempty"`
	Avatar       any    `json:"avatar,omitempty"`
	Desc         string `json:"desc,omitempty"`
	IPLocation   string `json:"ip_location,omitempty"`
	Follows      int    `json:"follows,omitempty"`
	Fans         int    `json:"fans,omitempty"`
	Interaction  int    `json:"interaction,omitempty"`
	TagList      string `json:"tag_list,omitempty"`
	LastModifyTS int64  `json:"last_modify_ts"`
}

func (c CreatorRecord) CSVHeader() []string {
	return []string{
		"user_id",
		"nickname",
		"gender",
		"avatar",
		"desc",
		"ip_location",
		"follows",
		"fans",
		"interaction",
		"tag_list",
		"last_modify_ts",
	}
}

func ExtractCreatorID(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		idx := strings.Index(s, "/user/profile/")
		if idx == -1 {
			return ""
		}
		s = s[idx+len("/user/profile/"):]
	}
	s = strings.Trim(s, "/")
	if q := strings.Index(s, "?"); q != -1 {
		s = s[:q]
	}
	return strings.TrimSpace(s)
}

func (c CreatorRecord) ToCSV() []string {
	avatarBytes, _ := json.Marshal(c.Avatar)
	return []string{
		c.UserID,
		c.Nickname,
		c.Gender,
		string(avatarBytes),
		c.Desc,
		c.IPLocation,
		fmt.Sprintf("%d", c.Follows),
		fmt.Sprintf("%d", c.Fans),
		fmt.Sprintf("%d", c.Interaction),
		c.TagList,
		fmt.Sprintf("%d", c.LastModifyTS),
	}
}

func BuildCreatorRecord(userID string, userPageData map[string]interface{}) (CreatorRecord, error) {
	basicAny := userPageData["basicInfo"]
	basic, _ := basicAny.(map[string]interface{})

	interactionsAny := userPageData["interactions"]
	interactions, _ := interactionsAny.([]interface{})

	follows := 0
	fans := 0
	interaction := 0
	for _, it := range interactions {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		t := fmt.Sprintf("%v", m["type"])
		c := toInt(m["count"])
		switch t {
		case "follows":
			follows = c
		case "fans":
			fans = c
		case "interaction":
			interaction = c
		}
	}

	tagsAny := userPageData["tags"]
	tags, _ := tagsAny.([]interface{})
	tagMap := map[string]string{}
	for _, t := range tags {
		m, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		tagType := strings.TrimSpace(fmt.Sprintf("%v", m["tagType"]))
		name := strings.TrimSpace(fmt.Sprintf("%v", m["name"]))
		if tagType == "" || name == "" || tagType == "<nil>" || name == "<nil>" {
			continue
		}
		tagMap[tagType] = name
	}
	tagBytes, _ := json.Marshal(tagMap)

	gender := ""
	switch toInt(basic["gender"]) {
	case 1:
		gender = "Female"
	case 0:
		gender = "Male"
	}

	return CreatorRecord{
		UserID:       userID,
		Nickname:     toString(basic["nickname"]),
		Gender:       gender,
		Avatar:       basic["images"],
		Desc:         toString(basic["desc"]),
		IPLocation:   toString(basic["ipLocation"]),
		Follows:      follows,
		Fans:         fans,
		Interaction:  interaction,
		TagList:      string(tagBytes),
		LastModifyTS: time.Now().Unix(),
	}, nil
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	if s == "<nil>" {
		return ""
	}
	return s
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}
