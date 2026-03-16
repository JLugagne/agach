package rolestest

import (
	"context"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/roles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRoleRepository is a function-based mock implementation of the RoleRepository interface.
// It allows flexible testing by injecting custom behavior for each method.
//
// Example usage:
//
//	mock := &MockRoleRepository{
//		CreateFunc: func(ctx context.Context, role domain.Role) error {
//			return nil
//		},
//		FindByIDFunc: func(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
//			return &domain.Role{ID: id, Slug: "developer", Name: "Developer"}, nil
//		},
//	}
type MockRoleRepository struct {
	CreateFunc     func(ctx context.Context, role domain.Role) error
	FindByIDFunc   func(ctx context.Context, id domain.RoleID) (*domain.Role, error)
	FindBySlugFunc func(ctx context.Context, slug string) (*domain.Role, error)
	ListFunc       func(ctx context.Context) ([]domain.Role, error)
	UpdateFunc     func(ctx context.Context, role domain.Role) error
	DeleteFunc     func(ctx context.Context, id domain.RoleID) error
	IsInUseFunc    func(ctx context.Context, slug string) (bool, error)
}

func (m *MockRoleRepository) Create(ctx context.Context, role domain.Role) error {
	if m.CreateFunc == nil {
		panic("called not defined CreateFunc")
	}
	return m.CreateFunc(ctx, role)
}

func (m *MockRoleRepository) FindByID(ctx context.Context, id domain.RoleID) (*domain.Role, error) {
	if m.FindByIDFunc == nil {
		panic("called not defined FindByIDFunc")
	}
	return m.FindByIDFunc(ctx, id)
}

func (m *MockRoleRepository) FindBySlug(ctx context.Context, slug string) (*domain.Role, error) {
	if m.FindBySlugFunc == nil {
		panic("called not defined FindBySlugFunc")
	}
	return m.FindBySlugFunc(ctx, slug)
}

func (m *MockRoleRepository) List(ctx context.Context) ([]domain.Role, error) {
	if m.ListFunc == nil {
		panic("called not defined ListFunc")
	}
	return m.ListFunc(ctx)
}

func (m *MockRoleRepository) Update(ctx context.Context, role domain.Role) error {
	if m.UpdateFunc == nil {
		panic("called not defined UpdateFunc")
	}
	return m.UpdateFunc(ctx, role)
}

func (m *MockRoleRepository) Delete(ctx context.Context, id domain.RoleID) error {
	if m.DeleteFunc == nil {
		panic("called not defined DeleteFunc")
	}
	return m.DeleteFunc(ctx, id)
}

func (m *MockRoleRepository) IsInUse(ctx context.Context, slug string) (bool, error) {
	if m.IsInUseFunc == nil {
		panic("called not defined IsInUseFunc")
	}
	return m.IsInUseFunc(ctx, slug)
}

// RolesContractTesting runs all contract tests for a RoleRepository implementation.
// Use this function to verify that your implementation adheres to the RoleRepository contract.
//
// Parameters:
//   - t: testing.T instance
//   - repo: the RoleRepository implementation to test
//
// Example usage in implementation tests:
//
//	func TestSQLiteRoleRepository(t *testing.T) {
//		repo := setupTestRepo(t)
//		defer cleanupTestRepo(t, repo)
//		rolestest.RolesContractTesting(t, repo)
//	}
func RolesContractTesting(t *testing.T, repo roles.RoleRepository) {
	ctx := context.Background()

	t.Run("Contract: Create stores role and FindByID retrieves it", func(t *testing.T) {
		role := domain.Role{
			ID:          domain.NewRoleID(),
			Slug:        "test-developer",
			Name:        "Test Developer",
			Icon:        "💻",
			Color:       "#3B82F6",
			Description: "A test developer role",
			TechStack:   []string{"Go", "React"},
			PromptHint:  "Focus on clean code",
			SortOrder:   1,
			CreatedAt:   time.Now(),
		}

		err := repo.Create(ctx, role)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindByID(ctx, role.ID)
		require.NoError(t, err, "FindByID should succeed")
		require.NotNil(t, retrieved, "Retrieved role must not be nil")
		assert.Equal(t, role.ID, retrieved.ID)
		assert.Equal(t, role.Slug, retrieved.Slug)
		assert.Equal(t, role.Name, retrieved.Name)
		assert.Equal(t, role.Icon, retrieved.Icon)
		assert.Equal(t, role.Color, retrieved.Color)
		assert.Equal(t, role.Description, retrieved.Description)
		assert.Equal(t, role.TechStack, retrieved.TechStack)
		assert.Equal(t, role.PromptHint, retrieved.PromptHint)
		assert.Equal(t, role.SortOrder, retrieved.SortOrder)
	})

	t.Run("Contract: FindByID returns error for non-existent role", func(t *testing.T) {
		nonExistentID := domain.NewRoleID()
		_, err := repo.FindByID(ctx, nonExistentID)
		assert.Error(t, err, "FindByID should return error for non-existent role")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrRoleNotFound)
	})

	t.Run("Contract: Create stores role and FindBySlug retrieves it", func(t *testing.T) {
		role := domain.Role{
			ID:        domain.NewRoleID(),
			Slug:      "test-designer",
			Name:      "Test Designer",
			SortOrder: 2,
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, role)
		require.NoError(t, err, "Create should succeed")

		retrieved, err := repo.FindBySlug(ctx, role.Slug)
		require.NoError(t, err, "FindBySlug should succeed")
		require.NotNil(t, retrieved, "Retrieved role must not be nil")
		assert.Equal(t, role.ID, retrieved.ID)
		assert.Equal(t, role.Slug, retrieved.Slug)
		assert.Equal(t, role.Name, retrieved.Name)
	})

	t.Run("Contract: FindBySlug returns error for non-existent slug", func(t *testing.T) {
		_, err := repo.FindBySlug(ctx, "non-existent-slug")
		assert.Error(t, err, "FindBySlug should return error for non-existent slug")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrRoleNotFound)
	})

	t.Run("Contract: Create returns error for duplicate slug", func(t *testing.T) {
		role1 := domain.Role{
			ID:        domain.NewRoleID(),
			Slug:      "duplicate-slug",
			Name:      "First Role",
			SortOrder: 1,
			CreatedAt: time.Now(),
		}

		role2 := domain.Role{
			ID:        domain.NewRoleID(),
			Slug:      "duplicate-slug",
			Name:      "Second Role",
			SortOrder: 2,
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, role1)
		require.NoError(t, err, "First Create should succeed")

		err = repo.Create(ctx, role2)
		assert.Error(t, err, "Second Create with duplicate slug should fail")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrRoleAlreadyExists)
	})

	t.Run("Contract: List returns all roles ordered by sort_order", func(t *testing.T) {
		// Create multiple roles
		roles := []domain.Role{
			{
				ID:        domain.NewRoleID(),
				Slug:      "list-role-3",
				Name:      "List Role 3",
				SortOrder: 3,
				CreatedAt: time.Now(),
			},
			{
				ID:        domain.NewRoleID(),
				Slug:      "list-role-1",
				Name:      "List Role 1",
				SortOrder: 1,
				CreatedAt: time.Now(),
			},
			{
				ID:        domain.NewRoleID(),
				Slug:      "list-role-2",
				Name:      "List Role 2",
				SortOrder: 2,
				CreatedAt: time.Now(),
			},
		}

		for _, role := range roles {
			err := repo.Create(ctx, role)
			require.NoError(t, err, "Create should succeed")
		}

		retrieved, err := repo.List(ctx)
		require.NoError(t, err, "List should succeed")
		require.NotEmpty(t, retrieved, "List should return roles")

		// Find our test roles in the list
		var testRoles []domain.Role
		for _, r := range retrieved {
			if r.Slug == "list-role-1" || r.Slug == "list-role-2" || r.Slug == "list-role-3" {
				testRoles = append(testRoles, r)
			}
		}

		require.Len(t, testRoles, 3, "Should find all test roles")

		// Verify ordering by sort_order
		assert.Equal(t, "list-role-1", testRoles[0].Slug, "First role should be list-role-1")
		assert.Equal(t, "list-role-2", testRoles[1].Slug, "Second role should be list-role-2")
		assert.Equal(t, "list-role-3", testRoles[2].Slug, "Third role should be list-role-3")
	})

	t.Run("Contract: Update modifies role data", func(t *testing.T) {
		role := domain.Role{
			ID:          domain.NewRoleID(),
			Slug:        "update-role",
			Name:        "Update Role",
			Icon:        "📝",
			Color:       "#EF4444",
			Description: "Original description",
			SortOrder:   1,
			CreatedAt:   time.Now(),
		}

		err := repo.Create(ctx, role)
		require.NoError(t, err, "Create should succeed")

		// Update the role
		role.Name = "Updated Role"
		role.Description = "Updated description"
		role.Icon = "✏️"
		role.Color = "#10B981"

		err = repo.Update(ctx, role)
		require.NoError(t, err, "Update should succeed")

		retrieved, err := repo.FindByID(ctx, role.ID)
		require.NoError(t, err, "FindByID should succeed")
		assert.Equal(t, "Updated Role", retrieved.Name)
		assert.Equal(t, "Updated description", retrieved.Description)
		assert.Equal(t, "✏️", retrieved.Icon)
		assert.Equal(t, "#10B981", retrieved.Color)
		assert.Equal(t, "update-role", retrieved.Slug, "Slug should remain unchanged")
	})

	t.Run("Contract: Update returns error for non-existent role", func(t *testing.T) {
		nonExistentRole := domain.Role{
			ID:        domain.NewRoleID(),
			Slug:      "non-existent",
			Name:      "Non-existent",
			CreatedAt: time.Now(),
		}

		err := repo.Update(ctx, nonExistentRole)
		assert.Error(t, err, "Update should return error for non-existent role")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrRoleNotFound)
	})

	t.Run("Contract: Delete removes role", func(t *testing.T) {
		role := domain.Role{
			ID:        domain.NewRoleID(),
			Slug:      "delete-role",
			Name:      "Delete Role",
			SortOrder: 1,
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, role)
		require.NoError(t, err, "Create should succeed")

		err = repo.Delete(ctx, role.ID)
		require.NoError(t, err, "Delete should succeed")

		_, err = repo.FindByID(ctx, role.ID)
		assert.Error(t, err, "FindByID should return error for deleted role")
		assert.ErrorIs(t, err, domain.ErrRoleNotFound)
	})

	t.Run("Contract: Delete returns error for non-existent role", func(t *testing.T) {
		nonExistentID := domain.NewRoleID()
		err := repo.Delete(ctx, nonExistentID)
		assert.Error(t, err, "Delete should return error for non-existent role")
		assert.True(t, domain.IsDomainError(err), "Error should be a domain error")
		assert.ErrorIs(t, err, domain.ErrRoleNotFound)
	})

	t.Run("Contract: IsInUse returns false for unused role", func(t *testing.T) {
		role := domain.Role{
			ID:        domain.NewRoleID(),
			Slug:      "unused-role",
			Name:      "Unused Role",
			SortOrder: 1,
			CreatedAt: time.Now(),
		}

		err := repo.Create(ctx, role)
		require.NoError(t, err, "Create should succeed")

		inUse, err := repo.IsInUse(ctx, role.Slug)
		require.NoError(t, err, "IsInUse should succeed")
		assert.False(t, inUse, "Unused role should return false")
	})
}
