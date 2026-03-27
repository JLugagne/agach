package websocket

import (
	"encoding/json"
	"net"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	maxClients        = 1000
	maxClientsPerIP   = 20
	broadcastBuffer   = 256
)

// Event represents a WebSocket event
type Event struct {
	Type      string      `json:"type"`
	ProjectID string      `json:"project_id,omitempty"`
	Data      interface{} `json:"data"`
}

// RelayHandlerFunc is called when a message of a registered type is received from a client.
type RelayHandlerFunc func(client *Client, msg []byte)

// Hub maintains the set of active WebSocket connections and broadcasts events
type Hub struct {
	clients      map[*Client]bool
	ipCount      map[string]int // per-IP connection count
	broadcast    chan Event
	register     chan *Client
	unregister   chan *Client
	relay        chan relayMessage
	stop         chan struct{}
	handlers     map[string]RelayHandlerFunc
	mu           sync.Mutex
	logger       *logrus.Logger
	droppedCount int64 // backpressure counter for broadcast overflow
}

// relayMessage carries a raw message from one client to be forwarded to others.
type relayMessage struct {
	from *Client
	data json.RawMessage
}

// Client represents a WebSocket client connection
type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	send       chan Event
	projectID  string
	remoteAddr string
	isDaemon   bool
	nodeID     string
	closed    bool
	closeMu   sync.Mutex
	writeMu   sync.Mutex // protects conn writes (WritePump, sendRaw, pings)
}

// DoCloseSend closes the send channel exactly once. Safe to call from multiple goroutines.
func (c *Client) DoCloseSend() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

// NewHub creates a new WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		ipCount:    make(map[string]int),
		broadcast:  make(chan Event, broadcastBuffer),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		relay:      make(chan relayMessage, broadcastBuffer),
		stop:       make(chan struct{}),
		handlers:   make(map[string]RelayHandlerFunc),
		logger:     logger,
	}
}

// RegisterHandler registers a handler for the given WebSocket message type.
func (h *Hub) RegisterHandler(msgType string, fn RelayHandlerFunc) {
	h.handlers[msgType] = fn
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case <-h.stop:
			h.mu.Lock()
			for client := range h.clients {
				delete(h.clients, client)
				client.DoCloseSend()
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
			// Per-IP connection limit
			ip := client.remoteAddr
			if ip != "" && h.ipCount[ip] >= maxClientsPerIP {
				h.mu.Unlock()
				h.logger.WithFields(logrus.Fields{"ip": ip, "limit": maxClientsPerIP}).Warn("Per-IP connection limit reached, rejecting")
				client.conn.Close()
				continue
			}
			h.clients[client] = true
			if ip != "" {
				h.ipCount[ip]++
			}
			h.mu.Unlock()
			h.logger.WithField("client_count", clientCount+1).Debug("Client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				ip := client.remoteAddr
				if ip != "" {
					h.ipCount[ip]--
					if h.ipCount[ip] <= 0 {
						delete(h.ipCount, ip)
					}
				}
			}
			h.mu.Unlock()
			HandleUnregister(h.clients, &h.mu, client, h.logger)

		case event := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				if event.ProjectID != "" && client.projectID != event.ProjectID {
					continue
				}
				select {
				case client.send <- event:
				default:
					h.logger.WithField("event_type", event.Type).Warn("Client send buffer full, dropping event and closing connection")
					delete(h.clients, client)
					client.DoCloseSend()
				}
			}
			h.mu.Unlock()

		case msg := <-h.relay:
			// Extract project_id from the relay message payload for filtering.
			msgProjectID := msg.from.projectID
			if msgProjectID == "" {
				var envelope struct {
					ProjectID string `json:"project_id"`
				}
				_ = json.Unmarshal(msg.data, &envelope)
				msgProjectID = envelope.ProjectID
			}

			h.mu.Lock()
			if msg.from.isDaemon {
				for client := range h.clients {
					if client.isDaemon {
						continue
					}
					if msgProjectID != "" && client.projectID != msgProjectID {
						continue
					}
					h.sendRaw(client, msg.data)
				}
			} else {
				for client := range h.clients {
					if !client.isDaemon {
						continue
					}
					if msgProjectID != "" && client.projectID != msgProjectID {
						continue
					}
					h.sendRaw(client, msg.data)
				}
			}
			h.mu.Unlock()
		}
	}
}

// sendRaw writes a raw JSON message to a client. Must be called with h.mu held.
func (h *Hub) sendRaw(client *Client, data json.RawMessage) {
	client.closeMu.Lock()
	if client.closed {
		client.closeMu.Unlock()
		return
	}
	client.closeMu.Unlock()
	client.writeMu.Lock()
	defer client.writeMu.Unlock()
	_ = client.conn.SetWriteDeadline(time.Now().Add(WriteWait))
	if err := client.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		h.logger.WithError(err).Debug("relay write failed")
	}
}

