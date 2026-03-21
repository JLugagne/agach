package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	maxMessageSize  = 4 * 1024
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = (pongWait * 9) / 10
	maxClients      = 1000
	broadcastBuffer = 256
)

// Event represents a WebSocket event
type Event struct {
	Type      string      `json:"type"`
	ProjectID string      `json:"project_id,omitempty"`
	Data      interface{} `json:"data"`
}

// Hub maintains the set of active WebSocket connections and broadcasts events
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	mu         sync.Mutex
	logger     *logrus.Logger
}

// Client represents a WebSocket client connection
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan Event
	projectID string
	closed    bool
	closeMu   sync.Mutex
}

// NewHub creates a new WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Event, broadcastBuffer),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
		logger:     logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case <-h.stop:
			h.mu.Lock()
			for client := range h.clients {
				delete(h.clients, client)
				h.closeClientSend(client)
			}
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			clientCount := len(h.clients)
			if clientCount >= maxClients {
				h.mu.Unlock()
				h.logger.WithField("max_clients", maxClients).Warn("Max client limit reached, rejecting connection")
				client.conn.Close()
				continue
			}
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.WithField("client_count", clientCount+1).Debug("Client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				h.closeClientSend(client)
			}
			h.mu.Unlock()
			h.logger.WithField("client_count", len(h.clients)).Debug("Client unregistered")

		case event := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				if client.projectID != "" && event.ProjectID != "" && client.projectID != event.ProjectID {
					continue
				}
				select {
				case client.send <- event:
				default:
					h.logger.WithField("event_type", event.Type).Warn("Client send buffer full, dropping event and closing connection")
					delete(h.clients, client)
					h.closeClientSend(client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// closeClientSend closes the client's send channel exactly once. Must be called with h.mu held.
func (h *Hub) closeClientSend(client *Client) {
	client.closeMu.Lock()
	defer client.closeMu.Unlock()
	if !client.closed {
		client.closed = true
		close(client.send)
	}
}

// Stop signals the hub's Run loop to exit cleanly.
func (h *Hub) Stop() {
	close(h.stop)
}

// Broadcast sends an event to all connected clients. Non-blocking: drops the
// event and logs if the broadcast channel is full.
func (h *Hub) Broadcast(event Event) {
	select {
	case h.broadcast <- event:
	default:
		h.logger.WithField("event_type", event.Type).Warn("Broadcast channel full, dropping event")
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.WithError(err).Warn("WebSocket read error")
			}
			if err == websocket.ErrReadLimit {
				// Drain the underlying TCP receive buffer to avoid sending RST to
				// the client, which would abort any in-flight write.
				nc := c.conn.UnderlyingConn()
				nc.SetDeadline(time.Now().Add(500 * time.Millisecond))
				buf := make([]byte, 4096)
				for {
					_, drainErr := nc.Read(buf)
					if drainErr != nil {
						break
					}
				}
			}
			break
		}
	}
}

// WritePump pumps events from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.hub.logger.WithError(err).Warn("WritePump: NextWriter failed, dropping remaining events")
				return
			}

			if err := json.NewEncoder(w).Encode(event); err != nil {
				c.hub.logger.WithError(err).Error("Failed to encode event")
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWS handles WebSocket requests from clients.
// projectID, if non-empty, scopes the client to only receive events for that project.
func (h *Hub) ServeWS(conn *websocket.Conn, projectID ...string) {
	pid := ""
	if len(projectID) > 0 {
		pid = projectID[0]
	}
	client := &Client{
		hub:       h,
		conn:      conn,
		send:      make(chan Event, 256),
		projectID: pid,
	}

	h.register <- client

	go client.WritePump()
	go client.ReadPump()
}
