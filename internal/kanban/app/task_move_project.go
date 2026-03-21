package app

import (
	"context"
	"errors"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// MoveTaskToProject moves a task from one project to another.
// The task lands in the "todo" column of the target project.
// Comments and dependencies are NOT moved (they are cascade-deleted with the source task).
// Blocking/won't-do flags are reset so the task starts fresh in the target project.
//
// Relationship constraint: the two projects must share the same parent (siblings),
// OR one must be a direct parent/child of the other.
func (a *App) MoveTaskToProject(ctx context.Context, sourceProjectID domain.ProjectID, taskID domain.TaskID, targetProjectID domain.ProjectID) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"sourceProjectID": sourceProjectID,
		"taskID":          taskID,
		"targetProjectID": targetProjectID,
	})

	if sourceProjectID == targetProjectID {
		return domain.ErrProjectsNotRelated
	}

	// Fetch the task from the source project
	task, err := a.tasks.FindByID(ctx, sourceProjectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task in source project")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	// Verify source project exists
	sourceProject, err := a.projects.FindByID(ctx, sourceProjectID)
	if err != nil {
		logger.WithError(err).Error("failed to find source project")
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if sourceProject == nil {
		return domain.ErrProjectNotFound
	}

	// Verify target project exists
	targetProject, err := a.projects.FindByID(ctx, targetProjectID)
	if err != nil {
		logger.WithError(err).Error("failed to find target project")
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if targetProject == nil {
		return domain.ErrProjectNotFound
	}

	// Verify projects are related (siblings or direct parent-child)
	if !projectsAreRelated(sourceProject, targetProject) {
		return domain.ErrProjectsNotRelated
	}

	// Get the "todo" column in the target project
	todoColumn, err := a.columns.FindBySlug(ctx, targetProjectID, domain.ColumnTodo)
	if err != nil {
		logger.WithError(err).Error("failed to find todo column in target project")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if todoColumn == nil {
		return domain.ErrColumnNotFound
	}

	// Build the new task for the target project.
	// Reset position to 0 (will be appended at the end below), reset blocking/wont-do flags.
	now := time.Now()
	newTask := domain.Task{
		ID:               domain.NewTaskID(),
		ColumnID:         todoColumn.ID,
		Title:            task.Title,
		Summary:          task.Summary,
		Description:      task.Description,
		Priority:         task.Priority,
		PriorityScore:    task.PriorityScore,
		Position:         0,
		CreatedByRole:    task.CreatedByRole,
		CreatedByAgent:   task.CreatedByAgent,
		AssignedRole:     task.AssignedRole,
		IsBlocked:        false,
		WontDoRequested:  false,
		FilesModified:    task.FilesModified,
		Resolution:       task.Resolution,
		ContextFiles:     task.ContextFiles,
		Tags:             task.Tags,
		EstimatedEffort:  task.EstimatedEffort,
		InputTokens:      task.InputTokens,
		OutputTokens:     task.OutputTokens,
		CacheReadTokens:  task.CacheReadTokens,
		CacheWriteTokens: task.CacheWriteTokens,
		Model:            task.Model,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Delete the task from the source project first (fail fast before creating in target).
	if err := a.tasks.Delete(ctx, sourceProjectID, taskID); err != nil {
		logger.WithError(err).Error("failed to delete task from source project")
		return err
	}

	// Create the task in the target project.
	// Source task is already deleted; if create fails the task is lost — callers should
	// retry or handle the error.
	if err := a.tasks.Create(ctx, targetProjectID, newTask); err != nil {
		logger.WithError(err).Error("failed to create task in target project")
		return err
	}

	logger.WithFields(map[string]interface{}{
		"newTaskID": newTask.ID,
	}).Info("task moved to project successfully")
	return nil
}

// projectsAreRelated returns true when two projects share the same parent (siblings)
// or one is the direct parent of the other.
func projectsAreRelated(a, b *domain.Project) bool {
	// Direct parent-child: a is parent of b
	if b.ParentID != nil && *b.ParentID == a.ID {
		return true
	}
	// Direct parent-child: b is parent of a
	if a.ParentID != nil && *a.ParentID == b.ID {
		return true
	}
	// Siblings: both share the same parent
	if a.ParentID != nil && b.ParentID != nil && *a.ParentID == *b.ParentID {
		return true
	}
	return false
}
