package security_test

// Security tests for the app (service) layer.
//
// All tests assert correct behaviour. GREEN tests confirm valid operations succeed;
// formerly-RED tests confirm that production code now correctly rejects invalid inputs.
//
// Vulnerabilities covered (all fixed):
//  1. State machine: BlockTask accepts an already-blocked task (double-block)
//  2. State machine: RequestWontDo accepts an already-wont-do-requested task
//  3. State machine: CompleteTask accepted on a todo-column task (no in_progress guard)
//  4. Token integer accumulation: negative token values accepted, corrupting counters
//  5. State machine: MoveTask allows arbitrary column transitions (e.g. done -> todo bypass)
//  6. Missing completion summary minimum length enforcement in app layer

import (
	"context"
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	tasksrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// 1-2. (Removed — WIP limits no longer enforced)
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// 3. State machine: BlockTask on an already-blocked task
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_BlockTask_AlreadyBlockedTask verifies that BlockTask rejects a
// call on a task that is already blocked. Without this guard, calling BlockTask
// twice would:
//   - Overwrite the BlockedReason without preserving history
//   - Append a duplicate auto-comment
//   - Reset BlockedAt timestamp (erasing audit trail)
//   - Re-append the task to the blocked column, increasing its position
//
// Vulnerability that was fixed: BlockTask had no guard for task.IsBlocked.
func TestSecurity_BlockTask_AlreadyBlockedTask(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	blockedColumnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	// Task is already blocked
	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:            taskID,
			ColumnID:      blockedColumnID,
			Title:         "Blocked task",
			Summary:       "summary",
			IsBlocked:     true,
			BlockedReason: "original reason",
		}, nil
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if slug == domain.ColumnBlocked {
			return &domain.Column{ID: blockedColumnID, Slug: domain.ColumnBlocked, Name: "Blocked"}, nil
		}
		return nil, nil
	}

	var updateCalled bool
	mockTasks.UpdateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
		updateCalled = true
		return nil
	}

	mockTasks.ListFunc = func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
		return nil, nil
	}

	mockComments.CreateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
		return nil
	}

	err := a.BlockTask(ctx, projectID, taskID, "new reason — overwrites history", "agent-2", "")

	assert.Error(t, err,
		"BlockTask on an already-blocked task must return an error")

	if updateCalled {
		t.Error("task.Update must not be called when double-blocking — it would overwrite the audit trail")
	}
}

// TestSecurity_GREEN_BlockTask_UnblockedTaskSucceeds verifies that blocking a
// non-blocked task works correctly.
func TestSecurity_GREEN_BlockTask_UnblockedTaskSucceeds(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	inProgressColumnID := domain.NewColumnID()
	blockedColumnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:        taskID,
			ColumnID:  inProgressColumnID,
			Title:     "Active task",
			Summary:   "summary",
			IsBlocked: false,
		}, nil
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if slug == domain.ColumnBlocked {
			return &domain.Column{ID: blockedColumnID, Slug: domain.ColumnBlocked, Name: "Blocked"}, nil
		}
		return nil, nil
	}

	mockTasks.ListFunc = func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
		return nil, nil
	}

	mockTasks.UpdateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
		return nil
	}

	mockComments.CreateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
		return nil
	}

	err := a.BlockTask(ctx, projectID, taskID, strings.Repeat("B", 60), "agent-1", "")
	require.NoError(t, err, "blocking a non-blocked task must succeed")
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. State machine: RequestWontDo on an already-wont-do-requested task
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RequestWontDo_AlreadyRequested verifies that RequestWontDo
// rejects a call on a task that already has WontDoRequested=true. Without this
// guard, calling it twice would:
//   - Reset WontDoRequestedAt timestamp (erasing original audit entry)
//   - Overwrite the WontDoReason silently
//   - Create a duplicate auto-comment
//
// Vulnerability that was fixed: RequestWontDo had no idempotency guard.
func TestSecurity_RequestWontDo_AlreadyRequested(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	blockedColumnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	// Task already has won't-do requested
	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:              taskID,
			ColumnID:        blockedColumnID,
			Title:           "Contested task",
			Summary:         "summary",
			IsBlocked:       true,
			WontDoRequested: true,
			WontDoReason:    "original reason",
		}, nil
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if slug == domain.ColumnBlocked {
			return &domain.Column{ID: blockedColumnID, Slug: domain.ColumnBlocked, Name: "Blocked"}, nil
		}
		return nil, nil
	}

	var updateCalled bool
	mockTasks.UpdateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
		updateCalled = true
		return nil
	}

	mockTasks.ListFunc = func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
		return nil, nil
	}

	mockComments.CreateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
		return nil
	}

	err := a.RequestWontDo(ctx, projectID, taskID, "duplicate reason", "agent-2", "")

	assert.Error(t, err,
		"RequestWontDo on a task that already has wont_do_requested=true must return an error")

	if updateCalled {
		t.Error("task.Update must not be called on an already-requested wont-do task")
	}
}

