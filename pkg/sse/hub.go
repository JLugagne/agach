package sse

import (
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	maxSubscribersPerProject = 1000
	heartbeatInterval        = time.Second
)

type subscriber struct {
	ch     chan string
	closed bool
}

// Hub manages SSE subscribers per project
type Hub struct {
	mu          sync.Mutex
	subscribers map[string][]*subscriber
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string][]*subscriber)}
}

func (h *Hub) Subscribe(projectID string) (chan string, func()) {
	if projectID == "" {
		return nil, func() {}
	}

	h.mu.Lock()
	if len(h.subscribers[projectID]) >= maxSubscribersPerProject {
		h.mu.Unlock()
		return nil, func() {}
	}

	ch := make(chan string, 10)
	sub := &subscriber{ch: ch}
	h.subscribers[projectID] = append(h.subscribers[projectID], sub)
	h.mu.Unlock()

	go h.runHeartbeat(projectID, sub)

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.removeSub(projectID, sub)
	}

	return ch, unsubscribe
}

func (h *Hub) runHeartbeat(projectID string, sub *subscriber) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.Lock()
		if sub.closed {
			h.mu.Unlock()
			return
		}
		select {
		case sub.ch <- ":":
		default:
		}
		h.mu.Unlock()
	}
}

func (h *Hub) removeSub(projectID string, sub *subscriber) {
	if sub.closed {
		return
	}
	subs := h.subscribers[projectID]
	for i, s := range subs {
		if s == sub {
			h.subscribers[projectID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	if len(h.subscribers[projectID]) == 0 {
		delete(h.subscribers, projectID)
	}
	sub.closed = true
	close(sub.ch)
}

func sanitize(data string) string {
	if idx := strings.IndexByte(data, '\n'); idx >= 0 {
		data = data[:idx]
	}
	if idx := strings.IndexByte(data, '\r'); idx >= 0 {
		data = data[:idx]
	}
	return data
}

func (h *Hub) Publish(projectID, data string) {
	data = sanitize(data)

	h.mu.Lock()
	subs := make([]*subscriber, len(h.subscribers[projectID]))
	copy(subs, h.subscribers[projectID])
	h.mu.Unlock()

	for _, sub := range subs {
		h.mu.Lock()
		if sub.closed {
			h.mu.Unlock()
			continue
		}
		select {
		case sub.ch <- data:
			h.mu.Unlock()
		default:
			logrus.WithField("project_id", projectID).Warn("SSE subscriber channel full, evicting slow consumer")
			h.removeSub(projectID, sub)
			h.mu.Unlock()
		}
	}
}

func (h *Hub) HasSubscribers(projectID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers[projectID]) > 0
}
