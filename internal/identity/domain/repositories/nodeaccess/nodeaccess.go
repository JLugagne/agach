package nodeaccess

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type NodeAccessRepository interface {
	GrantUser(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) error
	GrantTeam(ctx context.Context, nodeID domain.NodeID, teamID domain.TeamID) error
	RevokeUser(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) error
	RevokeTeam(ctx context.Context, nodeID domain.NodeID, teamID domain.TeamID) error
	ListByNode(ctx context.Context, nodeID domain.NodeID) ([]domain.NodeAccess, error)
	HasAccess(ctx context.Context, nodeID domain.NodeID, userID domain.UserID) (bool, error)
}
