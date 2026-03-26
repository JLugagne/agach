package app_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/sidecar/app"
	"github.com/JLugagne/agach-mcp/internal/sidecar/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAPI implements domain.ServerAPI for testing.
type mockAPI struct {
	createTaskFn             func(ctx context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error)
	addDependencyFn          func(ctx context.Context, taskID, dependsOnTaskID string) error
	completeTaskFn           func(ctx context.Context, taskID string, req domain.CompleteTaskRequest) error
	moveTaskFn               func(ctx context.Context, taskID, targetColumn string) error
	blockTaskFn              func(ctx context.Context, taskID string, req domain.BlockTaskRequest) error
	requestWontDoFn          func(ctx context.Context, taskID string, req domain.WontDoRequest) error
	updateFeatureChangelogFn func(ctx context.Context, req domain.FeatureChangelogsRequest) error
}

func (m *mockAPI) CreateTask(ctx context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error) {
	if m.createTaskFn != nil {
		return m.createTaskFn(ctx, req)
	}
	return domain.CreateTaskResponse{}, nil
}

func (m *mockAPI) AddDependency(ctx context.Context, taskID, dependsOnTaskID string) error {
	if m.addDependencyFn != nil {
		return m.addDependencyFn(ctx, taskID, dependsOnTaskID)
	}
	return nil
}

func (m *mockAPI) CompleteTask(ctx context.Context, taskID string, req domain.CompleteTaskRequest) error {
	if m.completeTaskFn != nil {
		return m.completeTaskFn(ctx, taskID, req)
	}
	return nil
}

func (m *mockAPI) MoveTask(ctx context.Context, taskID, targetColumn string) error {
	if m.moveTaskFn != nil {
		return m.moveTaskFn(ctx, taskID, targetColumn)
	}
	return nil
}

func (m *mockAPI) BlockTask(ctx context.Context, taskID string, req domain.BlockTaskRequest) error {
	if m.blockTaskFn != nil {
		return m.blockTaskFn(ctx, taskID, req)
	}
	return nil
}

func (m *mockAPI) RequestWontDo(ctx context.Context, taskID string, req domain.WontDoRequest) error {
	if m.requestWontDoFn != nil {
		return m.requestWontDoFn(ctx, taskID, req)
	}
	return nil
}

func (m *mockAPI) UpdateFeatureChangelogs(ctx context.Context, req domain.FeatureChangelogsRequest) error {
	if m.updateFeatureChangelogFn != nil {
		return m.updateFeatureChangelogFn(ctx, req)
	}
	return nil
}

