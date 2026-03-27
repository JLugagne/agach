package app

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	projectaccessrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projectaccess"
	"github.com/sirupsen/logrus"
)

// ProjectAccessService handles project access grant operations.
type ProjectAccessService struct {
	access projectaccessrepo.ProjectAccessRepository
	logger *logrus.Logger
}

func newProjectAccessService(access projectaccessrepo.ProjectAccessRepository, logger *logrus.Logger) *ProjectAccessService {
	return &ProjectAccessService{access: access, logger: logger}
}

var validProjectRoles = map[string]bool{
	"admin":  true,
	"member": true,
}

func (s *ProjectAccessService) GrantUserAccess(ctx context.Context, projectID domain.ProjectID, userID, role string) error {
	if !validProjectRoles[role] {
		return domain.ErrInvalidTaskData
	}
	return s.access.GrantUser(ctx, projectID, userID, role)
}

func (s *ProjectAccessService) RevokeUserAccess(ctx context.Context, projectID domain.ProjectID, userID string) error {
	return s.access.RevokeUser(ctx, projectID, userID)
}

func (s *ProjectAccessService) UpdateUserAccessRole(ctx context.Context, projectID domain.ProjectID, userID, role string) error {
	return s.access.UpdateUserRole(ctx, projectID, userID, role)
}

func (s *ProjectAccessService) GrantTeamAccess(ctx context.Context, projectID domain.ProjectID, teamID string) error {
	return s.access.GrantTeam(ctx, projectID, teamID)
}

func (s *ProjectAccessService) RevokeTeamAccess(ctx context.Context, projectID domain.ProjectID, teamID string) error {
	return s.access.RevokeTeam(ctx, projectID, teamID)
}

func (s *ProjectAccessService) ListProjectUserAccess(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectUserAccess, error) {
	return s.access.ListUserAccess(ctx, projectID)
}

func (s *ProjectAccessService) ListProjectTeamAccess(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectTeamAccess, error) {
	return s.access.ListTeamAccess(ctx, projectID)
}

func (s *ProjectAccessService) HasProjectAccess(ctx context.Context, projectID domain.ProjectID, userID string, teamIDs []string) (bool, error) {
	return s.access.HasAccess(ctx, projectID, userID, teamIDs)
}

func (s *ProjectAccessService) ListAccessibleProjectIDs(ctx context.Context, userID string, teamIDs []string) ([]domain.ProjectID, error) {
	return s.access.ListAccessibleProjectIDs(ctx, userID, teamIDs)
}
