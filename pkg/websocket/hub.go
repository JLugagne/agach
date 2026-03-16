package websocket

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
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
	mu         sync.RWMutex
	logger     *logrus.Logger
}

// Client represents a WebSocket client connection
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan Event
}

// NewHub creates a new WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.WithField("client_count", len(h.clients)).Debug("Client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.WithField("client_count", len(h.clients)).Debug("Client unregistered")

		case event := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- event:
				default:
					// Client buffer full, drop message and close connection
					h.logger.Warn("Client send buffer full, closing connection")
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends an event to all connected clients
func (h *Hub) Broadcast(event Event) {
	h.broadcast <- event
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.WithError(err).Warn("WebSocket read error")
			}
			break
		}
		// We don't expect clients to send messages, only receive
	}
}

// WritePump pumps events from the hub to the WebSocket connection
func (c *Client) WritePump() {
	defer func() {
		c.conn.Close()
	}()

	for event := range c.send {
		w, err := c.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}

		if err := json.NewEncoder(w).Encode(event); err != nil {
			c.hub.logger.WithError(err).Error("Failed to encode event")
		}

		if err := w.Close(); err != nil {
			return
		}
	}
}

// ServeWS handles WebSocket requests from clients
func (h *Hub) ServeWS(conn *websocket.Conn) {
	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan Event, 256),
	}

	h.register <- client

	// Start pumps in goroutines
	go client.WritePump()
	go client.ReadPump()
}