// Stop signals the hub's Run loop to exit cleanly.
func (h *Hub) Stop() {
	close(h.stop)
}

var (
	scriptBlockRe = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	htmlTagRe     = regexp.MustCompile(`<[^>]*>`)
)

// sanitizeEventData strips HTML tags from string values in the event data.
func sanitizeEventData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		cleaned := make(map[string]interface{}, len(v))
		for k, val := range v {
			cleaned[k] = sanitizeEventData(val)
		}
		return cleaned
	case map[string]string:
		cleaned := make(map[string]string, len(v))
		for k, val := range v {
			val = scriptBlockRe.ReplaceAllString(val, "")
			cleaned[k] = htmlTagRe.ReplaceAllString(val, "")
		}
		return cleaned
	case string:
		v = scriptBlockRe.ReplaceAllString(v, "")
		return htmlTagRe.ReplaceAllString(v, "")
	default:
		return data
	}
}

// Broadcast sends an event to all connected clients. Non-blocking: drops the
// event and logs if the broadcast channel is full. String values in the event
// data are sanitized to strip HTML tags.
func (h *Hub) Broadcast(event Event) {
	event.Data = sanitizeEventData(event.Data)
	select {
	case h.broadcast <- event:
	default:
		h.droppedCount++
		h.logger.WithField("event_type", event.Type).Warn("Broadcast channel full, dropping event (backpressure)")
	}
}

// NewRelayHandler builds a RelayHandlerFunc that forwards the raw message through the relay channel.
func (h *Hub) NewRelayHandler() RelayHandlerFunc {
	return func(client *Client, msg []byte) {
		select {
		case h.relay <- relayMessage{from: client, data: msg}:
		default:
			h.logger.Warn("Relay channel full, dropping message")
		}
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.WithError(err).Warn("WebSocket read error")
			}
			if err == websocket.ErrReadLimit {
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

		var peek struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &peek); err != nil {
			continue
		}
		if handler, ok := c.hub.handlers[peek.Type]; ok {
			handler(c, data)
		}
	}
}

// WritePump pumps events from the hub to the WebSocket connection.
// All writes are serialised via writeMu to prevent concurrent writes
// from sendRaw (relay/targeted messages) and this pump.
func (c *Client) WritePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			c.writeMu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.writeMu.Unlock()
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.writeMu.Unlock()
				return
			}
			if err := json.NewEncoder(w).Encode(event); err != nil {
				c.hub.logger.WithError(err).Error("WritePump: encode failed")
			}
			w.Close()
			c.writeMu.Unlock()

		case <-ticker.C:
			c.writeMu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			c.writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// ServeWSOption configures a WebSocket client.
type ServeWSOption func(*Client)

// WithProjectID scopes the client to only receive events for that project.
func WithProjectID(pid string) ServeWSOption {
	return func(c *Client) { c.projectID = pid }
}

// AsDaemon marks the client as a daemon connection for message relay.
func AsDaemon() ServeWSOption {
	return func(c *Client) { c.isDaemon = true }
}

// WithNodeID sets the node ID on a daemon client for targeted message delivery.
func WithNodeID(nodeID string) ServeWSOption {
	return func(c *Client) { c.nodeID = nodeID }
}

// SendToDaemon sends a raw JSON message to the daemon client with the given node ID.
// Returns false if no matching daemon is connected.
func (h *Hub) SendToDaemon(nodeID string, data json.RawMessage) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		if client.isDaemon && client.nodeID == nodeID {
			h.sendRaw(client, data)
			return true
		}
	}
	h.logger.WithField("node_id", nodeID).Warn("SendToDaemon: no matching daemon found")
	return false
}

// ConnectedDaemonNodeIDs returns the node IDs of all connected daemon clients.
func (h *Hub) ConnectedDaemonNodeIDs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var ids []string
	for client := range h.clients {
		if client.isDaemon && client.nodeID != "" {
			ids = append(ids, client.nodeID)
		}
	}
	return ids
}

// ServeWS handles WebSocket requests from clients.
// CanAcceptIP checks if the hub can accept another connection from the given IP.
// Call this before upgrading the WebSocket connection to reject early.
func (h *Hub) CanAcceptIP(ip string) bool {
	if ip == "" {
		return true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.ipCount[ip] < maxClientsPerIP
}

func (h *Hub) ServeWS(conn *websocket.Conn, opts ...ServeWSOption) {
	// Extract IP from remote address (strip port)
	addr := conn.RemoteAddr().String()
	if host, _, err := net.SplitHostPort(addr); err == nil {
		addr = host
	}

	client := &Client{
		hub:        h,
		conn:       conn,
		send:       make(chan Event, 256),
		remoteAddr: addr,
	}
	for _, opt := range opts {
		opt(client)
	}

	h.register <- client

	go client.WritePump()
	go client.ReadPump()
}
