package douyin

import "testing"

func TestCommentCSV(t *testing.T) {
	c := &Comment{
		CID:             "c1",
		Text:            "hi",
		CreateTime:      1,
		DiggCount:       2,
		NoteID:          "n1",
		ParentCommentID: "p1",
		User:            CommentUser{UID: "u", SecUID: "su", Nickname: "nick"},
	}
	if len(c.CSVHeader()) == 0 || len(c.ToCSV()) == 0 {
		t.Fatalf("csv fields empty")
	}
	if got := c.ToCSV()[0]; got != "n1" {
		t.Fatalf("expected note_id first col, got %q", got)
	}
}
