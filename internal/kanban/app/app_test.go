package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/app"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/columns/columnstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/comments/commentstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/dependencies/dependenciestest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/projects/projectstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/roles/rolestest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks/taskstest"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestApp creates a test app with all mocked repositories
func setupTestApp() (*app.App, *projectstest.MockProjectRepository, *rolestest.MockRoleRepository, *taskstest.MockTaskRepository, *columnstest.MockColumnRepository, *commentstest.MockCommentRepository, *dependenciestest.MockDependencyRepository) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	mockProjects := &projectstest.MockProjectRepository{}
	mockRoles := &rolestest.MockRoleRepository{}
	mockTasks := &taskstest.MockTaskRepository{}
	mockColumns := &columnstest.MockColumnRepository{}
	mockComments := &commentstest.MockCommentRepository{}
	mockDependencies := &dependenciestest.MockDependencyRepository{}

	a := app.NewApp(app.Config{
		Projects:     mockProjects,
		Roles:        mockRoles,
		Tasks:        mockTasks,
		Columns:      mockColumns,
		Comments:     mockComments,
		Dependencies: mockDependencies,
		Logger:       logger,
	})

	return a, mockProjects, mockRoles, mockTasks, mockColumns, mockComments, mockDependencies
}

// Project Command Tests

func TestApp_CreateProject_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	mockProjects.CreateFunc = func(ctx context.Context, project domain.Project) error {
		return nil
	}

	project, err := a.CreateProject(ctx, "Test Project", "Description", "/tmp/test", "", "architect", "agent1", nil)

	require.NoError(t, err)
	assert.NotEmpty(t, project.ID)
	assert.Equal(t, "Test Project", project.Name)
	assert.Equal(t, "Description", project.Description)
	assert.Equal(t, "architect", project.CreatedByRole)
	assert.Equal(t, "agent1", project.CreatedByAgent)
	assert.Nil(t, project.ParentID)
}

func TestApp_CreateProject_WithParent_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	parentID := domain.NewProjectID()
	parent := &domain.Project{
		ID:   parentID,
		Name: "Parent Project",
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == parentID {
			return parent, nil
		}
		return nil, errors.New("not found")
	}

	mockProjects.CreateFunc = func(ctx context.Context, project domain.Project) error {
		return nil
	}

	project, err := a.CreateProject(ctx, "Child Project", "Description", "/tmp/child", "", "architect", "agent1", &parentID)

	require.NoError(t, err)
	assert.NotEmpty(t, project.ID)
	assert.Equal(t, "Child Project", project.Name)
	assert.NotNil(t, project.ParentID)
	assert.Equal(t, parentID, *project.ParentID)
}

func TestApp_CreateProject_EmptyName_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, _, _, _ := setupTestApp()

	_, err := a.CreateProject(ctx, "", "Description", "/tmp/test", "", "architect", "agent1", nil)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNameRequired)
}

func TestApp_CreateProject_ParentNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	parentID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	_, err := a.CreateProject(ctx, "Child Project", "Description", "/tmp/child", "", "architect", "agent1", &parentID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestApp_UpdateProject_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	oldTime := time.Now().Add(-1 * time.Hour)
	existingProject := &domain.Project{
		ID:          projectID,
		Name:        "Old Name",
		Description: "Old Description",
		UpdatedAt:   oldTime,
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return existingProject, nil
		}
		return nil, errors.New("not found")
	}

	var updatedProject domain.Project
	mockProjects.UpdateFunc = func(ctx context.Context, project domain.Project) error {
		updatedProject = project
		return nil
	}

	err := a.UpdateProject(ctx, projectID, "New Name", "New Description", nil)

	require.NoError(t, err)
	assert.Equal(t, "New Name", updatedProject.Name)
	assert.Equal(t, "New Description", updatedProject.Description)
	assert.True(t, updatedProject.UpdatedAt.After(oldTime))
}

func TestApp_UpdateProject_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	err := a.UpdateProject(ctx, projectID, "New Name", "New Description", nil)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestApp_UpdateProject_SetsDefaultRole(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	existingProject := &domain.Project{
		ID:   projectID,
		Name: "Test Project",
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return existingProject, nil
		}
		return nil, errors.New("not found")
	}

	var updatedProject domain.Project
	mockProjects.UpdateFunc = func(ctx context.Context, project domain.Project) error {
		updatedProject = project
		return nil
	}

	role := "go-developer"
	err := a.UpdateProject(ctx, projectID, "Test Project", "", &role)

	require.NoError(t, err)
	assert.Equal(t, "go-developer", updatedProject.DefaultRole)
}

