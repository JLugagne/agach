package commands

import (
	"context"
	"encoding/json"
	"net/http"

	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	gorillaws "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type wsAuthQueries interface {
	ValidateJWT(ctx context.Context, token string) (identitydomain.Actor, error)
	ValidateDaemonJWT(ctx context.Context, token string) (identitydomain.DaemonActor, error)
}

// WSHandler handles WebSocket upgrade with token-based authentication.
type WSHandler struct {
	authQueries          wsAuthQueries
	hub                  *websocket.Hub
	upgrader             gorillaws.Upgrader
	logger               *logrus.Logger
	resourceManifestJSON json.RawMessage // pre-marshaled manifest sent to daemons on connect
}

// NewWSHandler creates a WSHandler. authQueries may be nil to skip authentication.
// manifestData is the pre-marshaled JSON of the resource manifest entries.
func NewWSHandler(authQueries wsAuthQueries, hub *websocket.Hub, logger *logrus.Logger, manifestData json.RawMessage) *WSHandler {
	return &WSHandler{
		authQueries:          authQueries,
		hub:                  hub,
		resourceManifestJSON: manifestData,
		upgrader: gorillaws.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		logger: logger,
	}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isDaemon := false
	var nodeID string
	if h.authQueries != nil {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, `{"status":"fail","error":{"code":"UNAUTHORIZED","message":"authentication required"}}`, http.StatusUnauthorized)
			return
		}
		if _, err := h.authQueries.ValidateJWT(r.Context(), token); err != nil {
			daemonActor, daemonErr := h.authQueries.ValidateDaemonJWT(r.Context(), token)
			if daemonErr != nil {
				http.Error(w, `{"status":"fail","error":{"code":"UNAUTHORIZED","message":"authentication required"}}`, http.StatusUnauthorized)
				return
			}
			isDaemon = true
			nodeID = daemonActor.NodeID.String()
		}
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade WebSocket")
		return
	}

	var opts []websocket.ServeWSOption
	if isDaemon {
		h.logger.WithField("node_id", nodeID).Info("WebSocket: daemon connected")
		opts = append(opts, websocket.AsDaemon(), websocket.WithNodeID(nodeID))
	} else {
		h.logger.Info("WebSocket: browser client connected")
	}
	h.hub.ServeWS(conn, opts...)

	// Send resource manifest to daemon after connection is established
	if isDaemon && len(h.resourceManifestJSON) > 0 {
		event, _ := json.Marshal(map[string]any{
			"type": "resource_manifest",
			"data": json.RawMessage(h.resourceManifestJSON),
		})
		h.hub.SendToDaemon(nodeID, event)
	}
}
