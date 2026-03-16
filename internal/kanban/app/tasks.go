package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
)

// Task Commands

func (a *App) CreateTask(ctx context.Context, projectID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string) (domain.Task, error) {
	logger := a.logger.WithContext(ctx).WithField("projectID", projectID)

	if title == "" {
		return domain.Task{}, domain.ErrTaskTitleRequired
	}
	if summary == "" {
		return domain.Task{}, domain.ErrSummaryRequired
	}

	// Verify project exists
	project, err := a.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return domain.Task{}, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return domain.Task{}, domain.ErrProjectNotFound
	}

	// Get the todo column
	todoColumn, err := a.columns.FindBySlug(ctx, projectID, domain.ColumnTodo)
	if err != nil {
		logger.WithError(err).Error("failed to find todo column")
		return domain.Task{}, errors.Join(domain.ErrColumnNotFound, err)
	}
	if todoColumn == nil {
		return domain.Task{}, domain.ErrColumnNotFound
	}

	// Calculate priority score
	priorityScore := getPriorityScore(priority)

	// Get next position in todo column
	existingTasks, err := a.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &todoColumn.Slug})
	if err != nil {
		logger.WithError(err).Error("failed to list existing tasks")
		return domain.Task{}, err
	}
	nextPosition := len(existingTasks)

	task := domain.Task{
		ID:              domain.NewTaskID(),
		ColumnID:        todoColumn.ID,
		Title:           title,
		Summary:         summary,
		Description:     description,
		Priority:        priority,
		PriorityScore:   priorityScore,
		Position:        nextPosition,
		CreatedByRole:   createdByRole,
		CreatedByAgent:  createdByAgent,
		AssignedRole:    assignedRole,
		ContextFiles:    contextFiles,
		Tags:            tags,
		EstimatedEffort: estimatedEffort,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := a.tasks.Create(ctx, projectID, task); err != nil {
		logger.WithError(err).Error("failed to create task")
		return domain.Task{}, err
	}

	logger.WithField("taskID", task.ID).Info("task created successfully")
	return task, nil
}

func (a *App) UpdateTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if title != nil {
		task.Title = *title
	}
	if description != nil {
		task.Description = *description
	}
	if assignedRole != nil {
		task.AssignedRole = *assignedRole
	}
	if estimatedEffort != nil {
		task.EstimatedEffort = *estimatedEffort
	}
	if resolution != nil {
		task.Resolution = *resolution
	}
	if priority != nil {
		task.Priority = *priority
		task.PriorityScore = getPriorityScore(*priority)
	}
	if contextFiles != nil {
		task.ContextFiles = *contextFiles
	}
	if tags != nil {
		task.Tags = *tags
	}
	if tokenUsage != nil {
		task.InputTokens += tokenUsage.InputTokens
		task.OutputTokens += tokenUsage.OutputTokens
		task.CacheReadTokens += tokenUsage.CacheReadTokens
		task.CacheWriteTokens += tokenUsage.CacheWriteTokens
		if tokenUsage.Model != "" {
			task.Model = tokenUsage.Model
		}
	}

	task.UpdatedAt = time.Now()

	if err := a.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to update task")
		return err
	}

	logger.Info("task updated successfully")
	return nil
}

func (a *App) UpdateTaskFiles(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, filesModified, contextFiles *[]string) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if filesModified != nil {
		task.FilesModified = *filesModified
	}
	if contextFiles != nil {
		task.ContextFiles = *contextFiles
	}

	task.UpdatedAt = time.Now()

	if err := a.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to update task files")
		return err
	}

	logger.Info("task files updated successfully")
	return nil
}

func (a *App) DeleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Verify task exists
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if err := a.tasks.Delete(ctx, projectID, taskID); err != nil {
		logger.WithError(err).Error("failed to delete task")
		return err
	}

	logger.Info("task deleted successfully")
	return nil
}

