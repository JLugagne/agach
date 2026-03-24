package app

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/nodeaccess/nodeaccesstest"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/nodes/nodestest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeActor() domain.Actor {
	return domain.Actor{
		UserID: domain.NewUserID(),
		Email:  "user@example.com",
		Role:   domain.RoleMember,
	}
}

func makeNode(ownerID domain.UserID) domain.Node {
	return domain.Node{
		ID:          domain.NewNodeID(),
		OwnerUserID: ownerID,
		Name:        "my-daemon",
		Mode:        domain.NodeModeDefault,
		Status:      domain.NodeStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestNodeService_ListNodes_ReturnsOwnedNodes(t *testing.T) {
	actor := makeActor()
	node := makeNode(actor.UserID)
	nodeRepo := &nodestest.MockNodeRepository{
		ListByOwnerFunc: func(_ context.Context, ownerID domain.UserID) ([]domain.Node, error) {
			assert.Equal(t, actor.UserID, ownerID)
			return []domain.Node{node}, nil
		},
	}
	svc := NewNodeQueriesService(nodeRepo, &nodeaccesstest.MockNodeAccessRepository{})
	result, err := svc.ListNodes(context.Background(), actor)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, node.ID, result[0].ID)
}

func TestNodeService_ListNodes_EmptyWhenNoNodes(t *testing.T) {
	actor := makeActor()
	nodeRepo := &nodestest.MockNodeRepository{
		ListByOwnerFunc: func(_ context.Context, _ domain.UserID) ([]domain.Node, error) {
			return []domain.Node{}, nil
		},
	}
	svc := NewNodeQueriesService(nodeRepo, &nodeaccesstest.MockNodeAccessRepository{})
	result, err := svc.ListNodes(context.Background(), actor)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestNodeService_GetNode_Success(t *testing.T) {
	actor := makeActor()
	node := makeNode(actor.UserID)
	nodeRepo := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, id domain.NodeID) (domain.Node, error) {
			assert.Equal(t, node.ID, id)
			return node, nil
		},
	}
	svc := NewNodeQueriesService(nodeRepo, &nodeaccesstest.MockNodeAccessRepository{})
	result, err := svc.GetNode(context.Background(), actor, node.ID)
	require.NoError(t, err)
	assert.Equal(t, node.ID, result.ID)
}

func TestNodeService_GetNode_NotOwner_ReturnsUnauthorized(t *testing.T) {
	actor := makeActor()
	otherOwner := domain.NewUserID()
	node := makeNode(otherOwner)
	nodeRepo := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, _ domain.NodeID) (domain.Node, error) {
			return node, nil
		},
	}
	svc := NewNodeQueriesService(nodeRepo, &nodeaccesstest.MockNodeAccessRepository{})
	_, err := svc.GetNode(context.Background(), actor, node.ID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestNodeService_RevokeNode_Success(t *testing.T) {
	actor := makeActor()
	node := makeNode(actor.UserID)
	var updatedNode domain.Node
	nodeRepo := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, _ domain.NodeID) (domain.Node, error) {
			return node, nil
		},
		UpdateFunc: func(_ context.Context, n domain.Node) error {
			updatedNode = n
			return nil
		},
	}
	svc := NewNodeService(nodeRepo, &nodeaccesstest.MockNodeAccessRepository{})
	err := svc.RevokeNode(context.Background(), actor, node.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.NodeStatusRevoked, updatedNode.Status)
	assert.NotNil(t, updatedNode.RevokedAt)
	assert.Empty(t, updatedNode.RefreshTokenHash)
}

func TestNodeService_RevokeNode_AlreadyRevoked(t *testing.T) {
	actor := makeActor()
	node := makeNode(actor.UserID)
	node.Status = domain.NodeStatusRevoked
	now := time.Now()
	node.RevokedAt = &now
	nodeRepo := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, _ domain.NodeID) (domain.Node, error) {
			return node, nil
		},
	}
	svc := NewNodeService(nodeRepo, &nodeaccesstest.MockNodeAccessRepository{})
	err := svc.RevokeNode(context.Background(), actor, node.ID)
	assert.ErrorIs(t, err, domain.ErrNodeRevoked)
}

