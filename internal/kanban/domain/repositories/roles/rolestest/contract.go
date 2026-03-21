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
	CreateFunc                   func(ctx context.Context, role domain.Role) error
	FindByIDFunc                 func(ctx context.Context, id domain.RoleID) (*domain.Role, error)
	FindBySlugFunc               func(ctx context.Context, slug string) (*domain.Role, error)
	ListFunc                     func(ctx context.Context) ([]domain.Role, error)
	UpdateFunc                   func(ctx context.Context, role domain.Role) error
	DeleteFunc                   func(ctx context.Context, id domain.RoleID) error
	IsInUseFunc                  func(ctx context.Context, slug string) (bool, error)
	CloneFunc                    func(ctx context.Context, sourceID domain.RoleID, newSlug, newName string) (domain.Role, error)
	AssignToProjectFunc          func(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error
	RemoveFromProjectFunc        func(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error
	ListByProjectFunc            func(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error)
	IsAssignedToProjectFunc      func(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) (bool, error)
	CopyGlobalRolesToProjectFunc func(ctx context.Context, projectID domain.ProjectID) error
	CreateInProjectFunc          func(ctx context.Context, projectID domain.ProjectID, role domain.Role) error
	FindBySlugInProjectFunc      func(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Role, error)
	FindByIDInProjectFunc        func(ctx context.Context, projectID domain.ProjectID, id domain.RoleID) (*domain.Role, error)
	ListInProjectFunc            func(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error)
	UpdateInProjectFunc          func(ctx context.Context, projectID domain.ProjectID, role domain.Role) error
	DeleteInProjectFunc          func(ctx context.Context, projectID domain.ProjectID, id domain.RoleID) error
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

func (m *MockRoleRepository) Clone(ctx context.Context, sourceID domain.RoleID, newSlug, newName string) (domain.Role, error) {
	if m.CloneFunc == nil {
		panic("called not defined CloneFunc")
	}
	return m.CloneFunc(ctx, sourceID, newSlug, newName)
}

func (m *MockRoleRepository) AssignToProject(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error {
	if m.AssignToProjectFunc == nil {
		panic("called not defined AssignToProjectFunc")
	}
	return m.AssignToProjectFunc(ctx, projectID, roleID)
}

func (m *MockRoleRepository) RemoveFromProject(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error {
	if m.RemoveFromProjectFunc == nil {
		panic("called not defined RemoveFromProjectFunc")
	}
	return m.RemoveFromProjectFunc(ctx, projectID, roleID)
}

func (m *MockRoleRepository) ListByProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error) {
	if m.ListByProjectFunc == nil {
		panic("called not defined ListByProjectFunc")
	}
	return m.ListByProjectFunc(ctx, projectID)
}

func (m *MockRoleRepository) IsAssignedToProject(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) (bool, error) {
	if m.IsAssignedToProjectFunc == nil {
		panic("called not defined IsAssignedToProjectFunc")
	}
	return m.IsAssignedToProjectFunc(ctx, projectID, roleID)
}

func (m *MockRoleRepository) CopyGlobalRolesToProject(ctx context.Context, projectID domain.ProjectID) error {
	if m.CopyGlobalRolesToProjectFunc == nil {
		return nil
	}
	return m.CopyGlobalRolesToProjectFunc(ctx, projectID)
}

func (m *MockRoleRepository) CreateInProject(ctx context.Context, projectID domain.ProjectID, role domain.Role) error {
	if m.CreateInProjectFunc == nil {
		return nil
	}
	return m.CreateInProjectFunc(ctx, projectID, role)
}

func (m *MockRoleRepository) FindBySlugInProject(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Role, error) {
	if m.FindBySlugInProjectFunc == nil {
		panic("called not defined FindBySlugInProjectFunc")
	}
	return m.FindBySlugInProjectFunc(ctx, projectID, slug)
}

func (m *MockRoleRepository) FindByIDInProject(ctx context.Context, projectID domain.ProjectID, id domain.RoleID) (*domain.Role, error) {
	if m.FindByIDInProjectFunc == nil {
		panic("called not defined FindByIDInProjectFunc")
	}
	return m.FindByIDInProjectFunc(ctx, projectID, id)
}

func (m *MockRoleRepository) ListInProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error) {
	if m.ListInProjectFunc == nil {
		panic("called not defined ListInProjectFunc")
	}
	return m.ListInProjectFunc(ctx, projectID)
}

func (m *MockRoleRepository) UpdateInProject(ctx context.Context, projectID domain.ProjectID, role domain.Role) error {
	if m.UpdateInProjectFunc == nil {
		panic("called not defined UpdateInProjectFunc")
	}
	return m.UpdateInProjectFunc(ctx, projectID, role)
}

func (m *MockRoleRepository) DeleteInProject(ctx context.Context, projectID domain.ProjectID, id domain.RoleID) error {
	if m.DeleteInProjectFunc == nil {
		panic("called not defined DeleteInProjectFunc")
	}
	return m.DeleteInProjectFunc(ctx, projectID, id)
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

// MockRoleExtended exposes the new method signatures added in the agent-management feature.
// It is a standalone struct and does NOT implement the full RoleRepository interface.
type MockRoleExtended struct {
	CloneFunc               func(ctx context.Context, sourceID domain.RoleID, newSlug, newName string) (domain.Role, error)
	AssignToProjectFunc     func(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error
	RemoveFromProjectFunc   func(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error
	ListByProjectFunc       func(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error)
	IsAssignedToProjectFunc func(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) (bool, error)
}

func (m *MockRoleExtended) Clone(ctx context.Context, sourceID domain.RoleID, newSlug, newName string) (domain.Role, error) {
	if m.CloneFunc == nil {
		panic("called not defined CloneFunc")
	}
	return m.CloneFunc(ctx, sourceID, newSlug, newName)
}

func (m *MockRoleExtended) AssignToProject(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error {
	if m.AssignToProjectFunc == nil {
		panic("called not defined AssignToProjectFunc")
	}
	return m.AssignToProjectFunc(ctx, projectID, roleID)
}

func (m *MockRoleExtended) RemoveFromProject(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error {
	if m.RemoveFromProjectFunc == nil {
		panic("called not defined RemoveFromProjectFunc")
	}
	return m.RemoveFromProjectFunc(ctx, projectID, roleID)
}

func (m *MockRoleExtended) ListByProject(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error) {
	if m.ListByProjectFunc == nil {
		panic("called not defined ListByProjectFunc")
	}
	return m.ListByProjectFunc(ctx, projectID)
}

func (m *MockRoleExtended) IsAssignedToProject(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) (bool, error) {
	if m.IsAssignedToProjectFunc == nil {
		panic("called not defined IsAssignedToProjectFunc")
	}
	return m.IsAssignedToProjectFunc(ctx, projectID, roleID)
}

// RoleExtendedContractTesting runs contract tests for the new Clone/AssignToProject/
// RemoveFromProject/ListByProject/IsAssignedToProject methods of RoleRepository.
func RoleExtendedContractTesting(t *testing.T, repo roles.RoleRepository) {
	ctx := context.Background()

	seedRole := func(t *testing.T, slug, name string) domain.Role {
		t.Helper()
		role := domain.Role{
			ID:          domain.NewRoleID(),
			Slug:        slug,
			Name:        name,
			Description: "Contract test role",
			TechStack:   []string{"Go"},
			PromptHint:  "Be helpful",
			SortOrder:   0,
			CreatedAt:   time.Now(),
		}
		require.NoError(t, repo.Create(ctx, role))
		return role
	}

	t.Run("Contract: Clone creates copy with new slug", func(t *testing.T) {
		original := seedRole(t, "original-"+domain.NewRoleID().String(), "Original")

		cloned, err := repo.Clone(ctx, original.ID, "clone-1-"+domain.NewRoleID().String(), "Clone 1")
		require.NoError(t, err)
		assert.NotEqual(t, original.ID, cloned.ID)
		assert.Equal(t, "Clone 1", cloned.Name)
		assert.Equal(t, original.Description, cloned.Description)
		assert.Equal(t, original.TechStack, cloned.TechStack)

		retrieved, err := repo.FindByID(ctx, original.ID)
		require.NoError(t, err)
		assert.Equal(t, original.ID, retrieved.ID, "original must still exist")
	})

	t.Run("Contract: Clone returns ErrRoleAlreadyExists for duplicate slug", func(t *testing.T) {
		original := seedRole(t, "orig-dup-"+domain.NewRoleID().String(), "Original Dup")

		_, err := repo.Clone(ctx, original.ID, original.Slug, "")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRoleAlreadyExists)
	})

	t.Run("Contract: AssignToProject and ListByProject", func(t *testing.T) {
		role := seedRole(t, "assign-role-"+domain.NewRoleID().String(), "Assign Role")
		projectID := domain.NewProjectID()

		err := repo.AssignToProject(ctx, projectID, role.ID)
		require.NoError(t, err)

		list, err := repo.ListByProject(ctx, projectID)
		require.NoError(t, err)
		found := false
		for _, r := range list {
			if r.ID == role.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "assigned role must appear in ListByProject")

		assigned, err := repo.IsAssignedToProject(ctx, projectID, role.ID)
		require.NoError(t, err)
		assert.True(t, assigned)
	})

	t.Run("Contract: AssignToProject idempotency", func(t *testing.T) {
		role := seedRole(t, "idempotent-role-"+domain.NewRoleID().String(), "Idempotent Role")
		projectID := domain.NewProjectID()

		err := repo.AssignToProject(ctx, projectID, role.ID)
		require.NoError(t, err)

		err = repo.AssignToProject(ctx, projectID, role.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAgentAlreadyInProject)
	})

	t.Run("Contract: RemoveFromProject", func(t *testing.T) {
		role := seedRole(t, "remove-role-"+domain.NewRoleID().String(), "Remove Role")
		projectID := domain.NewProjectID()

		require.NoError(t, repo.AssignToProject(ctx, projectID, role.ID))

		err := repo.RemoveFromProject(ctx, projectID, role.ID)
		require.NoError(t, err)

		list, err := repo.ListByProject(ctx, projectID)
		require.NoError(t, err)
		for _, r := range list {
			assert.NotEqual(t, role.ID, r.ID, "removed role must not appear in ListByProject")
		}

		assigned, err := repo.IsAssignedToProject(ctx, projectID, role.ID)
		require.NoError(t, err)
		assert.False(t, assigned)
	})

	t.Run("Contract: RemoveFromProject not-assigned", func(t *testing.T) {
		role := seedRole(t, "never-assigned-"+domain.NewRoleID().String(), "Never Assigned")
		projectID := domain.NewProjectID()

		err := repo.RemoveFromProject(ctx, projectID, role.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAgentNotInProject)
	})
}
