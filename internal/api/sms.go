package api

import (
	"encoding/json"
	"media-crawler-go/internal/sms"
	"net/http"
	"strings"
	"time"
)

type smsNotification struct {
	Platform       string `json:"platform"`
	CurrentNumber  string `json:"current_number"`
	FromNumber     string `json:"from_number"`
	SmsContent     string `json:"sms_content"`
	Timestamp      string `json:"timestamp"`
	Phone          string `json:"phone"`
	Content        string `json:"content"`
	Verification   string `json:"verification_code"`
	VerificationCode string `json:"code"`
}

func (s *Server) handleSMS(w http.ResponseWriter, r *http.Request) {
	var n smsNotification
	dec := json.NewDecoder(r.Body)
	_ = dec.Decode(&n)

	platform := strings.TrimSpace(n.Platform)
	if platform == "" {
		platform = "xhs"
	}

	phone := strings.TrimSpace(n.CurrentNumber)
	if phone == "" {
		phone = strings.TrimSpace(n.Phone)
	}

	extracted := strings.TrimSpace(n.Verification)
	if extracted == "" {
		extracted = strings.TrimSpace(n.VerificationCode)
	}
	content := strings.TrimSpace(n.SmsContent)
	if content == "" {
		content = strings.TrimSpace(n.Content)
	}
	if extracted == "" {
		extracted = sms.ExtractCode(content)
	}
	if phone != "" && extracted != "" {
		_ = sms.Store(platform, phone, extracted, 3*time.Minute)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	return

	code := strings.TrimSpace(n.Verification)
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}
	if code == "" {
		code = strings.TrimSpace(n.VerificationCode)
	}

	if code == "" {
		content := strings.TrimSpace(n.SmsContent)
		if content == "" {
			content = strings.TrimSpace(n.Content)
		}
		code = sms.ExtractCode(content)
	}

	if phone != "" && code != "" {
		_ = sms.Store(platform, phone, code, 3*time.Minute)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
