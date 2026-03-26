package service

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type NodeCommands interface {
	RevokeNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) error
	UpdateNodeAccess(ctx context.Context, actor domain.Actor, nodeID domain.NodeID, grantUserIDs []domain.UserID, grantTeamIDs []domain.TeamID, revokeUserIDs []domain.UserID, revokeTeamIDs []domain.TeamID) error
	RenameNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID, name string) error
	AdminRevokeNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) error
}

type NodeQueries interface {
	ListNodes(ctx context.Context, actor domain.Actor) ([]domain.Node, error)
	GetNode(ctx context.Context, actor domain.Actor, nodeID domain.NodeID) (domain.Node, error)
	ListAllNodes(ctx context.Context, actor domain.Actor) ([]domain.Node, error)
}