func (a *App) MoveTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnSlug domain.ColumnSlug) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID":        projectID,
		"taskID":           taskID,
		"targetColumnSlug": targetColumnSlug,
	})

	// Get task
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	// Get current column
	currentColumn, err := a.columns.FindByID(ctx, projectID, task.ColumnID)
	if err != nil {
		logger.WithError(err).Error("failed to find current column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if currentColumn == nil {
		return domain.ErrColumnNotFound
	}

	// Get target column
	targetColumn, err := a.columns.FindBySlug(ctx, projectID, targetColumnSlug)
	if err != nil {
		logger.WithError(err).Error("failed to find target column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if targetColumn == nil {
		return domain.ErrColumnNotFound
	}

	// If moving from in_progress back to todo, append resolution
	if currentColumn.Slug == domain.ColumnInProgress && targetColumnSlug == domain.ColumnTodo {
		resolutionNote := fmt.Sprintf("[Moved back to Todo by human on %s - task was not completed]",
			time.Now().Format("2006-01-02"))
		if task.Resolution == "" {
			task.Resolution = resolutionNote
		} else {
			task.Resolution += "\n\n" + resolutionNote
		}
	}

	// Clear blocking flags if moving out of blocked column
	if currentColumn.Slug == domain.ColumnBlocked {
		task.IsBlocked = false
		task.BlockedReason = ""
		task.BlockedAt = nil
		task.BlockedByAgent = ""
		task.WontDoRequested = false
		task.WontDoReason = ""
		task.WontDoRequestedBy = ""
		task.WontDoRequestedAt = nil
	}

	// Set blocking flags if moving to blocked column
	if targetColumnSlug == domain.ColumnBlocked {
		task.IsBlocked = true
		if task.BlockedAt == nil {
			now := time.Now()
			task.BlockedAt = &now
		}
	}

	// Check WIP limit for in_progress column
	if targetColumnSlug == domain.ColumnInProgress && targetColumn.WIPLimit > 0 {
		inProgressTasks, err := a.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &targetColumnSlug})
		if err != nil {
			logger.WithError(err).Error("failed to count in-progress tasks")
			return err
		}
		if len(inProgressTasks) >= targetColumn.WIPLimit {
			return domain.ErrWIPLimitExceeded
		}
	}

	// Get next position in target column
	targetTasks, err := a.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &targetColumnSlug})
	if err != nil {
		logger.WithError(err).Error("failed to list target column tasks")
		return err
	}

	task.ColumnID = targetColumn.ID
	task.Position = len(targetTasks)
	task.UpdatedAt = time.Now()

	if err := a.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to move task")
		return err
	}

	logger.Info("task moved successfully")
	return nil
}

func (a *App) StartTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	return a.MoveTask(ctx, projectID, taskID, domain.ColumnInProgress)
}

func (a *App) CompleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Get task
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	// Get done column
	doneColumn, err := a.columns.FindBySlug(ctx, projectID, domain.ColumnDone)
	if err != nil {
		logger.WithError(err).Error("failed to find done column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if doneColumn == nil {
		return domain.ErrColumnNotFound
	}

	// Get next position in done column
	doneTasks, err := a.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &doneColumn.Slug})
	if err != nil {
		logger.WithError(err).Error("failed to list done tasks")
		return err
	}

	now := time.Now()
	task.ColumnID = doneColumn.ID
	task.Position = len(doneTasks)
	task.CompletionSummary = completionSummary
	task.FilesModified = filesModified
	task.CompletedByAgent = completedByAgent
	task.CompletedAt = &now
	task.UpdatedAt = now
	if tokenUsage != nil {
		task.InputTokens += tokenUsage.InputTokens
		task.OutputTokens += tokenUsage.OutputTokens
		task.CacheReadTokens += tokenUsage.CacheReadTokens
		task.CacheWriteTokens += tokenUsage.CacheWriteTokens
		if tokenUsage.Model != "" {
			task.Model = tokenUsage.Model
		}
	}

	if err := a.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to complete task")
		return err
	}

	logger.Info("task completed successfully")
	return nil
}

func (a *App) BlockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, blockedReason, blockedByAgent string) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Get task
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	// Get blocked column
	blockedColumn, err := a.columns.FindBySlug(ctx, projectID, domain.ColumnBlocked)
	if err != nil {
		logger.WithError(err).Error("failed to find blocked column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if blockedColumn == nil {
		return domain.ErrColumnNotFound
	}

	// Get next position in blocked column
	blockedTasks, err := a.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &blockedColumn.Slug})
	if err != nil {
		logger.WithError(err).Error("failed to list blocked tasks")
		return err
	}

	now := time.Now()
	task.ColumnID = blockedColumn.ID
	task.Position = len(blockedTasks)
	task.IsBlocked = true
	task.BlockedReason = blockedReason
	task.BlockedByAgent = blockedByAgent
	task.BlockedAt = &now
	task.UpdatedAt = now

	if err := a.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to block task")
		return err
	}

	// Create auto-comment
	comment := fmt.Sprintf("Task blocked: %s", blockedReason)
	if _, err := a.CreateComment(ctx, projectID, taskID, blockedByAgent, "", domain.AuthorTypeAgent, comment); err != nil {
		logger.WithError(err).Warn("failed to create block comment")
	}

	logger.Info("task blocked successfully")
	return nil
}

