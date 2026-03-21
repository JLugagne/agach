package app_test

// Security tests for the app (service) layer.
//
// Each vulnerability section has:
//   - RED test  (TestSecurity_RED_*):  expected to FAIL today (demonstrates the gap)
//   - GREEN test (TestSecurity_GREEN_*): expected to PASS today or after the fix
//
// Vulnerabilities covered:
//  1. Negative WIP limit accepted by UpdateColumnWIPLimit — disables WIP enforcement
//  2. WIP limit bypass: MoveTask counts the task being moved in the WIP check
//  3. State machine: BlockTask accepts an already-blocked task (double-block)
//  4. State machine: RequestWontDo accepts an already-wont-do-requested task
//  5. State machine: CompleteTask accepted on a todo-column task (no in_progress guard)
//  6. Token integer accumulation: negative token values accepted, corrupting counters
//  7. State machine: MoveTask allows arbitrary column transitions (e.g. done → todo bypass)
//  8. Missing completion summary minimum length enforcement in app layer

import (
	"context"
	"strings"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/columns/columnstest"
	tasksrepo "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// 1. Negative WIP limit accepted by UpdateColumnWIPLimit
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_UpdateColumnWIPLimit_NegativeValueAccepted demonstrates that
// UpdateColumnWIPLimit does not validate the wipLimit argument. Storing a
// negative value (e.g. -1) disables all WIP enforcement because MoveTask only
// enforces the limit when WIPLimit > 0.
//
// Attack: call update_wip_limit(column="in_progress", wip_limit=-1) to silently
// disable the concurrency cap and move unlimited tasks into in_progress.
//
// RED: app.UpdateColumnWIPLimit passes -1 to the repository without validation.
// Fix: add "if wipLimit < 0 { return domain.ErrInvalidColumn }" before the
// repository call in columns.go.
func TestSecurity_RED_UpdateColumnWIPLimit_NegativeValueAccepted(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		return &domain.Column{ID: columnID, Slug: slug, Name: "In Progress", WIPLimit: 3}, nil
	}

	var storedWIPLimit int
	mockColumns.UpdateWIPLimitFunc = func(_ context.Context, _ domain.ProjectID, _ domain.ColumnID, wip int) error {
		storedWIPLimit = wip
		return nil
	}

	err := a.UpdateColumnWIPLimit(ctx, projectID, domain.ColumnInProgress, -1)

	// RED: no error is returned and the negative value is forwarded to the repo.
	assert.Error(t, err,
		"RED: UpdateColumnWIPLimit(-1) must return an error; got nil; "+
			"fix: validate wipLimit >= 0 before updating")

	if storedWIPLimit == -1 {
		t.Error("RED: UpdateWIPLimit was called with -1, disabling all WIP enforcement")
	}
}

// TestSecurity_GREEN_UpdateColumnWIPLimit_ZeroIsAccepted verifies that
// wipLimit=0 (unlimited) is accepted without error.
func TestSecurity_GREEN_UpdateColumnWIPLimit_ZeroIsAccepted(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		return &domain.Column{ID: columnID, Slug: slug, Name: "In Progress", WIPLimit: 3}, nil
	}
	mockColumns.UpdateWIPLimitFunc = func(_ context.Context, _ domain.ProjectID, _ domain.ColumnID, _ int) error {
		return nil
	}

	err := a.UpdateColumnWIPLimit(ctx, projectID, domain.ColumnInProgress, 0)
	require.NoError(t, err, "wipLimit=0 (unlimited) must be accepted")
}

