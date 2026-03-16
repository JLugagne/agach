package converters_test

import (
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/converters"
	"github.com/stretchr/testify/assert"
)

func TestToDomainPriority(t *testing.T) {
	t.Run("Empty string returns medium priority", func(t *testing.T) {
		result := converters.ToDomainPriority("")
		assert.Equal(t, domain.PriorityMedium, result)
	})

	t.Run("Critical priority converts correctly", func(t *testing.T) {
		result := converters.ToDomainPriority("critical")
		assert.Equal(t, domain.PriorityCritical, result)
	})

	t.Run("High priority converts correctly", func(t *testing.T) {
		result := converters.ToDomainPriority("high")
		assert.Equal(t, domain.PriorityHigh, result)
	})

	t.Run("Medium priority converts correctly", func(t *testing.T) {
		result := converters.ToDomainPriority("medium")
		assert.Equal(t, domain.PriorityMedium, result)
	})

	t.Run("Low priority converts correctly", func(t *testing.T) {
		result := converters.ToDomainPriority("low")
		assert.Equal(t, domain.PriorityLow, result)
	})

	t.Run("Arbitrary string is preserved as priority", func(t *testing.T) {
		result := converters.ToDomainPriority("urgent")
		assert.Equal(t, domain.Priority("urgent"), result)
	})
}

func TestToDomainTaskIDs(t *testing.T) {
	t.Run("Empty slice returns empty slice", func(t *testing.T) {
		result := converters.ToDomainTaskIDs([]string{})
		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})

	t.Run("Single ID converts correctly", func(t *testing.T) {
		result := converters.ToDomainTaskIDs([]string{"task-abc-123"})
		assert.Len(t, result, 1)
		assert.Equal(t, domain.TaskID("task-abc-123"), result[0])
	})

	t.Run("Multiple IDs convert correctly and preserve order", func(t *testing.T) {
		ids := []string{"id-1", "id-2", "id-3"}
		result := converters.ToDomainTaskIDs(ids)
		assert.Len(t, result, 3)
		assert.Equal(t, domain.TaskID("id-1"), result[0])
		assert.Equal(t, domain.TaskID("id-2"), result[1])
		assert.Equal(t, domain.TaskID("id-3"), result[2])
	})
}

