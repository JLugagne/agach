package app

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/columns"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/comments"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dependencies"
	featuresrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/features"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/sirupsen/logrus"
)

type TaskService struct {
	tasks        tasks.TaskRepository
	columns      columns.ColumnRepository
	dependencies dependencies.DependencyRepository
	features     featuresrepo.FeatureRepository
	projects     projects.ProjectRepository
	comments     comments.CommentRepository
	logger       *logrus.Logger
}

func newTaskService(
	tasks tasks.TaskRepository,
	columns columns.ColumnRepository,
	dependencies dependencies.DependencyRepository,
	features featuresrepo.FeatureRepository,
	projects projects.ProjectRepository,
	comments comments.CommentRepository,
	logger *logrus.Logger,
) *TaskService {
	return &TaskService{
		tasks:        tasks,
		columns:      columns,
		dependencies: dependencies,
		features:     features,
		projects:     projects,
		comments:     comments,
		logger:       logger,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, projectID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string, startInBacklog bool, featureID *domain.FeatureID) (domain.Task, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	if title == "" {
		return domain.Task{}, domain.ErrTaskTitleRequired
	}
	if summary == "" {
		return domain.Task{}, domain.ErrSummaryRequired
	}

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return domain.Task{}, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return domain.Task{}, domain.ErrProjectNotFound
	}

	if featureID != nil {
		featureProject, err := s.projects.FindByID(ctx, domain.ProjectID(string(*featureID)))
		if err != nil || featureProject == nil {
			return domain.Task{}, domain.ErrProjectNotFound
		}
		if featureProject.ParentID == nil || *featureProject.ParentID != projectID {
			return domain.Task{}, domain.ErrFeatureNotInProject
		}
	}

	var targetColumn *domain.Column
	if startInBacklog {
		targetColumn, err = s.columns.EnsureBacklog(ctx, projectID)
		if err != nil {
			logger.WithError(err).Error("failed to ensure backlog column")
			return domain.Task{}, errors.Join(domain.ErrColumnNotFound, err)
		}
	} else {
		targetColumn, err = s.columns.FindBySlug(ctx, projectID, domain.ColumnTodo)
		if err != nil {
			logger.WithError(err).Error("failed to find todo column")
			return domain.Task{}, errors.Join(domain.ErrColumnNotFound, err)
		}
	}
	if targetColumn == nil {
		return domain.Task{}, domain.ErrColumnNotFound
	}

	priorityScore := priority.Score()

	existingTasks, err := s.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &targetColumn.Slug})
	if err != nil {
		logger.WithError(err).Error("failed to list existing tasks")
		return domain.Task{}, err
	}
	nextPosition := len(existingTasks)

	task := domain.Task{
		ID:              domain.NewTaskID(),
		ColumnID:        targetColumn.ID,
		FeatureID:       featureID,
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

	if err := s.tasks.Create(ctx, projectID, task); err != nil {
		logger.WithError(err).Error("failed to create task")
		return domain.Task{}, err
	}

	logger.WithField("taskID", task.ID).Info("task created successfully")
	return task, nil
}

