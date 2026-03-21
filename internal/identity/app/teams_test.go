package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/identity/app"
	"github.com/JLugagne/agach-mcp/internal/identity/domain"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/teams/teamstest"
	"github.com/JLugagne/agach-mcp/internal/identity/domain/repositories/users/userstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func adminActor() domain.Actor {
	return domain.Actor{UserID: domain.NewUserID(), Email: "admin@example.com", Role: domain.RoleAdmin}
}

func memberActor() domain.Actor {
	return domain.Actor{UserID: domain.NewUserID(), Email: "member@example.com", Role: domain.RoleMember}
}

// ─────────────────────────────────────────────────────────────────────────────
// CreateTeam
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamService_CreateTeam_Success(t *testing.T) {
	ctx := context.Background()

	mockTeams := &teamstest.MockTeamRepository{
		FindBySlugFunc: func(_ context.Context, slug string) (domain.Team, error) {
			return domain.Team{}, domain.ErrTeamNotFound
		},
		CreateFunc: func(_ context.Context, team domain.Team) error {
			return nil
		},
	}

	svc := app.NewTeamService(mockTeams, &userstest.MockUserRepository{})

	team, err := svc.CreateTeam(ctx, adminActor(), "Engineering", "engineering", "Eng team")

	require.NoError(t, err)
	assert.NotEmpty(t, team.ID)
	assert.Equal(t, "Engineering", team.Name)
	assert.Equal(t, "engineering", team.Slug)
}

func TestTeamService_CreateTeam_NonAdmin_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, &userstest.MockUserRepository{})

	_, err := svc.CreateTeam(ctx, memberActor(), "Engineering", "engineering", "")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestTeamService_CreateTeam_EmptySlug_ReturnsError(t *testing.T) {
	ctx := context.Background()

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, &userstest.MockUserRepository{})

	_, err := svc.CreateTeam(ctx, adminActor(), "Engineering", "   ", "")

	require.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
}

func TestTeamService_CreateTeam_SlugConflict_ReturnsError(t *testing.T) {
	ctx := context.Background()

	mockTeams := &teamstest.MockTeamRepository{
		FindBySlugFunc: func(_ context.Context, slug string) (domain.Team, error) {
			return domain.Team{ID: domain.NewTeamID(), Slug: slug}, nil // slug exists
		},
	}

	svc := app.NewTeamService(mockTeams, &userstest.MockUserRepository{})

	_, err := svc.CreateTeam(ctx, adminActor(), "Engineering", "existing-slug", "")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTeamSlugConflict)
}

func TestTeamService_CreateTeam_SlugNormalized(t *testing.T) {
	ctx := context.Background()

	var createdTeam domain.Team
	mockTeams := &teamstest.MockTeamRepository{
		FindBySlugFunc: func(_ context.Context, slug string) (domain.Team, error) {
			return domain.Team{}, domain.ErrTeamNotFound
		},
		CreateFunc: func(_ context.Context, team domain.Team) error {
			createdTeam = team
			return nil
		},
	}

	svc := app.NewTeamService(mockTeams, &userstest.MockUserRepository{})

	_, err := svc.CreateTeam(ctx, adminActor(), "Engineering", "  Engineering-Team  ", "")

	require.NoError(t, err)
	assert.Equal(t, "engineering-team", createdTeam.Slug, "slug should be lowercased and trimmed")
}

// ─────────────────────────────────────────────────────────────────────────────
// UpdateTeam
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamService_UpdateTeam_Success(t *testing.T) {
	ctx := context.Background()

	mockTeams := &teamstest.MockTeamRepository{
		UpdateFunc: func(_ context.Context, team domain.Team) error {
			return nil
		},
	}

	svc := app.NewTeamService(mockTeams, &userstest.MockUserRepository{})

	team := domain.Team{ID: domain.NewTeamID(), Name: "Updated", Slug: "updated"}
	err := svc.UpdateTeam(ctx, adminActor(), team)

	require.NoError(t, err)
}

func TestTeamService_UpdateTeam_EmptySlug_ReturnsError(t *testing.T) {
	ctx := context.Background()

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, &userstest.MockUserRepository{})

	team := domain.Team{ID: domain.NewTeamID(), Name: "Updated", Slug: "  "}
	err := svc.UpdateTeam(ctx, adminActor(), team)

	require.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
}

