package xhs

import (
	"fmt"
	"strings"
	"time"
)

type Note struct {
	NoteId       string   `json:"note_id"`
	Title        string   `json:"title"`
	Desc         string   `json:"desc"`
	Type         string   `json:"type"`
	User         User     `json:"user"`
	ImageList    []Image  `json:"image_list"`
	Video        Video    `json:"video"`
	TagList      []Tag    `json:"tag_list"`
	InteractInfo Interact `json:"interact_info"`
	XsecToken    string   `json:"xsec_token"`
	XsecSource   string   `json:"xsec_source"`
}

func (n *Note) CSVHeader() []string {
	return []string{
		"NoteId", "Title", "Desc", "Type", "UserId", "Nickname",
		"LikedCount", "CollectedCount", "CommentCount", "ShareCount", "LastUpdateTime",
	}
}

func (n *Note) ToCSV() []string {
	// Clean desc to avoid CSV issues (though encoding/csv handles newlines/quotes)
	// Just standard behavior.
	return []string{
		n.NoteId,
		n.Title,
		n.Desc,
		n.Type,
		n.User.UserId,
		n.User.Nickname,
		n.InteractInfo.LikedCount,
		n.InteractInfo.CollectedCount,
		n.InteractInfo.CommentCount,
		n.InteractInfo.ShareCount,
		fmt.Sprintf("%d", time.Now().Unix()),
	}
}

type User struct {
	UserId   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type Image struct {
	UrlDefault string `json:"url_default"`
	Url        string `json:"url"` // Populated
}

type Video struct {
	Media Consumer `json:"media"`
}

type Consumer struct {
	Stream map[string][]StreamItem `json:"stream"`
}

type StreamItem struct {
	MasterUrl string `json:"master_url"`
}

type Tag struct {
	Name string `json:"name"`
}

type Interact struct {
	LikedCount     string `json:"liked_count"`
	CollectedCount string `json:"collected_count"`
	CommentCount   string `json:"comment_count"`
	ShareCount     string `json:"share_count"`
}

type Comment struct {
	Id                string    `json:"id"`
	Content           string    `json:"content"`
	CreateTime        int64     `json:"create_time"`
	User              User      `json:"user"`
	LikeCount         string    `json:"like_count"`
	SubComments       []Comment `json:"sub_comments"`
	SubCommentCursor  string    `json:"sub_comment_cursor"`
	SubCommentHasMore bool      `json:"sub_comment_has_more"`

	// Extra for CSV
	NoteId string `json:"-"`
}

func (c *Comment) CSVHeader() []string {
	return []string{"NoteId", "CommentId", "Content", "UserId", "Nickname", "LikeCount", "CreateTime"}
}

func (c *Comment) ToCSV() []string {
	return []string{
		c.NoteId,
		c.Id,
		c.Content,
		c.User.UserId,
		c.User.Nickname,
		c.LikeCount,
		fmt.Sprintf("%d", c.CreateTime),
	}
}

type CommentResult struct {
	HasMore  bool      `json:"has_more"`
	Cursor   string    `json:"cursor"`
	Comments []Comment `json:"comments"`
}

type SearchResult struct {
	HasMore bool         `json:"has_more"`
	Items   []SearchItem `json:"items"`
}

type CreatorNotesResult struct {
	HasMore bool   `json:"has_more"`
	Cursor  string `json:"cursor"`
	Notes   []Note `json:"notes"`
}

type SearchItem struct {
	Id         string `json:"id"`
	XsecSource string `json:"xsec_source"`
	XsecToken  string `json:"xsec_token"`
	ModelType  string `json:"model_type"`
	NoteCard   Note   `json:"note_card"`
}

// Helper to clean strings if needed, though csv writer handles it
func cleanStr(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}