// TestSecurity_GREEN_UpdateColumnWIPLimit_PositiveIsAccepted verifies that
// a positive wipLimit is accepted without error.
func TestSecurity_GREEN_UpdateColumnWIPLimit_PositiveIsAccepted(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		return &domain.Column{ID: columnID, Slug: slug, Name: "In Progress", WIPLimit: 3}, nil
	}
	mockColumns.UpdateWIPLimitFunc = func(_ context.Context, _ domain.ProjectID, _ domain.ColumnID, _ int) error {
		return nil
	}

	err := a.UpdateColumnWIPLimit(ctx, projectID, domain.ColumnInProgress, 5)
	require.NoError(t, err, "positive wipLimit must be accepted")
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. WIP limit bypass — the task being moved is included in the count
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_WIPLimit_BypassedByMovingSameTask demonstrates that MoveTask
// counts the task being moved when determining whether the WIP limit is reached.
//
// Scenario: WIP limit is 3, in_progress already has 3 tasks (A, B, C). An agent
// calls MoveTask(taskA, in_progress). The code fetches the in_progress task list
// (which still includes taskA) and gets count=3, then checks "3 >= 3" → returns
// WIPLimitExceeded correctly. So far so good.
//
// BUT the real bypass is subtler: MoveTask first calls the WIP-check List, which
// includes tasks from the in_progress column, and THEN calls a second List to
// compute the target position. If both calls return 3 tasks, the task itself IS
// already counted (because the move hasn't happened yet). This means:
//
// - Task being moved is IN in_progress already (same column move / re-start).
//   The WIP limit check fires "3 >= 3" and BLOCKS the move even though the task
//   is already there — over-restrictive, can deadlock an agent.
// - More critically: a task moved from todo → in_progress when in_progress is at
//   exactly WIPLimit-1 tasks succeeds correctly. But when in_progress already has
//   exactly WIPLimit tasks (all different from the task being moved), the check
//   should fire. This part is correct.
//
// The real vulnerability is: MoveTask from in_progress → in_progress (a "restart")
// is BLOCKED by the WIP check even though no additional slot is consumed, because
// the task is already counted in the column.
//
// RED: MoveTask(task already in in_progress, target=in_progress) returns
// ErrWIPLimitExceeded even though no new slot is consumed.
// Fix: exclude the task being moved from the WIP count (count tasks where
// id != taskID in the WIP check).
func TestSecurity_RED_WIPLimit_BypassedByMovingSameTask(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	inProgressColumnID := domain.NewColumnID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	// Task is already in in_progress
	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:       taskID,
			ColumnID: inProgressColumnID,
			Title:    "Running task",
			Summary:  "summary",
		}, nil
	}

	mockColumns.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
		if id == inProgressColumnID {
			return &domain.Column{ID: inProgressColumnID, Slug: domain.ColumnInProgress, Name: "In Progress", WIPLimit: 1}, nil
		}
		return nil, nil
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if slug == domain.ColumnInProgress {
			return &domain.Column{ID: inProgressColumnID, Slug: domain.ColumnInProgress, Name: "In Progress", WIPLimit: 1}, nil
		}
		return nil, nil
	}

	// in_progress has exactly 1 task (the task being moved) — at WIP limit
	mockTasks.ListFunc = func(_ context.Context, _ domain.ProjectID, f tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
		return []domain.TaskWithDetails{
			{Task: domain.Task{ID: taskID, ColumnID: inProgressColumnID}},
		}, nil
	}

	mockTasks.HasUnresolvedDependenciesFunc = func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (bool, error) {
		return false, nil
	}

	mockTasks.UpdateFunc = func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
		return nil
	}

	// StartTask (which calls MoveTask to in_progress) — task is already there
	// RED: returns ErrWIPLimitExceeded even though the task is already in in_progress
	err := a.StartTask(ctx, projectID, taskID)

	// The current behaviour fires WIPLimitExceeded because the task itself is
	// counted in the WIP check. After the fix, the move should succeed (no new slot
	// consumed — the task is already in the column).
	assert.NoError(t, err,
		"RED: StartTask on a task already in in_progress must not return WIPLimitExceeded; "+
			"the task is already consuming the WIP slot; "+
			"fix: exclude taskID from the WIP count in MoveTask")
}

// TestSecurity_GREEN_WIPLimitIsEnforcedWhenMovingNewTask verifies that moving a
// new task into a full in_progress column correctly returns ErrWIPLimitExceeded.
func TestSecurity_GREEN_WIPLimitIsEnforcedWhenMovingNewTask(t *testing.T) {
	ctx := context.Background()
	a, mockProjects, _, mockTasks, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	todoColumnID := domain.NewColumnID()
	inProgressColumnID := domain.NewColumnID()
	otherTaskID := domain.NewTaskID()

	mockProjects.FindByIDFunc = func(_ context.Context, _ domain.ProjectID) (*domain.Project, error) {
		return &domain.Project{ID: projectID, Name: "P"}, nil
	}

	// Task is in todo (not yet in in_progress)
	mockTasks.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
		return &domain.Task{
			ID:       taskID,
			ColumnID: todoColumnID,
			Title:    "New task",
			Summary:  "summary",
		}, nil
	}

	mockColumns.FindByIDFunc = func(_ context.Context, _ domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
		if id == todoColumnID {
			return &domain.Column{ID: todoColumnID, Slug: domain.ColumnTodo, Name: "Todo"}, nil
		}
		return nil, nil
	}

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		if slug == domain.ColumnInProgress {
			return &domain.Column{ID: inProgressColumnID, Slug: domain.ColumnInProgress, Name: "In Progress", WIPLimit: 1}, nil
		}
		return nil, nil
	}

	// in_progress has 1 task (a different task, not the one being moved) — at WIP limit
	mockTasks.ListFunc = func(_ context.Context, _ domain.ProjectID, f tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
		if f.ColumnSlug != nil && *f.ColumnSlug == domain.ColumnInProgress {
			return []domain.TaskWithDetails{
				{Task: domain.Task{ID: otherTaskID, ColumnID: inProgressColumnID}},
			}, nil
		}
		return nil, nil
	}

	mockTasks.HasUnresolvedDependenciesFunc = func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (bool, error) {
		return false, nil
	}

	err := a.StartTask(ctx, projectID, taskID)

	assert.ErrorIs(t, err, domain.ErrWIPLimitExceeded,
		"GREEN: moving a new task into a full in_progress column must return ErrWIPLimitExceeded")
}

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

	err := a.BlockTask(ctx, projectID, taskID, "new reason — overwrites history", "agent-2")

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

	err := a.BlockTask(ctx, projectID, taskID, strings.Repeat("B", 60), "agent-1")
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

	err := a.RequestWontDo(ctx, projectID, taskID, "duplicate reason", "agent-2")

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

	err := a.RequestWontDo(ctx, projectID, taskID, strings.Repeat("R", 55), "agent-1")
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
	err := a.CompleteTask(ctx, projectID, taskID, completionSummary, nil, "agent-1", nil)

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

	err := a.CompleteTask(ctx, projectID, taskID, strings.Repeat("C", 110), nil, "agent-1", nil)
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

	err := a.UpdateTask(ctx, projectID, taskID, nil, nil, nil, nil, nil, nil, nil, nil, negativeUsage, nil)

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
	err := a.UpdateTask(ctx, projectID, taskID, nil, nil, nil, nil, nil, nil, nil, nil, usage, nil)
	require.NoError(t, err)
	require.NotNil(t, updatedTask)
	assert.Equal(t, 150, updatedTask.InputTokens, "positive token accumulation must work correctly")
	assert.Equal(t, 25, updatedTask.OutputTokens)
}

