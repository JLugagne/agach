package nodestest

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type MockNodeRepository struct {
	CreateFunc           func(ctx context.Context, node domain.Node) error
	FindByIDFunc         func(ctx context.Context, id domain.NodeID) (domain.Node, error)
	ListByOwnerFunc      func(ctx context.Context, ownerID domain.UserID) ([]domain.Node, error)
	ListActiveByOwnerFunc func(ctx context.Context, ownerID domain.UserID) ([]domain.Node, error)
	UpdateFunc           func(ctx context.Context, node domain.Node) error
	UpdateLastSeenFunc   func(ctx context.Context, id domain.NodeID) error
	ListAllFunc          func(ctx context.Context) ([]domain.Node, error)
}

func (m *MockNodeRepository) Create(ctx context.Context, node domain.Node) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, node)
}

func (m *MockNodeRepository) FindByID(ctx context.Context, id domain.NodeID) (domain.Node, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockNodeRepository) ListByOwner(ctx context.Context, ownerID domain.UserID) ([]domain.Node, error) {
	if m.ListByOwnerFunc == nil {
		panic("called not defined ListByOwnerFunc")
	}
	return m.ListByOwnerFunc(ctx, ownerID)
}

func (m *MockNodeRepository) ListActiveByOwner(ctx context.Context, ownerID domain.UserID) ([]domain.Node, error) {
	if m.ListActiveByOwnerFunc == nil {
		panic("called not defined ListActiveByOwnerFunc")
	}
	return m.ListActiveByOwnerFunc(ctx, ownerID)
}

func (m *MockNodeRepository) Update(ctx context.Context, node domain.Node) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, node)
}

func (m *MockNodeRepository) UpdateLastSeen(ctx context.Context, id domain.NodeID) error {
	if m.UpdateLastSeenFunc == nil {
		panic("called not defined UpdateLastSeenFunc")
	}
	return m.UpdateLastSeenFunc(ctx, id)
}

func (m *MockNodeRepository) ListAll(ctx context.Context) ([]domain.Node, error) {
	if m.ListAllFunc == nil {
		panic("called not defined ListAllFunc")
	}
	return m.ListAllFunc(ctx)
}