func (s *TaskService) BulkCreateTasks(ctx context.Context, projectID domain.ProjectID, inputs []service.BulkTaskInput) ([]domain.Task, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	if len(inputs) == 0 {
		return nil, nil
	}

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	backlogColumn, err := s.columns.EnsureBacklog(ctx, projectID)
	if err != nil {
		return nil, errors.Join(domain.ErrColumnNotFound, err)
	}
	if backlogColumn == nil {
		return nil, domain.ErrColumnNotFound
	}
	todoColumn, err := s.columns.FindBySlug(ctx, projectID, domain.ColumnTodo)
	if err != nil {
		return nil, errors.Join(domain.ErrColumnNotFound, err)
	}
	if todoColumn == nil {
		return nil, domain.ErrColumnNotFound
	}

	for i, input := range inputs {
		if input.Title == "" {
			return nil, fmt.Errorf("inputs[%d]: %w", i, domain.ErrTaskTitleRequired)
		}
		if input.Summary == "" {
			return nil, fmt.Errorf("inputs[%d]: %w", i, domain.ErrSummaryRequired)
		}
		if input.FeatureID != nil {
			featureProject, err := s.projects.FindByID(ctx, domain.ProjectID(string(*input.FeatureID)))
			if err != nil || featureProject == nil {
				return nil, fmt.Errorf("inputs[%d]: %w", i, domain.ErrProjectNotFound)
			}
			if featureProject.ParentID == nil || *featureProject.ParentID != projectID {
				return nil, fmt.Errorf("inputs[%d]: %w", i, domain.ErrFeatureNotInProject)
			}
		}
	}

	backlogSlug := domain.ColumnBacklog
	todoSlug := domain.ColumnTodo
	backlogTasks, err := s.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &backlogSlug})
	if err != nil {
		return nil, err
	}
	todoTasks, err := s.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &todoSlug})
	if err != nil {
		return nil, err
	}
	backlogPos := len(backlogTasks)
	todoPos := len(todoTasks)

	now := time.Now()
	domainTasks := make([]domain.Task, 0, len(inputs))
	for _, input := range inputs {
		priority := input.Priority
		if priority == "" {
			priority = domain.PriorityMedium
		}

		var columnID domain.ColumnID
		var position int
		if input.StartInBacklog {
			columnID = backlogColumn.ID
			position = backlogPos
			backlogPos++
		} else {
			columnID = todoColumn.ID
			position = todoPos
			todoPos++
		}

		domainTasks = append(domainTasks, domain.Task{
			ID:              domain.NewTaskID(),
			ColumnID:        columnID,
			FeatureID:       input.FeatureID,
			Title:           input.Title,
			Summary:         input.Summary,
			Description:     input.Description,
			Priority:        priority,
			PriorityScore:   priority.Score(),
			Position:        position,
			CreatedByRole:   input.CreatedByRole,
			CreatedByAgent:  input.CreatedByAgent,
			AssignedRole:    input.AssignedRole,
			ContextFiles:    input.ContextFiles,
			Tags:            input.Tags,
			EstimatedEffort: input.EstimatedEffort,
			CreatedAt:       now,
			UpdatedAt:       now,
		})
	}

	if err := s.tasks.BulkCreate(ctx, projectID, domainTasks); err != nil {
		logger.WithError(err).Error("failed to bulk create tasks")
		return nil, err
	}

	for i, input := range inputs {
		for _, depID := range input.DependsOn {
			dep := domain.TaskDependency{
				ID:              domain.NewDependencyID(),
				TaskID:          domainTasks[i].ID,
				DependsOnTaskID: depID,
				CreatedAt:       time.Now(),
			}
			if err := s.dependencies.Create(ctx, projectID, dep); err != nil {
				return nil, fmt.Errorf("task %s dependency %s: %w", domainTasks[i].ID, depID, err)
			}
		}
	}

	logger.WithField("count", len(domainTasks)).Info("tasks bulk created successfully")
	return domainTasks, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.FeatureID, clearFeature bool) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if title != nil {
		if *title == "" {
			return domain.ErrTaskTitleRequired
		}
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
		task.PriorityScore = priority.Score()
	}
	if contextFiles != nil {
		task.ContextFiles = *contextFiles
	}
	if tags != nil {
		task.Tags = *tags
	}
	if tokenUsage != nil {
		if tokenUsage.InputTokens < 0 || tokenUsage.OutputTokens < 0 || tokenUsage.CacheReadTokens < 0 || tokenUsage.CacheWriteTokens < 0 {
			return domain.ErrInvalidTaskData
		}
		task.InputTokens = addClamped(task.InputTokens, tokenUsage.InputTokens)
		task.OutputTokens = addClamped(task.OutputTokens, tokenUsage.OutputTokens)
		task.CacheReadTokens = addClamped(task.CacheReadTokens, tokenUsage.CacheReadTokens)
		task.CacheWriteTokens = addClamped(task.CacheWriteTokens, tokenUsage.CacheWriteTokens)
		if tokenUsage.Model != "" {
			task.Model = tokenUsage.Model
		}
		if tokenUsage.ColdStartInputTokens > 0 {
			task.ColdStartInputTokens = tokenUsage.ColdStartInputTokens
			task.ColdStartOutputTokens = tokenUsage.ColdStartOutputTokens
			task.ColdStartCacheReadTokens = tokenUsage.ColdStartCacheReadTokens
			task.ColdStartCacheWriteTokens = tokenUsage.ColdStartCacheWriteTokens
		}
	}
	if humanEstimateSeconds != nil {
		task.HumanEstimateSeconds = *humanEstimateSeconds
	}
	if clearFeature {
		task.FeatureID = nil
	} else if featureID != nil {
		task.FeatureID = featureID
	}

	task.UpdatedAt = time.Now()

	if err := s.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to update task")
		return err
	}

	logger.Info("task updated successfully")
	return nil
}

