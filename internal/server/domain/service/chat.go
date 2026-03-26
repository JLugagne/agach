package service

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// ChatService defines the chat session operations used by inbound handlers.
type ChatService interface {
	CreateSession(ctx context.Context, featureID domain.FeatureID, projectID domain.ProjectID, nodeID string) (*domain.ChatSession, error)
	GetSession(ctx context.Context, id domain.ChatSessionID) (*domain.ChatSession, error)
	ListSessions(ctx context.Context, featureID domain.FeatureID) ([]domain.ChatSession, error)
	EndSession(ctx context.Context, id domain.ChatSessionID, state domain.ChatSessionState) error
	UpdateJSONLPath(ctx context.Context, id domain.ChatSessionID, path string) error
	UpdateTokenUsage(ctx context.Context, id domain.ChatSessionID, usage domain.TokenUsage) error
}