func TestApp_UpdateProject_ClearsDefaultRole(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	existingProject := &domain.Project{
		ID:          projectID,
		Name:        "Test Project",
		DefaultRole: "go-developer",
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return existingProject, nil
		}
		return nil, errors.New("not found")
	}

	var updatedProject domain.Project
	mockProjects.UpdateFunc = func(ctx context.Context, project domain.Project) error {
		updatedProject = project
		return nil
	}

	role := ""
	err := a.UpdateProject(ctx, projectID, "Test Project", "", &role)

	require.NoError(t, err)
	assert.Equal(t, "", updatedProject.DefaultRole)
}

func TestApp_UpdateProject_NilDefaultRole_DoesNotChange(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	existingProject := &domain.Project{
		ID:          projectID,
		Name:        "Test Project",
		DefaultRole: "go-developer",
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return existingProject, nil
		}
		return nil, errors.New("not found")
	}

	var updatedProject domain.Project
	mockProjects.UpdateFunc = func(ctx context.Context, project domain.Project) error {
		updatedProject = project
		return nil
	}

	err := a.UpdateProject(ctx, projectID, "Test Project", "", nil)

	require.NoError(t, err)
	assert.Equal(t, "go-developer", updatedProject.DefaultRole)
}

func TestApp_DeleteProject_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	existingProject := &domain.Project{
		ID:   projectID,
		Name: "Test Project",
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return existingProject, nil
		}
		return nil, errors.New("not found")
	}

	mockProjects.DeleteFunc = func(ctx context.Context, id domain.ProjectID) ([]domain.ProjectID, error) {
		return []domain.ProjectID{id}, nil
	}

	err := a.DeleteProject(ctx, projectID)

	require.NoError(t, err)
}

func TestApp_DeleteProject_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	err := a.DeleteProject(ctx, projectID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

// Project Query Tests

func TestApp_GetProject_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	expectedProject := &domain.Project{
		ID:          projectID,
		Name:        "Test Project",
		Description: "Test Description",
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return expectedProject, nil
		}
		return nil, errors.New("not found")
	}

	project, err := a.GetProject(ctx, projectID)

	require.NoError(t, err)
	assert.Equal(t, expectedProject, project)
}

func TestApp_GetProject_NotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetProject(ctx, projectID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestApp_ListProjects_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	expectedProjects := []domain.Project{
		{ID: domain.NewProjectID(), Name: "Project 1"},
		{ID: domain.NewProjectID(), Name: "Project 2"},
	}

	mockProjects.ListFunc = func(ctx context.Context, parentID *domain.ProjectID) ([]domain.Project, error) {
		if parentID == nil {
			return expectedProjects, nil
		}
		return []domain.Project{}, nil
	}

	projects, err := a.ListProjects(ctx)

	require.NoError(t, err)
	assert.Equal(t, expectedProjects, projects)
}

func TestApp_ListSubProjects_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	parentID := domain.NewProjectID()
	parent := &domain.Project{
		ID:   parentID,
		Name: "Parent Project",
	}

	expectedSubProjects := []domain.Project{
		{ID: domain.NewProjectID(), Name: "Child 1", ParentID: &parentID},
		{ID: domain.NewProjectID(), Name: "Child 2", ParentID: &parentID},
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == parentID {
			return parent, nil
		}
		return nil, errors.New("not found")
	}

	mockProjects.ListFunc = func(ctx context.Context, pid *domain.ProjectID) ([]domain.Project, error) {
		if pid != nil && *pid == parentID {
			return expectedSubProjects, nil
		}
		return []domain.Project{}, nil
	}

	subProjects, err := a.ListSubProjects(ctx, parentID)

	require.NoError(t, err)
	assert.Equal(t, expectedSubProjects, subProjects)
}

func TestApp_ListSubProjects_ParentNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	parentID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	_, err := a.ListSubProjects(ctx, parentID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestApp_GetProjectSummary_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.GetSummaryFunc = func(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error) {
		if id == projectID {
			return &domain.ProjectSummary{
				TodoCount:       2,
				InProgressCount: 1,
				DoneCount:       1,
				BlockedCount:    1,
			}, nil
		}
		return nil, errors.New("not found")
	}

	summary, err := a.GetProjectSummary(ctx, projectID)

	require.NoError(t, err)
	assert.Equal(t, 2, summary.TodoCount)
	assert.Equal(t, 1, summary.InProgressCount)
	assert.Equal(t, 1, summary.DoneCount)
	assert.Equal(t, 1, summary.BlockedCount)
}

func TestApp_GetProjectSummary_ProjectNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.GetSummaryFunc = func(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetProjectSummary(ctx, projectID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}
