package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/teams"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/service"
)

type teamService struct {
	teams teams.TeamRepository
	users users.UserRepository
}

// NewTeamService wires a TeamCommands+TeamQueries implementation.
func NewTeamService(t teams.TeamRepository, u users.UserRepository) service.TeamCommands {
	return &teamService{teams: t, users: u}
}

// NewTeamQueriesService returns the read-side of the team service.
func NewTeamQueriesService(t teams.TeamRepository, u users.UserRepository) service.TeamQueries {
	return &teamService{teams: t, users: u}
}

var (
	_ service.TeamCommands = (*teamService)(nil)
	_ service.TeamQueries  = (*teamService)(nil)
)

func (s *teamService) CreateTeam(ctx context.Context, actor domain.Actor, name, slug, description string) (domain.Team, error) {
	if !actor.IsAdmin() {
		return domain.Team{}, domain.ErrForbidden
	}
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return domain.Team{}, &domain.Error{Code: "SLUG_REQUIRED", Message: "team slug is required"}
	}
	_, err := s.teams.FindBySlug(ctx, slug)
	if err == nil {
		return domain.Team{}, domain.ErrTeamSlugConflict
	}
	if !errors.Is(err, domain.ErrTeamNotFound) {
		return domain.Team{}, err
	}
	now := time.Now()
	t := domain.Team{
		ID:          domain.NewTeamID(),
		Name:        strings.TrimSpace(name),
		Slug:        slug,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.teams.Create(ctx, t); err != nil {
		return domain.Team{}, err
	}
	return t, nil
}

func (s *teamService) UpdateTeam(ctx context.Context, actor domain.Actor, team domain.Team) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	if strings.TrimSpace(team.Slug) == "" {
		return &domain.Error{Code: "SLUG_REQUIRED", Message: "team slug is required"}
	}
	team.UpdatedAt = time.Now()
	return s.teams.Update(ctx, team)
}

func (s *teamService) DeleteTeam(ctx context.Context, actor domain.Actor, id domain.TeamID) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	return s.teams.Delete(ctx, id)
}

func (s *teamService) AddUserToTeam(ctx context.Context, actor domain.Actor, userID domain.UserID, teamID domain.TeamID) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	return s.users.AddToTeam(ctx, userID, teamID)
}

func (s *teamService) RemoveUserFromTeam(ctx context.Context, actor domain.Actor, userID domain.UserID, teamID domain.TeamID) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	return s.users.RemoveFromTeam(ctx, userID, teamID)
}

func (s *teamService) SetUserRole(ctx context.Context, actor domain.Actor, userID domain.UserID, role domain.MemberRole) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	u.Role = role
	u.UpdatedAt = time.Now()
	return s.users.Update(ctx, u)
}

func (s *teamService) BlockUser(ctx context.Context, actor domain.Actor, userID domain.UserID) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	now := time.Now()
	u.BlockedAt = &now
	u.UpdatedAt = now
	return s.users.Update(ctx, u)
}

func (s *teamService) UnblockUser(ctx context.Context, actor domain.Actor, userID domain.UserID) error {
	if !actor.IsAdmin() {
		return domain.ErrForbidden
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	u.BlockedAt = nil
	u.UpdatedAt = time.Now()
	return s.users.Update(ctx, u)
}

func (s *teamService) ListTeams(ctx context.Context) ([]domain.Team, error) {
	return s.teams.List(ctx)
}

func (s *teamService) GetTeam(ctx context.Context, id domain.TeamID) (domain.Team, error) {
	return s.teams.FindByID(ctx, id)
}

func (s *teamService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.users.ListAll(ctx)
}

func (s *teamService) ListTeamMembers(ctx context.Context, teamID domain.TeamID) ([]domain.User, error) {
	return s.users.ListByTeam(ctx, teamID)
}
