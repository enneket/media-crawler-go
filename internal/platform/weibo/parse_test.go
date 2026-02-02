package weibo

import "testing"

func TestParseStatusID(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"4KjD8oZ4D", "4KjD8oZ4D"},
		{"https://m.weibo.cn/status/4KjD8oZ4D", "4KjD8oZ4D"},
		{"https://m.weibo.cn/statuses/show?id=4KjD8oZ4D", "4KjD8oZ4D"},
		{"https://weibo.com/123456/4KjD8oZ4D", "4KjD8oZ4D"},
	}
	for _, tt := range tests {
		got, _, err := ParseStatusID(tt.in)
		if err != nil {
			t.Fatalf("ParseStatusID(%q) err: %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("ParseStatusID(%q)=%q want=%q", tt.in, got, tt.want)
		}
	}
}
