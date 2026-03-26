package chatstest

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/chats"
)

var _ chats.ChatSessionRepository = (*MockChatSession)(nil)

type MockChatSession struct {
	CreateFunc           func(ctx context.Context, session domain.ChatSession) error
	FindByIDFunc         func(ctx context.Context, id domain.ChatSessionID) (*domain.ChatSession, error)
	FindByFeatureFunc    func(ctx context.Context, featureID domain.FeatureID) ([]domain.ChatSession, error)
	UpdateFunc           func(ctx context.Context, session domain.ChatSession) error
	UpdateStateFunc      func(ctx context.Context, id domain.ChatSessionID, state domain.ChatSessionState, endedAt *time.Time) error
	UpdateJSONLPathFunc  func(ctx context.Context, id domain.ChatSessionID, path string) error
	UpdateTokenUsageFunc func(ctx context.Context, id domain.ChatSessionID, usage domain.TokenUsage) error
}

func (m *MockChatSession) Create(ctx context.Context, session domain.ChatSession) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, session)
}

func (m *MockChatSession) FindByID(ctx context.Context, id domain.ChatSessionID) (*domain.ChatSession, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockChatSession) FindByFeature(ctx context.Context, featureID domain.FeatureID) ([]domain.ChatSession, error) {
	if m.FindByFeatureFunc == nil {
		panic("called not defined FindByFeatureFunc")
	}
	return m.FindByFeatureFunc(ctx, featureID)
}

func (m *MockChatSession) Update(ctx context.Context, session domain.ChatSession) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, session)
}

func (m *MockChatSession) UpdateState(ctx context.Context, id domain.ChatSessionID, state domain.ChatSessionState, endedAt *time.Time) error {
	if m.UpdateStateFunc == nil {
		panic("called not defined UpdateStateFunc")
	}
	return m.UpdateStateFunc(ctx, id, state, endedAt)
}

func (m *MockChatSession) UpdateJSONLPath(ctx context.Context, id domain.ChatSessionID, path string) error {
	if m.UpdateJSONLPathFunc == nil {
		panic("called not defined UpdateJSONLPathFunc")
	}
	return m.UpdateJSONLPathFunc(ctx, id, path)
}

func (m *MockChatSession) UpdateTokenUsage(ctx context.Context, id domain.ChatSessionID, usage domain.TokenUsage) error {
	if m.UpdateTokenUsageFunc == nil {
		panic("called not defined UpdateTokenUsageFunc")
	}
	return m.UpdateTokenUsageFunc(ctx, id, usage)
}