func (a *App) UnblockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Get task
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	// Verify task is in blocked column
	currentColumn, err := a.columns.FindByID(ctx, projectID, task.ColumnID)
	if err != nil {
		logger.WithError(err).Error("failed to find current column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if currentColumn == nil {
		return domain.ErrColumnNotFound
	}

	if currentColumn.Slug != domain.ColumnBlocked {
		return domain.ErrTaskNotInBlocked
	}

	// Move to todo
	return a.MoveTask(ctx, projectID, taskID, domain.ColumnTodo)
}

func (a *App) RequestWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, wontDoReason, wontDoRequestedBy string) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Get task
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	// Get blocked column
	blockedColumn, err := a.columns.FindBySlug(ctx, projectID, domain.ColumnBlocked)
	if err != nil {
		logger.WithError(err).Error("failed to find blocked column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if blockedColumn == nil {
		return domain.ErrColumnNotFound
	}

	// Get next position in blocked column
	blockedTasks, err := a.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &blockedColumn.Slug})
	if err != nil {
		logger.WithError(err).Error("failed to list blocked tasks")
		return err
	}

	now := time.Now()
	task.ColumnID = blockedColumn.ID
	task.Position = len(blockedTasks)
	task.IsBlocked = true
	task.WontDoRequested = true
	task.WontDoReason = wontDoReason
	task.WontDoRequestedBy = wontDoRequestedBy
	task.WontDoRequestedAt = &now
	task.UpdatedAt = now

	if err := a.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to request won't-do")
		return err
	}

	// Create auto-comment
	comment := fmt.Sprintf("Won't-do requested: %s", wontDoReason)
	if _, err := a.CreateComment(ctx, projectID, taskID, wontDoRequestedBy, "", domain.AuthorTypeAgent, comment); err != nil {
		logger.WithError(err).Warn("failed to create won't-do comment")
	}

	logger.Info("won't-do requested successfully")
	return nil
}

func (a *App) ApproveWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Get task
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	// Verify task has won't-do requested
	if !task.WontDoRequested {
		return domain.ErrWontDoNotRequested
	}

	// Get done column
	doneColumn, err := a.columns.FindBySlug(ctx, projectID, domain.ColumnDone)
	if err != nil {
		logger.WithError(err).Error("failed to find done column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if doneColumn == nil {
		return domain.ErrColumnNotFound
	}

	// Get next position in done column
	doneTasks, err := a.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &doneColumn.Slug})
	if err != nil {
		logger.WithError(err).Error("failed to list done tasks")
		return err
	}

	// Move to done column, keep wont_do_requested=true as state marker
	now := time.Now()
	task.ColumnID = doneColumn.ID
	task.Position = len(doneTasks)
	task.IsBlocked = false
	task.BlockedReason = ""
	task.BlockedAt = nil
	task.BlockedByAgent = ""
	task.CompletedAt = &now
	task.UpdatedAt = now

	if err := a.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to approve won't-do")
		return err
	}

	// Create auto-comment
	comment := fmt.Sprintf("Won't-do approved: task moved to done with won't-do state")
	if _, err := a.CreateComment(ctx, projectID, taskID, "human", "", domain.AuthorTypeHuman, comment); err != nil {
		logger.WithError(err).Warn("failed to create approval comment")
	}

	logger.Info("won't-do approved and task moved to done successfully")
	return nil
}

func (a *App) RejectWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, reason string) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Get task
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	// Verify task has won't-do requested
	if !task.WontDoRequested {
		return domain.ErrWontDoNotRequested
	}

	// Clear won't-do flags
	task.WontDoRequested = false
	task.WontDoReason = ""
	task.WontDoRequestedBy = ""
	task.WontDoRequestedAt = nil

	// Get todo column
	todoColumn, err := a.columns.FindBySlug(ctx, projectID, domain.ColumnTodo)
	if err != nil {
		logger.WithError(err).Error("failed to find todo column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if todoColumn == nil {
		return domain.ErrColumnNotFound
	}

	// Get next position in todo column
	todoTasks, err := a.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &todoColumn.Slug})
	if err != nil {
		logger.WithError(err).Error("failed to list todo tasks")
		return err
	}

	// Move to todo
	task.ColumnID = todoColumn.ID
	task.Position = len(todoTasks)
	task.IsBlocked = false
	task.BlockedReason = ""
	task.BlockedAt = nil
	task.BlockedByAgent = ""
	task.UpdatedAt = time.Now()

	if err := a.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to reject won't-do")
		return err
	}

	// Create auto-comment
	comment := fmt.Sprintf("Won't-do rejected: %s", reason)
	if _, err := a.CreateComment(ctx, projectID, taskID, "human", "", domain.AuthorTypeHuman, comment); err != nil {
		logger.WithError(err).Warn("failed to create rejection comment")
	}

	logger.Info("won't-do rejected successfully")
	return nil
}