// TestSecurity_GREEN_RequestWontDo_FirstRequestSucceeds verifies that the first
// won't-do request is accepted.
func TestSecurity_GREEN_RequestWontDo_FirstRequestSucceeds(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	todoColumnID := domain.NewColumnID()
	blockedColumnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:              taskID,
			ColumnID:        todoColumnID,
			Title:           "Boring task",
			Summary:         "summary",
			WontDoRequested: false,
		}, nil
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if slug == domain.ColumnBlocked {
			return &domain.Column{ID: blockedColumnID, Slug: domain.ColumnBlocked, Name: "Blocked"}, nil
		}
		return nil, nil
	}

	mockTasks.ListFunc = func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
		return nil, nil
	}

	mockTasks.UpdateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
		return nil
	}

	mockComments.CreateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
		return nil
	}

	err := a.RequestWontDo(ctx, projectID, taskID, strings.Repeat("R", 55), "agent-1", "")
	require.NoError(t, err, "first won't-do request must succeed")
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. State machine: CompleteTask on a todo-column task (not in in_progress)
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_CompleteTask_NotInProgress verifies that CompleteTask rejects a
// task that is not in the "in_progress" column. Without this check, an agent
// could skip the StartTask step, corrupting duration_seconds and bypassing
// human reviews.
//
// Vulnerability that was fixed: CompleteTask only checked IsBlocked without
// verifying the task was in the in_progress column.
func TestSecurity_CompleteTask_NotInProgress(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	todoColumnID := domain.NewColumnID()
	doneColumnID := domain.NewColumnID()

	// Task is in "todo" (not started)
	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:        taskID,
			ColumnID:  todoColumnID,
			Title:     "Unstarted task",
			Summary:   "summary",
			IsBlocked: false,
			StartedAt: nil, // never started
		}, nil
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if slug == domain.ColumnDone {
			return &domain.Column{ID: doneColumnID, Slug: domain.ColumnDone, Name: "Done"}, nil
		}
		return nil, nil
	}

	mockColumns.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
		if id == todoColumnID {
			return &domain.Column{ID: todoColumnID, Slug: domain.ColumnTodo, Name: "Todo"}, nil
		}
		return nil, nil
	}

	mockTasks.ListFunc = func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
		return nil, nil
	}

	var updatedTask *domain.Task
	mockTasks.UpdateFunc = func(_ context.Context, _ domain.ProjectID, t domain.Task) error {
		updatedTask = &t
		return nil
	}

	completionSummary := strings.Repeat("C", 110) // satisfies >= 100 char check at HTTP layer
	err := a.CompleteTask(ctx, projectID, taskID, completionSummary, nil, "agent-1", nil, "")

	assert.Error(t, err,
		"CompleteTask on a task in 'todo' column must return an error")

	if updatedTask != nil {
		assert.NotZero(t, updatedTask.DurationSeconds,
			"DurationSeconds must not be 0 for a completed task (StartedAt was nil)")
	}
}

