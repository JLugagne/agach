package app

import (
	"context"
	"errors"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

func (a *App) GetWIPSlots(ctx context.Context, projectID domain.ProjectID) (*domain.WIPSlotsInfo, error) {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	column, err := a.columns.FindBySlug(ctx, projectID, domain.ColumnInProgress)
	if err != nil {
		logger.WithError(err).Error("failed to find in_progress column")
		return nil, err
	}

	count, err := a.tasks.CountByColumn(ctx, projectID, column.ID)
	if err != nil {
		logger.WithError(err).Error("failed to count in_progress tasks")
		return nil, err
	}

	freeSlots := -1
	if column.WIPLimit > 0 {
		freeSlots = column.WIPLimit - count
		if freeSlots < 0 {
			freeSlots = 0
		}
	}

	return &domain.WIPSlotsInfo{
		WIPLimit:   column.WIPLimit,
		InProgress: count,
		FreeSlots:  freeSlots,
	}, nil
}
