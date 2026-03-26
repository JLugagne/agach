package app_test

// Additional coverage tests for uncovered branches in task_move_project.go
// and roles.go error paths.

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// projectsAreRelated — uncovered relationship directions
// ─────────────────────────────────────────────────────────────────────────────

// TestApp_MoveTaskToProject_BIsParentOfA covers: a.ParentID == b.ID
func TestApp_MoveTaskToProject_BIsParentOfA_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, _, _ := setupTestApp()

	sourceProjectID := domain.NewProjectID()
	targetProjectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()
	targetColumnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(_ context.Context, pid domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return &domain.Task{ID: tid, ColumnID: columnID, Title: "Task", Summary: "S"}, nil
	}

	// sourceProject has targetProject as parent (b is parent of a)
	mockProjects.FindByIDFunc = func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
		switch id {
		case sourceProjectID:
			return &domain.Project{ID: sourceProjectID, Name: "Source", ParentID: &targetProjectID}, nil
		case targetProjectID:
			return &domain.Project{ID: targetProjectID, Name: "Target", ParentID: nil}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, _ domain.ColumnSlug) (*domain.Column, error) {
		return &domain.Column{ID: targetColumnID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
	}

	mockTasks.CreateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Task) error { return nil }
	mockTasks.DeleteFunc = func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) error { return nil }

	err := a.MoveTaskToProject(ctx, sourceProjectID, taskID, targetProjectID)
	require.NoError(t, err)
}

// TestApp_MoveTaskToProject_SiblingProjects covers: shared parent ID
func TestApp_MoveTaskToProject_SiblingProjects_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, _, _ := setupTestApp()

	parentID := domain.NewProjectID()
	sourceProjectID := domain.NewProjectID()
	targetProjectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()
	targetColumnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return &domain.Task{ID: tid, ColumnID: columnID, Title: "Task", Summary: "S"}, nil
	}

	// Both share the same parent — siblings
	mockProjects.FindByIDFunc = func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
		switch id {
		case sourceProjectID:
			return &domain.Project{ID: sourceProjectID, Name: "Source", ParentID: &parentID}, nil
		case targetProjectID:
			return &domain.Project{ID: targetProjectID, Name: "Target", ParentID: &parentID}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, _ domain.ColumnSlug) (*domain.Column, error) {
		return &domain.Column{ID: targetColumnID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
	}

	mockTasks.CreateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Task) error { return nil }
	mockTasks.DeleteFunc = func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) error { return nil }

	err := a.MoveTaskToProject(ctx, sourceProjectID, taskID, targetProjectID)
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// MoveTaskToProject — nil project / nil column / create error / delete error
// ─────────────────────────────────────────────────────────────────────────────

func TestApp_MoveTaskToProject_NilSourceProject_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, _, _, _ := setupTestApp()

	sourceProjectID := domain.NewProjectID()
	targetProjectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return &domain.Task{ID: tid, ColumnID: columnID, Title: "Task", Summary: "S"}, nil
	}

	mockProjects.FindByIDFunc = func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == sourceProjectID {
			return nil, nil // nil, no error
		}
		return &domain.Project{ID: id}, nil
	}

	err := a.MoveTaskToProject(ctx, sourceProjectID, taskID, targetProjectID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestApp_MoveTaskToProject_NilTargetProject_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, _, _, _ := setupTestApp()

	sourceProjectID := domain.NewProjectID()
	targetProjectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return &domain.Task{ID: tid, ColumnID: columnID, Title: "Task", Summary: "S"}, nil
	}

	mockProjects.FindByIDFunc = func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == sourceProjectID {
			return &domain.Project{ID: sourceProjectID}, nil
		}
		return nil, nil // nil target project
	}

	err := a.MoveTaskToProject(ctx, sourceProjectID, taskID, targetProjectID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestApp_MoveTaskToProject_NilTodoColumn_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, _, _ := setupTestApp()

	sourceProjectID := domain.NewProjectID()
	targetProjectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return &domain.Task{ID: tid, ColumnID: columnID, Title: "Task", Summary: "S"}, nil
	}

	// Related projects (parent-child)
	mockProjects.FindByIDFunc = func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == sourceProjectID {
			return &domain.Project{ID: sourceProjectID, ParentID: nil}, nil
		}
		return &domain.Project{ID: targetProjectID, ParentID: &sourceProjectID}, nil
	}

	// nil column returned (no error)
	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, _ domain.ColumnSlug) (*domain.Column, error) {
		return nil, nil
	}

	err := a.MoveTaskToProject(ctx, sourceProjectID, taskID, targetProjectID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

func TestApp_MoveTaskToProject_CreateError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, _, _ := setupTestApp()

	sourceProjectID := domain.NewProjectID()
	targetProjectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()
	targetColumnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return &domain.Task{ID: tid, ColumnID: columnID, Title: "Task", Summary: "S"}, nil
	}

	mockProjects.FindByIDFunc = func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == sourceProjectID {
			return &domain.Project{ID: sourceProjectID, ParentID: nil}, nil
		}
		return &domain.Project{ID: targetProjectID, ParentID: &sourceProjectID}, nil
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, _ domain.ColumnSlug) (*domain.Column, error) {
		return &domain.Column{ID: targetColumnID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
	}

	mockTasks.CreateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
		return errors.New("create error")
	}

	err := a.MoveTaskToProject(ctx, sourceProjectID, taskID, targetProjectID)
	require.Error(t, err)
}