func TestBulkCreateTasks_Simple(t *testing.T) {
	var createdTitles []string
	mock := &mockAPI{
		createTaskFn: func(_ context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error) {
			createdTitles = append(createdTitles, req.Title)
			return domain.CreateTaskResponse{ID: "task-" + req.Title}, nil
		},
	}

	a := app.New(mock, "")
	created, err := a.BulkCreateTasks(context.Background(), []domain.BulkTaskInput{
		{Title: "one", Summary: "s1"},
		{Title: "two", Summary: "s2"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"one", "two"}, createdTitles)
	assert.Len(t, created, 2)
	assert.Equal(t, "task-one", created[0].ID)
	assert.Equal(t, "task-two", created[1].ID)
}

func TestBulkCreateTasks_IntraBatchRefs(t *testing.T) {
	var deps []struct{ taskID, depID string }
	mock := &mockAPI{
		createTaskFn: func(_ context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error) {
			return domain.CreateTaskResponse{ID: "id-" + req.Title}, nil
		},
		addDependencyFn: func(_ context.Context, taskID, dependsOnTaskID string) error {
			deps = append(deps, struct{ taskID, depID string }{taskID, dependsOnTaskID})
			return nil
		},
	}

	a := app.New(mock, "")
	created, err := a.BulkCreateTasks(context.Background(), []domain.BulkTaskInput{
		{Ref: "first", Title: "A", Summary: "s"},
		{Ref: "second", Title: "B", Summary: "s", DependsOn: []string{"first"}},
		{Title: "C", Summary: "s", DependsOn: []string{"second", "external-id"}},
	})

	require.NoError(t, err)
	assert.Len(t, created, 3)
	assert.Equal(t, "first", created[0].Ref)
	assert.Equal(t, "second", created[1].Ref)
	assert.Equal(t, "", created[2].Ref)

	require.Len(t, deps, 3)
	// B depends on A (resolved from ref "first" → "id-A")
	assert.Equal(t, "id-B", deps[0].taskID)
	assert.Equal(t, "id-A", deps[0].depID)
	// C depends on B (resolved from ref "second" → "id-B")
	assert.Equal(t, "id-C", deps[1].taskID)
	assert.Equal(t, "id-B", deps[1].depID)
	// C depends on external-id (not a ref, passed through as-is)
	assert.Equal(t, "id-C", deps[2].taskID)
	assert.Equal(t, "external-id", deps[2].depID)
}

func TestBulkCreateTasks_CreateError_ReturnsPartial(t *testing.T) {
	callCount := 0
	mock := &mockAPI{
		createTaskFn: func(_ context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error) {
			callCount++
			if callCount == 2 {
				return domain.CreateTaskResponse{}, fmt.Errorf("server error")
			}
			return domain.CreateTaskResponse{ID: "id-" + req.Title}, nil
		},
	}

	a := app.New(mock, "")
	created, err := a.BulkCreateTasks(context.Background(), []domain.BulkTaskInput{
		{Title: "ok", Summary: "s"},
		{Title: "fail", Summary: "s"},
		{Title: "never", Summary: "s"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "fail")
	// Should return the one that was created before the error
	assert.Len(t, created, 1)
	assert.Equal(t, "id-ok", created[0].ID)
}

func TestBulkAddDependencies(t *testing.T) {
	var deps []struct{ taskID, depID string }
	mock := &mockAPI{
		addDependencyFn: func(_ context.Context, taskID, depID string) error {
			deps = append(deps, struct{ taskID, depID string }{taskID, depID})
			return nil
		},
	}

	a := app.New(mock, "")
	err := a.BulkAddDependencies(context.Background(), []domain.BulkDependencyInput{
		{TaskID: "t1", DependsOnTaskID: "t0"},
		{TaskID: "t2", DependsOnTaskID: "t1"},
	})

	require.NoError(t, err)
	assert.Len(t, deps, 2)
}

func TestBulkAddDependencies_Error(t *testing.T) {
	mock := &mockAPI{
		addDependencyFn: func(_ context.Context, _, _ string) error {
			return fmt.Errorf("cycle detected")
		},
	}

	a := app.New(mock, "")
	err := a.BulkAddDependencies(context.Background(), []domain.BulkDependencyInput{
		{TaskID: "t1", DependsOnTaskID: "t0"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detected")
}

func TestCompleteTask_WritesSummaryFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	mock := &mockAPI{}
	a := app.New(mock, "feat-123")

	err := a.CompleteTask(context.Background(), "task-abc", domain.CompleteTaskRequest{
		CompletionSummary: "All tests pass, feature implemented.",
		CompletedByAgent:  "green",
	})

	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(dir, "features", "feat-123", "task-abc_SUMMARY.md"))
	require.NoError(t, err)
	assert.Equal(t, "All tests pass, feature implemented.", string(content))
}

func TestCompleteTask_APIError_NoSummaryFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	mock := &mockAPI{
		completeTaskFn: func(_ context.Context, _ string, _ domain.CompleteTaskRequest) error {
			return fmt.Errorf("task not in progress")
		},
	}
	a := app.New(mock, "feat-123")

	err := a.CompleteTask(context.Background(), "task-abc", domain.CompleteTaskRequest{
		CompletionSummary: "should not be written",
	})

	require.Error(t, err)
	_, statErr := os.Stat(filepath.Join(dir, "features", "feat-123", "task-abc_SUMMARY.md"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestBlockTask_WritesSummaryFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	mock := &mockAPI{}
	a := app.New(mock, "feat-456")

	err := a.BlockTask(context.Background(), "task-xyz", domain.BlockTaskRequest{
		BlockedReason:  "Waiting for upstream API to be ready.",
		BlockedByAgent: "red",
	})

	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(dir, "features", "feat-456", "task-xyz_SUMMARY.md"))
	require.NoError(t, err)
	assert.Equal(t, "Waiting for upstream API to be ready.", string(content))
}

func TestWontDoTask_WritesSummaryFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	mock := &mockAPI{}
	a := app.New(mock, "feat-789")

	err := a.WontDoTask(context.Background(), "task-wont", domain.WontDoRequest{
		WontDoReason:      "Requirements changed, this task is no longer needed.",
		WontDoRequestedBy: "reviewer",
	})

	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(dir, "features", "feat-789", "task-wont_SUMMARY.md"))
	require.NoError(t, err)
	assert.Equal(t, "Requirements changed, this task is no longer needed.", string(content))
}

func TestWriteSummary_NoFeatureID_SkipsFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	mock := &mockAPI{}
	a := app.New(mock, "") // no feature ID

	err := a.CompleteTask(context.Background(), "task-1", domain.CompleteTaskRequest{
		CompletionSummary: "done",
	})

	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(dir, "features"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestRunTask(t *testing.T) {
	var movedTo string
	mock := &mockAPI{
		moveTaskFn: func(_ context.Context, taskID, targetColumn string) error {
			movedTo = targetColumn
			return nil
		},
	}

	a := app.New(mock, "")
	err := a.RunTask(context.Background(), "task-1")

	require.NoError(t, err)
	assert.Equal(t, "in_progress", movedTo)
}

func TestUpdateFeatureChangelogs(t *testing.T) {
	var captured domain.FeatureChangelogsRequest
	mock := &mockAPI{
		updateFeatureChangelogFn: func(_ context.Context, req domain.FeatureChangelogsRequest) error {
			captured = req
			return nil
		},
	}

	userCL := "Added login page"
	techCL := "New auth middleware"
	a := app.New(mock, "")
	err := a.UpdateFeatureChangelogs(context.Background(), domain.FeatureChangelogsRequest{
		UserChangelog: &userCL,
		TechChangelog: &techCL,
	})

	require.NoError(t, err)
	assert.Equal(t, &userCL, captured.UserChangelog)
	assert.Equal(t, &techCL, captured.TechChangelog)
}