func TestTeamService_UpdateTeam_NonAdmin_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, &userstest.MockUserRepository{})

	err := svc.UpdateTeam(ctx, memberActor(), domain.Team{ID: domain.NewTeamID()})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// ─────────────────────────────────────────────────────────────────────────────
// DeleteTeam
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamService_DeleteTeam_Success(t *testing.T) {
	ctx := context.Background()

	mockTeams := &teamstest.MockTeamRepository{
		DeleteFunc: func(_ context.Context, id domain.TeamID) error {
			return nil
		},
	}

	svc := app.NewTeamService(mockTeams, &userstest.MockUserRepository{})

	err := svc.DeleteTeam(ctx, adminActor(), domain.NewTeamID())

	require.NoError(t, err)
}

func TestTeamService_DeleteTeam_NonAdmin_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, &userstest.MockUserRepository{})

	err := svc.DeleteTeam(ctx, memberActor(), domain.NewTeamID())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// ─────────────────────────────────────────────────────────────────────────────
// AddUserToTeam
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamService_AddUserToTeam_Success(t *testing.T) {
	ctx := context.Background()

	userID := domain.NewUserID()
	teamID := domain.NewTeamID()

	var updatedUser domain.User
	mockUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return domain.User{ID: id, Email: "user@example.com"}, nil
		},
		UpdateFunc: func(_ context.Context, u domain.User) error {
			updatedUser = u
			return nil
		},
	}

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, mockUsers)

	err := svc.AddUserToTeam(ctx, adminActor(), userID, teamID)

	require.NoError(t, err)
	require.NotNil(t, updatedUser.TeamID)
	assert.Equal(t, teamID, *updatedUser.TeamID)
}

func TestTeamService_AddUserToTeam_NonAdmin_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, &userstest.MockUserRepository{})

	err := svc.AddUserToTeam(ctx, memberActor(), domain.NewUserID(), domain.NewTeamID())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestTeamService_AddUserToTeam_UserNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, mockUsers)

	err := svc.AddUserToTeam(ctx, adminActor(), domain.NewUserID(), domain.NewTeamID())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

// ─────────────────────────────────────────────────────────────────────────────
// RemoveUserFromTeam
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamService_RemoveUserFromTeam_Success(t *testing.T) {
	ctx := context.Background()

	teamID := domain.NewTeamID()
	var updatedUser domain.User
	mockUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return domain.User{ID: id, TeamID: &teamID}, nil
		},
		UpdateFunc: func(_ context.Context, u domain.User) error {
			updatedUser = u
			return nil
		},
	}

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, mockUsers)

	err := svc.RemoveUserFromTeam(ctx, adminActor(), domain.NewUserID())

	require.NoError(t, err)
	assert.Nil(t, updatedUser.TeamID, "TeamID should be cleared")
}

func TestTeamService_RemoveUserFromTeam_NonAdmin_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, &userstest.MockUserRepository{})

	err := svc.RemoveUserFromTeam(ctx, memberActor(), domain.NewUserID())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

// ─────────────────────────────────────────────────────────────────────────────
// SetUserRole
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamService_SetUserRole_Success(t *testing.T) {
	ctx := context.Background()

	var updatedUser domain.User
	mockUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return domain.User{ID: id, Role: domain.RoleMember}, nil
		},
		UpdateFunc: func(_ context.Context, u domain.User) error {
			updatedUser = u
			return nil
		},
	}

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, mockUsers)

	err := svc.SetUserRole(ctx, adminActor(), domain.NewUserID(), domain.RoleAdmin)

	require.NoError(t, err)
	assert.Equal(t, domain.RoleAdmin, updatedUser.Role)
}

func TestTeamService_SetUserRole_NonAdmin_ReturnsForbidden(t *testing.T) {
	ctx := context.Background()

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, &userstest.MockUserRepository{})

	err := svc.SetUserRole(ctx, memberActor(), domain.NewUserID(), domain.RoleAdmin)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)
}

func TestTeamService_SetUserRole_UserNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, mockUsers)

	err := svc.SetUserRole(ctx, adminActor(), domain.NewUserID(), domain.RoleAdmin)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

