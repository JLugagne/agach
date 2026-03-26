package app

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/chats"
)

// ChatService handles chat session operations
type ChatService struct {
	repo chats.ChatSessionRepository
}

// NewChatService creates a new ChatService
func NewChatService(repo chats.ChatSessionRepository) *ChatService {
	return &ChatService{repo: repo}
}

// CreateSession creates a new chat session
func (s *ChatService) CreateSession(ctx context.Context, featureID domain.FeatureID, projectID domain.ProjectID, nodeID string) (*domain.ChatSession, error) {
	now := time.Now().UTC()
	session := domain.ChatSession{
		ID:        domain.NewChatSessionID(),
		FeatureID: featureID,
		ProjectID: projectID,
		NodeID:    nodeID,
		State:     domain.ChatStateActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, session); err != nil {
		return nil, err
	}
	return &session, nil
}

// GetSession retrieves a chat session by ID
func (s *ChatService) GetSession(ctx context.Context, id domain.ChatSessionID) (*domain.ChatSession, error) {
	return s.repo.FindByID(ctx, id)
}

// ListSessions retrieves all chat sessions for a feature
func (s *ChatService) ListSessions(ctx context.Context, featureID domain.FeatureID) ([]domain.ChatSession, error) {
	return s.repo.FindByFeature(ctx, featureID)
}

// EndSession ends a chat session
func (s *ChatService) EndSession(ctx context.Context, id domain.ChatSessionID, state domain.ChatSessionState) error {
	now := time.Now().UTC()
	return s.repo.UpdateState(ctx, id, state, &now)
}

// UpdateJSONLPath updates the JSONL path for a session
func (s *ChatService) UpdateJSONLPath(ctx context.Context, id domain.ChatSessionID, path string) error {
	return s.repo.UpdateJSONLPath(ctx, id, path)
}

// UpdateTokenUsage updates token counts and model for a session
func (s *ChatService) UpdateTokenUsage(ctx context.Context, id domain.ChatSessionID, usage domain.TokenUsage) error {
	return s.repo.UpdateTokenUsage(ctx, id, usage)
}
