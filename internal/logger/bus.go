package logger

import "sync"

type bus struct {
	mu   sync.RWMutex
	subs map[chan []byte]struct{}
}

func newBus() *bus {
	return &bus{subs: map[chan []byte]struct{}{}}
}

func (b *bus) subscribe(buffer int) (chan []byte, func()) {
	if buffer < 1 {
		buffer = 1
	}
	ch := make(chan []byte, buffer)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() { b.unsubscribe(ch) }
}

func (b *bus) unsubscribe(ch chan []byte) {
	b.mu.Lock()
	if _, ok := b.subs[ch]; ok {
		delete(b.subs, ch)
		close(ch)
	}
	b.mu.Unlock()
}

func (b *bus) count() int {
	b.mu.RLock()
	n := len(b.subs)
	b.mu.RUnlock()
	return n
}

func (b *bus) publish(msg []byte) {
	if len(msg) == 0 {
		return
	}
	b.mu.RLock()
	if len(b.subs) == 0 {
		b.mu.RUnlock()
		return
	}
	copied := append([]byte(nil), msg...)
	for ch := range b.subs {
		select {
		case ch <- copied:
		default:
		}
	}
	b.mu.RUnlock()
}

var defaultBus = newBus()

func Subscribe() (<-chan []byte, func()) {
	ch, cancel := defaultBus.subscribe(256)
	return ch, cancel
}
