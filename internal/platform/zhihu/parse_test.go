package zhihu

import "testing"

func TestParseZhihuID(t *testing.T) {
	tests := []struct {
		in    string
		wantQ string
		wantA string
	}{
		{"123", "123", ""},
		{"https://www.zhihu.com/question/123", "123", ""},
		{"https://www.zhihu.com/question/123/answer/456", "123", "456"},
	}
	for _, tt := range tests {
		q, a, noteID, err := ParseZhihuID(tt.in)
		if err != nil {
			t.Fatalf("ParseZhihuID(%q) err=%v", tt.in, err)
		}
		if q != tt.wantQ || a != tt.wantA {
			t.Fatalf("ParseZhihuID(%q) got q=%q a=%q want q=%q a=%q", tt.in, q, a, tt.wantQ, tt.wantA)
		}
		if noteID == "" {
			t.Fatalf("ParseZhihuID(%q) noteID empty", tt.in)
		}
	}
}
