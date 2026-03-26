package chats

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// ChatSessionRepository defines operations for managing chat sessions
type ChatSessionRepository interface {
	// Create creates a new chat session
	Create(ctx context.Context, session domain.ChatSession) error

	// FindByID retrieves a chat session by ID
	FindByID(ctx context.Context, id domain.ChatSessionID) (*domain.ChatSession, error)

	// FindByFeature retrieves all chat sessions for a feature
	FindByFeature(ctx context.Context, featureID domain.FeatureID) ([]domain.ChatSession, error)

	// Update updates a chat session
	Update(ctx context.Context, session domain.ChatSession) error

	// UpdateState updates only the state of a chat session
	UpdateState(ctx context.Context, id domain.ChatSessionID, state domain.ChatSessionState, endedAt *time.Time) error

	// UpdateJSONLPath updates the JSONL file path for a chat session
	UpdateJSONLPath(ctx context.Context, id domain.ChatSessionID, path string) error

	// UpdateTokenUsage updates the token usage for a chat session
	UpdateTokenUsage(ctx context.Context, id domain.ChatSessionID, usage domain.TokenUsage) error
}
