package security_test

// Security tests for the app (service) layer.
//
// Each vulnerability section has:
//   - RED test  (TestSecurity_RED_*):  expected to FAIL today (demonstrates the gap)
//   - GREEN test (TestSecurity_GREEN_*): expected to PASS today or after the fix
//
// Vulnerabilities covered:
//  1. State machine: BlockTask accepts an already-blocked task (double-block)
//  2. State machine: RequestWontDo accepts an already-wont-do-requested task
//  3. State machine: CompleteTask accepted on a todo-column task (no in_progress guard)
//  4. Token integer accumulation: negative token values accepted, corrupting counters
//  5. State machine: MoveTask allows arbitrary column transitions (e.g. done → todo bypass)
//  6. Missing completion summary minimum length enforcement in app layer

import (
	"context"
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	tasksrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// 1-2. (Removed — WIP limits no longer enforced)
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// 3. State machine: BlockTask on an already-blocked task
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_BlockTask_AlreadyBlockedTask demonstrates that BlockTask does
// not verify the task's current state before proceeding. Calling BlockTask on a
// task that is already blocked:
//   - Overwrites the BlockedReason without preserving history
//   - Appends a duplicate auto-comment
//   - Resets BlockedAt timestamp (erasing audit trail)
//   - Re-appends the task to the blocked column, increasing its position
//
// RED: BlockTask has no guard "if task.IsBlocked { return ErrTaskBlocked }".
// Fix: add that guard at the top of BlockTask().
func TestSecurity_RED_BlockTask_AlreadyBlockedTask(t *testing.T) {
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

	// RED: no error is returned; the task is updated and a duplicate comment created.
	assert.Error(t, err,
		"RED: BlockTask on an already-blocked task must return an error (e.g. ErrTaskBlocked); "+
			"got nil; fix: add 'if task.IsBlocked { return domain.ErrTaskBlocked }' guard at top of BlockTask()")

	if updateCalled {
		t.Error("RED: task.Update was called — double-blocking overwrites audit trail")
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

// TestSecurity_RED_RequestWontDo_AlreadyRequested demonstrates that
// RequestWontDo has no idempotency guard. Calling it twice on the same task:
//   - Resets WontDoRequestedAt timestamp (erasing original audit entry)
//   - Overwrites the WontDoReason silently
//   - Creates a duplicate auto-comment
//
// RED: RequestWontDo has no guard "if task.WontDoRequested { return ErrWontDoNotRequested }".
// (There is no suitable existing error; one must be added, or ErrInvalidTaskData used.)
// Fix: add "if task.WontDoRequested { return domain.ErrInvalidTaskData }" at the
// top of RequestWontDo().
func TestSecurity_RED_RequestWontDo_AlreadyRequested(t *testing.T) {
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

	// RED: no error returned; audit trail is silently overwritten.
	assert.Error(t, err,
		"RED: RequestWontDo on a task that already has wont_do_requested=true must return an error; "+
			"got nil; fix: add 'if task.WontDoRequested { return domain.ErrInvalidTaskData }' guard")

	if updateCalled {
		t.Error("RED: task.Update was called on an already-requested wont-do task — audit trail overwritten")
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

// TestSecurity_RED_CompleteTask_NotInProgress demonstrates that CompleteTask
// does not verify the task is in the "in_progress" column. An agent can mark a
// task as completed while it is still in "todo", skipping the StartTask step
// entirely. This corrupts duration_seconds (StartedAt is nil → 0 duration) and
// bypasses any human reviews triggered by the in_progress transition.
//
// RED: CompleteTask only checks IsBlocked; it does not check the current column.
// Fix: add a column check — require task to be in ColumnInProgress before
// completing (or at minimum not in todo/backlog/blocked).
func TestSecurity_RED_CompleteTask_NotInProgress(t *testing.T) {
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

	// RED: no error — task in "todo" is directly moved to "done",
	// with DurationSeconds=0 (StartedAt was nil) and no StartTask audit.
	assert.Error(t, err,
		"RED: CompleteTask on a task in 'todo' column must return an error; "+
			"the task was never started; fix: check that the task is in ColumnInProgress before completing")

	if updatedTask != nil {
		assert.NotZero(t, updatedTask.DurationSeconds,
			"RED: DurationSeconds is 0 because StartedAt was nil — task was never started yet was 'completed'")
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

// TestSecurity_RED_NegativeTokenValuesCorruptCounter demonstrates that
// UpdateTask accumulates token values without range validation.  Passing
// negative token counts (e.g. from a buggy or malicious agent) decrements the
// running total, producing incorrect statistics and potentially wrapping to
// negative values in subsequent reports.
//
// RED: app.UpdateTask does "task.InputTokens += tokenUsage.InputTokens" without
// checking tokenUsage.InputTokens >= 0.
// Fix: add "if tokenUsage.InputTokens < 0 { return domain.ErrInvalidTaskData }"
// (or silently clamp to 0) before accumulating.
func TestSecurity_RED_NegativeTokenValuesCorruptCounter(t *testing.T) {
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

	// After the fix, UpdateTask should return an error for negative token values.
	// Currently it accepts them and decrements the counter to a negative value.
	if err == nil {
		// Demonstrate the vulnerable state: tokens went negative
		require.NotNil(t, updatedTask, "UpdateTask must have been called")
		assert.GreaterOrEqual(t, updatedTask.InputTokens, 0,
			"RED: InputTokens is now %d (negative); negative token usage was accepted and corrupted "+
				"the counter; fix: validate tokenUsage fields >= 0 before accumulating",
			updatedTask.InputTokens)
	}
	// If err != nil, the fix is already in place — that's the desired outcome.
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

// TestSecurity_RED_CreateComment_InvalidAuthorTypeAccepted demonstrates that
// CreateComment at the app layer accepts any AuthorType string without validation.
// An agent can call create_comment with author_type="human" to impersonate a
// human reviewer in the audit log.
//
// RED: app.CreateComment delegates directly to the comment repository without
// validating that authorType is one of AuthorTypeAgent or AuthorTypeHuman.
// Fix: add validation "if authorType != domain.AuthorTypeAgent && authorType != domain.AuthorTypeHuman { return domain.ErrInvalidCommentData }"
// at the top of CreateComment().
func TestSecurity_RED_CreateComment_InvalidAuthorTypeAccepted(t *testing.T) {
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

	// RED: no error; the comment is stored with author_type="admin".
	assert.Error(t, err,
		"RED: CreateComment with author_type='admin' must return an error; "+
			"got nil; fix: validate authorType is one of {agent, human}")

	if storedComment != nil {
		assert.NotEqual(t, domain.AuthorType("admin"), storedComment.AuthorType,
			"RED: comment with spoofed author_type='admin' was stored in the repository")
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

// TestSecurity_RED_CreateTask_EmptyTitleIsRejected verifies that CreateTask
// returns ErrTaskTitleRequired when title is empty.
// (This is a GREEN check — the guard is already present. Including it here as
// a regression guard against future refactoring that removes it.)
func TestSecurity_GREEN_CreateTask_EmptyTitleIsRejected(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, _, _, _, _ := setupTestApp()

	projectID := domain.NewProjectID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	_, err := a.CreateTask(ctx, projectID, "", "summary", "desc",
		domain.PriorityMedium, "", "", "", nil, nil, "", false, nil)

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

	_, err := a.CreateTask(ctx, projectID, "Valid title", "", "desc",
		domain.PriorityMedium, "", "", "", nil, nil, "", false, nil)

	assert.ErrorIs(t, err, domain.ErrSummaryRequired,
		"empty task summary must be rejected with ErrSummaryRequired")
}
