package xhs

import "testing"

func TestBuildCookies(t *testing.T) {
	cookies := buildCookies("a1=foo; web_session=bar; bad; k=v=more")
	if len(cookies) != 3 {
		t.Fatalf("expected 3 cookies, got %d", len(cookies))
	}
	if cookies[0].Name != "a1" || cookies[0].Value != "foo" {
		t.Fatalf("unexpected cookie 0: %+v", cookies[0])
	}
	if cookies[1].Name != "web_session" || cookies[1].Value != "bar" {
		t.Fatalf("unexpected cookie 1: %+v", cookies[1])
	}
	if cookies[2].Name != "k" || cookies[2].Value != "v=more" {
		t.Fatalf("unexpected cookie 2: %+v", cookies[2])
	}
}
