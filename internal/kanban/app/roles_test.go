package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Role Command Tests

func TestApp_CreateRole_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.FindBySlugFunc = func(ctx context.Context, slug string) (*domain.Role, error) {
		// Return error to indicate role doesn't exist
		return nil, errors.New("not found")
	}

	mockRoles.CreateFunc = func(ctx context.Context, role domain.Role) error {
		return nil
	}

	role, err := a.CreateRole(ctx, "architect", "System Architect", "📐", "#3B82F6", "Designs system architecture", "Focus on clean architecture", "", []string{"Go", "PostgreSQL"}, 0)

	require.NoError(t, err)
	assert.NotEmpty(t, role.ID)
	assert.Equal(t, "architect", role.Slug)
	assert.Equal(t, "System Architect", role.Name)
	assert.Equal(t, "📐", role.Icon)
	assert.Equal(t, "#3B82F6", role.Color)
	assert.Equal(t, "Designs system architecture", role.Description)
	assert.Equal(t, []string{"Go", "PostgreSQL"}, role.TechStack)
	assert.Equal(t, "Focus on clean architecture", role.PromptHint)
}

func TestApp_CreateRole_EmptySlug_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, _ := setupTestApp()

	_, err := a.CreateRole(ctx, "", "System Architect", "📐", "#3B82F6", "Description", "", "", nil, 0)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrRoleSlugRequired)
}

func TestApp_CreateRole_EmptyName_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, _ := setupTestApp()

	_, err := a.CreateRole(ctx, "architect", "", "📐", "#3B82F6", "Description", "", "", nil, 0)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrRoleNameRequired)
}

func TestApp_UpdateRole_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	existingRole := &domain.Role{
		ID:   roleID,
		Slug: "architect",
		Name: "Old Name",
	}

	mockRoles.FindByIDFunc = func(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
		if id == roleID {
			return existingRole, nil
		}
		return nil, errors.New("not found")
	}

	var updatedRole domain.Role
	mockRoles.UpdateFunc = func(ctx context.Context, role domain.Role) error {
		updatedRole = role
		return nil
	}

	err := a.UpdateRole(ctx, roleID, "New Name", "🏗️", "#10B981", "New Description", "New hint", "", []string{"Go"}, 0)

	require.NoError(t, err)
	assert.Equal(t, "New Name", updatedRole.Name)
	assert.Equal(t, "🏗️", updatedRole.Icon)
	assert.Equal(t, "#10B981", updatedRole.Color)
	assert.Equal(t, "New Description", updatedRole.Description)
	assert.Equal(t, []string{"Go"}, updatedRole.TechStack)
	assert.Equal(t, "New hint", updatedRole.PromptHint)
}

func TestApp_UpdateRole_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()

	mockRoles.FindByIDFunc = func(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
		return nil, errors.New("not found")
	}

	err := a.UpdateRole(ctx, roleID, "New Name", "", "", "", "", "", nil, 0)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestApp_DeleteRole_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	existingRole := &domain.Role{
		ID:   roleID,
		Slug: "architect",
		Name: "System Architect",
	}

	mockRoles.FindByIDFunc = func(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
		if id == roleID {
			return existingRole, nil
		}
		return nil, errors.New("not found")
	}

	mockRoles.DeleteFunc = func(ctx context.Context, id domain.RoleID) error {
		return nil
	}

	err := a.DeleteRole(ctx, roleID)

	require.NoError(t, err)
}

func TestApp_DeleteRole_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()

	mockRoles.FindByIDFunc = func(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
		return nil, errors.New("not found")
	}

	err := a.DeleteRole(ctx, roleID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

// Role Query Tests

func TestApp_GetRole_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	expectedRole := &domain.Role{
		ID:   roleID,
		Slug: "architect",
		Name: "System Architect",
	}

	mockRoles.FindByIDFunc = func(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
		if id == roleID {
			return expectedRole, nil
		}
		return nil, errors.New("not found")
	}

	role, err := a.GetRole(ctx, roleID)

	require.NoError(t, err)
	assert.Equal(t, expectedRole, role)
}

func TestApp_GetRole_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()

	mockRoles.FindByIDFunc = func(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetRole(ctx, roleID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestApp_GetRoleBySlug_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	expectedRole := &domain.Role{
		ID:   domain.NewRoleID(),
		Slug: "architect",
		Name: "System Architect",
	}

	mockRoles.FindBySlugFunc = func(ctx context.Context, slug string) (*domain.Role, error) {
		if slug == "architect" {
			return expectedRole, nil
		}
		return nil, errors.New("not found")
	}

	role, err := a.GetRoleBySlug(ctx, "architect")

	require.NoError(t, err)
	assert.Equal(t, expectedRole, role)
}

func TestApp_GetRoleBySlug_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.FindBySlugFunc = func(ctx context.Context, slug string) (*domain.Role, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetRoleBySlug(ctx, "nonexistent")

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestApp_ListRoles_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	expectedRoles := []domain.Role{
		{ID: domain.NewRoleID(), Slug: "architect", Name: "System Architect"},
		{ID: domain.NewRoleID(), Slug: "developer", Name: "Developer"},
	}

	mockRoles.ListFunc = func(ctx context.Context) ([]domain.Role, error) {
		return expectedRoles, nil
	}

	roles, err := a.ListRoles(ctx)

	require.NoError(t, err)
	assert.Equal(t, expectedRoles, roles)
}

// Per-project role command tests

func TestApp_CreateProjectRole_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockRoles.FindBySlugInProjectFunc = func(ctx context.Context, pid domain.ProjectID, slug string) (*domain.Role, error) {
		return nil, errors.New("not found")
	}

	var createdRole domain.Role
	mockRoles.CreateInProjectFunc = func(ctx context.Context, pid domain.ProjectID, role domain.Role) error {
		createdRole = role
		return nil
	}

	role, err := a.CreateProjectRole(ctx, projectID, "dev", "Developer", "", "#000", "desc", "hint", "", []string{"Go"}, 1)

	require.NoError(t, err)
	assert.NotEmpty(t, role.ID)
	assert.Equal(t, "dev", role.Slug)
	assert.Equal(t, "Developer", role.Name)
	assert.NotEmpty(t, createdRole.ID)
}

