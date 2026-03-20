package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_GetWIPSlots_UnlimitedWIP(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnInProgress {
			return &domain.Column{ID: columnID, Slug: domain.ColumnInProgress, Name: "In Progress", WIPLimit: 0}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.CountByColumnFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (int, error) {
		if pid == projectID && cid == columnID {
			return 5, nil
		}
		return 0, errors.New("not found")
	}

	info, err := a.GetWIPSlots(ctx, projectID)

	require.NoError(t, err)
	assert.Equal(t, 0, info.WIPLimit)
	assert.Equal(t, 5, info.InProgress)
	assert.Equal(t, -1, info.FreeSlots)
}

func TestApp_GetWIPSlots_SlotsAvailable(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnInProgress {
			return &domain.Column{ID: columnID, Slug: domain.ColumnInProgress, Name: "In Progress", WIPLimit: 3}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.CountByColumnFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (int, error) {
		if pid == projectID && cid == columnID {
			return 1, nil
		}
		return 0, errors.New("not found")
	}

	info, err := a.GetWIPSlots(ctx, projectID)

	require.NoError(t, err)
	assert.Equal(t, 3, info.WIPLimit)
	assert.Equal(t, 1, info.InProgress)
	assert.Equal(t, 2, info.FreeSlots)
}

func TestApp_GetWIPSlots_NoFreeSlots(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if pid == projectID && slug == domain.ColumnInProgress {
			return &domain.Column{ID: columnID, Slug: domain.ColumnInProgress, Name: "In Progress", WIPLimit: 2}, nil
		}
		return nil, errors.New("not found")
	}

	mockTasks.CountByColumnFunc = func(ctx context.Context, pid domain.ProjectID, cid domain.ColumnID) (int, error) {
		if pid == projectID && cid == columnID {
			return 2, nil
		}
		return 0, errors.New("not found")
	}

	info, err := a.GetWIPSlots(ctx, projectID)

	require.NoError(t, err)
	assert.Equal(t, 2, info.WIPLimit)
	assert.Equal(t, 2, info.InProgress)
	assert.Equal(t, 0, info.FreeSlots)
}

func TestApp_GetWIPSlots_ProjectNotFound_ReturnsError(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
		return nil, errors.New("not found")
	}

	mockColumns.FindBySlugFunc = func(ctx context.Context, pid domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		return nil, errors.New("project not found")
	}

	_, err := a.GetWIPSlots(ctx, projectID)

	assert.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
	assert.ErrorIs(t, err, domain.ErrProjectNotFound)
}
