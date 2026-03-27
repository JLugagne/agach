package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Column Query Tests

func TestApp_GetColumn_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()
	expectedColumn := &domain.Column{
		ID:   columnID,
		Slug: domain.ColumnTodo,
		Name: "To Do",
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return &domain.Project{ID: projectID, Name: "Test Project"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		if pid == projectID && cid == columnID {
			return expectedColumn, nil
		}
		return nil, errors.New("not found")
	}

	column, err := a.GetColumn(ctx, projectID, columnID)

	require.NoError(t, err)
	assert.Equal(t, expectedColumn, column)
}

func TestApp_GetColumn_ProjectNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetColumn(ctx, projectID, columnID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestApp_GetColumn_ColumnNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return &domain.Project{ID: projectID, Name: "Test Project"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindByIDFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (*domain.Column, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetColumn(ctx, projectID, columnID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

func TestApp_GetColumnBySlug_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	expectedColumn := &domain.Column{
		ID:   domain.NewColumnID(),
		Slug: domain.ColumnInProgress,
		Name: "In Progress",
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return &domain.Project{ID: projectID, Name: "Test Project"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnInProgress {
			return expectedColumn, nil
		}
		return nil, errors.New("not found")
	}

	column, err := a.GetColumnBySlug(ctx, projectID, domain.ColumnInProgress)

	require.NoError(t, err)
	assert.Equal(t, expectedColumn, column)
}

func TestApp_GetColumnBySlug_ProjectNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetColumnBySlug(ctx, projectID, domain.ColumnTodo)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}

func TestApp_GetColumnBySlug_ColumnNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return &domain.Project{ID: projectID, Name: "Test Project"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		return nil, errors.New("not found")
	}

	_, err := a.GetColumnBySlug(ctx, projectID, domain.ColumnTodo)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

func TestApp_ListColumns_Success(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	expectedColumns := []domain.Column{
		{ID: domain.NewColumnID(), Slug: domain.ColumnTodo, Name: "To Do", Position: 0},
		{ID: domain.NewColumnID(), Slug: domain.ColumnInProgress, Name: "In Progress", Position: 1},
		{ID: domain.NewColumnID(), Slug: domain.ColumnDone, Name: "Done", Position: 2},
		{ID: domain.NewColumnID(), Slug: domain.ColumnBlocked, Name: "Blocked", Position: 3},
	}

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		if id == projectID {
			return &domain.Project{ID: projectID, Name: "Test Project"}, nil
		}
		return nil, errors.New("not found")
	}

	mockColumns.ListFunc = func(ctx context.Context, pid domain.ProjectID) ([]domain.Column, error) {
		if pid == projectID {
			return expectedColumns, nil
		}
		return nil, errors.New("not found")
	}

	columns, err := a.ListColumns(ctx, projectID)

	require.NoError(t, err)
	assert.Equal(t, expectedColumns, columns)
}

func TestApp_ListColumns_ProjectNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	_, err := a.ListColumns(ctx, projectID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}
