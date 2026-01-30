package xhs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func ExtractInitialState(html string) (map[string]interface{}, error) {
	raw, err := extractBalancedJSONAfter(html, "window.__INITIAL_STATE__=")
	if err != nil {
		return nil, err
	}

	raw = strings.ReplaceAll(raw, ":undefined", ":null")
	raw = strings.ReplaceAll(raw, "undefined", "null")

	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.UseNumber()
	var out map[string]interface{}
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func ExtractCreatorUserPageData(html string) (map[string]interface{}, error) {
	state, err := ExtractInitialState(html)
	if err != nil {
		return nil, err
	}
	userAny, ok := state["user"]
	if !ok || userAny == nil {
		return nil, fmt.Errorf("missing state.user")
	}
	userMap, ok := userAny.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid state.user type")
	}
	updAny, ok := userMap["userPageData"]
	if !ok || updAny == nil {
		return nil, fmt.Errorf("missing state.user.userPageData")
	}
	updMap, ok := updAny.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid state.user.userPageData type")
	}
	return updMap, nil
}

func extractBalancedJSONAfter(text, marker string) (string, error) {
	idx := strings.Index(text, marker)
	if idx == -1 {
		return "", fmt.Errorf("marker not found: %s", marker)
	}
	s := text[idx+len(marker):]
	s = strings.TrimLeft(s, " \t\r\n")
	start := strings.IndexByte(s, '{')
	if start == -1 {
		return "", fmt.Errorf("json start not found after marker")
	}
	s = s[start:]

	depth := 0
	inString := false
	escape := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[:i+1], nil
			}
		}
	}
	return "", fmt.Errorf("unterminated json object")
}
