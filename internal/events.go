package internal

import "sync"

// EventHub is a minimal fan-out for "something changed" notifications. It backs
// the dashboard's server-sent-events stream so the UI can update the instant a
// label prints or a fault toggles, instead of polling.
type EventHub struct {
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

func NewEventHub() *EventHub {
	return &EventHub{subs: make(map[chan struct{}]struct{})}
}

// Subscribe registers a listener and returns its notification channel. The
// channel is buffered by one so a burst of updates coalesces into a single
// pending signal. Callers must Unsubscribe when done.
func (h *EventHub) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subs[ch] = struct{}{}
	return ch
}

func (h *EventHub) Unsubscribe(ch chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
}

// Publish signals every subscriber without blocking. A subscriber that already
// has a pending signal is skipped, since it will pick up the latest state when
// it wakes.
func (h *EventHub) Publish() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
