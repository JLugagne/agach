package apikeys

import (
	"context"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type APIKeyRepository interface {
	Create(ctx context.Context, key domain.APIKey) error
	FindByID(ctx context.Context, id domain.APIKeyID) (domain.APIKey, error)
	FindByHash(ctx context.Context, hash string) (domain.APIKey, error)
	Revoke(ctx context.Context, id domain.APIKeyID) error
	UpdateLastUsed(ctx context.Context, id domain.APIKeyID, at time.Time) error
	ListByUser(ctx context.Context, userID domain.UserID) ([]domain.APIKey, error)
}
