package sse

import (
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	maxSubscribersPerProject = 1000
	maxDataSize              = 64 * 1024 // 64 KB max SSE message
	heartbeatInterval        = time.Second
)

type subscriber struct {
	ch     chan string
	closed bool
	done   chan struct{}
}

// Hub manages SSE subscribers per project
type Hub struct {
	mu          sync.Mutex
	subscribers map[string][]*subscriber
	logger      *logrus.Logger
}

func NewHub(logger *logrus.Logger) *Hub {
	return &Hub{subscribers: make(map[string][]*subscriber), logger: logger}
}

func isValidProjectID(id string) bool {
	if id == "" || len(id) > 500 {
		return false
	}
	// Must contain at least one dash or pipe (UUID format, composite IDs)
	// to reject simple strings like "admin", "*"
	if !strings.ContainsAny(id, "-|:") {
		return false
	}
	// Reject obvious attack payloads
	if strings.ContainsAny(id, "<>\"'&;\\") {
		return false
	}
	return true
}

func (h *Hub) Subscribe(projectID string) (chan string, func()) {
	if !isValidProjectID(projectID) {
		return nil, func() {}
	}

	h.mu.Lock()
	if len(h.subscribers[projectID]) >= maxSubscribersPerProject {
		h.mu.Unlock()
		return nil, func() {}
	}

	ch := make(chan string, 10)
	sub := &subscriber{ch: ch, done: make(chan struct{})}
	h.subscribers[projectID] = append(h.subscribers[projectID], sub)
	h.mu.Unlock()

	go h.runHeartbeat(projectID, sub)

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.removeSub(projectID, sub)
		})
	}

	return ch, unsubscribe
}

func (h *Hub) runHeartbeat(projectID string, sub *subscriber) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-sub.done:
			return
		case <-ticker.C:
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
}

func (h *Hub) removeSub(projectID string, sub *subscriber) {
	if sub.closed {
		return
	}
	close(sub.done)
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
	data = strings.ReplaceAll(data, "\x00", "")
	if len(data) > maxDataSize {
		data = data[:maxDataSize]
	}
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
			h.logger.WithField("project_id", projectID).Warn("SSE subscriber channel full, evicting slow consumer")
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