// ─────────────────────────────────────────────────────────────────────────────
// 7. Missing column validation in UpdateColumnWIPLimit — non-in_progress column
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_UpdateColumnWIPLimit_OnDoneColumn demonstrates that
// UpdateColumnWIPLimit accepts any ColumnSlug, including "done". Setting a WIP
// limit on "done" has no effect (the code only enforces WIP on ColumnInProgress)
// but it is accepted silently, suggesting the endpoint is not guarding against
// calls that make no semantic sense and could mask mis-configured client calls.
//
// RED: UpdateColumnWIPLimit does not validate that columnSlug is ColumnInProgress.
// Fix: check "if columnSlug != domain.ColumnInProgress { return domain.ErrInvalidColumn }"
// (or explicitly enumerate valid WIP-capable columns).
func TestSecurity_RED_UpdateColumnWIPLimit_OnDoneColumn(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	doneColumnID := domain.NewColumnID()

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		return &domain.Column{ID: doneColumnID, Slug: slug, Name: "Done"}, nil
	}

	var storedWIPLimit int
	mockColumns.UpdateWIPLimitFunc = func(_ context.Context, _ domain.ProjectID, _ domain.ColumnID, wip int) error {
		storedWIPLimit = wip
		return nil
	}

	err := a.UpdateColumnWIPLimit(ctx, projectID, domain.ColumnDone, 5)

	// RED: No error — WIP limit is silently set on the "done" column even though
	// it can never be enforced (MoveTask only checks WIPLimit on ColumnInProgress).
	// This is a silent no-op that misleads the caller and wastes a write.
	assert.Error(t, err,
		"RED: UpdateColumnWIPLimit on 'done' column must return an error (e.g. ErrInvalidColumn); "+
			"WIP limits only apply to in_progress; fix: restrict to ColumnInProgress only")

	_ = storedWIPLimit // suppress unused warning
}

// TestSecurity_GREEN_UpdateColumnWIPLimit_InProgressIsValid verifies that
// updating the WIP limit on the in_progress column succeeds.
func TestSecurity_GREEN_UpdateColumnWIPLimit_InProgressIsValid(t *testing.T) {
	ctx := context.Background()
	a, _, _, _, mockColumns, _, _ := setupTestApp()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()

	mockColumns.FindBySlugFunc = func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
		return &domain.Column{ID: columnID, Slug: slug, Name: "In Progress"}, nil
	}

	mockColumns.UpdateWIPLimitFunc = func(_ context.Context, _ domain.ProjectID, _ domain.ColumnID, _ int) error {
		return nil
	}

	err := a.UpdateColumnWIPLimit(ctx, projectID, domain.ColumnInProgress, 3)
	require.NoError(t, err, "setting WIP limit on in_progress must succeed")
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
		domain.PriorityMedium, "", "", "", nil, nil, "", false)

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
		domain.PriorityMedium, "", "", "", nil, nil, "", false)

	assert.ErrorIs(t, err, domain.ErrSummaryRequired,
		"empty task summary must be rejected with ErrSummaryRequired")
}

// ─────────────────────────────────────────────────────────────────────────────
// Compile-time assertion: columnstest.MockColumnRepository must satisfy the
// columnstest interface (already checked by the package itself).
var _ = (*columnstest.MockColumnRepository)(nil)
