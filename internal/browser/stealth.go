package browser

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"media-crawler-go/internal/config"

	"github.com/playwright-community/playwright-go"
)

const stealthScript = `(function () {
  try {
    Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
  } catch (e) {}

  try {
    window.chrome = window.chrome || { runtime: {} };
  } catch (e) {}

  try {
    Object.defineProperty(navigator, 'languages', { get: () => ['zh-CN', 'zh', 'en-US', 'en'] });
  } catch (e) {}

  try {
    Object.defineProperty(navigator, 'plugins', { get: () => [1, 2, 3, 4, 5] });
  } catch (e) {}

  try {
    const originalQuery = window.navigator.permissions && window.navigator.permissions.query;
    if (originalQuery) {
      window.navigator.permissions.query = (parameters) => (
        parameters && parameters.name === 'notifications'
          ? Promise.resolve({ state: Notification.permission })
          : originalQuery(parameters)
      );
    }
  } catch (e) {}

  try {
    const getParameter = WebGLRenderingContext.prototype.getParameter;
    WebGLRenderingContext.prototype.getParameter = function (parameter) {
      if (parameter === 37445) return 'Intel Inc.';
      if (parameter === 37446) return 'Intel Iris OpenGL Engine';
      return getParameter.call(this, parameter);
    };
  } catch (e) {}
})();`

var (
	stealthOnce  sync.Once
	stealthFinal string
)

func resolvedStealthScript() string {
	stealthOnce.Do(func() {
		candidates := make([]string, 0, 4)
		if p := resolveStealthPath(strings.TrimSpace(config.AppConfig.StealthScriptPath)); p != "" {
			candidates = append(candidates, p)
		}
		if p := resolveStealthPath("libs/stealth.min.js"); p != "" {
			candidates = append(candidates, p)
		}
		if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
			if p := resolveStealthPath(filepath.Join(filepath.Dir(exe), "libs/stealth.min.js")); p != "" {
				candidates = append(candidates, p)
			}
		}
		for _, p := range candidates {
			if b, err := os.ReadFile(p); err == nil {
				if s := strings.TrimSpace(string(b)); s != "" {
					stealthFinal = s
					return
				}
			}
		}
		stealthFinal = stealthScript
	})
	return stealthFinal
}

func resolveStealthPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)

	if st, err := os.Stat(p); err == nil {
		if st.IsDir() {
			fp := filepath.Join(p, "stealth.min.js")
			if st2, err2 := os.Stat(fp); err2 == nil && !st2.IsDir() {
				return fp
			}
			return ""
		}
		return p
	}

	if filepath.IsAbs(p) {
		return ""
	}

	if exe, err := os.Executable(); err == nil && strings.TrimSpace(exe) != "" {
		fp := filepath.Join(filepath.Dir(exe), p)
		if st, err := os.Stat(fp); err == nil && !st.IsDir() {
			return fp
		}
		if st, err := os.Stat(filepath.Dir(fp)); err == nil && st.IsDir() {
			fp2 := filepath.Join(filepath.Dir(fp), "stealth.min.js")
			if st2, err2 := os.Stat(fp2); err2 == nil && !st2.IsDir() {
				return fp2
			}
		}
	}
	return ""
}

func InjectStealthToPage(page playwright.Page) error {
	if page == nil {
		return nil
	}
	return page.AddInitScript(playwright.Script{Content: playwright.String(resolvedStealthScript())})
}

func InjectStealthToContext(ctx playwright.BrowserContext) error {
	if ctx == nil {
		return nil
	}
	return ctx.AddInitScript(playwright.Script{Content: playwright.String(resolvedStealthScript())})
}