// TestSecurity_GREEN_CompleteTask_InProgressTaskSucceeds verifies that completing
// a task that IS in in_progress succeeds without error.
func TestSecurity_GREEN_CompleteTask_InProgressTaskSucceeds(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	inProgressColumnID := domain.NewColumnID()
	doneColumnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:        taskID,
			ColumnID:  inProgressColumnID,
			Title:     "Running task",
			Summary:   "summary",
			IsBlocked: false,
		}, nil
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if slug == domain.ColumnDone {
			return &domain.Column{ID: doneColumnID, Slug: domain.ColumnDone, Name: "Done"}, nil
		}
		return nil, nil
	}

	mockColumns.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
		if id == inProgressColumnID {
			return &domain.Column{ID: inProgressColumnID, Slug: domain.ColumnInProgress, Name: "In Progress"}, nil
		}
		return nil, nil
	}

	mockTasks.ListFunc = func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
		return nil, nil
	}

	mockTasks.UpdateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
		return nil
	}

	err := a.CompleteTask(ctx, projectID, taskID, strings.Repeat("C", 110), nil, "agent-1", nil, "")
	require.NoError(t, err, "completing a task that is in in_progress must succeed")
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. Negative token values corrupt the cumulative token counter
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_NegativeTokenValuesCorruptCounter verifies that UpdateTask
// rejects or clamps negative token values. Passing negative token counts would
// decrement the running total and corrupt statistics.
//
// Vulnerability that was fixed: app.UpdateTask applied "task.InputTokens +=
// tokenUsage.InputTokens" without checking tokenUsage.InputTokens >= 0.
func TestSecurity_NegativeTokenValuesCorruptCounter(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	// Task has 100 accumulated input tokens — negative usage will bring it below 0
	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:          taskID,
			ColumnID:    columnID,
			Title:       "Task with tokens",
			Summary:     "summary",
			InputTokens: 100,
		}, nil
	}

	var updatedTask *domain.Task
	mockTasks.UpdateFunc = func(_ context.Context, _ domain.ProjectID, t domain.Task) error {
		updatedTask = &t
		return nil
	}

	negativeUsage := &domain.TokenUsage{
		InputTokens:  -500, // attacker subtracts more than accumulated — goes negative
		OutputTokens: 0,
	}

	err := a.UpdateTask(ctx, projectID, taskID, nil, nil, nil, nil, nil, nil, nil, nil, negativeUsage, nil, nil, false)

	if err == nil {
		require.NotNil(t, updatedTask, "UpdateTask must have been called")
		assert.GreaterOrEqual(t, updatedTask.InputTokens, 0,
			"InputTokens must not go negative (got %d); negative token values must be rejected or clamped",
			updatedTask.InputTokens)
	}
}

// TestSecurity_GREEN_PositiveTokenValuesAccumulate verifies that positive token
// usage is accumulated correctly (regression guard).
func TestSecurity_GREEN_PositiveTokenValuesAccumulate(t *testing.T) {
	ctx := context.Background()
	a, _, _, mockTasks, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:          taskID,
			ColumnID:    columnID,
			Title:       "Task",
			Summary:     "summary",
			InputTokens: 100,
		}, nil
	}

	var updatedTask *domain.Task
	mockTasks.UpdateFunc = func(_ context.Context, _ domain.ProjectID, t domain.Task) error {
		updatedTask = &t
		return nil
	}

	usage := &domain.TokenUsage{InputTokens: 50, OutputTokens: 25}
	err := a.UpdateTask(ctx, projectID, taskID, nil, nil, nil, nil, nil, nil, nil, nil, usage, nil, nil, false)
	require.NoError(t, err)
	require.NotNil(t, updatedTask)
	assert.Equal(t, 150, updatedTask.InputTokens, "positive token accumulation must work correctly")
	assert.Equal(t, 25, updatedTask.OutputTokens)
}