// Task Queries

func (a *App) GetTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (*domain.Task, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to get task")
		return nil, errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	return task, nil
}

func (a *App) ListTasks(ctx context.Context, projectID domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
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

	taskList, err := a.tasks.List(ctx, projectID, filters)
	if err != nil {
		logger.WithError(err).Error("failed to list tasks")
		return nil, err
	}

	return taskList, nil
}

func (a *App) GetNextTask(ctx context.Context, projectID domain.ProjectID, role string, subProjectID *domain.ProjectID) (*domain.Task, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID":    projectID,
		"role":         role,
		"subProjectID": subProjectID,
	})

	// Determine which project IDs to search
	projectIDs := []domain.ProjectID{projectID}
	if subProjectID != nil {
		// Get the full subtree of the specified sub-project
		tree, err := a.projects.GetTree(ctx, *subProjectID)
		if err != nil {
			logger.WithError(err).Error("failed to get project tree")
			return nil, errors.Join(domain.ErrProjectNotFound, err)
		}
		projectIDs = make([]domain.ProjectID, 0, len(tree))
		for _, p := range tree {
			projectIDs = append(projectIDs, p.ID)
		}
	}

	// Search across all project IDs for the highest priority ready task
	todoSlug := domain.ColumnTodo
	falseVal := false
	filters := tasks.TaskFilters{
		ColumnSlug:      &todoSlug,
		AssignedRole:    &role,
		IsBlocked:       &falseVal,
		WontDoRequested: &falseVal,
	}

	var bestTask *domain.Task
	var bestProjectID domain.ProjectID

	for _, pid := range projectIDs {
		taskList, err := a.tasks.List(ctx, pid, filters)
		if err != nil {
			logger.WithError(err).WithField("searchProjectID", pid).Error("failed to list tasks")
			return nil, err
		}

		for i := range taskList {
			task := &taskList[i].Task

			// Check if all dependencies are resolved (in done column)
			hasUnresolved, err := a.tasks.HasUnresolvedDependencies(ctx, pid, task.ID)
			if err != nil {
				logger.WithError(err).WithField("taskID", task.ID).Error("failed to check dependencies")
				return nil, err
			}
			if hasUnresolved {
				continue
			}

			// Compare with current best: higher priority score wins, then earlier created_at
			if bestTask == nil ||
				task.PriorityScore > bestTask.PriorityScore ||
				(task.PriorityScore == bestTask.PriorityScore && task.CreatedAt.Before(bestTask.CreatedAt)) {
				bestTask = task
				bestProjectID = pid
			}
			// Since tasks are already sorted by priority DESC, created_at ASC,
			// the first valid task in this project is the best for this project.
			// But we still need to compare across projects.
			break
		}
	}

	if bestTask == nil {
		return nil, domain.ErrNoAvailableTasks
	}

	_ = bestProjectID
	return bestTask, nil
}

func (a *App) GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error) {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	depContext, err := a.dependencies.GetDependencyContext(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to get dependency context")
		return nil, err
	}

	return depContext, nil
}

// MarkTaskSeen marks a task as seen (idempotent — only sets seen_at if currently NULL)
func (a *App) MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	logger := a.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	// Verify task exists
	task, err := a.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if err := a.tasks.MarkTaskSeen(ctx, projectID, taskID); err != nil {
		logger.WithError(err).Error("failed to mark task as seen")
		return err
	}

	logger.Info("task marked as seen")
	return nil
}

// Helper functions

func getPriorityScore(priority domain.Priority) int {
	switch priority {
	case domain.PriorityCritical:
		return 400
	case domain.PriorityHigh:
		return 300
	case domain.PriorityMedium:
		return 200
	case domain.PriorityLow:
		return 100
	default:
		return 200
	}
}
