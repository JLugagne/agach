package app

import (
	"context"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dependencies"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/sirupsen/logrus"
)

type DependencyService struct {
	dependencies dependencies.DependencyRepository
	tasks        tasks.TaskRepository
	logger       *logrus.Logger
}

func newDependencyService(dependencies dependencies.DependencyRepository, tasks tasks.TaskRepository, logger *logrus.Logger) *DependencyService {
	return &DependencyService{
		dependencies: dependencies,
		tasks:        tasks,
		logger:       logger,
	}
}

func (s *DependencyService) AddDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error {
	if taskID == dependsOnTaskID {
		return domain.ErrCannotDependOnSelf
	}

	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID":       projectID,
		"taskID":          taskID,
		"dependsOnTaskID": dependsOnTaskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	dependsOnTask, err := s.tasks.FindByID(ctx, projectID, dependsOnTaskID)
	if err != nil {
		logger.WithError(err).Error("failed to find depends-on task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if dependsOnTask == nil {
		return domain.ErrTaskNotFound
	}

	wouldCycle, err := s.dependencies.WouldCreateCycle(ctx, projectID, taskID, dependsOnTaskID)
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

	if err := s.dependencies.Create(ctx, projectID, dep); err != nil {
		logger.WithError(err).Error("failed to add dependency")
		return err
	}

	logger.Info("dependency added successfully")
	return nil
}

func (s *DependencyService) RemoveDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID":       projectID,
		"taskID":          taskID,
		"dependsOnTaskID": dependsOnTaskID,
	})

	if err := s.dependencies.Delete(ctx, projectID, taskID, dependsOnTaskID); err != nil {
		logger.WithError(err).Error("failed to remove dependency")
		return err
	}

	logger.Info("dependency removed successfully")
	return nil
}

func (s *DependencyService) GetDependencyTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return nil, errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	deps, err := s.dependencies.List(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to list dependencies")
		return nil, err
	}

	result := make([]domain.Task, 0, len(deps))
	for _, dep := range deps {
		t, err := s.tasks.FindByID(ctx, projectID, dep.DependsOnTaskID)
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

func (s *DependencyService) GetDependentTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return nil, errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	deps, err := s.dependencies.ListDependents(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to list dependents")
		return nil, err
	}

	result := make([]domain.Task, 0, len(deps))
	for _, dep := range deps {
		t, err := s.tasks.FindByID(ctx, projectID, dep.TaskID)
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

func (s *DependencyService) ListDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return nil, errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	deps, err := s.dependencies.List(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to list dependencies")
		return nil, err
	}

	return deps, nil
}
