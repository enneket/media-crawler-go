package browser

import (
	"os"
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
		p := strings.TrimSpace(config.AppConfig.StealthScriptPath)
		if p == "" {
			if _, err := os.Stat("libs/stealth.min.js"); err == nil {
				p = "libs/stealth.min.js"
			}
		}
		if p != "" {
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
