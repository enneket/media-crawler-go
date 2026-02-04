package weibo

import "strconv"

type Comment struct {
	NoteID          string
	CommentID       string
	ParentCommentID string
	Content         string
	CreateTime      int64
	LikeCount       int64
	UserID          string
	UserNickname    string
}

func (c Comment) CSVHeader() []string {
	return []string{"note_id", "comment_id", "parent_comment_id", "content", "create_time", "like_count", "user_id", "user_nickname"}
}

func (c Comment) ToCSV() []string {
	return []string{
		c.NoteID,
		c.CommentID,
		c.ParentCommentID,
		c.Content,
		strconv.FormatInt(c.CreateTime, 10),
		strconv.FormatInt(c.LikeCount, 10),
		c.UserID,
		c.UserNickname,
	}
}

