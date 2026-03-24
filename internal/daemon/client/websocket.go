package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4 * 1024
	reconnectMin   = 1 * time.Second
	reconnectMax   = 30 * time.Second
)

type WSEvent struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	ProjectID string          `json:"project_id,omitempty"`
	Data      json.RawMessage `json:"data"`
}

type WSEventHandler func(event WSEvent)

type WSClient struct {
	wsURL   string
	token   string
	logger  *logrus.Logger
	handler WSEventHandler

	conn       *websocket.Conn
	connMu     sync.Mutex
	connected  bool
	doneCh     chan struct{}
	stopCh     chan struct{}
	reconnects int
}

func NewWSClient(wsURL, token string, logger *logrus.Logger, handler WSEventHandler) *WSClient {
	return &WSClient{
		wsURL:   wsURL,
		token:   token,
		logger:  logger,
		handler: handler,
	}
}

func (c *WSClient) SetToken(token string) {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	c.token = token
}

func (c *WSClient) Connect(ctx context.Context) error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	u, err := url.Parse(c.wsURL)
	if err != nil {
		return fmt.Errorf("parse ws url: %w", err)
	}
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{}
	conn, resp, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("websocket dial: status %d: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("websocket dial: %w", err)
	}

	c.conn = conn
	c.connected = true
	c.stopCh = make(chan struct{})
	c.doneCh = make(chan struct{})

	go c.run()

	return nil
}

func (c *WSClient) Disconnect() {
	c.connMu.Lock()
	if !c.connected {
		c.connMu.Unlock()
		return
	}
	stopCh := c.stopCh
	doneCh := c.doneCh
	c.connMu.Unlock()

	close(stopCh)
	<-doneCh
}

func (c *WSClient) IsConnected() bool {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.connected
}

// Send sends a message to the server via the WebSocket connection.
func (c *WSClient) Send(msg interface{}) error {
	c.connMu.Lock()
	conn := c.conn
	connected := c.connected
	c.connMu.Unlock()

	if !connected || conn == nil {
		return fmt.Errorf("not connected")
	}
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteJSON(msg)
}

func (c *WSClient) run() {
	c.connMu.Lock()
	doneCh := c.doneCh
	c.connMu.Unlock()

	defer func() {
		c.handleDisconnect()
		close(doneCh)
	}()

	readDone := make(chan struct{})

	go func() {
		c.readPump()
		close(readDone)
	}()

	c.writePump(readDone)

	<-readDone
}

func (c *WSClient) readPump() {
	c.connMu.Lock()
	conn := c.conn
	stopCh := c.stopCh
	c.connMu.Unlock()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-stopCh:
			default:
				c.logger.WithError(err).Debug("websocket read error")
			}
			return
		}

		var event WSEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			c.logger.WithError(err).Warn("failed to parse ws event")
			continue
		}

		if c.handler != nil {
			c.handler(event)
		}
	}
}

func (c *WSClient) writePump(readDone <-chan struct{}) {
	c.connMu.Lock()
	conn := c.conn
	stopCh := c.stopCh
	c.connMu.Unlock()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		case <-readDone:
			return
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.WithError(err).Debug("websocket ping error")
				return
			}
		}
	}
}

func (c *WSClient) handleDisconnect() {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.connected = false
}

func (c *WSClient) RunWithReconnect(ctx context.Context) error {
	for {
		err := c.Connect(ctx)
		if err != nil {
			c.logger.WithError(err).Warn("websocket connect failed, will retry")
		} else {
			c.reconnects = 0
			c.logger.Debug("websocket connected")

			c.connMu.Lock()
			doneCh := c.doneCh
			c.connMu.Unlock()

			select {
			case <-ctx.Done():
				c.Disconnect()
				return ctx.Err()
			case <-doneCh:
				c.logger.Debug("websocket disconnected, will reconnect")
			}
		}

		delay := c.reconnectDelay()
		c.reconnects++

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

func (c *WSClient) reconnectDelay() time.Duration {
	delay := reconnectMin
	for i := 0; i < c.reconnects; i++ {
		delay *= 2
		if delay > reconnectMax {
			return reconnectMax
		}
	}
	return delay
}
