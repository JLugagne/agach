package app

import (
	"context"
	"errors"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// Column Commands

func (a *App) UpdateColumnWIPLimit(ctx context.Context, projectID domain.ProjectID, columnSlug domain.ColumnSlug, wipLimit int) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID":  projectID,
		"columnSlug": columnSlug,
		"wipLimit":   wipLimit,
	})

	if columnSlug != domain.ColumnInProgress {
		return domain.ErrInvalidColumn
	}

	if wipLimit < 0 {
		return domain.ErrInvalidColumn
	}

	column, err := a.columns.FindBySlug(ctx, projectID, columnSlug)
	if err != nil {
		logger.WithError(err).Error("failed to find column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}

	if err := a.columns.UpdateWIPLimit(ctx, projectID, column.ID, wipLimit); err != nil {
		logger.WithError(err).Error("failed to update WIP limit")
		return err
	}

	logger.Info("WIP limit updated")
	return nil
}

// Column Queries (columns are read-only, created automatically with projects)

func (a *App) GetColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (*domain.Column, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"columnID":  columnID,
	})

	// Verify project exists
	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	column, err := a.columns.FindByID(ctx, projectID, columnID)
	if err != nil {
		logger.WithError(err).Error("failed to get column")
		return nil, errors.Join(domain.ErrColumnNotFound, err)
	}
	if column == nil {
		return nil, domain.ErrColumnNotFound
	}

	return column, nil
}

func (a *App) GetColumnBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"slug":      slug,
	})

	// Verify project exists
	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	column, err := a.columns.FindBySlug(ctx, projectID, slug)
	if err != nil {
		logger.WithError(err).Error("failed to get column by slug")
		return nil, errors.Join(domain.ErrColumnNotFound, err)
	}
	if column == nil {
		return nil, domain.ErrColumnNotFound
	}

	return column, nil
}

func (a *App) ListColumns(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error) {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	// Verify project exists
	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	columns, err := a.columns.List(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to list columns")
		return nil, err
	}

	return columns, nil
}
