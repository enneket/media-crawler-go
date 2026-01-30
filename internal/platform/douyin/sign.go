package douyin

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/dop251/goja"
)

type Signer struct {
	runtime *goja.Runtime
}

func NewSigner() (*Signer, error) {
	rt := goja.New()
	if err := rt.Set("btoa", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return rt.ToValue("")
		}
		s := call.Arguments[0].String()
		enc := base64.StdEncoding.EncodeToString([]byte(s))
		return rt.ToValue(enc)
	}); err != nil {
		return nil, fmt.Errorf("set btoa: %w", err)
	}
	js, err := loadSignJS()
	if err != nil {
		return nil, err
	}
	if _, err := rt.RunString(js); err != nil {
		return nil, fmt.Errorf("load douyin.js: %w", err)
	}
	return &Signer{runtime: rt}, nil
}

func (s *Signer) SignDetail(params url.Values, ua string) (string, error) {
	return s.callFn("sign_datail", params, ua)
}

func (s *Signer) SignReply(params url.Values, ua string) (string, error) {
	return s.callFn("sign_reply", params, ua)
}

func (s *Signer) callFn(name string, params url.Values, ua string) (string, error) {
	fn, ok := goja.AssertFunction(s.runtime.Get(name))
	if !ok {
		return "", fmt.Errorf("js function %s not found", name)
	}
	query := params.Encode()
	val, err := fn(goja.Undefined(), s.runtime.ToValue(query), s.runtime.ToValue(ua))
	if err != nil {
		return "", err
	}
	str, ok := val.Export().(string)
	if !ok {
		return "", fmt.Errorf("%s returned non-string", name)
	}
	return str, nil
}

func loadSignJS() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "douyin.js")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
