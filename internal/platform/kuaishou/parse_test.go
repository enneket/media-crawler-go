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
