package logger

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

type BroadcastHandler struct {
	next   slog.Handler
	attrs  []slog.Attr
	groups []string
}

func NewBroadcastHandler(next slog.Handler) slog.Handler {
	if next == nil {
		return nil
	}
	return &BroadcastHandler{next: next}
}

func (h *BroadcastHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *BroadcastHandler) Handle(ctx context.Context, r slog.Record) error {
	err := h.next.Handle(ctx, r)

	evt := map[string]any{
		"time":  formatTime(r.Time),
		"level": r.Level.String(),
		"msg":   r.Message,
	}

	attrs := map[string]any{}
	for _, a := range h.attrs {
		addAttr(attrs, h.groups, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		addAttr(attrs, h.groups, a)
		return true
	})
	evt["attrs"] = attrs
	addEvent(Event{
		Time:  evt["time"].(string),
		Level: evt["level"].(string),
		Msg:   evt["msg"].(string),
		Attrs: attrs,
	})

	if defaultBus.count() > 0 {
		b, mErr := json.Marshal(evt)
		if mErr == nil {
			b = append(b, '\n')
			defaultBus.publish(b)
		}
	}
	return err
}

func (h *BroadcastHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := &BroadcastHandler{
		next:   h.next.WithAttrs(attrs),
		attrs: append(append([]slog.Attr(nil), h.attrs...), attrs...),
	}
	if len(h.groups) > 0 {
		out.groups = append([]string(nil), h.groups...)
	}
	return out
}

func (h *BroadcastHandler) WithGroup(name string) slog.Handler {
	name = stringsTrimSpace(name)
	out := &BroadcastHandler{
		next:   h.next.WithGroup(name),
		attrs: append([]slog.Attr(nil), h.attrs...),
	}
	if len(h.groups) > 0 {
		out.groups = append([]string(nil), h.groups...)
	}
	if name != "" {
		out.groups = append(out.groups, name)
	}
	return out
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func stringsTrimSpace(s string) string {
	i := 0
	j := len(s)
	for i < j {
		if s[i] != ' ' && s[i] != '\n' && s[i] != '\t' && s[i] != '\r' {
			break
		}
		i++
	}
	for j > i {
		c := s[j-1]
		if c != ' ' && c != '\n' && c != '\t' && c != '\r' {
			break
		}
		j--
	}
	return s[i:j]
}

func addAttr(dst map[string]any, groups []string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}

	m := ensureGroupPath(dst, groups)
	if a.Value.Kind() == slog.KindGroup {
		gm := map[string]any{}
		for _, ga := range a.Value.Group() {
			addAttr(gm, nil, ga)
		}
		m[a.Key] = gm
		return
	}
	m[a.Key] = valueToAny(a.Value)
}

func ensureGroupPath(dst map[string]any, groups []string) map[string]any {
	if len(groups) == 0 {
		return dst
	}
	cur := dst
	for _, g := range groups {
		g = stringsTrimSpace(g)
		if g == "" {
			continue
		}
		next, ok := cur[g].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[g] = next
		}
		cur = next
	}
	return cur
}

func valueToAny(v slog.Value) any {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindBool:
		return v.Bool()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().UTC().Format(time.RFC3339Nano)
	case slog.KindGroup:
		m := map[string]any{}
		for _, a := range v.Group() {
			addAttr(m, nil, a)
		}
		return m
	default:
		return v.Any()
	}
}
