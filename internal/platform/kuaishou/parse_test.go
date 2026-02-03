package kuaishou

import "testing"

func TestParseKSID(t *testing.T) {
	tests := []struct {
		in     string
		wantID string
	}{
		{"abc123", "abc123"},
		{"https://www.kuaishou.com/short-video/abc123", "abc123"},
		{"https://www.kuaishou.com/photo/xyz_9", "xyz_9"},
	}
	for _, tt := range tests {
		got, noteID, err := ParseKSID(tt.in)
		if err != nil && got != tt.wantID {
			t.Fatalf("ParseKSID(%q) err=%v", tt.in, err)
		}
		if got != tt.wantID {
			t.Fatalf("ParseKSID(%q) got=%q want=%q", tt.in, got, tt.wantID)
		}
		if noteID == "" {
			t.Fatalf("ParseKSID(%q) noteID empty", tt.in)
		}
	}
}

func TestExtractDetailURLsFromHTML(t *testing.T) {
	html := `
<a href="/short-video/abc123">x</a>
<a href="/photo/xyz_9">y</a>
<a href="https://www.kuaishou.com/short-video/abc123">dup</a>
`
	got := ExtractDetailURLsFromHTML(html, "https://www.kuaishou.com", 10)
	if len(got) != 2 {
		t.Fatalf("got=%v", got)
	}
	if got[0] != "https://www.kuaishou.com/short-video/abc123" {
		t.Fatalf("first=%s", got[0])
	}
	if got[1] != "https://www.kuaishou.com/photo/xyz_9" {
		t.Fatalf("second=%s", got[1])
	}
}

func TestParseKSCreatorID(t *testing.T) {
	id, err := ParseKSCreatorID("https://www.kuaishou.com/profile/user_1")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if id != "user_1" {
		t.Fatalf("id=%s", id)
	}

	id, err = ParseKSCreatorID("http://example.local/profile/user_2")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if id != "user_2" {
		t.Fatalf("id=%s", id)
	}
}
