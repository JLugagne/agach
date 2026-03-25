package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/google/uuid"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

var validPriorities = map[domain.Priority]bool{
	domain.PriorityCritical: true,
	domain.PriorityHigh:     true,
	domain.PriorityMedium:   true,
	domain.PriorityLow:      true,
}

// ToDomainPriority converts string to domain.Priority, returning PriorityMedium for invalid/empty values.
func ToDomainPriority(priority string) domain.Priority {
	if priority == "" || len(priority) > 20 {
		return domain.PriorityMedium
	}
	p := domain.Priority(priority)
	if !validPriorities[p] {
		return domain.PriorityMedium
	}
	return p
}

// ToDomainTaskIDs converts []string to []domain.TaskID, skipping non-UUID entries.
func ToDomainTaskIDs(ids []string) []domain.TaskID {
	result := make([]domain.TaskID, 0, len(ids))
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			continue
		}
		result = append(result, domain.TaskID(id))
	}
	return result
}

// ToPublicTask converts domain.Task to pkgserver.TaskResponse
func ToPublicTask(task domain.Task) pkgserver.TaskResponse {
	var featureID *string
	if task.FeatureID != nil {
		s := string(*task.FeatureID)
		featureID = &s
	}

	return pkgserver.TaskResponse{
		ID:                string(task.ID),
		ColumnID:          string(task.ColumnID),
		FeatureID:         featureID,
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
		InputTokens:       clampInt(task.InputTokens),
		OutputTokens:      clampInt(task.OutputTokens),
		CacheReadTokens:   clampInt(task.CacheReadTokens),
		CacheWriteTokens:  clampInt(task.CacheWriteTokens),
		Model:                task.Model,
		SessionID:            task.SessionID,
		SeenAt:               task.SeenAt,
		StartedAt:            task.StartedAt,
		DurationSeconds:      clampInt(task.DurationSeconds),
		HumanEstimateSeconds: task.HumanEstimateSeconds,
		CreatedAt:            task.CreatedAt,
		UpdatedAt:            task.UpdatedAt,
	}
}

// ToPublicTasks converts []domain.Task to []pkgserver.TaskResponse
func ToPublicTasks(ts []domain.Task) []pkgserver.TaskResponse {
	return MapSlice(ts, ToPublicTask)
}

// ToPublicTaskWithDetails converts domain.TaskWithDetails to pkgserver.TaskWithDetailsResponse
func ToPublicTaskWithDetails(task domain.TaskWithDetails) pkgserver.TaskWithDetailsResponse {
	return pkgserver.TaskWithDetailsResponse{
		TaskResponse:      ToPublicTask(task.Task),
		HasUnresolvedDeps: task.HasUnresolvedDeps,
		CommentCount:      task.CommentCount,
	}
}

// ToPublicTasksWithDetails converts []domain.TaskWithDetails to []pkgserver.TaskWithDetailsResponse
func ToPublicTasksWithDetails(tasks []domain.TaskWithDetails) []pkgserver.TaskWithDetailsResponse {
	return MapSlice(tasks, ToPublicTaskWithDetails)
}

// ToPublicDependencyContext converts domain.DependencyContext to pkgserver.DependencyContextResponse
func ToPublicDependencyContext(ctx domain.DependencyContext) pkgserver.DependencyContextResponse {
	return pkgserver.DependencyContextResponse{
		TaskID:            string(ctx.TaskID),
		Title:             ctx.Title,
		CompletionSummary: ctx.CompletionSummary,
		FilesModified:     ctx.FilesModified,
	}
}

// ToPublicDependencyContexts converts []domain.DependencyContext to []pkgserver.DependencyContextResponse
func ToPublicDependencyContexts(contexts []domain.DependencyContext) []pkgserver.DependencyContextResponse {
	return MapSlice(contexts, ToPublicDependencyContext)
}
