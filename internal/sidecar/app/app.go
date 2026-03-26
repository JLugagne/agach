package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JLugagne/agach-mcp/internal/sidecar/domain"
)

// App orchestrates sidecar tool operations.
type App struct {
	api       domain.ServerAPI
	featureID string
}

// New creates a new App.
func New(api domain.ServerAPI, featureID string) *App {
	return &App{api: api, featureID: featureID}
}

// BulkCreateTasks creates multiple tasks, resolving intra-batch dependency references.
// Tasks can reference each other using the "ref" field in depends_on.
func (a *App) BulkCreateTasks(ctx context.Context, tasks []domain.BulkTaskInput) ([]domain.CreatedTask, error) {
	// Map ref -> created task ID for intra-batch dependency resolution
	refToID := make(map[string]string)
	created := make([]domain.CreatedTask, 0, len(tasks))

	for _, t := range tasks {
		resp, err := a.api.CreateTask(ctx, domain.CreateTaskRequest{
			Title:           t.Title,
			Summary:         t.Summary,
			Description:     t.Description,
			Priority:        t.Priority,
			AssignedRole:    t.AssignedRole,
			ContextFiles:    t.ContextFiles,
			Tags:            t.Tags,
			EstimatedEffort: t.EstimatedEffort,
		})
		if err != nil {
			return created, fmt.Errorf("create task %q: %w", t.Title, err)
		}

		if t.Ref != "" {
			refToID[t.Ref] = resp.ID
		}
		created = append(created, domain.CreatedTask{Ref: t.Ref, ID: resp.ID})

		// Add dependencies for this task
		for _, dep := range t.DependsOn {
			depID := dep
			// Resolve intra-batch reference
			if resolved, ok := refToID[dep]; ok {
				depID = resolved
			}
			if err := a.api.AddDependency(ctx, resp.ID, depID); err != nil {
				return created, fmt.Errorf("add dependency %s -> %s: %w", resp.ID, depID, err)
			}
		}
	}

	return created, nil
}

// BulkAddDependencies adds multiple dependencies in sequence.
func (a *App) BulkAddDependencies(ctx context.Context, deps []domain.BulkDependencyInput) error {
	for _, d := range deps {
		if err := a.api.AddDependency(ctx, d.TaskID, d.DependsOnTaskID); err != nil {
			return fmt.Errorf("add dependency %s -> %s: %w", d.TaskID, d.DependsOnTaskID, err)
		}
	}
	return nil
}

// CompleteTask completes a task and writes the summary file.
func (a *App) CompleteTask(ctx context.Context, taskID string, req domain.CompleteTaskRequest) error {
	if err := a.api.CompleteTask(ctx, taskID, req); err != nil {
		return err
	}
	a.writeSummary(taskID, req.CompletionSummary)
	return nil
}

// RunTask starts a task by moving it to in_progress.
func (a *App) RunTask(ctx context.Context, taskID string) error {
	return a.api.MoveTask(ctx, taskID, "in_progress")
}

// BlockTask blocks a task with a reason and writes the summary file.
func (a *App) BlockTask(ctx context.Context, taskID string, req domain.BlockTaskRequest) error {
	if err := a.api.BlockTask(ctx, taskID, req); err != nil {
		return err
	}
	a.writeSummary(taskID, req.BlockedReason)
	return nil
}

// WontDoTask requests a task be marked as won't do and writes the summary file.
func (a *App) WontDoTask(ctx context.Context, taskID string, req domain.WontDoRequest) error {
	if err := a.api.RequestWontDo(ctx, taskID, req); err != nil {
		return err
	}
	a.writeSummary(taskID, req.WontDoReason)
	return nil
}

// UpdateFeatureChangelogs updates the feature's user and/or tech changelogs.
func (a *App) UpdateFeatureChangelogs(ctx context.Context, req domain.FeatureChangelogsRequest) error {
	return a.api.UpdateFeatureChangelogs(ctx, req)
}

// writeSummary writes the summary content to features/{featureID}/{taskID}_SUMMARY.md.
// Best-effort: errors are silently ignored since the API call already succeeded.
func (a *App) writeSummary(taskID, content string) {
	if a.featureID == "" || content == "" {
		return
	}
	dir := filepath.Join("features", a.featureID)
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, taskID+"_SUMMARY.md")
	os.WriteFile(path, []byte(content), 0644)
}