func (s *TaskService) UpdateTaskFiles(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, filesModified, contextFiles *[]string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
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

	if err := s.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to update task files")
		return err
	}

	logger.Info("task files updated successfully")
	return nil
}

func (s *TaskService) DeleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if err := s.tasks.Delete(ctx, projectID, taskID); err != nil {
		logger.WithError(err).Error("failed to delete task")
		return err
	}

	logger.Info("task deleted successfully")
	return nil
}

func (s *TaskService) MoveTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnSlug domain.ColumnSlug, nodeID string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID":        projectID,
		"taskID":           taskID,
		"targetColumnSlug": targetColumnSlug,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	currentColumn, err := s.columns.FindByID(ctx, projectID, task.ColumnID)
	if err != nil {
		logger.WithError(err).Error("failed to find current column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if currentColumn == nil {
		return domain.ErrColumnNotFound
	}

	targetColumn, err := s.columns.FindBySlug(ctx, projectID, targetColumnSlug)
	if err != nil {
		logger.WithError(err).Error("failed to find target column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if targetColumn == nil {
		return domain.ErrColumnNotFound
	}

	if currentColumn.Slug == domain.ColumnInProgress && targetColumnSlug == domain.ColumnTodo {
		resolutionNote := fmt.Sprintf("[Moved back to Todo by human on %s - task was not completed]",
			time.Now().Format("2006-01-02"))
		if task.Resolution == "" {
			task.Resolution = resolutionNote
		} else {
			task.Resolution += "\n\n" + resolutionNote
		}
	}

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

	if targetColumnSlug == domain.ColumnBlocked {
		if task.BlockedReason == "" {
			return domain.ErrBlockedReasonRequired
		}
		task.IsBlocked = true
		if task.BlockedAt == nil {
			now := time.Now()
			task.BlockedAt = &now
		}
	}

	if targetColumnSlug == domain.ColumnInProgress {
		now := time.Now()
		task.StartedAt = &now
	}

	if targetColumnSlug == domain.ColumnInProgress {
		hasUnresolved, err := s.tasks.HasUnresolvedDependencies(ctx, projectID, taskID)
		if err != nil {
			logger.WithError(err).Error("failed to check unresolved dependencies")
			return err
		}
		if hasUnresolved {
			return errors.Join(domain.ErrUnresolvedDependencies, fmt.Errorf("cannot move task to in_progress: dependencies not resolved"))
		}
	}

	targetTasks, err := s.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &targetColumnSlug})
	if err != nil {
		logger.WithError(err).Error("failed to list target column tasks")
		return err
	}

	task.ColumnID = targetColumn.ID
	task.Position = len(targetTasks)
	task.UpdatedAt = time.Now()
	if nodeID != "" {
		task.NodeID = nodeID
	}

	if err := s.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to move task")
		return err
	}

	logger.Info("task moved successfully")
	return nil
}

func (s *TaskService) StartTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, nodeID string) error {
	return s.MoveTask(ctx, projectID, taskID, domain.ColumnInProgress, nodeID)
}