func TestNodeService_RevokeNode_NotOwner(t *testing.T) {
	actor := makeActor()
	otherOwner := domain.NewUserID()
	node := makeNode(otherOwner)
	nodeRepo := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, _ domain.NodeID) (domain.Node, error) {
			return node, nil
		},
	}
	svc := NewNodeService(nodeRepo, &nodeaccesstest.MockNodeAccessRepository{})
	err := svc.RevokeNode(context.Background(), actor, node.ID)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestNodeService_UpdateNodeAccess_SharedMode(t *testing.T) {
	actor := makeActor()
	node := makeNode(actor.UserID)
	node.Mode = domain.NodeModeShared
	grantUserID := domain.NewUserID()
	grantTeamID := domain.NewTeamID()
	revokeUserID := domain.NewUserID()
	revokeTeamID := domain.NewTeamID()
	var grantedUsers, revokedUsers []domain.UserID
	var grantedTeams, revokedTeams []domain.TeamID
	nodeRepo := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, _ domain.NodeID) (domain.Node, error) {
			return node, nil
		},
	}
	accessRepo := &nodeaccesstest.MockNodeAccessRepository{
		GrantUserFunc: func(_ context.Context, _ domain.NodeID, uid domain.UserID) error {
			grantedUsers = append(grantedUsers, uid)
			return nil
		},
		GrantTeamFunc: func(_ context.Context, _ domain.NodeID, tid domain.TeamID) error {
			grantedTeams = append(grantedTeams, tid)
			return nil
		},
		RevokeUserFunc: func(_ context.Context, _ domain.NodeID, uid domain.UserID) error {
			revokedUsers = append(revokedUsers, uid)
			return nil
		},
		RevokeTeamFunc: func(_ context.Context, _ domain.NodeID, tid domain.TeamID) error {
			revokedTeams = append(revokedTeams, tid)
			return nil
		},
	}
	svc := NewNodeService(nodeRepo, accessRepo)
	err := svc.UpdateNodeAccess(context.Background(), actor, node.ID,
		[]domain.UserID{grantUserID},
		[]domain.TeamID{grantTeamID},
		[]domain.UserID{revokeUserID},
		[]domain.TeamID{revokeTeamID},
	)
	require.NoError(t, err)
	assert.Equal(t, []domain.UserID{grantUserID}, grantedUsers)
	assert.Equal(t, []domain.TeamID{grantTeamID}, grantedTeams)
	assert.Equal(t, []domain.UserID{revokeUserID}, revokedUsers)
	assert.Equal(t, []domain.TeamID{revokeTeamID}, revokedTeams)
}

func TestNodeService_UpdateNodeAccess_DefaultMode_Fails(t *testing.T) {
	actor := makeActor()
	node := makeNode(actor.UserID)
	nodeRepo := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, _ domain.NodeID) (domain.Node, error) {
			return node, nil
		},
	}
	svc := NewNodeService(nodeRepo, &nodeaccesstest.MockNodeAccessRepository{})
	err := svc.UpdateNodeAccess(context.Background(), actor, node.ID, nil, nil, nil, nil)
	require.Error(t, err)
	var domErr *domain.Error
	assert.ErrorAs(t, err, &domErr)
	assert.Equal(t, "NODE_NOT_SHARED", domErr.Code)
}

func TestNodeService_RenameNode_Success(t *testing.T) {
	actor := makeActor()
	node := makeNode(actor.UserID)
	var updatedNode domain.Node
	nodeRepo := &nodestest.MockNodeRepository{
		FindByIDFunc: func(_ context.Context, _ domain.NodeID) (domain.Node, error) {
			return node, nil
		},
		UpdateFunc: func(_ context.Context, n domain.Node) error {
			updatedNode = n
			return nil
		},
	}
	svc := NewNodeService(nodeRepo, &nodeaccesstest.MockNodeAccessRepository{})
	err := svc.RenameNode(context.Background(), actor, node.ID, "new-name")
	require.NoError(t, err)
	assert.Equal(t, "new-name", updatedNode.Name)
}
