package browser

import (
	"os"
	"path/filepath"
	"strings"
)

func PrepareUserDataDir(base string, save bool, platform string) (string, func(), error) {
	base = strings.TrimSpace(base)
	platform = strings.TrimSpace(platform)
	if base == "" {
		base = "browser_data"
	}
	if base == "browser_data" && platform != "" {
		base = filepath.Join(base, platform)
	}

	if save {
		abs, err := filepath.Abs(base)
		if err != nil {
			return "", nil, err
		}
		if err := os.MkdirAll(abs, 0755); err != nil {
			return "", nil, err
		}
		return abs, func() {}, nil
	}

	prefix := "media-crawler-"
	if platform != "" {
		prefix += platform + "-"
	}
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", nil, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

