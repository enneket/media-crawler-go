package proxy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStaticProviderFromList(t *testing.T) {
	t.Setenv("IP_PROXY_LIST", "http://user:pass@1.1.1.1:8080, 2.2.2.2:9090")
	p := NewStaticFromConfigOrEnv()
	got, err := p.GetProxies(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetProxies err: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].IP != "1.1.1.1" || got[0].Port != 8080 || got[0].User != "user" || got[0].Password != "pass" {
		t.Fatalf("unexpected proxy[0]: %#v", got[0])
	}
	if got[1].IP != "2.2.2.2" || got[1].Port != 9090 || got[1].Protocol != "http" {
		t.Fatalf("unexpected proxy[1]: %#v", got[1])
	}
}

func TestStaticProviderFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proxies.txt")
	if err := os.WriteFile(path, []byte("# comment\n1.1.1.1:8080\n\nhttp://2.2.2.2:9090\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	t.Setenv("IP_PROXY_FILE", path)
	t.Setenv("IP_PROXY_LIST", "")

	p := NewStaticFromConfigOrEnv()
	got, err := p.GetProxies(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetProxies err: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].IP != "1.1.1.1" || got[0].Port != 8080 {
		t.Fatalf("unexpected proxy[0]: %#v", got[0])
	}
	if got[1].IP != "2.2.2.2" || got[1].Port != 9090 {
		t.Fatalf("unexpected proxy[1]: %#v", got[1])
	}
}

