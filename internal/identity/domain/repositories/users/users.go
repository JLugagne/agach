package users

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
	FindByID(ctx context.Context, id domain.UserID) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	Update(ctx context.Context, user domain.User) error
	ListAll(ctx context.Context) ([]domain.User, error)
	ListByTeam(ctx context.Context, teamID domain.TeamID) ([]domain.User, error)
	FindBySSO(ctx context.Context, provider, subject string) (domain.User, error)
	AddToTeam(ctx context.Context, userID domain.UserID, teamID domain.TeamID) error
	RemoveFromTeam(ctx context.Context, userID domain.UserID, teamID domain.TeamID) error
	ListTeamIDs(ctx context.Context, userID domain.UserID) ([]domain.TeamID, error)
}
