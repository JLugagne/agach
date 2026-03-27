package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/inbound/commands"
	"github.com/JLugagne/agach-mcp/internal/pkg/controller"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mock node services
// ─────────────────────────────────────────────────────────────────────────────

type mockNodeCommands struct {
	revokeNodeFunc       func(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) error
	updateNodeAccessFunc func(ctx context.Context, actor domain.Actor, nodeID domain.NodeID, grantUserIDs []domain.UserID, grantTeamIDs []domain.TeamID, revokeUserIDs []domain.UserID, revokeTeamIDs []domain.TeamID) error
	renameNodeFunc       func(ctx context.Context, actor domain.Actor, nodeID domain.NodeID, name string) error
	adminRevokeNodeFunc  func(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) error
}

func (m *mockNodeCommands) RevokeNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) error {
	return m.revokeNodeFunc(ctx, actor, nodeID)
}

func (m *mockNodeCommands) UpdateNodeAccess(ctx context.Context, actor domain.Actor, nodeID domain.NodeID, grantUserIDs []domain.UserID, grantTeamIDs []domain.TeamID, revokeUserIDs []domain.UserID, revokeTeamIDs []domain.TeamID) error {
	return m.updateNodeAccessFunc(ctx, actor, nodeID, grantUserIDs, grantTeamIDs, revokeUserIDs, revokeTeamIDs)
}

func (m *mockNodeCommands) RenameNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID, name string) error {
	return m.renameNodeFunc(ctx, actor, nodeID, name)
}

func (m *mockNodeCommands) AdminRevokeNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) error {
	if m.adminRevokeNodeFunc == nil {
		return nil
	}
	return m.adminRevokeNodeFunc(ctx, actor, nodeID)
}

type mockNodeQueries struct {
	listNodesFunc    func(ctx context.Context, actor domain.Actor) ([]domain.Node, error)
	getNodeFunc      func(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) (domain.Node, error)
	listAllNodesFunc func(ctx context.Context, actor domain.Actor) ([]domain.Node, error)
}

func (m *mockNodeQueries) ListNodes(ctx context.Context, actor domain.Actor) ([]domain.Node, error) {
	return m.listNodesFunc(ctx, actor)
}

func (m *mockNodeQueries) GetNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) (domain.Node, error) {
	return m.getNodeFunc(ctx, actor, nodeID)
}

func (m *mockNodeQueries) ListAllNodes(ctx context.Context, actor domain.Actor) ([]domain.Node, error) {
	if m.listAllNodesFunc == nil {
		return nil, nil
	}
	return m.listAllNodesFunc(ctx, actor)
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func newNodesTestHandler(cmds *mockNodeCommands, qrs *mockNodeQueries, authQrs *mockAuthQueries) *mux.Router {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	ctrl := controller.NewController(logger)
	h := commands.NewNodesHandler(cmds, qrs, authQrs, ctrl)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return r
}

func memberAuthQueriesForNodes() *mockAuthQueries {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "member@example.com", Role: domain.RoleMember}
	return &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}
}

func unauthorizedAuthQueriesForNodes() *mockAuthQueries {
	return &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return domain.Actor{}, domain.ErrUnauthorized
		},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ListNodes
// ─────────────────────────────────────────────────────────────────────────────

func TestNodesHandler_ListNodes_Success(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}
	now := time.Now()
	nodes := []domain.Node{
		{
			ID:          domain.NewNodeID(),
			OwnerUserID: actor.UserID,
			Name:        "my-node",
			Mode:        domain.NodeModeDefault,
			Status:      domain.NodeStatusActive,
			CreatedAt:   now,
		},
	}

	authQrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}

	qrs := &mockNodeQueries{
		listNodesFunc: func(_ context.Context, _ domain.Actor) ([]domain.Node, error) {
			return nodes, nil
		},
	}

	router := newNodesTestHandler(&mockNodeCommands{}, qrs, authQrs)

	req := httptest.NewRequest("GET", "/api/nodes", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])
	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok)
	nodesArr, ok := data["nodes"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 1, len(nodesArr))
}

func TestNodesHandler_ListNodes_Unauthenticated(t *testing.T) {
	router := newNodesTestHandler(&mockNodeCommands{}, &mockNodeQueries{}, unauthorizedAuthQueriesForNodes())

	req := httptest.NewRequest("GET", "/api/nodes", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// GetNode
// ─────────────────────────────────────────────────────────────────────────────

func TestNodesHandler_GetNode_Success(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}
	nodeID := domain.NewNodeID()
	now := time.Now()
	node := domain.Node{
		ID:          nodeID,
		OwnerUserID: actor.UserID,
		Name:        "my-node",
		Mode:        domain.NodeModeDefault,
		Status:      domain.NodeStatusActive,
		CreatedAt:   now,
	}

	authQrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}

	qrs := &mockNodeQueries{
		getNodeFunc: func(_ context.Context, _ domain.Actor, _ domain.NodeID) (domain.Node, error) {
			return node, nil
		},
	}

	router := newNodesTestHandler(&mockNodeCommands{}, qrs, authQrs)

	req := httptest.NewRequest("GET", "/api/nodes/"+nodeID.String(), nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])
}

