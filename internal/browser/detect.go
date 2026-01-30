package browser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func detectBrowserBinary(customPath string) (string, error) {
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			return customPath, nil
		}
		return "", fmt.Errorf("custom browser path not found: %s", customPath)
	}

	if v := os.Getenv("CHROME_PATH"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v, nil
		}
	}

	candidates := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"microsoft-edge",
		"msedge",
	}
	for _, name := range candidates {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}

	switch runtime.GOOS {
	case "darwin":
		macCandidates := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
		for _, p := range macCandidates {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	case "windows":
		home := os.Getenv("USERPROFILE")
		winCandidates := []string{
			filepath.Join(home, "AppData", "Local", "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(home, "AppData", "Local", "Microsoft", "Edge", "Application", "msedge.exe"),
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		}
		for _, p := range winCandidates {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	return "", fmt.Errorf("no chrome/chromium binary found; set CUSTOM_BROWSER_PATH or CHROME_PATH")
}
