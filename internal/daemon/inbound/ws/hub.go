package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	wsutil "github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	"github.com/JLugagne/agach-mcp/pkg/daemonws"
)

const (
	sendBuffer = 256
)

// Client represents a connected WebSocket client.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan daemonws.Message
	closed bool
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
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
				client.DoCloseSend()
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
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan daemonws.Message, sendBuffer),
		ctx:    ctx,
		cancel: cancel,
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

// DoCloseSend closes the send channel exactly once. Safe to call from multiple goroutines.
func (c *Client) DoCloseSend() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

func (c *Client) readPump() {
	defer func() {
		c.cancel()
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(wsutil.PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(wsutil.PongWait))
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

		resp, err := handler(c.ctx, msg)
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
	wsutil.RunWritePump(c.conn, c.send, func(conn *websocket.Conn, msg daemonws.Message) error {
		return conn.WriteJSON(msg)
	}, c.hub.logger)
}
