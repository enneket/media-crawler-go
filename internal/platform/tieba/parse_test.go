package tieba

import "testing"

func TestParseThreadID(t *testing.T) {
	tests := []struct {
		in     string
		wantID string
		wantOK bool
	}{
		{"123456", "123456", true},
		{"https://tieba.baidu.com/p/123456", "123456", true},
		{"https://tieba.baidu.com/f?kw=golang&ie=utf-8&kz=7890", "7890", true},
		{"kz=7890", "7890", true},
	}
	for _, tt := range tests {
		gotID, noteID, err := ParseThreadID(tt.in)
		if tt.wantOK && err != nil {
			t.Fatalf("ParseThreadID(%q) err=%v", tt.in, err)
		}
		if tt.wantOK && gotID != tt.wantID {
			t.Fatalf("ParseThreadID(%q) gotID=%q want=%q", tt.in, gotID, tt.wantID)
		}
		if tt.wantOK && noteID == "" {
			t.Fatalf("ParseThreadID(%q) noteID empty", tt.in)
		}
	}
}
