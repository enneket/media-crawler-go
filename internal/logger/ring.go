package logger

import "sync"

type Event struct {
	Time  string         `json:"time"`
	Level string         `json:"level"`
	Msg   string         `json:"msg"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

var (
	ringMu   sync.Mutex
	ringMax  = 2000
	ringLogs []Event
)

func addEvent(evt Event) {
	ringMu.Lock()
	if ringMax < 1 {
		ringMax = 1
	}
	if len(ringLogs) < ringMax {
		ringLogs = append(ringLogs, evt)
		ringMu.Unlock()
		return
	}
	copy(ringLogs, ringLogs[1:])
	ringLogs[len(ringLogs)-1] = evt
	ringMu.Unlock()
}

func Recent(limit int) []Event {
	ringMu.Lock()
	if limit <= 0 || limit > len(ringLogs) {
		limit = len(ringLogs)
	}
	start := len(ringLogs) - limit
	out := append([]Event(nil), ringLogs[start:]...)
	ringMu.Unlock()
	return out
}