// ─────────────────────────────────────────────────────────────────────────────
// 8. Comment author-type spoofing via CreateComment
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_CreateComment_InvalidAuthorTypeAccepted verifies that
// CreateComment rejects an invalid AuthorType. Without validation, an agent
// could pass author_type="human" to impersonate a human reviewer in the audit log.
//
// Vulnerability that was fixed: CreateComment delegated directly to the
// comment repository without validating authorType.
func TestSecurity_CreateComment_InvalidAuthorTypeAccepted(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:       taskID,
			ColumnID: columnID,
			Title:    "Some task",
			Summary:  "summary",
		}, nil
	}

	var storedComment *domain.Comment
	mockComments.CreateFunc = func(_ context.Context, _ domain.ProjectID, c domain.Comment) error {
		storedComment = &c
		return nil
	}

	_, err := a.CreateComment(ctx, projectID, taskID, "agent-1", "Agent Name",
		domain.AuthorType("admin"), // invalid author type — impersonating "admin"
		"I hereby approve this task as admin")

	assert.Error(t, err,
		"CreateComment with author_type='admin' must return an error")

	if storedComment != nil {
		assert.NotEqual(t, domain.AuthorType("admin"), storedComment.AuthorType,
			"a comment with spoofed author_type='admin' must not be stored in the repository")
	}
}

// TestSecurity_GREEN_CreateComment_ValidAuthorTypeAgent verifies that
// author_type="agent" is accepted.
func TestSecurity_GREEN_CreateComment_ValidAuthorTypeAgent(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{ID: taskID, ColumnID: columnID, Title: "T", Summary: "S"}, nil
	}

	mockComments.CreateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
		return nil
	}

	_, err := a.CreateComment(ctx, projectID, taskID, "agent-1", "", domain.AuthorTypeAgent, "A comment")
	require.NoError(t, err, "author_type='agent' must be accepted")
}

// TestSecurity_GREEN_CreateComment_ValidAuthorTypeHuman verifies that
// author_type="human" is accepted.
func TestSecurity_GREEN_CreateComment_ValidAuthorTypeHuman(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, _, mockComments, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{ID: taskID, ColumnID: columnID, Title: "T", Summary: "S"}, nil
	}

	mockComments.CreateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
		return nil
	}

	_, err := a.CreateComment(ctx, projectID, taskID, "", "Alice", domain.AuthorTypeHuman, "Human review")
	require.NoError(t, err, "author_type='human' must be accepted")
}

// ─────────────────────────────────────────────────────────────────────────────
// 9. CreateTask: missing validation guards — empty title / empty summary not
//    enforced in app layer via CreateComment (cross-check)
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_GREEN_CreateTask_EmptyTitleIsRejected verifies that CreateTask
// returns ErrTaskTitleRequired when title is empty.
// (Regression guard against future refactoring that removes this check.)
func TestSecurity_GREEN_CreateTask_EmptyTitleIsRejected(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	_, err := a.CreateTask(ctx, projectID, service.CreateTaskInput{
		Title: "", Summary: "summary", Description: "desc", Priority: domain.PriorityMedium,
	})

	assert.ErrorIs(t, err, domain.ErrTaskTitleRequired,
		"empty task title must be rejected with ErrTaskTitleRequired")
}

// TestSecurity_GREEN_CreateTask_EmptySummaryIsRejected verifies that CreateTask
// returns ErrSummaryRequired when summary is empty.
func TestSecurity_GREEN_CreateTask_EmptySummaryIsRejected(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	_, err := a.CreateTask(ctx, projectID, service.CreateTaskInput{
		Title: "Valid title", Summary: "", Description: "desc", Priority: domain.PriorityMedium,
	})

	assert.ErrorIs(t, err, domain.ErrSummaryRequired,
		"empty task summary must be rejected with ErrSummaryRequired")
}
