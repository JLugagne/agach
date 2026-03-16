package converters

import (
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// ToDomainPriority converts string to domain.Priority
func ToDomainPriority(priority string) domain.Priority {
	if priority == "" {
		return domain.PriorityMedium
	}
	return domain.Priority(priority)
}

// ToDomainTaskIDs converts []string to []domain.TaskID
func ToDomainTaskIDs(ids []string) []domain.TaskID {
	result := make([]domain.TaskID, len(ids))
	for i, id := range ids {
		result[i] = domain.TaskID(id)
	}
	return result
}

// ToPublicTask converts domain.Task to pkgkanban.TaskResponse
func ToPublicTask(task domain.Task) pkgkanban.TaskResponse {
	return pkgkanban.TaskResponse{
		ID:                string(task.ID),
		ColumnID:          string(task.ColumnID),
		Title:             task.Title,
		Summary:           task.Summary,
		Description:       task.Description,
		Priority:          string(task.Priority),
		PriorityScore:     task.PriorityScore,
		Position:          task.Position,
		CreatedByRole:     task.CreatedByRole,
		CreatedByAgent:    task.CreatedByAgent,
		AssignedRole:      task.AssignedRole,
		IsBlocked:         task.IsBlocked,
		BlockedReason:     task.BlockedReason,
		BlockedAt:         task.BlockedAt,
		BlockedByAgent:    task.BlockedByAgent,
		WontDoRequested:   task.WontDoRequested,
		WontDoReason:      task.WontDoReason,
		WontDoRequestedBy: task.WontDoRequestedBy,
		WontDoRequestedAt: task.WontDoRequestedAt,
		CompletionSummary: task.CompletionSummary,
		CompletedByAgent:  task.CompletedByAgent,
		CompletedAt:       task.CompletedAt,
		FilesModified:     task.FilesModified,
		Resolution:        task.Resolution,
		ContextFiles:      task.ContextFiles,
		Tags:              task.Tags,
		EstimatedEffort:   task.EstimatedEffort,
		InputTokens:       task.InputTokens,
		OutputTokens:      task.OutputTokens,
		CacheReadTokens:   task.CacheReadTokens,
		CacheWriteTokens:  task.CacheWriteTokens,
		Model:             task.Model,
		SeenAt:            task.SeenAt,
		CreatedAt:         task.CreatedAt,
		UpdatedAt:         task.UpdatedAt,
	}
}

// ToPublicTaskWithDetails converts domain.TaskWithDetails to pkgkanban.TaskWithDetailsResponse
func ToPublicTaskWithDetails(task domain.TaskWithDetails) pkgkanban.TaskWithDetailsResponse {
	return pkgkanban.TaskWithDetailsResponse{
		TaskResponse:      ToPublicTask(task.Task),
		HasUnresolvedDeps: task.HasUnresolvedDeps,
		CommentCount:      task.CommentCount,
	}
}

// ToPublicTasksWithDetails converts []domain.TaskWithDetails to []pkgkanban.TaskWithDetailsResponse
func ToPublicTasksWithDetails(tasks []domain.TaskWithDetails) []pkgkanban.TaskWithDetailsResponse {
	result := make([]pkgkanban.TaskWithDetailsResponse, len(tasks))
	for i, t := range tasks {
		result[i] = ToPublicTaskWithDetails(t)
	}
	return result
}

// ToPublicDependencyContext converts domain.DependencyContext to pkgkanban.DependencyContextResponse
func ToPublicDependencyContext(ctx domain.DependencyContext) pkgkanban.DependencyContextResponse {
	return pkgkanban.DependencyContextResponse{
		TaskID:            string(ctx.TaskID),
		Title:             ctx.Title,
		CompletionSummary: ctx.CompletionSummary,
		FilesModified:     ctx.FilesModified,
	}
}

// ToPublicDependencyContexts converts []domain.DependencyContext to []pkgkanban.DependencyContextResponse
func ToPublicDependencyContexts(contexts []domain.DependencyContext) []pkgkanban.DependencyContextResponse {
	result := make([]pkgkanban.DependencyContextResponse, len(contexts))
	for i, c := range contexts {
		result[i] = ToPublicDependencyContext(c)
	}
	return result
}