func TestApp_MoveTaskToProject_DeleteError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, _, _ := setupTestApp()

	sourceProjectID := domain.NewProjectID()
	targetProjectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()
	targetColumnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, tid domain.TaskID) (*domain.Task, error) {
		return &domain.Task{ID: tid, ColumnID: columnID, Title: "Task", Summary: "S"}, nil
	}

	mockProjects.FindByIDFunc = func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == sourceProjectID {
			return &domain.Project{ID: sourceProjectID, ParentID: nil}, nil
		}
		return &domain.Project{ID: targetProjectID, ParentID: &sourceProjectID}, nil
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, _ domain.ColumnSlug) (*domain.Column, error) {
		return &domain.Column{ID: targetColumnID, Slug: domain.ColumnTodo, Name: "To Do"}, nil
	}

	mockTasks.CreateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Task) error { return nil }
	mockTasks.DeleteFunc = func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) error {
		return errors.New("delete error")
	}

	err := a.MoveTaskToProject(ctx, sourceProjectID, taskID, targetProjectID)
	require.Error(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// Roles — missing nil-return and delete-error paths
// ─────────────────────────────────────────────────────────────────────────────

func TestApp_DeleteRole_NilRoleReturned_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	mockRoles.FindByIDFunc = func(_ context.Context, id domain.RoleID) (*domain.Role, error) {
		return nil, nil // nil role, no error
	}

	err := a.DeleteAgent(ctx, roleID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestApp_DeleteRole_DeleteError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	mockRoles.FindByIDFunc = func(_ context.Context, id domain.RoleID) (*domain.Role, error) {
		return &domain.Role{ID: roleID, Slug: "test", Name: "Test"}, nil
	}
	mockRoles.DeleteFunc = func(_ context.Context, id domain.RoleID) error {
		return errors.New("delete failed")
	}

	err := a.DeleteAgent(ctx, roleID)
	require.Error(t, err)
}

func TestApp_UpdateRole_NilRoleReturned_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	mockRoles.FindByIDFunc = func(_ context.Context, id domain.RoleID) (*domain.Role, error) {
		return nil, nil // nil role, no error
	}

	err := a.UpdateAgent(ctx, roleID, "New Name", "", "", "", "", "", "", "", nil, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestApp_UpdateRole_UpdateError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	mockRoles.FindByIDFunc = func(_ context.Context, id domain.RoleID) (*domain.Role, error) {
		return &domain.Role{ID: roleID, Slug: "test", Name: "Test"}, nil
	}
	mockRoles.UpdateFunc = func(_ context.Context, role domain.Role) error {
		return errors.New("update failed")
	}

	err := a.UpdateAgent(ctx, roleID, "New Name", "", "", "", "", "", "", "", nil, 0)
	require.Error(t, err)
}

func TestApp_UpdateRole_WithSortOrder_UpdatesField(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	existingRole := &domain.Role{ID: roleID, Slug: "test", Name: "Test", SortOrder: 0}
	mockRoles.FindByIDFunc = func(_ context.Context, id domain.RoleID) (*domain.Role, error) {
		return existingRole, nil
	}
	var updated domain.Role
	mockRoles.UpdateFunc = func(_ context.Context, role domain.Role) error {
		updated = role
		return nil
	}

	err := a.UpdateAgent(ctx, roleID, "", "", "", "", "", "", "", "", nil, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, updated.SortOrder)
}

func TestApp_UpdateRole_WithPromptTemplate_UpdatesField(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	existingRole := &domain.Role{ID: roleID, Slug: "test", Name: "Test"}
	mockRoles.FindByIDFunc = func(_ context.Context, id domain.RoleID) (*domain.Role, error) {
		return existingRole, nil
	}
	var updated domain.Role
	mockRoles.UpdateFunc = func(_ context.Context, role domain.Role) error {
		updated = role
		return nil
	}

	err := a.UpdateAgent(ctx, roleID, "", "", "", "", "", "my template", "", "", nil, 0)
	require.NoError(t, err)
	assert.Equal(t, "my template", updated.PromptTemplate)
}

func TestApp_ListRoles_RepositoryError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.ListFunc = func(_ context.Context) ([]domain.Role, error) {
		return nil, errors.New("db error")
	}

	_, err := a.ListAgents(ctx)
	require.Error(t, err)
}