func (s *TaskService) ReorderTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, newPosition int) error {
	if newPosition < 0 {
		return domain.ErrInvalidTaskData
	}

	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID":   projectID,
		"taskID":      taskID,
		"newPosition": newPosition,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if err := s.tasks.ReorderTask(ctx, projectID, taskID, newPosition); err != nil {
		logger.WithError(err).Error("failed to reorder task")
		return err
	}

	logger.Info("task reordered successfully")
	return nil
}

func (s *TaskService) CompleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage, nodeID string) error {
	if completionSummary == "" {
		return domain.ErrCompletionSummaryRequired
	}

	if tokenUsage != nil {
		if tokenUsage.InputTokens < 0 || tokenUsage.OutputTokens < 0 || tokenUsage.CacheReadTokens < 0 || tokenUsage.CacheWriteTokens < 0 {
			return domain.ErrInvalidTaskData
		}
	}

	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if task.IsBlocked {
		return domain.ErrTaskBlocked
	}

	currentColumn, _ := s.columns.FindByID(ctx, projectID, task.ColumnID)
	if currentColumn == nil {
		inProgressColumn, _ := s.columns.FindBySlug(ctx, projectID, domain.ColumnInProgress)
		if inProgressColumn != nil && task.ColumnID != inProgressColumn.ID {
			return domain.ErrInvalidTaskData
		}
	} else if currentColumn.Slug != domain.ColumnInProgress {
		return domain.ErrInvalidTaskData
	}

	doneColumn, err := s.columns.FindBySlug(ctx, projectID, domain.ColumnDone)
	if err != nil {
		logger.WithError(err).Error("failed to find done column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if doneColumn == nil {
		return domain.ErrColumnNotFound
	}

	doneTasks, err := s.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &doneColumn.Slug})
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
	if nodeID != "" {
		task.NodeID = nodeID
	}
	if task.StartedAt != nil {
		task.DurationSeconds = int(task.CompletedAt.Sub(*task.StartedAt).Seconds())
	}
	if tokenUsage != nil {
		task.InputTokens = addClamped(task.InputTokens, tokenUsage.InputTokens)
		task.OutputTokens = addClamped(task.OutputTokens, tokenUsage.OutputTokens)
		task.CacheReadTokens = addClamped(task.CacheReadTokens, tokenUsage.CacheReadTokens)
		task.CacheWriteTokens = addClamped(task.CacheWriteTokens, tokenUsage.CacheWriteTokens)
		if tokenUsage.Model != "" {
			task.Model = tokenUsage.Model
		}
	}

	if err := s.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to complete task")
		return err
	}

	logger.Info("task completed successfully")
	return nil
}

func (s *TaskService) BlockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, blockedReason, blockedByAgent, nodeID string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if blockedReason == "" {
		return domain.ErrBlockedReasonRequired
	}

	if task.IsBlocked {
		return domain.ErrTaskBlocked
	}

	blockedColumn, err := s.columns.FindBySlug(ctx, projectID, domain.ColumnBlocked)
	if err != nil {
		logger.WithError(err).Error("failed to find blocked column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if blockedColumn == nil {
		return domain.ErrColumnNotFound
	}

	blockedTasks, err := s.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &blockedColumn.Slug})
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
	if nodeID != "" {
		task.NodeID = nodeID
	}

	if err := s.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to block task")
		return err
	}

	comment := domain.Comment{
		ID:         domain.NewCommentID(),
		TaskID:     taskID,
		AuthorRole: blockedByAgent,
		AuthorType: domain.AuthorTypeAgent,
		Content:    fmt.Sprintf("Task blocked: %s", blockedReason),
		CreatedAt:  time.Now(),
	}
	if err := s.comments.Create(ctx, projectID, comment); err != nil {
		logger.WithError(err).Warn("failed to create block comment")
	}

	logger.Info("task blocked successfully")
	return nil
}

func (s *TaskService) UnblockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, nodeID string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	currentColumn, err := s.columns.FindByID(ctx, projectID, task.ColumnID)
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

	return s.MoveTask(ctx, projectID, taskID, domain.ColumnTodo, nodeID)
}

