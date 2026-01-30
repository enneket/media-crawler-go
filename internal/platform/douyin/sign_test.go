package douyin

import (
	"net/url"
	"testing"
)

func TestSignerSignDetail(t *testing.T) {
	s, err := NewSigner()
	if err != nil {
		t.Fatalf("NewSigner err: %v", err)
	}
	params := url.Values{}
	params.Set("aweme_id", "7525082444551310602")
	params.Set("aid", "6383")
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	ab, err := s.SignDetail(params, ua)
	if err != nil {
		t.Fatalf("SignDetail err: %v", err)
	}
	if ab == "" {
		t.Fatalf("expected non-empty a_bogus")
	}
}
