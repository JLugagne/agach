package app

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// IncrementToolUsage increments the tool usage counter.
// If the project is a sub-project, it resolves the root project and increments there.
func (a *App) IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID).WithField("toolName", toolName)

	if a.toolUsage == nil {
		return nil
	}

	// Resolve root project ID
	rootID, err := a.resolveRootProjectID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Warn("failed to resolve root project for tool usage tracking")
		return nil // non-blocking: don't fail the MCP call
	}

	if err := a.toolUsage.IncrementToolUsage(ctx, rootID, toolName); err != nil {
		logger.WithError(err).Warn("failed to increment tool usage")
		return nil // non-blocking
	}

	return nil
}

// GetToolUsageForProject returns tool usage stats for a project.
func (a *App) GetToolUsageForProject(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error) {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	if a.toolUsage == nil {
		return []domain.ToolUsageStat{}, nil
	}

	stats, err := a.toolUsage.ListToolUsage(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to get tool usage stats")
		return nil, err
	}

	return stats, nil
}

// resolveRootProjectID walks up the parent chain to find the root project.
func (a *App) resolveRootProjectID(ctx context.Context, projectID domain.ProjectID) (domain.ProjectID, error) {
	current := projectID
	for {
		project, err := a.projects.FindByID(ctx, current)
		if err != nil {
			return "", err
		}
		if project.ParentID == nil {
			return current, nil
		}
		current = *project.ParentID
	}
}
