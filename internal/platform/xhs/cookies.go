package xhs

import (
	"strings"

	"github.com/playwright-community/playwright-go"
)

func buildCookies(cookieStr string) []playwright.OptionalCookie {
	if strings.TrimSpace(cookieStr) == "" {
		return nil
	}
	cookies := make([]playwright.OptionalCookie, 0)
	for _, cookieItem := range strings.Split(cookieStr, ";") {
		parts := strings.SplitN(strings.TrimSpace(cookieItem), "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := parts[1]
		if name == "" || value == "" {
			continue
		}
		cookies = append(cookies, playwright.OptionalCookie{
			Name:   name,
			Value:  value,
			Domain: playwright.String(".xiaohongshu.com"),
			Path:   playwright.String("/"),
		})
	}
	return cookies
}
