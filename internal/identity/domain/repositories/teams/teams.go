package teams

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type TeamRepository interface {
	Create(ctx context.Context, team domain.Team) error
	FindByID(ctx context.Context, id domain.TeamID) (domain.Team, error)
	FindBySlug(ctx context.Context, slug string) (domain.Team, error)
	List(ctx context.Context) ([]domain.Team, error)
	Update(ctx context.Context, team domain.Team) error
	Delete(ctx context.Context, id domain.TeamID) error
}
