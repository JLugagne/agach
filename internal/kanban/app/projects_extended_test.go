package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ListProjectsWithSummary Tests

func TestApp_ListProjectsWithSummary_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID1 := domain.NewProjectID()
	projectID2 := domain.NewProjectID()

	mockProjects.ListFunc = func(ctx context.Context, parentID *domain.ProjectID) ([]domain.Project, error) {
		if parentID == nil {
			return []domain.Project{
				{ID: projectID1, Name: "Project 1"},
				{ID: projectID2, Name: "Project 2"},
			}, nil
		}
		return []domain.Project{}, nil
	}

	mockProjects.GetSummaryFunc = func(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error) {
		switch id {
		case projectID1:
			return &domain.ProjectSummary{TodoCount: 1, DoneCount: 1}, nil
		case projectID2:
			return &domain.ProjectSummary{}, nil
		}
		return nil, errors.New("not found")
	}

	mockProjects.CountChildrenFunc = func(ctx context.Context, id domain.ProjectID) (int, error) {
		return 0, nil
	}

	result, err := a.ListProjectsWithSummary(ctx)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	// Project 1 has 1 todo, 1 done
	assert.Equal(t, 1, result[0].TaskSummary.TodoCount)
	assert.Equal(t, 1, result[0].TaskSummary.DoneCount)
	// Project 2 has no tasks
	assert.Equal(t, 0, result[1].TaskSummary.TodoCount)
}

func TestApp_ListProjectsWithSummary_Empty_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	mockProjects.ListFunc = func(ctx context.Context, parentID *domain.ProjectID) ([]domain.Project, error) {
		return []domain.Project{}, nil
	}

	result, err := a.ListProjectsWithSummary(ctx)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// ListSubProjectsWithSummary Tests

func TestApp_ListSubProjectsWithSummary_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	parentID := domain.NewProjectID()
	child1ID := domain.NewProjectID()
	child2ID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		switch id {
		case parentID:
			return &domain.Project{ID: parentID, Name: "Parent Project"}, nil
		case child1ID:
			return &domain.Project{ID: child1ID, Name: "Child 1", ParentID: &parentID}, nil
		case child2ID:
			return &domain.Project{ID: child2ID, Name: "Child 2", ParentID: &parentID}, nil
		}
		return nil, errors.New("not found")
	}

	mockProjects.ListFunc = func(ctx context.Context, pid *domain.ProjectID) ([]domain.Project, error) {
		if pid != nil && *pid == parentID {
			return []domain.Project{
				{ID: child1ID, Name: "Child 1", ParentID: &parentID},
				{ID: child2ID, Name: "Child 2", ParentID: &parentID},
			}, nil
		}
		return []domain.Project{}, nil
	}

	mockProjects.GetSummaryFunc = func(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error) {
		if id == child1ID {
			return &domain.ProjectSummary{TodoCount: 1}, nil
		}
		return &domain.ProjectSummary{}, nil
	}

	mockProjects.CountChildrenFunc = func(ctx context.Context, id domain.ProjectID) (int, error) {
		return 0, nil
	}

	result, err := a.ListSubProjectsWithSummary(ctx, parentID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestApp_ListSubProjectsWithSummary_ParentNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	parentID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	_, err := a.ListSubProjectsWithSummary(ctx, parentID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

// GetProjectInfo Tests

func TestApp_GetProjectInfo_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	childID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		switch id {
		case projectID:
			return &domain.Project{ID: projectID, Name: "Main Project"}, nil
		case childID:
			return &domain.Project{ID: childID, Name: "Child Project", ParentID: &projectID}, nil
		}
		return nil, errors.New("not found")
	}

	mockProjects.ListFunc = func(ctx context.Context, pid *domain.ProjectID) ([]domain.Project, error) {
		if pid != nil && *pid == projectID {
			return []domain.Project{
				{ID: childID, Name: "Child Project", ParentID: &projectID},
			}, nil
		}
		return []domain.Project{}, nil
	}

	mockProjects.GetSummaryFunc = func(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error) {
		if id == projectID {
			return &domain.ProjectSummary{TodoCount: 1}, nil
		}
		return &domain.ProjectSummary{}, nil
	}

	mockProjects.CountChildrenFunc = func(ctx context.Context, id domain.ProjectID) (int, error) {
		if id == projectID {
			return 1, nil
		}
		return 0, nil
	}

	info, err := a.GetProjectInfo(ctx, projectID)

	require.NoError(t, err)
	assert.Equal(t, projectID, info.Project.ID)
	assert.Equal(t, "Main Project", info.Project.Name)
	assert.Equal(t, 1, info.TaskSummary.TodoCount)
	assert.Len(t, info.Children, 1)
	assert.Equal(t, childID, info.Children[0].ID)
	// Breadcrumb should contain just the project itself (no parent)
	assert.Len(t, info.Breadcrumb, 1)
	assert.Equal(t, projectID, info.Breadcrumb[0].ID)
}

func TestApp_GetProjectInfo_WithParent_BuildsBreadcrumb(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	grandparentID := domain.NewProjectID()
	parentID := domain.NewProjectID()
	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		switch id {
		case grandparentID:
			return &domain.Project{ID: grandparentID, Name: "Grandparent"}, nil
		case parentID:
			return &domain.Project{ID: parentID, Name: "Parent", ParentID: &grandparentID}, nil
		case projectID:
			return &domain.Project{ID: projectID, Name: "Child", ParentID: &parentID}, nil
		}
		return nil, errors.New("not found")
	}

	mockProjects.ListFunc = func(ctx context.Context, pid *domain.ProjectID) ([]domain.Project, error) {
		return []domain.Project{}, nil
	}

	mockProjects.GetSummaryFunc = func(ctx context.Context, id domain.ProjectID) (*domain.ProjectSummary, error) {
		return &domain.ProjectSummary{}, nil
	}

	mockProjects.CountChildrenFunc = func(ctx context.Context, id domain.ProjectID) (int, error) {
		return 0, nil
	}

	info, err := a.GetProjectInfo(ctx, projectID)

	require.NoError(t, err)
	assert.Len(t, info.Breadcrumb, 3)
	assert.Equal(t, grandparentID, info.Breadcrumb[0].ID)
	assert.Equal(t, parentID, info.Breadcrumb[1].ID)
	assert.Equal(t, projectID, info.Breadcrumb[2].ID)
}

func TestApp_GetProjectInfo_ProjectNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetProjectInfo(ctx, projectID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}
