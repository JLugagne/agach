package websocket

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// closeable is the minimal interface a client must satisfy to allow the hub to close its send channel.
type closeable interface {
	DoCloseSend()
}

// HandleUnregister removes a client from the map and closes its send channel.
// Must be called with mu not held.
func HandleUnregister[C interface {
	comparable
	closeable
}](clients map[C]bool, mu *sync.Mutex, client C, logger *logrus.Logger) {
	mu.Lock()
	if _, ok := clients[client]; ok {
		delete(clients, client)
		client.DoCloseSend()
	}
	mu.Unlock()
	logger.WithField("client_count", len(clients)).Debug("Client unregistered")
}

// RunWritePump is the shared write-side pump used by all hub implementations.
// It reads messages from send, serialises them with writeMsg, and sends periodic pings.
// Returns when the send channel is closed or a write error occurs.
func RunWritePump[M any](
	conn *websocket.Conn,
	send <-chan M,
	writeMsg func(*websocket.Conn, M) error,
	logger *logrus.Logger,
) {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case msg, ok := <-send:
			conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := writeMsg(conn, msg); err != nil {
				if logger != nil {
					logger.WithError(err).Warn("WritePump: write failed")
				}
				return
			}

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