func TestApp_CreateProjectRole_EmptySlug_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, _ := setupTestApp()

	_, err := a.CreateProjectRole(ctx, domain.NewProjectID(), "", "Developer", "", "", "", "", "", nil, 0)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleSlugRequired)
}

func TestApp_CreateProjectRole_EmptyName_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, _ := setupTestApp()

	_, err := a.CreateProjectRole(ctx, domain.NewProjectID(), "dev", "", "", "", "", "", "", nil, 0)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleNameRequired)
}

func TestApp_CreateProjectRole_AlreadyExists_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	existingRole := &domain.Role{ID: domain.NewRoleID(), Slug: "dev", Name: "Developer"}

	mockRoles.FindBySlugInProjectFunc = func(ctx context.Context, pid domain.ProjectID, slug string) (*domain.Role, error) {
		return existingRole, nil
	}

	_, err := a.CreateProjectRole(ctx, projectID, "dev", "Developer", "", "", "", "", "", nil, 0)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleAlreadyExists)
}

func TestApp_UpdateProjectRole_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	roleID := domain.NewRoleID()
	existingRole := &domain.Role{ID: roleID, Slug: "dev", Name: "Old Name"}

	mockRoles.FindByIDInProjectFunc = func(ctx context.Context, pid domain.ProjectID, rid domain.RoleID) (*domain.Role, error) {
		if pid == projectID && rid == roleID {
			return existingRole, nil
		}
		return nil, errors.New("not found")
	}

	var updatedRole domain.Role
	mockRoles.UpdateInProjectFunc = func(ctx context.Context, pid domain.ProjectID, role domain.Role) error {
		updatedRole = role
		return nil
	}

	err := a.UpdateProjectRole(ctx, projectID, roleID, "New Name", "", "", "", "", "", nil, 0)

	require.NoError(t, err)
	assert.Equal(t, "New Name", updatedRole.Name)
}

func TestApp_UpdateProjectRole_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.FindByIDInProjectFunc = func(ctx context.Context, pid domain.ProjectID, rid domain.RoleID) (*domain.Role, error) {
		return nil, errors.New("not found")
	}

	err := a.UpdateProjectRole(ctx, domain.NewProjectID(), domain.NewRoleID(), "New Name", "", "", "", "", "", nil, 0)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestApp_DeleteProjectRole_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	roleID := domain.NewRoleID()

	mockRoles.FindByIDInProjectFunc = func(ctx context.Context, pid domain.ProjectID, rid domain.RoleID) (*domain.Role, error) {
		if pid == projectID && rid == roleID {
			return &domain.Role{ID: roleID, Slug: "dev", Name: "Developer"}, nil
		}
		return nil, errors.New("not found")
	}

	mockRoles.DeleteInProjectFunc = func(ctx context.Context, pid domain.ProjectID, rid domain.RoleID) error {
		return nil
	}

	err := a.DeleteProjectRole(ctx, projectID, roleID)

	require.NoError(t, err)
}

func TestApp_DeleteProjectRole_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.FindByIDInProjectFunc = func(ctx context.Context, pid domain.ProjectID, rid domain.RoleID) (*domain.Role, error) {
		return nil, errors.New("not found")
	}

	err := a.DeleteProjectRole(ctx, domain.NewProjectID(), domain.NewRoleID())

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestApp_ListProjectRoles_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	expectedRoles := []domain.Role{
		{ID: domain.NewRoleID(), Slug: "dev", Name: "Developer"},
	}

	mockRoles.ListInProjectFunc = func(ctx context.Context, pid domain.ProjectID) ([]domain.Role, error) {
		if pid == projectID {
			return expectedRoles, nil
		}
		return nil, errors.New("not found")
	}

	roles, err := a.ListProjectRoles(ctx, projectID)

	require.NoError(t, err)
	assert.Equal(t, expectedRoles, roles)
}

func TestApp_GetProjectRoleBySlug_Success(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	expectedRole := &domain.Role{ID: domain.NewRoleID(), Slug: "dev", Name: "Developer"}

	mockRoles.FindBySlugInProjectFunc = func(ctx context.Context, pid domain.ProjectID, slug string) (*domain.Role, error) {
		if pid == projectID && slug == "dev" {
			return expectedRole, nil
		}
		return nil, errors.New("not found")
	}

	role, err := a.GetProjectRoleBySlug(ctx, projectID, "dev")

	require.NoError(t, err)
	assert.Equal(t, expectedRole, role)
}

func TestApp_GetProjectRoleBySlug_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.FindBySlugInProjectFunc = func(ctx context.Context, pid domain.ProjectID, slug string) (*domain.Role, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetProjectRoleBySlug(ctx, domain.NewProjectID(), "nonexistent")

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}
