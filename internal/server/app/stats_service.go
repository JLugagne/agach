package app

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/toolusage"
	"github.com/sirupsen/logrus"
)

type StatsService struct {
	toolUsage toolusage.ToolUsageRepository
	tasks     tasks.TaskRepository
	projects  projects.ProjectRepository
	logger    *logrus.Logger
}

func newStatsService(
	toolUsage toolusage.ToolUsageRepository,
	tasks tasks.TaskRepository,
	projects projects.ProjectRepository,
	logger *logrus.Logger,
) *StatsService {
	return &StatsService{
		toolUsage: toolUsage,
		tasks:     tasks,
		projects:  projects,
		logger:    logger,
	}
}

func (s *StatsService) IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID).WithField("toolName", toolName)

	if s.toolUsage == nil {
		return nil
	}

	rootID, err := s.resolveRootProjectID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Warn("failed to resolve root project for tool usage tracking")
		return nil
	}

	if err := s.toolUsage.IncrementToolUsage(ctx, rootID, toolName); err != nil {
		logger.WithError(err).Warn("failed to increment tool usage")
		return nil
	}

	return nil
}

func (s *StatsService) GetToolUsageForProject(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	if s.toolUsage == nil {
		return []domain.ToolUsageStat{}, nil
	}

	stats, err := s.toolUsage.ListToolUsage(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to get tool usage stats")
		return nil, err
	}

	return stats, nil
}

func (s *StatsService) GetTimeline(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID).WithField("days", days)

	entries, err := s.tasks.GetTimeline(ctx, projectID, days)
	if err != nil {
		logger.WithError(err).Error("failed to get timeline")
		return nil, err
	}

	logger.Info("timeline retrieved successfully")
	return entries, nil
}

func (s *StatsService) GetColdStartStats(ctx context.Context, projectID domain.ProjectID) ([]domain.AgentColdStartStat, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	rootID, err := s.resolveRootProjectID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to resolve root project ID")
		return nil, err
	}

	stats, err := s.tasks.GetColdStartStats(ctx, rootID)
	if err != nil {
		logger.WithError(err).Error("failed to get cold start stats")
		return nil, err
	}

	return stats, nil
}

func (s *StatsService) GetModelTokenStats(ctx context.Context, projectID domain.ProjectID) ([]domain.ModelTokenStat, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	rootID, err := s.resolveRootProjectID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to resolve root project ID")
		return nil, err
	}

	stats, err := s.tasks.GetModelTokenStats(ctx, rootID)
	if err != nil {
		logger.WithError(err).Error("failed to get model token stats")
		return nil, err
	}

	return stats, nil
}

func (s *StatsService) ListModelPricing(ctx context.Context) ([]domain.ModelPricing, error) {
	return s.projects.ListModelPricing(ctx)
}

func (s *StatsService) resolveRootProjectID(ctx context.Context, projectID domain.ProjectID) (domain.ProjectID, error) {
	current := projectID
	for {
		project, err := s.projects.FindByID(ctx, current)
		if err != nil {
			return "", err
		}
		if project.ParentID == nil {
			return current, nil
		}
		current = *project.ParentID
	}
}
