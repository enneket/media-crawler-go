package douyin

import "testing"

func TestExtractAwemeID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"7525082444551310602", "7525082444551310602"},
		{"https://www.douyin.com/video/7525082444551310602", "7525082444551310602"},
		{"https://www.douyin.com/user/xxx?modal_id=7471165520058862848", "7471165520058862848"},
		{"https://v.douyin.com/iF12345ABC/", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := ExtractAwemeID(c.in)
		if got != c.want {
			t.Fatalf("ExtractAwemeID(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestExtractSecUserID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"MS4wLjABAAAATJPY7LAlaa5X", "MS4wLjABAAAATJPY7LAlaa5X"},
		{"https://www.douyin.com/user/MS4wLjABAAAATJPY7LAlaa5X?from_tab_name=main", "MS4wLjABAAAATJPY7LAlaa5X"},
		{"", ""},
	}
	for _, c := range cases {
		got := ExtractSecUserID(c.in)
		if got != c.want {
			t.Fatalf("ExtractSecUserID(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