// ─────────────────────────────────────────────────────────────────────────────
// ListTeams / GetTeam
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamService_ListTeams_Success(t *testing.T) {
	ctx := context.Background()

	expected := []domain.Team{
		{ID: domain.NewTeamID(), Name: "Team A"},
		{ID: domain.NewTeamID(), Name: "Team B"},
	}

	mockTeams := &teamstest.MockTeamRepository{
		ListFunc: func(_ context.Context) ([]domain.Team, error) {
			return expected, nil
		},
	}

	svc := app.NewTeamQueriesService(mockTeams, &userstest.MockUserRepository{})

	teams, err := svc.ListTeams(ctx)

	require.NoError(t, err)
	assert.Len(t, teams, 2)
}

func TestTeamService_GetTeam_Success(t *testing.T) {
	ctx := context.Background()

	teamID := domain.NewTeamID()
	expected := domain.Team{ID: teamID, Name: "Engineering"}

	mockTeams := &teamstest.MockTeamRepository{
		FindByIDFunc: func(_ context.Context, id domain.TeamID) (domain.Team, error) {
			return expected, nil
		},
	}

	svc := app.NewTeamQueriesService(mockTeams, &userstest.MockUserRepository{})

	team, err := svc.GetTeam(ctx, teamID)

	require.NoError(t, err)
	assert.Equal(t, expected.ID, team.ID)
	assert.Equal(t, expected.Name, team.Name)
}

func TestTeamService_GetTeam_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()

	mockTeams := &teamstest.MockTeamRepository{
		FindByIDFunc: func(_ context.Context, id domain.TeamID) (domain.Team, error) {
			return domain.Team{}, domain.ErrTeamNotFound
		},
	}

	svc := app.NewTeamQueriesService(mockTeams, &userstest.MockUserRepository{})

	_, err := svc.GetTeam(ctx, domain.NewTeamID())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTeamNotFound)
}

// ─────────────────────────────────────────────────────────────────────────────
// ListUsers / ListTeamMembers
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamService_ListUsers_Success(t *testing.T) {
	ctx := context.Background()

	expected := []domain.User{
		{ID: domain.NewUserID(), Email: "a@example.com"},
		{ID: domain.NewUserID(), Email: "b@example.com"},
	}

	mockUsers := &userstest.MockUserRepository{
		ListAllFunc: func(_ context.Context) ([]domain.User, error) {
			return expected, nil
		},
	}

	svc := app.NewTeamQueriesService(&teamstest.MockTeamRepository{}, mockUsers)

	users, err := svc.ListUsers(ctx)

	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestTeamService_ListTeamMembers_Success(t *testing.T) {
	ctx := context.Background()

	teamID := domain.NewTeamID()
	expected := []domain.User{
		{ID: domain.NewUserID(), Email: "member@example.com"},
	}

	mockUsers := &userstest.MockUserRepository{
		ListByTeamFunc: func(_ context.Context, id domain.TeamID) ([]domain.User, error) {
			assert.Equal(t, teamID, id)
			return expected, nil
		},
	}

	svc := app.NewTeamQueriesService(&teamstest.MockTeamRepository{}, mockUsers)

	members, err := svc.ListTeamMembers(ctx, teamID)

	require.NoError(t, err)
	assert.Len(t, members, 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// Repository errors bubble up
// ─────────────────────────────────────────────────────────────────────────────

func TestTeamService_RemoveUserFromTeam_UserNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()

	mockUsers := &userstest.MockUserRepository{
		FindByIDFunc: func(_ context.Context, id domain.UserID) (domain.User, error) {
			return domain.User{}, domain.ErrUserNotFound
		},
	}

	svc := app.NewTeamService(&teamstest.MockTeamRepository{}, mockUsers)

	err := svc.RemoveUserFromTeam(ctx, adminActor(), domain.NewUserID())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestTeamService_CreateTeam_RepoError_Bubbles(t *testing.T) {
	ctx := context.Background()

	repoErr := errors.New("db connection lost")
	mockTeams := &teamstest.MockTeamRepository{
		FindBySlugFunc: func(_ context.Context, slug string) (domain.Team, error) {
			return domain.Team{}, domain.ErrTeamNotFound
		},
		CreateFunc: func(_ context.Context, team domain.Team) error {
			return repoErr
		},
	}

	svc := app.NewTeamService(mockTeams, &userstest.MockUserRepository{})

	_, err := svc.CreateTeam(ctx, adminActor(), "Engineering", "engineering", "")

	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
}
