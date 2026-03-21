package service

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
)

// TeamCommands handles team and membership mutations.
type TeamCommands interface {
	CreateTeam(ctx context.Context, actor domain.Actor, name, slug, description string) (domain.Team, error)
	UpdateTeam(ctx context.Context, actor domain.Actor, team domain.Team) error
	DeleteTeam(ctx context.Context, actor domain.Actor, id domain.TeamID) error
	AddUserToTeam(ctx context.Context, actor domain.Actor, userID domain.UserID, teamID domain.TeamID) error
	RemoveUserFromTeam(ctx context.Context, actor domain.Actor, userID domain.UserID) error
	SetUserRole(ctx context.Context, actor domain.Actor, userID domain.UserID, role domain.MemberRole) error
}

// TeamQueries handles team and membership lookups.
type TeamQueries interface {
	ListTeams(ctx context.Context) ([]domain.Team, error)
	GetTeam(ctx context.Context, id domain.TeamID) (domain.Team, error)
	ListUsers(ctx context.Context) ([]domain.User, error)
	ListTeamMembers(ctx context.Context, teamID domain.TeamID) ([]domain.User, error)
}
