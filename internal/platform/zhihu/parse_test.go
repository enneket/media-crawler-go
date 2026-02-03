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

func TestExtractDetailURLsFromHTML(t *testing.T) {
	html := `
<a href="/question/123">q</a>
<a href="/question/123/answer/456">a</a>
<a href="https://www.zhihu.com/question/789/answer/111">b</a>
<a href="https://www.zhihu.com/question/789">c</a>
`
	got := ExtractDetailURLsFromHTML(html, 10)
	if len(got) < 3 {
		t.Fatalf("got=%v", got)
	}
	if got[0] != "https://www.zhihu.com/question/123" {
		t.Fatalf("first=%s", got[0])
	}
	if got[1] != "https://www.zhihu.com/question/123/answer/456" {
		t.Fatalf("second=%s", got[1])
	}
	if got[2] != "https://www.zhihu.com/question/789/answer/111" {
		t.Fatalf("third=%s", got[2])
	}
}