func (s *TaskService) RequestWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, wontDoReason, wontDoRequestedBy, nodeID string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if wontDoReason == "" {
		return domain.ErrWontDoReasonRequired
	}

	if task.WontDoRequested {
		return domain.ErrInvalidTaskData
	}

	blockedColumn, err := s.columns.FindBySlug(ctx, projectID, domain.ColumnBlocked)
	if err != nil {
		logger.WithError(err).Error("failed to find blocked column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if blockedColumn == nil {
		return domain.ErrColumnNotFound
	}

	blockedTasks, err := s.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &blockedColumn.Slug})
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
	if nodeID != "" {
		task.NodeID = nodeID
	}

	if err := s.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to request won't-do")
		return err
	}

	comment := domain.Comment{
		ID:         domain.NewCommentID(),
		TaskID:     taskID,
		AuthorRole: wontDoRequestedBy,
		AuthorType: domain.AuthorTypeAgent,
		Content:    fmt.Sprintf("Won't-do requested: %s", wontDoReason),
		CreatedAt:  time.Now(),
	}
	if err := s.comments.Create(ctx, projectID, comment); err != nil {
		logger.WithError(err).Warn("failed to create won't-do comment")
	}

	logger.Info("won't-do requested successfully")
	return nil
}

func (s *TaskService) ApproveWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if !task.WontDoRequested {
		return domain.ErrWontDoNotRequested
	}

	doneColumn, err := s.columns.FindBySlug(ctx, projectID, domain.ColumnDone)
	if err != nil {
		logger.WithError(err).Error("failed to find done column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if doneColumn == nil {
		return domain.ErrColumnNotFound
	}

	doneTasks, err := s.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &doneColumn.Slug})
	if err != nil {
		logger.WithError(err).Error("failed to list done tasks")
		return err
	}

	now := time.Now()
	task.ColumnID = doneColumn.ID
	task.Position = len(doneTasks)
	task.IsBlocked = false
	task.BlockedReason = ""
	task.BlockedAt = nil
	task.BlockedByAgent = ""
	task.CompletedAt = &now
	task.UpdatedAt = now
	task.CompletionSummary = "Won't do (approved): " + task.WontDoReason

	if err := s.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to approve won't-do")
		return err
	}

	comment := domain.Comment{
		ID:         domain.NewCommentID(),
		TaskID:     taskID,
		AuthorRole: "human",
		AuthorType: domain.AuthorTypeHuman,
		Content:    "Won't-do approved: task moved to done with won't-do state",
		CreatedAt:  time.Now(),
	}
	if err := s.comments.Create(ctx, projectID, comment); err != nil {
		logger.WithError(err).Warn("failed to create approval comment")
	}

	logger.Info("won't-do approved and task moved to done successfully")
	return nil
}

func (s *TaskService) RejectWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, reason string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if !task.WontDoRequested {
		return domain.ErrWontDoNotRequested
	}

	currentColumn, _ := s.columns.FindByID(ctx, projectID, task.ColumnID)
	if currentColumn == nil {
		blockedColumn, _ := s.columns.FindBySlug(ctx, projectID, domain.ColumnBlocked)
		if blockedColumn != nil && task.ColumnID != blockedColumn.ID {
			return domain.ErrTaskNotInBlocked
		}
	} else if currentColumn.Slug != domain.ColumnBlocked {
		return domain.ErrTaskNotInBlocked
	}

	task.WontDoRequested = false
	task.WontDoReason = ""
	task.WontDoRequestedBy = ""
	task.WontDoRequestedAt = nil

	todoColumn, err := s.columns.FindBySlug(ctx, projectID, domain.ColumnTodo)
	if err != nil {
		logger.WithError(err).Error("failed to find todo column")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if todoColumn == nil {
		return domain.ErrColumnNotFound
	}

	todoTasks, err := s.tasks.List(ctx, projectID, tasks.TaskFilters{ColumnSlug: &todoColumn.Slug})
	if err != nil {
		logger.WithError(err).Error("failed to list todo tasks")
		return err
	}

	task.ColumnID = todoColumn.ID
	task.Position = len(todoTasks)
	task.IsBlocked = false
	task.BlockedReason = ""
	task.BlockedAt = nil
	task.BlockedByAgent = ""
	task.UpdatedAt = time.Now()

	if err := s.tasks.Update(ctx, projectID, *task); err != nil {
		logger.WithError(err).Error("failed to reject won't-do")
		return err
	}

	comment := domain.Comment{
		ID:         domain.NewCommentID(),
		TaskID:     taskID,
		AuthorRole: "human",
		AuthorType: domain.AuthorTypeHuman,
		Content:    fmt.Sprintf("Won't-do rejected: %s", reason),
		CreatedAt:  time.Now(),
	}
	if err := s.comments.Create(ctx, projectID, comment); err != nil {
		logger.WithError(err).Warn("failed to create rejection comment")
	}

	logger.Info("won't-do rejected successfully")
	return nil
}

