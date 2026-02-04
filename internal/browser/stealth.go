package browser

import "github.com/playwright-community/playwright-go"

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

func InjectStealthToPage(page playwright.Page) error {
	if page == nil {
		return nil
	}
	return page.AddInitScript(playwright.Script{Content: playwright.String(stealthScript)})
}

func InjectStealthToContext(ctx playwright.BrowserContext) error {
	if ctx == nil {
		return nil
	}
	return ctx.AddInitScript(playwright.Script{Content: playwright.String(stealthScript)})
}
