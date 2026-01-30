package xhs

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

func md5Hex(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// Signer handles the XHS signing logic using Playwright
type Signer struct {
	Page playwright.Page
	mu   sync.Mutex
}

func NewSigner(page playwright.Page) *Signer {
	return &Signer{Page: page}
}

func (s *Signer) GetB1() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, err := s.Page.Evaluate("() => window.localStorage.getItem('b1')")
	if err != nil {
		return ""
	}
	if v, ok := val.(string); ok {
		return v
	}
	return ""
}

func (s *Signer) CallMnsv2(signStr, md5Str string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	signStrEscaped := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(signStr, "\\", "\\\\"), "'", "\\'"), "\n", "\\n")
	md5StrEscaped := strings.ReplaceAll(strings.ReplaceAll(md5Str, "\\", "\\\\"), "'", "\\'")

	script := fmt.Sprintf("window.mnsv2('%s', '%s')", signStrEscaped, md5StrEscaped)
	val, err := s.Page.Evaluate(script)
	if err != nil {
		return "", err
	}
	if v, ok := val.(string); ok {
		return v, nil
	}
	return "", fmt.Errorf("mnsv2 returned non-string")
}

func (s *Signer) Sign(uri string, data interface{}, a1 string, method string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	signStr := buildSignString(uri, data, method)
	md5Str := md5Hex(signStr)
	
	x3, err := s.callMnsv2Locked(signStr, md5Str)
	if err != nil {
		return nil, err
	}

	dataType := "object"
	if _, ok := data.(string); ok {
		dataType = "string"
	} else if data == nil {
		dataType = "string" // or object? Python: "object" if dict/list else "string". If nil? Python code says "if isinstance(data, (dict, list))". nil is NoneType -> string.
	}

	xs := buildXsPayload(x3, dataType)
	xt := fmt.Sprintf("%d", time.Now().UnixMilli())
	b1 := s.getB1Locked()
	xsCommon := buildXsCommon(a1, b1, xs, xt)
	traceId := getTraceId()

	return map[string]string{
		"X-S":          xs,
		"X-T":          xt,
		"x-S-Common":   xsCommon,
		"X-B3-Traceid": traceId,
	}, nil
}

func (s *Signer) getB1Locked() string {
	val, err := s.Page.Evaluate("() => window.localStorage.getItem('b1')")
	if err != nil {
		return ""
	}
	if v, ok := val.(string); ok {
		return v
	}
	return ""
}

func (s *Signer) callMnsv2Locked(signStr, md5Str string) (string, error) {
	signStrEscaped := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(signStr, "\\", "\\\\"), "'", "\\'"), "\n", "\\n")
	md5StrEscaped := strings.ReplaceAll(strings.ReplaceAll(md5Str, "\\", "\\\\"), "'", "\\'")

	script := fmt.Sprintf("window.mnsv2('%s', '%s')", signStrEscaped, md5StrEscaped)
	val, err := s.Page.Evaluate(script)
	if err != nil {
		return "", err
	}
	if v, ok := val.(string); ok {
		return v, nil
	}
	return "", fmt.Errorf("mnsv2 returned non-string")
}
