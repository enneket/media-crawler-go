package douyin

import (
	"fmt"
	"net/url"
	"sync"
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

func TestSignerConcurrent(t *testing.T) {
	s, err := NewSigner()
	if err != nil {
		t.Fatalf("NewSigner err: %v", err)
	}
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

	var wg sync.WaitGroup
	errCh := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			params := url.Values{}
			params.Set("aweme_id", "7525082444551310602")
			params.Set("aid", "6383")
			ab, err := s.SignDetail(params, ua)
			if err != nil {
				errCh <- err
				return
			}
			if ab == "" {
				errCh <- fmt.Errorf("empty a_bogus")
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent sign failed: %v", err)
		}
	}
}
