package nodes

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type NodeRepository interface {
	Create(ctx context.Context, node domain.Node) error
	FindByID(ctx context.Context, id domain.NodeID) (domain.Node, error)
	ListByOwner(ctx context.Context, ownerID domain.UserID) ([]domain.Node, error)
	ListActiveByOwner(ctx context.Context, ownerID domain.UserID) ([]domain.Node, error)
	Update(ctx context.Context, node domain.Node) error
	UpdateLastSeen(ctx context.Context, id domain.NodeID) error
	ListAll(ctx context.Context) ([]domain.Node, error)
}
