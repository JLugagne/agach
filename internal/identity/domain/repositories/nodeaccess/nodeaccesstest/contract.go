package nodeaccesstest

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type MockNodeAccessRepository struct {
	GrantUserFunc  func(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) error
	GrantTeamFunc  func(ctx context.Context, nodeID domain.NodeID, teamID domain.TeamID) error
	RevokeUserFunc func(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) error
	RevokeTeamFunc func(ctx context.Context, nodeID domain.NodeID, teamID domain.TeamID) error
	ListByNodeFunc func(ctx context.Context, nodeID domain.NodeID) ([]domain.NodeAccess, error)
	HasAccessFunc  func(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) (bool, error)
}

func (m *MockNodeAccessRepository) GrantUser(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) error {
	if m.GrantUserFunc == nil {
		panic("called not defined GrantUserFunc")
	}
	return m.GrantUserFunc(ctx, nodeID, userID)
}

func (m *MockNodeAccessRepository) GrantTeam(ctx context.Context, nodeID domain.NodeID, teamID domain.TeamID) error {
	if m.GrantTeamFunc == nil {
		panic("called not defined GrantTeamFunc")
	}
	return m.GrantTeamFunc(ctx, nodeID, teamID)
}

func (m *MockNodeAccessRepository) RevokeUser(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) error {
	if m.RevokeUserFunc == nil {
		panic("called not defined RevokeUserFunc")
	}
	return m.RevokeUserFunc(ctx, nodeID, userID)
}

func (m *MockNodeAccessRepository) RevokeTeam(ctx context.Context, nodeID domain.NodeID, teamID domain.TeamID) error {
	if m.RevokeTeamFunc == nil {
		panic("called not defined RevokeTeamFunc")
	}
	return m.RevokeTeamFunc(ctx, nodeID, teamID)
}

func (m *MockNodeAccessRepository) ListByNode(ctx context.Context, nodeID domain.NodeID) ([]domain.NodeAccess, error) {
	if m.ListByNodeFunc == nil {
		panic("called not defined ListByNodeFunc")
	}
	return m.ListByNodeFunc(ctx, nodeID)
}

func (m *MockNodeAccessRepository) HasAccess(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) (bool, error) {
	if m.HasAccessFunc == nil {
		panic("called not defined HasAccessFunc")
	}
	return m.HasAccessFunc(ctx, nodeID, userID)
}