func TestNodesHandler_GetNode_NotFound(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}
	nodeID := domain.NewNodeID()

	authQrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}

	qrs := &mockNodeQueries{
		getNodeFunc: func(_ context.Context, _ domain.Actor, _ domain.NodeID) (domain.Node, error) {
			return domain.Node{}, domain.ErrNodeNotFound
		},
	}

	router := newNodesTestHandler(&mockNodeCommands{}, qrs, authQrs)

	req := httptest.NewRequest("GET", "/api/nodes/"+nodeID.String(), nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "fail", resp["status"])
}

// ─────────────────────────────────────────────────────────────────────────────
// RevokeNode
// ─────────────────────────────────────────────────────────────────────────────

func TestNodesHandler_RevokeNode_Success(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}
	nodeID := domain.NewNodeID()

	authQrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}

	cmds := &mockNodeCommands{
		revokeNodeFunc: func(_ context.Context, _ domain.Actor, _ domain.NodeID) error {
			return nil
		},
	}

	router := newNodesTestHandler(cmds, &mockNodeQueries{}, authQrs)

	req := httptest.NewRequest("DELETE", "/api/nodes/"+nodeID.String(), nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestNodesHandler_RevokeNode_NotFound(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}
	nodeID := domain.NewNodeID()

	authQrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}

	cmds := &mockNodeCommands{
		revokeNodeFunc: func(_ context.Context, _ domain.Actor, _ domain.NodeID) error {
			return domain.ErrNodeNotFound
		},
	}

	router := newNodesTestHandler(cmds, &mockNodeQueries{}, authQrs)

	req := httptest.NewRequest("DELETE", "/api/nodes/"+nodeID.String(), nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// RenameNode
// ─────────────────────────────────────────────────────────────────────────────

func TestNodesHandler_RenameNode_Success(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}
	nodeID := domain.NewNodeID()
	now := time.Now()
	updatedNode := domain.Node{
		ID:          nodeID,
		OwnerUserID: actor.UserID,
		Name:        "renamed-node",
		Mode:        domain.NodeModeDefault,
		Status:      domain.NodeStatusActive,
		CreatedAt:   now,
	}

	authQrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}

	cmds := &mockNodeCommands{
		renameNodeFunc: func(_ context.Context, _ domain.Actor, _ domain.NodeID, _ string) error {
			return nil
		},
	}

	qrs := &mockNodeQueries{
		getNodeFunc: func(_ context.Context, _ domain.Actor, _ domain.NodeID) (domain.Node, error) {
			return updatedNode, nil
		},
	}

	router := newNodesTestHandler(cmds, qrs, authQrs)

	body, _ := json.Marshal(map[string]string{"name": "renamed-node"})
	req := httptest.NewRequest("PATCH", "/api/nodes/"+nodeID.String()+"/name", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "success", resp["status"])
	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok)
	node, ok := data["node"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "renamed-node", node["name"])
}

// ─────────────────────────────────────────────────────────────────────────────
// UpdateAccess
// ─────────────────────────────────────────────────────────────────────────────

func TestNodesHandler_UpdateAccess_SharedMode(t *testing.T) {
	actor := domain.Actor{UserID: domain.NewUserID(), Email: "user@example.com", Role: domain.RoleMember}
	nodeID := domain.NewNodeID()
	userID := domain.NewUserID()
	teamID := domain.NewTeamID()

	authQrs := &mockAuthQueries{
		validateJWTFunc: func(_ context.Context, _ string) (domain.Actor, error) {
			return actor, nil
		},
	}

	cmds := &mockNodeCommands{
		updateNodeAccessFunc: func(_ context.Context, _ domain.Actor, _ domain.NodeID, grantUserIDs []domain.UserID, grantTeamIDs []domain.TeamID, revokeUserIDs []domain.UserID, revokeTeamIDs []domain.TeamID) error {
			assert.Equal(t, 1, len(grantUserIDs))
			assert.Equal(t, 1, len(grantTeamIDs))
			assert.Equal(t, 1, len(revokeUserIDs))
			assert.Equal(t, 1, len(revokeTeamIDs))
			return nil
		},
	}

	router := newNodesTestHandler(cmds, &mockNodeQueries{}, authQrs)

	body, _ := json.Marshal(map[string][]string{
		"grant_user_ids":  {userID.String()},
		"grant_team_ids":  {teamID.String()},
		"revoke_user_ids": {domain.NewUserID().String()},
		"revoke_team_ids": {domain.NewTeamID().String()},
	})
	req := httptest.NewRequest("PUT", "/api/nodes/"+nodeID.String()+"/access", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}
