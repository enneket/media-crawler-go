package store

import "strconv"

type UnifiedComment struct {
	Platform        string
	NoteID          string
	CommentID       string
	ParentCommentID string
	Content         string
	CreateTime      int64
	LikeCount       int64
	UserID          string
	UserSecUID      string
	UserNickname    string
}

func (c *UnifiedComment) CSVHeader() []string {
	return []string{
		"platform",
		"note_id",
		"comment_id",
		"parent_comment_id",
		"content",
		"create_time",
		"like_count",
		"user_id",
		"user_sec_uid",
		"user_nickname",
	}
}

func (c *UnifiedComment) ToCSV() []string {
	return []string{
		c.Platform,
		c.NoteID,
		c.CommentID,
		c.ParentCommentID,
		c.Content,
		strconv.FormatInt(c.CreateTime, 10),
		strconv.FormatInt(c.LikeCount, 10),
		c.UserID,
		c.UserSecUID,
		c.UserNickname,
	}
}