func (s *TaskService) UpdateTaskSessionID(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error {
	return s.tasks.UpdateSessionID(ctx, projectID, taskID, sessionID)
}

func (s *TaskService) GetTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (*domain.Task, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to get task")
		return nil, errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return nil, domain.ErrTaskNotFound
	}

	return task, nil
}

func (s *TaskService) ListTasks(ctx context.Context, projectID domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
	logger := s.logger.WithContext(ctx).WithField("projectID", projectID)

	project, err := s.projects.FindByID(ctx, projectID)
	if err != nil {
		logger.WithError(err).Error("failed to find project")
		return nil, errors.Join(domain.ErrProjectNotFound, err)
	}
	if project == nil {
		return nil, domain.ErrProjectNotFound
	}

	taskList, err := s.tasks.List(ctx, projectID, filters)
	if err != nil {
		logger.WithError(err).Error("failed to list tasks")
		return nil, err
	}

	return taskList, nil
}

func (s *TaskService) GetNextTask(ctx context.Context, projectID domain.ProjectID, role string, featureID *domain.ProjectID) (*domain.Task, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"role":      role,
		"featureID": featureID,
	})

	projectIDs := []domain.ProjectID{projectID}
	if featureID != nil {
		tree, err := s.projects.GetTree(ctx, *featureID)
		if err != nil {
			logger.WithError(err).Error("failed to get project tree")
			return nil, errors.Join(domain.ErrProjectNotFound, err)
		}
		projectIDs = make([]domain.ProjectID, 0, len(tree))
		for _, p := range tree {
			projectIDs = append(projectIDs, p.ID)
		}
	}

	todoSlug := domain.ColumnTodo
	falseVal := false
	filters := tasks.TaskFilters{
		ColumnSlug:      &todoSlug,
		IsBlocked:       &falseVal,
		WontDoRequested: &falseVal,
	}
	if role != "" {
		filters.AssignedRole = &role
	}

	var bestTask *domain.Task
	var bestProjectID domain.ProjectID

	for _, pid := range projectIDs {
		taskList, err := s.tasks.List(ctx, pid, filters)
		if err != nil {
			logger.WithError(err).WithField("searchProjectID", pid).Error("failed to list tasks")
			return nil, err
		}

		for i := range taskList {
			task := &taskList[i].Task

			if role == "" {
				if task.AssignedRole != "" {
					continue
				}
			} else {
				if task.AssignedRole != role && task.AssignedRole != "" {
					continue
				}
			}

			hasUnresolved, err := s.tasks.HasUnresolvedDependencies(ctx, pid, task.ID)
			if err != nil {
				logger.WithError(err).WithField("taskID", task.ID).Error("failed to check dependencies")
				return nil, err
			}
			if hasUnresolved {
				continue
			}

			if bestTask == nil ||
				task.PriorityScore > bestTask.PriorityScore ||
				(task.PriorityScore == bestTask.PriorityScore && task.CreatedAt.Before(bestTask.CreatedAt)) {
				bestTask = task
				bestProjectID = pid
			}
			break
		}
	}

	if bestTask == nil {
		return nil, domain.ErrNoAvailableTasks
	}

	_ = bestProjectID
	return bestTask, nil
}

