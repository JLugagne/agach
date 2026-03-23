package app

import (
	"context"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// Dependency Commands

func (a *App) AddDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID":       projectID,
		"taskID":          taskID,
		"dependsOnTaskID": dependsOnTaskID,
	})

	// Verify both tasks exist
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	dependsOnTask, err := a.tasks.FindByID(ctx, projectID, dependsOnTaskID)
	if err != nil {
		logger.WithError(err).Error("failed to find depends-on task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if dependsOnTask == nil {
		return domain.ErrTaskNotFound
	}

	// Check if would create cycle
	wouldCycle, err := a.dependencies.WouldCreateCycle(ctx, projectID, taskID, dependsOnTaskID)
	if err != nil {
		logger.WithError(err).Error("failed to check for cycles")
		return err
	}
	if wouldCycle {
		return domain.ErrDependencyCycle
	}

	dep := domain.TaskDependency{
		ID:              domain.NewDependencyID(),
		TaskID:          taskID,
		DependsOnTaskID: dependsOnTaskID,
		CreatedAt:       time.Now(),
	}

	if err := a.dependencies.Create(ctx, projectID, dep); err != nil {
		logger.WithError(err).Error("failed to add dependency")
		return err
	}

	logger.Info("dependency added successfully")
	return nil
}

func (a *App) RemoveDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID":       projectID,
		"taskID":          taskID,
		"dependsOnTaskID": dependsOnTaskID,
	})

	if err := a.dependencies.Delete(ctx, projectID, taskID, dependsOnTaskID); err != nil {
		logger.WithError(err).Error("failed to remove dependency")
		return err
	}

	logger.Info("dependency removed successfully")
	return nil
}

// Dependency Queries

// GetDependencyTasks returns the task objects that this task depends on
func (a *App) GetDependencyTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Verify task exists
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return nil, errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	deps, err := a.dependencies.List(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to list dependencies")
		return nil, err
	}

	result := make([]domain.Task, 0, len(deps))
	for _, dep := range deps {
		t, err := a.tasks.FindByID(ctx, projectID, dep.DependsOnTaskID)
		if err != nil {
			logger.WithError(err).WithField("dependsOnTaskID", dep.DependsOnTaskID).Error("failed to find dependency task")
			return nil, err
		}
		if t != nil {
			result = append(result, *t)
		}
	}

	return result, nil
}

// GetDependentTasks returns the task objects that depend on this task
func (a *App) GetDependentTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Verify task exists
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return nil, errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	deps, err := a.dependencies.ListDependents(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to list dependents")
		return nil, err
	}

	result := make([]domain.Task, 0, len(deps))
	for _, dep := range deps {
		t, err := a.tasks.FindByID(ctx, projectID, dep.TaskID)
		if err != nil {
			logger.WithError(err).WithField("dependentTaskID", dep.TaskID).Error("failed to find dependent task")
			return nil, err
		}
		if t != nil {
			result = append(result, *t)
		}
	}

	return result, nil
}

func (a *App) ListDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Verify task exists
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return nil, errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	deps, err := a.dependencies.List(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to list dependencies")
		return nil, err
	}

	return deps, nil
}
