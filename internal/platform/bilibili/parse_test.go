package bilibili

import "testing"

func TestParseVideoID(t *testing.T) {
	tests := []struct {
		in     string
		noteID string
	}{
		{"BV1Q5411W7bH", "BV1Q5411W7BH"},
		{"bv1q5411w7bh", "BV1Q5411W7BH"},
		{"https://www.bilibili.com/video/BV1Q5411W7bH/", "BV1Q5411W7BH"},
		{"av170001", "av170001"},
		{"https://www.bilibili.com/video/av170001", "av170001"},
		{"https://www.bilibili.com/video/BV1Q5411W7bH?p=1", "BV1Q5411W7BH"},
	}
	for _, tt := range tests {
		_, _, got, err := ParseVideoID(tt.in)
		if err != nil {
			t.Fatalf("ParseVideoID(%q) err: %v", tt.in, err)
		}
		if got != tt.noteID {
			t.Fatalf("ParseVideoID(%q) noteID=%q want=%q", tt.in, got, tt.noteID)
		}
	}
}

func TestParseCreatorID(t *testing.T) {
	tests := []struct {
		in  string
		mid string
	}{
		{"123456", "123456"},
		{"https://space.bilibili.com/123456", "123456"},
		{"https://space.bilibili.com/123456/", "123456"},
		{"https://www.bilibili.com?mid=123456", "123456"},
	}
	for _, tt := range tests {
		got, err := ParseCreatorID(tt.in)
		if err != nil {
			t.Fatalf("ParseCreatorID(%q) err: %v", tt.in, err)
		}
		if got != tt.mid {
			t.Fatalf("ParseCreatorID(%q) mid=%q want=%q", tt.in, got, tt.mid)
		}
	}
}
