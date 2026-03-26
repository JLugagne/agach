package app

import (
	"context"
	"errors"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/columns"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects"
	"github.com/sirupsen/logrus"
)

type ColumnService struct {
	columns  columns.ColumnRepository
	projects projects.ProjectRepository
	logger   *logrus.Logger
}

func newColumnService(columns columns.ColumnRepository, projects projects.ProjectRepository, logger *logrus.Logger) *ColumnService {
	return &ColumnService{
		columns:  columns,
		projects: projects,
		logger:   logger,
	}
}

func (s *ColumnService) GetColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (*domain.Column, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"columnID":  columnID,
	})

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	column, err := s.columns.FindByID(ctx, projectID, columnID)
	if err != nil {
		logger.WithError(err).Error("failed to get column")
		return nil, errors.Join(domain.ErrColumnNotFound, err)
	}
	if column == nil {
		return nil, domain.ErrColumnNotFound
	}

	return column, nil
}

func (s *ColumnService) GetColumnBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"slug":      slug,
	})

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	column, err := s.columns.FindBySlug(ctx, projectID, slug)
	if err != nil {
		logger.WithError(err).Error("failed to get column by slug")
		return nil, errors.Join(domain.ErrColumnNotFound, err)
	}
	if column == nil {
		return nil, domain.ErrColumnNotFound
	}

	return column, nil
}

func (s *ColumnService) ListColumns(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	cols, err := s.columns.List(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to list columns")
		return nil, err
	}

	return cols, nil
}
