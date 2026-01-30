package douyin

import "fmt"

type CommentUser struct {
	UID      string `json:"uid"`
	SecUID   string `json:"sec_uid"`
	Nickname string `json:"nickname"`
}

type Comment struct {
	CID        string      `json:"cid"`
	Text       string      `json:"text"`
	CreateTime int64       `json:"create_time"`
	DiggCount  int64       `json:"digg_count"`
	User       CommentUser `json:"user"`

	NoteID          string `json:"-"`
	ParentCommentID string `json:"-"`
}

func (c *Comment) CSVHeader() []string {
	return []string{
		"note_id",
		"comment_id",
		"parent_comment_id",
		"text",
		"create_time",
		"digg_count",
		"user_uid",
		"user_sec_uid",
		"user_nickname",
	}
}

func (c *Comment) ToCSV() []string {
	return []string{
		c.NoteID,
		c.CID,
		c.ParentCommentID,
		c.Text,
		fmt.Sprintf("%d", c.CreateTime),
		fmt.Sprintf("%d", c.DiggCount),
		c.User.UID,
		c.User.SecUID,
		c.User.Nickname,
	}
}
