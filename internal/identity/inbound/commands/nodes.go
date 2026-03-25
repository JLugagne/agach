package commands

import (
	"errors"
	"net/http"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
	"github.com/JLugagne/agach-mcp/internal/pkg/apierror"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/gorilla/mux"
)

// NodesHandler handles node management HTTP endpoints.
type NodesHandler struct {
	nodeCommands service.NodeCommands
	nodeQueries  service.NodeQueries
	authQueries  service.AuthQueries
	controller   *controller.Controller
}

// NewNodesHandler creates a new nodes handler.
func NewNodesHandler(
	nodeCommands service.NodeCommands,
	nodeQueries service.NodeQueries,
	authQueries service.AuthQueries,
	ctrl *controller.Controller,
) *NodesHandler {
	return &NodesHandler{
		nodeCommands: nodeCommands,
		nodeQueries:  nodeQueries,
		authQueries:  authQueries,
		controller:   ctrl,
	}
}

// RegisterRoutes registers node routes on the router.
func (h *NodesHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/nodes", h.ListNodes).Methods("GET")
	router.HandleFunc("/api/nodes/{id}", h.GetNode).Methods("GET")
	router.HandleFunc("/api/nodes/{id}", h.RevokeNode).Methods("DELETE")
	router.HandleFunc("/api/nodes/{id}/name", h.RenameNode).Methods("PATCH")
	router.HandleFunc("/api/nodes/{id}/access", h.UpdateAccess).Methods("PUT")
}

// Response types

type listNodesResponse struct {
	Nodes []nodeDetailResponse `json:"nodes"`
}

type nodeDetailResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Mode       string     `json:"mode"`
	Status     string     `json:"status"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type renameNodeRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

type updateAccessRequest struct {
	GrantUserIDs  []string `json:"grant_user_ids"`
	GrantTeamIDs  []string `json:"grant_team_ids"`
	RevokeUserIDs []string `json:"revoke_user_ids"`
	RevokeTeamIDs []string `json:"revoke_team_ids"`
}

// ListNodes handles GET /api/nodes.
func (h *NodesHandler) ListNodes(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromRequest(w, r, h.controller, h.authQueries)
	if !ok {
		return
	}

	nodes, err := h.nodeQueries.ListNodes(r.Context(), actor)
	if err != nil {
		h.controller.SendError(w, r, err)
		return
	}

	resp := listNodesResponse{Nodes: make([]nodeDetailResponse, len(nodes))}
	for i, n := range nodes {
		resp.Nodes[i] = toNodeDetailResponse(n)
	}

	h.controller.SendSuccess(w, r, resp)
}

// GetNode handles GET /api/nodes/{id}.
func (h *NodesHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromRequest(w, r, h.controller, h.authQueries)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	nodeID, err := domain.ParseNodeID(vars["id"])
	if err != nil {
		status := http.StatusBadRequest
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_NODE_ID", Message: "invalid node ID format"})
		return
	}

	node, err := h.nodeQueries.GetNode(r.Context(), actor, nodeID)
	if err != nil {
		h.handleNodeError(w, r, err)
		return
	}

	h.controller.SendSuccess(w, r, map[string]any{"node": toNodeDetailResponse(node)})
}

// RevokeNode handles DELETE /api/nodes/{id}.
func (h *NodesHandler) RevokeNode(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromRequest(w, r, h.controller, h.authQueries)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	nodeID, err := domain.ParseNodeID(vars["id"])
	if err != nil {
		status := http.StatusBadRequest
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_NODE_ID", Message: "invalid node ID format"})
		return
	}

	if err := h.nodeCommands.RevokeNode(r.Context(), actor, nodeID); err != nil {
		h.handleNodeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RenameNode handles PATCH /api/nodes/{id}/name.
func (h *NodesHandler) RenameNode(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromRequest(w, r, h.controller, h.authQueries)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	nodeID, err := domain.ParseNodeID(vars["id"])
	if err != nil {
		status := http.StatusBadRequest
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_NODE_ID", Message: "invalid node ID format"})
		return
	}

	var req renameNodeRequest
	if err := h.controller.DecodeAndValidate(r, &req, nil); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	if err := h.nodeCommands.RenameNode(r.Context(), actor, nodeID, req.Name); err != nil {
		h.handleNodeError(w, r, err)
		return
	}

	// Fetch updated node
	node, _ := h.nodeQueries.GetNode(r.Context(), actor, nodeID)
	h.controller.SendSuccess(w, r, map[string]any{"node": toNodeDetailResponse(node)})
}

// UpdateAccess handles PUT /api/nodes/{id}/access.
func (h *NodesHandler) UpdateAccess(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromRequest(w, r, h.controller, h.authQueries)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	nodeID, err := domain.ParseNodeID(vars["id"])
	if err != nil {
		status := http.StatusBadRequest
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "INVALID_NODE_ID", Message: "invalid node ID format"})
		return
	}

	var req updateAccessRequest
	if err := h.controller.DecodeAndValidate(r, &req, nil); err != nil {
		h.controller.SendFail(w, r, nil, err)
		return
	}

	// Parse IDs (with validation)
	grantUserIDs, grantTeamIDs, revokeUserIDs, revokeTeamIDs, parseErr := h.parseAccessIDs(req)
	if parseErr != nil {
		status := http.StatusBadRequest
		h.controller.SendFail(w, r, &status, parseErr)
		return
	}

	if err := h.nodeCommands.UpdateNodeAccess(r.Context(), actor, nodeID, grantUserIDs, grantTeamIDs, revokeUserIDs, revokeTeamIDs); err != nil {
		h.handleNodeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func toNodeDetailResponse(n domain.Node) nodeDetailResponse {
	return nodeDetailResponse{
		ID:         n.ID.String(),
		Name:       n.Name,
		Mode:       string(n.Mode),
		Status:     string(n.Status),
		LastSeenAt: n.LastSeenAt,
		RevokedAt:  n.RevokedAt,
		CreatedAt:  n.CreatedAt,
	}
}

func (h *NodesHandler) parseAccessIDs(req updateAccessRequest) ([]domain.UserID, []domain.TeamID, []domain.UserID, []domain.TeamID, error) {
	grantUserIDs := make([]domain.UserID, len(req.GrantUserIDs))
	for i, s := range req.GrantUserIDs {
		id, err := domain.ParseUserID(s)
		if err != nil {
			return nil, nil, nil, nil, &apierror.Error{Code: "INVALID_USER_ID", Message: "invalid user ID format"}
		}
		grantUserIDs[i] = id
	}

	grantTeamIDs := make([]domain.TeamID, len(req.GrantTeamIDs))
	for i, s := range req.GrantTeamIDs {
		id, err := domain.ParseTeamID(s)
		if err != nil {
			return nil, nil, nil, nil, &apierror.Error{Code: "INVALID_TEAM_ID", Message: "invalid team ID format"}
		}
		grantTeamIDs[i] = id
	}

	revokeUserIDs := make([]domain.UserID, len(req.RevokeUserIDs))
	for i, s := range req.RevokeUserIDs {
		id, err := domain.ParseUserID(s)
		if err != nil {
			return nil, nil, nil, nil, &apierror.Error{Code: "INVALID_USER_ID", Message: "invalid user ID format"}
		}
		revokeUserIDs[i] = id
	}

	revokeTeamIDs := make([]domain.TeamID, len(req.RevokeTeamIDs))
	for i, s := range req.RevokeTeamIDs {
		id, err := domain.ParseTeamID(s)
		if err != nil {
			return nil, nil, nil, nil, &apierror.Error{Code: "INVALID_TEAM_ID", Message: "invalid team ID format"}
		}
		revokeTeamIDs[i] = id
	}

	return grantUserIDs, grantTeamIDs, revokeUserIDs, revokeTeamIDs, nil
}

func (h *NodesHandler) handleNodeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNodeNotFound):
		status := http.StatusNotFound
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "NODE_NOT_FOUND", Message: "node not found"})
	case errors.Is(err, domain.ErrNodeRevoked):
		status := http.StatusGone
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "NODE_REVOKED", Message: "node has been revoked"})
	case errors.Is(err, domain.ErrUnauthorized):
		status := http.StatusForbidden
		h.controller.SendFail(w, r, &status, &apierror.Error{Code: "FORBIDDEN", Message: "you do not own this node"})
	default:
		h.controller.SendError(w, r, err)
	}
}
