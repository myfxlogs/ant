package service

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type debateV2EventHub struct {
	mu   sync.Mutex
	subs map[uuid.UUID]map[chan struct{}]struct{}
}

func newDebateV2EventHub() *debateV2EventHub {
	return &debateV2EventHub{subs: map[uuid.UUID]map[chan struct{}]struct{}{}}
}

func (h *debateV2EventHub) subscribe(ctx context.Context, sessionID uuid.UUID) <-chan struct{} {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	if h.subs[sessionID] == nil {
		h.subs[sessionID] = map[chan struct{}]struct{}{}
	}
	h.subs[sessionID][ch] = struct{}{}
	h.mu.Unlock()
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		if set := h.subs[sessionID]; set != nil {
			delete(set, ch)
			if len(set) == 0 {
				delete(h.subs, sessionID)
			}
		}
		h.mu.Unlock()
		close(ch)
	}()
	return ch
}

func (h *debateV2EventHub) publish(sessionID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[sessionID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