func TestToPublicTask(t *testing.T) {
	t.Run("Converts minimal task correctly", func(t *testing.T) {
		now := time.Now()
		task := domain.Task{
			ID:        domain.TaskID("task-123"),
			ColumnID:  domain.ColumnID("col-456"),
			Title:     "Fix bug",
			Summary:   "Brief description",
			Priority:  domain.PriorityHigh,
			CreatedAt: now,
			UpdatedAt: now,
		}

		result := converters.ToPublicTask(task)

		assert.Equal(t, "task-123", result.ID)
		assert.Equal(t, "col-456", result.ColumnID)
		assert.Equal(t, "Fix bug", result.Title)
		assert.Equal(t, "Brief description", result.Summary)
		assert.Equal(t, "high", result.Priority)
		assert.Equal(t, now, result.CreatedAt)
		assert.Equal(t, now, result.UpdatedAt)
	})

	t.Run("Converts full task with all fields", func(t *testing.T) {
		now := time.Now()
		blockedAt := now.Add(-1 * time.Hour)
		completedAt := now.Add(-30 * time.Minute)
		wontDoAt := now.Add(-15 * time.Minute)

		task := domain.Task{
			ID:                domain.TaskID("task-full"),
			ColumnID:          domain.ColumnID("col-done"),
			Title:             "Implement feature",
			Summary:           "Summary of work",
			Description:       "Full description here",
			Priority:          domain.PriorityCritical,
			PriorityScore:     400,
			Position:          2,
			CreatedByRole:     "backend",
			CreatedByAgent:    "agent-007",
			AssignedRole:      "frontend",
			IsBlocked:         true,
			BlockedReason:     "Waiting for API",
			BlockedAt:         &blockedAt,
			BlockedByAgent:    "agent-007",
			WontDoRequested:   true,
			WontDoReason:      "Out of scope",
			WontDoRequestedBy: "agent-007",
			WontDoRequestedAt: &wontDoAt,
			CompletionSummary: "Done everything",
			CompletedByAgent:  "agent-007",
			CompletedAt:       &completedAt,
			FilesModified:     []string{"main.go", "util.go"},
			Resolution:        "Resolved by refactoring",
			ContextFiles:      []string{"context.md"},
			Tags:              []string{"backend", "api"},
			EstimatedEffort:   "M",
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		result := converters.ToPublicTask(task)

		assert.Equal(t, "task-full", result.ID)
		assert.Equal(t, "col-done", result.ColumnID)
		assert.Equal(t, "Implement feature", result.Title)
		assert.Equal(t, "Summary of work", result.Summary)
		assert.Equal(t, "Full description here", result.Description)
		assert.Equal(t, "critical", result.Priority)
		assert.Equal(t, 400, result.PriorityScore)
		assert.Equal(t, 2, result.Position)
		assert.Equal(t, "backend", result.CreatedByRole)
		assert.Equal(t, "agent-007", result.CreatedByAgent)
		assert.Equal(t, "frontend", result.AssignedRole)
		assert.True(t, result.IsBlocked)
		assert.Equal(t, "Waiting for API", result.BlockedReason)
		assert.Equal(t, &blockedAt, result.BlockedAt)
		assert.Equal(t, "agent-007", result.BlockedByAgent)
		assert.True(t, result.WontDoRequested)
		assert.Equal(t, "Out of scope", result.WontDoReason)
		assert.Equal(t, "agent-007", result.WontDoRequestedBy)
		assert.Equal(t, &wontDoAt, result.WontDoRequestedAt)
		assert.Equal(t, "Done everything", result.CompletionSummary)
		assert.Equal(t, "agent-007", result.CompletedByAgent)
		assert.Equal(t, &completedAt, result.CompletedAt)
		assert.Equal(t, []string{"main.go", "util.go"}, result.FilesModified)
		assert.Equal(t, "Resolved by refactoring", result.Resolution)
		assert.Equal(t, []string{"context.md"}, result.ContextFiles)
		assert.Equal(t, []string{"backend", "api"}, result.Tags)
		assert.Equal(t, "M", result.EstimatedEffort)
		assert.Equal(t, now, result.CreatedAt)
		assert.Equal(t, now, result.UpdatedAt)
	})

	t.Run("Nil pointer time fields remain nil", func(t *testing.T) {
		task := domain.Task{
			ID:       domain.TaskID("task-nil"),
			ColumnID: domain.ColumnID("col-1"),
			Title:    "No times",
			Summary:  "Summary",
		}

		result := converters.ToPublicTask(task)

		assert.Nil(t, result.BlockedAt)
		assert.Nil(t, result.WontDoRequestedAt)
		assert.Nil(t, result.CompletedAt)
	})

	t.Run("Slice fields are preserved correctly", func(t *testing.T) {
		task := domain.Task{
			ID:            domain.TaskID("task-slices"),
			ColumnID:      domain.ColumnID("col-1"),
			Title:         "Slices task",
			Summary:       "Summary",
			FilesModified: []string{"a.go", "b.go", "c.go"},
			ContextFiles:  []string{"spec.md"},
			Tags:          []string{"tag1", "tag2"},
		}

		result := converters.ToPublicTask(task)

		assert.Equal(t, []string{"a.go", "b.go", "c.go"}, result.FilesModified)
		assert.Equal(t, []string{"spec.md"}, result.ContextFiles)
		assert.Equal(t, []string{"tag1", "tag2"}, result.Tags)
	})
}

func TestToPublicTaskWithDetails(t *testing.T) {
	t.Run("Converts task with unresolved deps and comment count", func(t *testing.T) {
		now := time.Now()
		taskWithDetails := domain.TaskWithDetails{
			Task: domain.Task{
				ID:        domain.TaskID("task-details"),
				ColumnID:  domain.ColumnID("col-todo"),
				Title:     "Task with details",
				Summary:   "Summary",
				Priority:  domain.PriorityLow,
				CreatedAt: now,
				UpdatedAt: now,
			},
			HasUnresolvedDeps: true,
			CommentCount:      5,
		}

		result := converters.ToPublicTaskWithDetails(taskWithDetails)

		assert.Equal(t, "task-details", result.ID)
		assert.Equal(t, "col-todo", result.ColumnID)
		assert.Equal(t, "Task with details", result.Title)
		assert.Equal(t, "low", result.Priority)
		assert.True(t, result.HasUnresolvedDeps)
		assert.Equal(t, 5, result.CommentCount)
	})

	t.Run("Converts task without unresolved deps", func(t *testing.T) {
		taskWithDetails := domain.TaskWithDetails{
			Task: domain.Task{
				ID:       domain.TaskID("task-clean"),
				ColumnID: domain.ColumnID("col-done"),
				Title:    "Clean task",
				Summary:  "Summary",
			},
			HasUnresolvedDeps: false,
			CommentCount:      0,
		}

		result := converters.ToPublicTaskWithDetails(taskWithDetails)

		assert.Equal(t, "task-clean", result.ID)
		assert.False(t, result.HasUnresolvedDeps)
		assert.Equal(t, 0, result.CommentCount)
	})
}

func TestToPublicTasksWithDetails(t *testing.T) {
	t.Run("Empty slice returns empty slice", func(t *testing.T) {
		result := converters.ToPublicTasksWithDetails([]domain.TaskWithDetails{})
		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})

	t.Run("Single task converts correctly", func(t *testing.T) {
		tasks := []domain.TaskWithDetails{
			{
				Task: domain.Task{
					ID:       domain.TaskID("task-one"),
					ColumnID: domain.ColumnID("col-1"),
					Title:    "One",
					Summary:  "Summary",
				},
				HasUnresolvedDeps: false,
				CommentCount:      2,
			},
		}

		result := converters.ToPublicTasksWithDetails(tasks)

		assert.Len(t, result, 1)
		assert.Equal(t, "task-one", result[0].ID)
		assert.Equal(t, 2, result[0].CommentCount)
	})

	t.Run("Multiple tasks convert correctly and preserve order", func(t *testing.T) {
		tasks := []domain.TaskWithDetails{
			{
				Task: domain.Task{
					ID:       domain.TaskID("task-a"),
					ColumnID: domain.ColumnID("col-1"),
					Title:    "Alpha",
					Summary:  "Summary A",
				},
				CommentCount: 1,
			},
			{
				Task: domain.Task{
					ID:       domain.TaskID("task-b"),
					ColumnID: domain.ColumnID("col-2"),
					Title:    "Beta",
					Summary:  "Summary B",
				},
				HasUnresolvedDeps: true,
				CommentCount:      3,
			},
			{
				Task: domain.Task{
					ID:       domain.TaskID("task-c"),
					ColumnID: domain.ColumnID("col-3"),
					Title:    "Gamma",
					Summary:  "Summary C",
				},
				CommentCount: 0,
			},
		}

		result := converters.ToPublicTasksWithDetails(tasks)

		assert.Len(t, result, 3)
		assert.Equal(t, "task-a", result[0].ID)
		assert.Equal(t, "Alpha", result[0].Title)
		assert.Equal(t, 1, result[0].CommentCount)
		assert.Equal(t, "task-b", result[1].ID)
		assert.True(t, result[1].HasUnresolvedDeps)
		assert.Equal(t, 3, result[1].CommentCount)
		assert.Equal(t, "task-c", result[2].ID)
		assert.Equal(t, 0, result[2].CommentCount)
	})
}

func TestToPublicDependencyContext(t *testing.T) {
	t.Run("Converts dependency context with files modified", func(t *testing.T) {
		ctx := domain.DependencyContext{
			TaskID:            domain.TaskID("dep-task-123"),
			Title:             "Completed dependency",
			CompletionSummary: "All work done",
			FilesModified:     []string{"service.go", "handler.go"},
		}

		result := converters.ToPublicDependencyContext(ctx)

		assert.Equal(t, "dep-task-123", result.TaskID)
		assert.Equal(t, "Completed dependency", result.Title)
		assert.Equal(t, "All work done", result.CompletionSummary)
		assert.Equal(t, []string{"service.go", "handler.go"}, result.FilesModified)
	})

	t.Run("Converts dependency context with empty files modified", func(t *testing.T) {
		ctx := domain.DependencyContext{
			TaskID:            domain.TaskID("dep-empty"),
			Title:             "No files",
			CompletionSummary: "Done",
			FilesModified:     []string{},
		}

		result := converters.ToPublicDependencyContext(ctx)

		assert.Equal(t, "dep-empty", result.TaskID)
		assert.Equal(t, "No files", result.Title)
		assert.Equal(t, "Done", result.CompletionSummary)
		assert.Equal(t, []string{}, result.FilesModified)
	})

	t.Run("Converts dependency context with nil files modified", func(t *testing.T) {
		ctx := domain.DependencyContext{
			TaskID:            domain.TaskID("dep-nil"),
			Title:             "Nil files",
			CompletionSummary: "Summary",
			FilesModified:     nil,
		}

		result := converters.ToPublicDependencyContext(ctx)

		assert.Equal(t, "dep-nil", result.TaskID)
		assert.Nil(t, result.FilesModified)
	})
}

func TestToPublicDependencyContexts(t *testing.T) {
	t.Run("Empty slice returns empty slice", func(t *testing.T) {
		result := converters.ToPublicDependencyContexts([]domain.DependencyContext{})
		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})

	t.Run("Single context converts correctly", func(t *testing.T) {
		contexts := []domain.DependencyContext{
			{
				TaskID:            domain.TaskID("single-dep"),
				Title:             "Single",
				CompletionSummary: "Done",
				FilesModified:     []string{"only.go"},
			},
		}

		result := converters.ToPublicDependencyContexts(contexts)

		assert.Len(t, result, 1)
		assert.Equal(t, "single-dep", result[0].TaskID)
		assert.Equal(t, "Single", result[0].Title)
	})

	t.Run("Multiple contexts convert correctly and preserve order", func(t *testing.T) {
		contexts := []domain.DependencyContext{
			{
				TaskID:            domain.TaskID("dep-1"),
				Title:             "First dep",
				CompletionSummary: "First done",
				FilesModified:     []string{"first.go"},
			},
			{
				TaskID:            domain.TaskID("dep-2"),
				Title:             "Second dep",
				CompletionSummary: "Second done",
				FilesModified:     []string{"second.go", "third.go"},
			},
		}

		result := converters.ToPublicDependencyContexts(contexts)

		assert.Len(t, result, 2)
		assert.Equal(t, "dep-1", result[0].TaskID)
		assert.Equal(t, "First dep", result[0].Title)
		assert.Equal(t, []string{"first.go"}, result[0].FilesModified)
		assert.Equal(t, "dep-2", result[1].TaskID)
		assert.Equal(t, "Second dep", result[1].Title)
		assert.Equal(t, []string{"second.go", "third.go"}, result[1].FilesModified)
	})
}
