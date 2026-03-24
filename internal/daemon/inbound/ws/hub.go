package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github.com/JLugagne/agach-mcp/pkg/daemonws"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	sendBuffer = 256
)

// Client represents a connected WebSocket client.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan daemonws.Message
	closed bool
	mu     sync.Mutex
}

// Hub manages daemon WebSocket connections with targeted request/response messaging.
type Hub struct {
	clients    map[*Client]bool
	handlers   map[string]HandlerFunc
	register   chan *Client
	unregister chan *Client
	events     chan daemonws.Message
	mu         sync.RWMutex
	logger     *logrus.Logger
}

// NewHub creates a new daemon WebSocket hub.
func NewHub(logger *logrus.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		handlers:   make(map[string]HandlerFunc),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		events:     make(chan daemonws.Message, sendBuffer),
		logger:     logger,
	}
}

// RegisterHandler registers a handler for a message type.
func (h *Hub) RegisterHandler(msgType string, handler HandlerFunc) {
	h.handlers[msgType] = handler
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.closeSend()
			}
			h.mu.Unlock()

		case msg := <-h.events:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					// drop if full
				}
			}
			h.mu.RUnlock()
		}
	}
}

// HandleConnection handles a new WebSocket connection.
func (h *Hub) HandleConnection(conn *websocket.Conn) {
	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan daemonws.Message, sendBuffer),
	}
	h.register <- client

	go client.writePump()
	go client.readPump()
}

// Send sends a message to the connected client.
func (h *Hub) Send(msg daemonws.Message) error {
	return nil
}

// SendEvent pushes an event to all connected clients.
func (h *Hub) SendEvent(msg daemonws.Message) {
	select {
	case h.events <- msg:
	default:
	}
}

func (c *Client) closeSend() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

func (c *Client) readPump() {
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
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				c.hub.logger.WithError(err).Warn("WebSocket read error")
			}
			return
		}

		var msg daemonws.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.hub.logger.WithError(err).Warn("Invalid message format")
			continue
		}

		handler, ok := c.hub.handlers[msg.Type]
		if !ok {
			resp := daemonws.Message{
				Type:      daemonws.TypeError,
				RequestID: msg.RequestID,
				Error:     "unknown message type: " + msg.Type,
			}
			select {
			case c.send <- resp:
			default:
			}
			continue
		}

		resp, err := handler(context.Background(), msg)
		if err != nil {
			resp = daemonws.Message{
				Type:      daemonws.TypeError,
				RequestID: msg.RequestID,
				Error:     err.Error(),
			}
		}
		if resp.RequestID == "" {
			resp.RequestID = msg.RequestID
		}

		select {
		case c.send <- resp:
		default:
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
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
