package sse

import (
	"sync"
)

// Hub manages SSE subscribers per project
type Hub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan string
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string][]chan string)}
}

func (h *Hub) Subscribe(projectID string) (chan string, func()) {
	ch := make(chan string, 10)
	h.mu.Lock()
	h.subscribers[projectID] = append(h.subscribers[projectID], ch)
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subs := h.subscribers[projectID]
		for i, s := range subs {
			if s == ch {
				h.subscribers[projectID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		close(ch)
	}
	return ch, unsubscribe
}

func (h *Hub) Publish(projectID, data string) {
	h.mu.RLock()
	subs := make([]chan string, len(h.subscribers[projectID]))
	copy(subs, h.subscribers[projectID])
	h.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- data:
		default:
		}
	}
}

func (h *Hub) HasSubscribers(projectID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers[projectID]) > 0
}

