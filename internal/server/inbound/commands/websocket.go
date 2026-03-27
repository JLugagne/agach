package commands

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"

	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/pkg/websocket"
	domain "github.com/JLugagne/agach-mcp/internal/server/domain"
	gorillaws "github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type wsAuthQueries interface {
	ValidateJWT(ctx context.Context, token string) (identitydomain.Actor, error)
	ValidateDaemonJWT(ctx context.Context, token string) (identitydomain.DaemonActor, error)
	GetUserTeamIDs(ctx context.Context, userID identitydomain.UserID) ([]identitydomain.TeamID, error)
}

// WSProjectAccessChecker verifies a user has access to a project.
type WSProjectAccessChecker interface {
	HasProjectAccess(ctx context.Context, projectID domain.ProjectID, userID string, teamIDs []string) (bool, error)
}

// WSHandler handles WebSocket upgrade with token-based authentication.
type WSHandler struct {
	authQueries          wsAuthQueries
	accessChecker        WSProjectAccessChecker
	hub                  *websocket.Hub
	upgrader             gorillaws.Upgrader
	logger               *logrus.Logger
	resourceManifestJSON json.RawMessage // pre-marshaled manifest sent to daemons on connect
}

// NewWSHandler creates a WSHandler. authQueries may be nil to skip authentication.
// manifestData is the pre-marshaled JSON of the resource manifest entries.
func NewWSHandler(authQueries wsAuthQueries, hub *websocket.Hub, logger *logrus.Logger, manifestData json.RawMessage, accessChecker ...WSProjectAccessChecker) *WSHandler {
	var ac WSProjectAccessChecker
	if len(accessChecker) > 0 {
		ac = accessChecker[0]
	}
	return &WSHandler{
		authQueries:          authQueries,
		accessChecker:        ac,
		hub:                  hub,
		resourceManifestJSON: manifestData,
		upgrader: gorillaws.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				u, err := url.Parse(origin)
				if err != nil {
					return false
				}
				return u.Host == r.Host
			},
		},
		logger: logger,
	}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Per-IP connection limit — reject before upgrade to avoid resource waste.
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if !h.hub.CanAcceptIP(ip) {
			http.Error(w, `{"status":"fail","error":{"code":"TOO_MANY_CONNECTIONS","message":"too many connections from this IP"}}`, http.StatusTooManyRequests)
			return
		}
	}

	isDaemon := false
	var nodeID string
	var actor identitydomain.Actor
	if h.authQueries != nil {
		ticket := r.URL.Query().Get("ticket")
		if ticket == "" {
			http.Error(w, `{"status":"fail","error":{"code":"UNAUTHORIZED","message":"authentication required"}}`, http.StatusUnauthorized)
			return
		}
		validActor, err := h.authQueries.ValidateJWT(r.Context(), ticket)
		if err != nil {
			daemonActor, daemonErr := h.authQueries.ValidateDaemonJWT(r.Context(), ticket)
			if daemonErr != nil {
				http.Error(w, `{"status":"fail","error":{"code":"UNAUTHORIZED","message":"authentication required"}}`, http.StatusUnauthorized)
				return
			}
			isDaemon = true
			nodeID = daemonActor.NodeID.String()
		} else {
			actor = validActor
		}
	}

	// For browser clients, verify project access if a project_id is specified.
	var projectID string
	if !isDaemon {
		projectID = r.URL.Query().Get("project_id")
		if projectID != "" && h.accessChecker != nil && !actor.IsZero() {
			// Resolve team IDs for the user.
			var teamIDs []string
			if h.authQueries != nil {
				tids, _ := h.authQueries.GetUserTeamIDs(r.Context(), actor.UserID)
				for _, tid := range tids {
					teamIDs = append(teamIDs, tid.String())
				}
			}
			pid, parseErr := domain.ParseProjectID(projectID)
			if parseErr != nil {
				http.Error(w, `{"status":"fail","error":{"code":"INVALID_PROJECT_ID","message":"invalid project ID"}}`, http.StatusBadRequest)
				return
			}
			ok, accessErr := h.accessChecker.HasProjectAccess(r.Context(), pid, actor.UserID.String(), teamIDs)
			if accessErr != nil || !ok {
				http.Error(w, `{"status":"fail","error":{"code":"FORBIDDEN","message":"no access to this project"}}`, http.StatusForbidden)
				return
			}
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
		if projectID != "" {
			opts = append(opts, websocket.WithProjectID(projectID))
		}
		h.logger.WithField("project_id", projectID).Info("WebSocket: browser client connected")
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