func (s *TaskService) GetNextTasks(ctx context.Context, projectID domain.ProjectID, role string, count int, featureID *domain.ProjectID) ([]domain.Task, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"role":      role,
		"count":     count,
		"featureID": featureID,
	})

	if count <= 0 {
		count = 1
	}

	targetProjectID := projectID
	if featureID != nil {
		targetProjectID = *featureID
	}

	results, err := s.tasks.GetNextTasks(ctx, targetProjectID, role, count)
	if err != nil {
		logger.WithError(err).Error("failed to get next tasks")
		return nil, err
	}

	return results, nil
}

func (s *TaskService) GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	depContext, err := s.dependencies.GetDependencyContext(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to get dependency context")
		return nil, err
	}

	return depContext, nil
}

func (s *TaskService) MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"projectID": projectID,
		"taskID":    taskID,
	})

	task, err := s.tasks.FindByID(ctx, projectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	if err := s.tasks.MarkTaskSeen(ctx, projectID, taskID); err != nil {
		logger.WithError(err).Error("failed to mark task as seen")
		return err
	}

	logger.Info("task marked as seen")
	return nil
}

func (s *TaskService) MoveTaskToProject(ctx context.Context, sourceProjectID domain.ProjectID, taskID domain.TaskID, targetProjectID domain.ProjectID) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{
		"sourceProjectID": sourceProjectID,
		"taskID":          taskID,
		"targetProjectID": targetProjectID,
	})

	if sourceProjectID == targetProjectID {
		return domain.ErrProjectsNotRelated
	}

	task, err := s.tasks.FindByID(ctx, sourceProjectID, taskID)
	if err != nil {
		logger.WithError(err).Error("failed to find task in source project")
		return errors.Join(domain.ErrTaskNotFound, err)
	}
	if task == nil {
		return domain.ErrTaskNotFound
	}

	sourceProject, err := s.projects.FindByID(ctx, sourceProjectID)
	if err != nil {
		logger.WithError(err).Error("failed to find source project")
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if sourceProject == nil {
		return domain.ErrProjectNotFound
	}

	targetProject, err := s.projects.FindByID(ctx, targetProjectID)
	if err != nil {
		logger.WithError(err).Error("failed to find target project")
		return errors.Join(domain.ErrProjectNotFound, err)
	}
	if targetProject == nil {
		return domain.ErrProjectNotFound
	}

	if !projectsAreRelated(sourceProject, targetProject) {
		return domain.ErrProjectsNotRelated
	}

	todoColumn, err := s.columns.FindBySlug(ctx, targetProjectID, domain.ColumnTodo)
	if err != nil {
		logger.WithError(err).Error("failed to find todo column in target project")
		return errors.Join(domain.ErrColumnNotFound, err)
	}
	if todoColumn == nil {
		return domain.ErrColumnNotFound
	}

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

	if err := s.tasks.Delete(ctx, sourceProjectID, taskID); err != nil {
		logger.WithError(err).Error("failed to delete task from source project")
		return err
	}

	if err := s.tasks.Create(ctx, targetProjectID, newTask); err != nil {
		logger.WithError(err).Error("failed to create task in target project")
		return err
	}

	logger.WithFields(map[string]interface{}{
		"newTaskID": newTask.ID,
	}).Info("task moved to project successfully")
	return nil
}

func addClamped(a, b int) int {
	if b > 0 && a > math.MaxInt-b {
		return math.MaxInt
	}
	return a + b
}

func projectsAreRelated(a, b *domain.Project) bool {
	if b.ParentID != nil && *b.ParentID == a.ID {
		return true
	}
	if a.ParentID != nil && *a.ParentID == b.ID {
		return true
	}
	if a.ParentID != nil && b.ParentID != nil && *a.ParentID == *b.ParentID {
		return true
	}
	return false
}
