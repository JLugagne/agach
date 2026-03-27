package projectaccess

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// ProjectAccessRepository defines operations for managing project access grants.
type ProjectAccessRepository interface {
	// GrantUser grants a user access to a project with a given role.
	GrantUser(ctx context.Context, projectID domain.ProjectID, userID, role string) error

	// RevokeUser revokes a user's access to a project.
	RevokeUser(ctx context.Context, projectID domain.ProjectID, userID string) error

	// UpdateUserRole updates a user's role on a project.
	UpdateUserRole(ctx context.Context, projectID domain.ProjectID, userID, role string) error

	// GrantTeam grants a team access to a project.
	GrantTeam(ctx context.Context, projectID domain.ProjectID, teamID string) error

	// RevokeTeam revokes a team's access to a project.
	RevokeTeam(ctx context.Context, projectID domain.ProjectID, teamID string) error

	// ListUserAccess returns all user access grants for a project.
	ListUserAccess(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectUserAccess, error)

	// ListTeamAccess returns all team access grants for a project.
	ListTeamAccess(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectTeamAccess, error)

	// HasAccess checks whether a user has access to a project, either directly
	// or via team membership. teamIDs should be the user's current team IDs.
	HasAccess(ctx context.Context, projectID domain.ProjectID, userID string, teamIDs []string) (bool, error)

	// ListAccessibleProjectIDs returns all project IDs the user can access
	// (directly or via team membership).
	ListAccessibleProjectIDs(ctx context.Context, userID string, teamIDs []string) ([]domain.ProjectID, error)
}
