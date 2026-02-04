package proxy

import "testing"

func TestNewProvider_JiSuHTTPAliases(t *testing.T) {
	for _, name := range []string{"jisuhttp", "jishuhttp", "jishu_http"} {
		p, err := NewProvider(name)
		if err != nil {
			t.Fatalf("NewProvider(%q) err: %v", name, err)
		}
		if p.Name() != ProviderJiSuHTTP {
			t.Fatalf("NewProvider(%q) got %q, want %q", name, p.Name(), ProviderJiSuHTTP)
		}
	}
}
