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

	role, err := a.CreateRole(ctx, "architect", "System Architect", "📐", "#3B82F6", "Designs system architecture", "Focus on clean architecture", []string{"Go", "PostgreSQL"}, 0)

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

	_, err := a.CreateRole(ctx, "", "System Architect", "📐", "#3B82F6", "Description", "", nil, 0)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrRoleSlugRequired)
}

func TestApp_CreateRole_EmptyName_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, _ := setupTestApp()

	_, err := a.CreateRole(ctx, "architect", "", "📐", "#3B82F6", "Description", "", nil, 0)

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

	err := a.UpdateRole(ctx, roleID, "New Name", "🏗️", "#10B981", "New Description", "New hint", []string{"Go"}, 0)

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

	err := a.UpdateRole(ctx, roleID, "New Name", "", "", "", "", nil, 0)

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