func TestApp_CreateRole_AlreadyExists_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	existing := &domain.Role{ID: domain.NewRoleID(), Slug: "test", Name: "Test"}
	mockRoles.FindBySlugFunc = func(_ context.Context, slug string) (*domain.Role, error) {
		return existing, nil // role found — already exists
	}

	_, err := a.CreateAgent(ctx, "test", "Test Role", "", "", "", "", "", "", "", nil, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleAlreadyExists)
}

func TestApp_CreateRole_CreateError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.FindBySlugFunc = func(_ context.Context, slug string) (*domain.Role, error) {
		return nil, errors.New("not found")
	}
	mockRoles.CreateFunc = func(_ context.Context, role domain.Role) error {
		return errors.New("db error")
	}

	_, err := a.CreateAgent(ctx, "test", "Test Role", "", "", "", "", "", "", "", nil, 0)
	require.Error(t, err)
}

func TestApp_ListProjectRoles_RepositoryError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.ListInProjectFunc = func(_ context.Context, pid domain.ProjectID) ([]domain.Role, error) {
		return nil, errors.New("db error")
	}

	_, err := a.ListProjectAgents(ctx, domain.NewProjectID())
	require.Error(t, err)
}

func TestApp_DeleteProjectRole_NilRoleReturned_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.FindByIDInProjectFunc = func(_ context.Context, pid domain.ProjectID, rid domain.RoleID) (*domain.Role, error) {
		return nil, nil // nil, no error
	}

	err := a.DeleteProjectAgent(ctx, domain.NewProjectID(), domain.NewRoleID())
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestApp_DeleteProjectRole_DeleteError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	mockRoles.FindByIDInProjectFunc = func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) (*domain.Role, error) {
		return &domain.Role{ID: roleID, Slug: "test", Name: "Test"}, nil
	}
	mockRoles.DeleteInProjectFunc = func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) error {
		return errors.New("delete failed")
	}

	err := a.DeleteProjectAgent(ctx, domain.NewProjectID(), roleID)
	require.Error(t, err)
}

func TestApp_UpdateProjectRole_NilRoleReturned_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.FindByIDInProjectFunc = func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) (*domain.Role, error) {
		return nil, nil // nil, no error
	}

	err := a.UpdateProjectAgent(ctx, domain.NewProjectID(), domain.NewRoleID(), "New Name", "", "", "", "", "", "", "", nil, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestApp_UpdateProjectRole_UpdateError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	mockRoles.FindByIDInProjectFunc = func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) (*domain.Role, error) {
		return &domain.Role{ID: roleID, Slug: "test", Name: "Test"}, nil
	}
	mockRoles.UpdateInProjectFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Role) error {
		return errors.New("update error")
	}

	err := a.UpdateProjectAgent(ctx, domain.NewProjectID(), roleID, "New Name", "", "", "", "", "", "", "", nil, 0)
	require.Error(t, err)
}

func TestApp_UpdateProjectRole_WithAllFields_UpdatesAll(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	roleID := domain.NewRoleID()
	existing := &domain.Role{ID: roleID, Slug: "test", Name: "Old"}
	mockRoles.FindByIDInProjectFunc = func(_ context.Context, _ domain.ProjectID, _ domain.RoleID) (*domain.Role, error) {
		return existing, nil
	}
	var updated domain.Role
	mockRoles.UpdateInProjectFunc = func(_ context.Context, _ domain.ProjectID, role domain.Role) error {
		updated = role
		return nil
	}

	err := a.UpdateProjectAgent(ctx, domain.NewProjectID(), roleID, "New Name", "icon", "color", "desc", "hint", "template", "", "", []string{"Go"}, 3)
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "icon", updated.Icon)
	assert.Equal(t, "color", updated.Color)
	assert.Equal(t, "desc", updated.Description)
	assert.Equal(t, "hint", updated.PromptHint)
	assert.Equal(t, "template", updated.PromptTemplate)
	assert.Equal(t, []string{"Go"}, updated.TechStack)
	assert.Equal(t, 3, updated.SortOrder)
}

func TestApp_GetProjectRoleBySlug_NilRoleReturned_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.FindBySlugInProjectFunc = func(_ context.Context, _ domain.ProjectID, _ string) (*domain.Role, error) {
		return nil, nil // nil, no error
	}

	_, err := a.GetProjectAgentBySlug(ctx, domain.NewProjectID(), "test")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRoleNotFound)
}

func TestApp_CreateProjectRole_CreateError_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, mockRoles, _, _, _, _ := setupTestApp()

	mockRoles.FindBySlugInProjectFunc = func(_ context.Context, _ domain.ProjectID, _ string) (*domain.Role, error) {
		return nil, errors.New("not found")
	}
	mockRoles.CreateInProjectFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Role) error {
		return errors.New("db error")
	}

	_, err := a.CreateProjectAgent(ctx, domain.NewProjectID(), "test", "Test Role", "", "", "", "", "", "", "", nil, 0)
	require.Error(t, err)
}
