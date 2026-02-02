package crawler

import "strings"

func DetectRiskHint(body string) string {
	s := strings.TrimSpace(body)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "captcha") || strings.Contains(lower, "recaptcha") {
		return "captcha"
	}
	if strings.Contains(s, "验证码") || strings.Contains(s, "人机验证") || strings.Contains(s, "安全验证") {
		return "captcha"
	}
	if strings.Contains(s, "请通过验证") || strings.Contains(s, "访问验证") {
		return "captcha"
	}
	if strings.Contains(lower, "forbidden") || strings.Contains(lower, "access denied") {
		return "forbidden"
	}
	return ""
}
