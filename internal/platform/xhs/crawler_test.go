package xhs

import "testing"

func TestExtractNoteId(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{url: "https://www.xiaohongshu.com/explore/64a123", want: "64a123"},
		{url: "https://www.xiaohongshu.com/explore/64a123?xsec_token=abc&xsec_source=pc_search", want: "64a123"},
		{url: "https://www.xiaohongshu.com/explore/64a123/", want: ""},
		{url: "", want: ""},
	}

	for _, tt := range tests {
		got := extractNoteId(tt.url)
		if got != tt.want {
			t.Fatalf("extractNoteId(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
