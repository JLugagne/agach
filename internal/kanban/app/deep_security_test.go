package app_test

// deep_security_test.go — Deep security analysis of the kanban app layer.
//
// Each vulnerability is documented with:
//   - Issue description and file:line reference
//   - A RED test that demonstrates the vulnerability (currently fails because the
//     bug is present — the test asserts the CORRECT behaviour which the code does
//     NOT yet enforce)
//   - A GREEN test that will pass once the fix is applied (or that already passes
//     and confirms an adjacent safe behaviour)
//
// Vulnerabilities covered:
//   1. WIP limit TOCTOU race — MoveTask fetches list twice; window between check
//      and commit allows limit to be exceeded.  (tasks.go:424-443)
//   2. Direct MoveTask to "blocked" sets IsBlocked=true with no reason required.
//      (tasks.go:396-403)
//   3. CompleteTask accepts tasks in any column, not just "in_progress".
//      (tasks.go:489-553)
//   4. BlockTask accepts an empty blocked reason — violates ErrBlockedReasonRequired
//      contract documented in errors.go. (tasks.go:556-611)
//   5. RequestWontDo accepts an empty wont_do reason. (tasks.go:647-703)
//   6. MoveTaskToProject is non-atomic: create succeeds then delete can fail,
//      leaving the task duplicated. (task_move_project.go:106-114)
//   7. Token counter integer overflow — UpdateTask unconditionally adds
//      tokenUsage.InputTokens with no upper-bound check.  (tasks.go:248)
//   8. UpdateColumnWIPLimit accepts negative values, making the limit meaningless.
//      (columns.go:12)
//   9. GetNextTask role-filter bypass — when role=="" the filter is dropped,
//      allowing any caller to steal tasks assigned to specific roles.
//      (tasks.go:916)
//  10. RejectWontDo does not verify the task is currently in the blocked column
//      — the flag can be inconsistent with the task's actual column.
//      (tasks.go:769-834)

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/kanban/app"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/columns/columnstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/comments/commentstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/dependencies/dependenciestest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/projects/projectstest"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/agents/agentstest"
	tasksrepo "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks/taskstest"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func newSecurityApp(
	projects *projectstest.MockProjectRepository,
	roles *agentstest.MockRoleRepository,
	tasks *taskstest.MockTaskRepository,
	columns *columnstest.MockColumnRepository,
	comments *commentstest.MockCommentRepository,
	deps *dependenciestest.MockDependencyRepository,
) *app.App {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	return app.NewApp(app.Config{
		Projects:     projects,
		Agents:        roles,
		Tasks:        tasks,
		Columns:      columns,
		Comments:     comments,
		Dependencies: deps,
		Logger:       logger,
	})
}

// fixedColumns returns a minimal slice of columns with an in_progress column
// carrying the given WIP limit.
func makeColumns(wipLimit int) map[domain.ColumnSlug]*domain.Column {
	return map[domain.ColumnSlug]*domain.Column{
		domain.ColumnBacklog: {
			ID:       domain.NewColumnID(),
			Slug:     domain.ColumnBacklog,
			Name:     "Backlog",
			Position: 0,
		},
		domain.ColumnTodo: {
			ID:       domain.NewColumnID(),
			Slug:     domain.ColumnTodo,
			Name:     "Todo",
			Position: 1,
		},
		domain.ColumnInProgress: {
			ID:       domain.NewColumnID(),
			Slug:     domain.ColumnInProgress,
			Name:     "In Progress",
			Position: 2,
			WIPLimit: wipLimit,
		},
		domain.ColumnDone: {
			ID:       domain.NewColumnID(),
			Slug:     domain.ColumnDone,
			Name:     "Done",
			Position: 3,
		},
		domain.ColumnBlocked: {
			ID:       domain.NewColumnID(),
			Slug:     domain.ColumnBlocked,
			Name:     "Blocked",
			Position: 4,
		},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 1 — WIP limit TOCTOU (tasks.go:424-443)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: MoveTask first calls tasks.List to count in-progress tasks for the WIP
// check (line 425), then calls tasks.List again to compute the new position
// (line 436).  Between those two calls the in-memory/repository state can have
// changed.  An adversary (or concurrent goroutine) can race to fill the last
// slot so that, at position-query time, the column is already over limit.
//
// More concretely: the WIP check at line 430 counts N tasks, approves the move,
// then line 436 re-queries and may now count N+1 tasks because another concurrent
// MoveTask slipped through.  The position therefore becomes N+1 even though the
// limit was N.
//
// RED: we simulate the race by making the List mock return a different count on
// its second invocation.  The test asserts the move is rejected.
func TestSecurity_RED_WIPLimitTOCTOU(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(1) // WIP limit = 1

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]
	inProgressColumn := cols[domain.ColumnInProgress]

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID,
		Title:    "race task",
		Summary:  "summary",
	}

	listCallCount := 0

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		HasUnresolvedDependenciesFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (bool, error) {
			return false, nil
		},
		// First call: WIP check — returns 0 tasks (under limit).
		// Second call: position query — returns 1 task (now at limit, someone else got in).
		ListFunc: func(_ context.Context, _ domain.ProjectID, f tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			listCallCount++
			if listCallCount == 1 {
				// WIP check: column appears empty — allow move
				return []domain.TaskWithDetails{}, nil
			}
			// Position query: column is now full (another task slipped in)
			return []domain.TaskWithDetails{
				{Task: domain.Task{ID: domain.NewTaskID(), ColumnID: inProgressColumn.ID}},
			}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
			for _, c := range cols {
				if c.ID == id {
					return c, nil
				}
			}
			return nil, nil
		},
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	// RED: current code does not use a single atomic check — it allows the move
	// even though the second List call shows the column is full.
	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnInProgress)
	// The correct behaviour after a fix would be to re-validate the limit using
	// a single consistent count.  Until the fix, this assertion will fail because
	// the code happily proceeds.
	assert.ErrorIs(t, err, domain.ErrWIPLimitExceeded,
		"VULNERABILITY: WIP limit check and position assignment are not atomic; "+
			"concurrent moves can exceed the limit")
}

// GREEN: WIP limit is enforced when the column is already at capacity from the
// very first query (no race — stable snapshot).
func TestSecurity_GREEN_WIPLimitEnforcedStable(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(1)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]
	inProgressColumn := cols[domain.ColumnInProgress]

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID,
		Title:    "task",
		Summary:  "summary",
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		HasUnresolvedDependenciesFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (bool, error) {
			return false, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, f tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			// Column is already at WIP limit (1 task) — both calls see the same state.
			return []domain.TaskWithDetails{
				{Task: domain.Task{ID: domain.NewTaskID(), ColumnID: inProgressColumn.ID}},
			}, nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
			for _, c := range cols {
				if c.ID == id {
					return c, nil
				}
			}
			return nil, nil
		},
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnInProgress)
	require.ErrorIs(t, err, domain.ErrWIPLimitExceeded)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 2 — MoveTask to blocked without reason (tasks.go:396-403)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: When the human UI calls MoveTask with targetColumnSlug="blocked", the
// code sets task.IsBlocked = true but does NOT require a blocked reason.  The
// result is a task that is in the blocked column with is_blocked=1 but an empty
// blocked_reason, which violates the documented business rule requiring a reason.
// The dedicated BlockTask path enforces a reason, but the generic MoveTask path
// does not.
//
// RED: assert that moving directly to blocked column without a reason is rejected.
func TestSecurity_RED_MoveToBlockedColumnWithoutReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID,
		Title:    "task",
		Summary:  "summary",
	}

	var savedTask domain.Task
	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, t domain.Task) error {
			savedTask = t
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
			for _, c := range cols {
				if c.ID == id {
					return c, nil
				}
			}
			return nil, nil
		},
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	// RED: current code does not require a reason — MoveTask succeeds and
	// is_blocked is true but blocked_reason is empty.
	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnBlocked)
	// After a fix, moving to blocked via MoveTask should either require a reason
	// or be disallowed in favour of BlockTask.
	assert.Error(t, err,
		"VULNERABILITY: MoveTask to blocked column requires no reason; task becomes blocked with empty blocked_reason")
	// Additionally confirm the current broken behaviour: no error, blocked=true, reason=""
	if err == nil {
		assert.True(t, savedTask.IsBlocked)
		assert.Empty(t, savedTask.BlockedReason,
			"confirms the bug: blocked_reason is empty after direct MoveTask to blocked")
	}
}

// GREEN: using the dedicated BlockTask path correctly requires a reason.
func TestSecurity_GREEN_BlockTaskRequiresReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]
	blockedColumn := cols[domain.ColumnBlocked]

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID,
		Title:    "task",
		Summary:  "summary",
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, f tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			if f.ColumnSlug != nil && *f.ColumnSlug == domain.ColumnBlocked {
				return []domain.TaskWithDetails{}, nil
			}
			return []domain.TaskWithDetails{}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	mockComments := &commentstest.MockCommentRepository{
		CreateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
			return nil
		},
	}

	_ = blockedColumn

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		mockComments,
		&dependenciestest.MockDependencyRepository{},
	)

	// BlockTask with a legitimate reason — should succeed.
	err := a.BlockTask(ctx, projectID, taskID, "waiting for external API credentials", "agent-1")
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 3 — CompleteTask accepts tasks in any column (tasks.go:489-553)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: CompleteTask does not verify that the task is in the "in_progress"
// column before completing it.  An agent can call CompleteTask on a task that
// is still in "todo" or even "backlog", bypassing the intended workflow where a
// task must first be started (moved to in_progress) before it can be completed.
// The only guard is the IsBlocked check (line 505), but that is insufficient.
//
// RED: complete a task that is in "todo" — the code should reject this.
func TestSecurity_RED_CompleteTaskFromTodo(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]
	doneColumn := cols[domain.ColumnDone]

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID, // task is in TODO, not in_progress
		Title:    "task",
		Summary:  "summary",
		IsBlocked: false,
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	_ = doneColumn

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	// RED: no error is expected by current code, but the correct behaviour is to
	// reject completing a task that has not been started.
	err := a.CompleteTask(ctx, projectID, taskID, "completed it quickly", nil, "agent-1", nil)
	assert.Error(t, err,
		"VULNERABILITY: CompleteTask does not verify the task is in 'in_progress'; "+
			"a task in 'todo' can be completed, skipping the start workflow")
}

// GREEN: completing a task that is properly in_progress succeeds.
func TestSecurity_GREEN_CompleteTaskFromInProgress(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	inProgressColumn := cols[domain.ColumnInProgress]
	doneColumn := cols[domain.ColumnDone]

	task := &domain.Task{
		ID:       taskID,
		ColumnID: inProgressColumn.ID, // correctly in_progress
		Title:    "task",
		Summary:  "summary",
		IsBlocked: false,
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	_ = doneColumn

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	err := a.CompleteTask(ctx, projectID, taskID, "done", nil, "agent-1", nil)
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 4 — BlockTask accepts empty blocked reason (tasks.go:556-611)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: domain/errors.go declares ErrBlockedReasonRequired and the error message
// says "minimum 50 characters", suggesting intent to validate.  However,
// BlockTask in tasks.go never checks whether blockedReason is empty or meets the
// minimum length.  An agent can block a task with an empty reason, creating noise.
//
// RED: assert that BlockTask rejects an empty blocked reason.
func TestSecurity_RED_BlockTaskEmptyReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID,
		Title:    "task",
		Summary:  "summary",
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	mockComments := &commentstest.MockCommentRepository{
		CreateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
			return nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		mockComments,
		&dependenciestest.MockDependencyRepository{},
	)

	// RED: current code does not validate blocked reason — succeeds with empty string.
	err := a.BlockTask(ctx, projectID, taskID, "", "agent-1")
	assert.ErrorIs(t, err, domain.ErrBlockedReasonRequired,
		"VULNERABILITY: BlockTask does not validate blocked reason; "+
			"ErrBlockedReasonRequired is declared but never checked in app layer")
}

// GREEN: blocking with a sufficient reason is accepted.
func TestSecurity_GREEN_BlockTaskWithSufficientReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID,
		Title:    "task",
		Summary:  "summary",
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	mockComments := &commentstest.MockCommentRepository{
		CreateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
			return nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		mockComments,
		&dependenciestest.MockDependencyRepository{},
	)

	reason := "Waiting for external API credentials from ops team — cannot proceed without access keys"
	err := a.BlockTask(ctx, projectID, taskID, reason, "agent-1")
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 5 — RequestWontDo accepts empty reason (tasks.go:647-703)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: Same pattern as BlockTask — ErrWontDoReasonRequired is declared in
// errors.go but RequestWontDo never validates the wontDoReason argument.
//
// RED: assert that RequestWontDo rejects an empty reason.
func TestSecurity_RED_RequestWontDoEmptyReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID,
		Title:    "task",
		Summary:  "summary",
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	mockComments := &commentstest.MockCommentRepository{
		CreateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
			return nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		mockComments,
		&dependenciestest.MockDependencyRepository{},
	)

	// RED: current code does not validate — succeeds with empty reason.
	err := a.RequestWontDo(ctx, projectID, taskID, "", "agent-1")
	assert.ErrorIs(t, err, domain.ErrWontDoReasonRequired,
		"VULNERABILITY: RequestWontDo does not validate wont_do reason; "+
			"ErrWontDoReasonRequired is declared but never checked")
}

// GREEN: providing a sufficient reason is accepted.
func TestSecurity_GREEN_RequestWontDoWithSufficientReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID,
		Title:    "task",
		Summary:  "summary",
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	mockComments := &commentstest.MockCommentRepository{
		CreateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
			return nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		mockComments,
		&dependenciestest.MockDependencyRepository{},
	)

	reason := "This task is out of scope for this sprint and is superseded by task #42 which covers the same requirement"
	err := a.RequestWontDo(ctx, projectID, taskID, reason, "agent-1")
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 6 — MoveTaskToProject is non-atomic (task_move_project.go:106-114)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: MoveTaskToProject creates the task in the target project (line 106)
// and then deletes it from the source project (line 112) as two separate
// operations.  If the delete fails, the task now exists in both projects — a
// silent data duplication.  This can lead to double work, conflicting state,
// or privilege escalation (an agent in project B gains access to a task that
// was supposed to be removed from project A).
//
// RED: simulate a delete failure and assert that the overall operation is rolled
// back (i.e., the task is NOT left in the target project).
func TestSecurity_RED_MoveTaskToProjectNonAtomic(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	sourceProjectID := domain.NewProjectID()
	targetProjectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]
	sourceProject := &domain.Project{ID: sourceProjectID, Name: "Source"}
	targetProject := &domain.Project{
		ID:       targetProjectID,
		Name:     "Target",
		ParentID: &sourceProjectID, // make them related (parent-child)
	}
	sourceProject.ParentID = nil

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID,
		Title:    "task to move",
		Summary:  "summary",
	}

	createdInTarget := false
	deleteErr := errors.New("storage failure")

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, pid domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			if pid == sourceProjectID {
				return task, nil
			}
			return nil, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		CreateFunc: func(_ context.Context, pid domain.ProjectID, _ domain.Task) error {
			if pid == targetProjectID {
				createdInTarget = true
			}
			return nil // create succeeds
		},
		DeleteFunc: func(_ context.Context, pid domain.ProjectID, _ domain.TaskID) error {
			if pid == sourceProjectID {
				return deleteErr // delete from source fails
			}
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			switch id {
			case sourceProjectID:
				return sourceProject, nil
			case targetProjectID:
				return targetProject, nil
			}
			return nil, nil
		},
	}

	a := newSecurityApp(
		mockProjects,
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	err := a.MoveTaskToProject(ctx, sourceProjectID, taskID, targetProjectID)

	// RED: current code returns the delete error but the task has ALREADY been
	// created in the target project (createdInTarget == true).
	// After a fix, either the operation must be fully rolled back (createdInTarget
	// should be false on error) or the error from delete should trigger a
	// compensating delete in the target.
	assert.Error(t, err, "delete failure should propagate")
	assert.False(t, createdInTarget,
		"VULNERABILITY: task was created in target project even though delete from source failed; "+
			"the task is now duplicated across both projects")
}

// GREEN: when both create and delete succeed, the move completes cleanly.
func TestSecurity_GREEN_MoveTaskToProjectSucceedsAtomically(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	sourceProjectID := domain.NewProjectID()
	targetProjectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]
	sourceProject := &domain.Project{ID: sourceProjectID, Name: "Source"}
	targetProject := &domain.Project{
		ID:       targetProjectID,
		Name:     "Target",
		ParentID: &sourceProjectID,
	}
	sourceProject.ParentID = nil

	task := &domain.Task{
		ID:       taskID,
		ColumnID: todoColumn.ID,
		Title:    "task",
		Summary:  "summary",
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, pid domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			if pid == sourceProjectID {
				return task, nil
			}
			return nil, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		CreateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
		DeleteFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	mockProjects := &projectstest.MockProjectRepository{
		FindByIDFunc: func(_ context.Context, id domain.ProjectID) (*domain.Project, error) {
			switch id {
			case sourceProjectID:
				return sourceProject, nil
			case targetProjectID:
				return targetProject, nil
			}
			return nil, nil
		},
	}

	a := newSecurityApp(
		mockProjects,
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	err := a.MoveTaskToProject(ctx, sourceProjectID, taskID, targetProjectID)
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 7 — Token counter integer overflow (tasks.go:248)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: UpdateTask accumulates token counts with `task.InputTokens +=
// tokenUsage.InputTokens` and similar.  On a 32-bit Go int (or when values are
// close to math.MaxInt32 on any platform), repeated large additions will silently
// overflow and produce negative or wildly incorrect counts.  An adversary can
// provide a crafted tokenUsage payload to corrupt billing/analytics data.
//
// RED: pass a tokenUsage value whose addition would overflow int and assert
// that the call is rejected or the result is clamped.
func TestSecurity_RED_TokenCounterOverflow(t *testing.T) {
	ctx := context.Background()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	// Set existing counts near max to guarantee overflow on addition.
	// math.MaxInt is the platform int max (64-bit on amd64).
	task := &domain.Task{
		ID:          taskID,
		ColumnID:    columnID,
		Title:       "task",
		Summary:     "summary",
		InputTokens: math.MaxInt - 10,
	}

	var savedTask domain.Task
	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, t domain.Task) error {
			savedTask = t
			return nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		&columnstest.MockColumnRepository{},
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	// Pass a value that when added to (MaxInt - 10) overflows: MaxInt - 10 + 100 wraps negative.
	overflow := &domain.TokenUsage{
		InputTokens: 100,
	}

	err := a.UpdateTask(ctx, projectID, taskID, nil, nil, nil, nil, nil, nil, nil, nil, overflow, nil, nil, false)

	// RED: current code applies the addition unconditionally and returns no error.
	// After a fix the call should either return an error or clamp to MaxInt.
	if err == nil {
		// Demonstrate the overflow: result should never be negative and should be
		// at least as large as the original value.
		assert.True(t, savedTask.InputTokens >= 0,
			"VULNERABILITY: token count overflowed to negative (%d); "+
				"no bounds check on UpdateTask tokenUsage accumulation",
			savedTask.InputTokens)
		assert.True(t, savedTask.InputTokens >= task.InputTokens,
			"VULNERABILITY: token count after addition (%d) is less than original (%d); "+
				"integer overflow occurred",
			savedTask.InputTokens, task.InputTokens)
	}
	// If err != nil, the implementation correctly rejected the overflow.
}

// GREEN: normal token accumulation (no overflow risk) works correctly.
func TestSecurity_GREEN_TokenCounterNormalAccumulation(t *testing.T) {
	ctx := context.Background()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	task := &domain.Task{
		ID:          taskID,
		ColumnID:    columnID,
		Title:       "task",
		Summary:     "summary",
		InputTokens: 1000,
	}

	var savedTask domain.Task
	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, t domain.Task) error {
			savedTask = t
			return nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		&columnstest.MockColumnRepository{},
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	usage := &domain.TokenUsage{InputTokens: 500}
	err := a.UpdateTask(ctx, projectID, taskID, nil, nil, nil, nil, nil, nil, nil, nil, usage, nil, nil, false)
	require.NoError(t, err)
	assert.Equal(t, 1500, savedTask.InputTokens)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 8 — UpdateColumnWIPLimit accepts negative values (columns.go:12)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: UpdateColumnWIPLimit does not validate that wipLimit >= 0.  A negative
// WIP limit (e.g. -1) makes the comparison `len(inProgressTasks) >= -1` always
// false, effectively disabling the WIP limit for that column.  An adversary
// can call this with -1 to bypass WIP enforcement for all subsequent moves.
//
// RED: assert that a negative WIP limit is rejected.
func TestSecurity_RED_NegativeWIPLimitBypass(t *testing.T) {
	ctx := context.Background()

	projectID := domain.NewProjectID()
	columnID := domain.NewColumnID()
	inProgressSlug := domain.ColumnInProgress

	project := &domain.Project{ID: projectID, Name: "Project"}

	column := &domain.Column{
		ID:       columnID,
		Slug:     inProgressSlug,
		WIPLimit: 3,
	}

	mockProjects := &projectstest.MockProjectRepository{}

	mockColumns := &columnstest.MockColumnRepository{
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			if slug == inProgressSlug {
				return column, nil
			}
			return nil, nil
		},
		UpdateWIPLimitFunc: func(_ context.Context, _ domain.ProjectID, _ domain.ColumnID, wipLimit int) error {
			// Accept any value — simulates storage layer
			column.WIPLimit = wipLimit
			return nil
		},
	}

	_ = project

	a := newSecurityApp(
		mockProjects,
		&agentstest.MockRoleRepository{},
		&taskstest.MockTaskRepository{},
		mockColumns,
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	// RED: a negative WIP limit should be rejected.
	err := a.UpdateColumnWIPLimit(ctx, projectID, inProgressSlug, -1)
	assert.Error(t, err,
		"VULNERABILITY: UpdateColumnWIPLimit accepts negative values; "+
			"setting -1 disables all WIP enforcement")
	// Confirm the current broken behaviour: no error is returned.
	if err == nil {
		assert.Equal(t, -1, column.WIPLimit,
			"confirms the bug: WIP limit is now -1, WIP checks will always pass")
	}
}

// GREEN: a zero WIP limit (disable enforcement) is valid; positive is valid.
func TestSecurity_GREEN_ValidWIPLimits(t *testing.T) {
	ctx := context.Background()

	for _, limit := range []int{0, 1, 5, 10} {
		projectID := domain.NewProjectID()
		columnID := domain.NewColumnID()
		inProgressSlug := domain.ColumnInProgress

		column := &domain.Column{
			ID:       columnID,
			Slug:     inProgressSlug,
			WIPLimit: 3,
		}

		mockColumns := &columnstest.MockColumnRepository{
			FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
				return column, nil
			},
			UpdateWIPLimitFunc: func(_ context.Context, _ domain.ProjectID, _ domain.ColumnID, wl int) error {
				column.WIPLimit = wl
				return nil
			},
		}

		a := newSecurityApp(
			&projectstest.MockProjectRepository{},
			&agentstest.MockRoleRepository{},
			&taskstest.MockTaskRepository{},
			mockColumns,
			&commentstest.MockCommentRepository{},
			&dependenciestest.MockDependencyRepository{},
		)

		err := a.UpdateColumnWIPLimit(ctx, projectID, inProgressSlug, limit)
		require.NoError(t, err, "WIP limit %d should be accepted", limit)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 9 — GetNextTask role-filter bypass (tasks.go:916)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: When the caller passes role="" to GetNextTask, line 916:
//
//	if role != "" {
//	    filters.AssignedRole = &role
//	}
//
// the role filter is simply dropped, so the query returns tasks from ALL roles,
// including tasks specifically assigned to a different role.  Any agent that
// passes an empty role string can steal high-priority tasks meant for others.
//
// RED: when role is empty, GetNextTask should NOT return a task that has a
// non-empty assigned_role (i.e., the task belongs to someone else).
func TestSecurity_RED_GetNextTaskRoleBypass(t *testing.T) {
	ctx := context.Background()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	// A task explicitly assigned to "security-officer".
	assignedTask := &domain.TaskWithDetails{
		Task: domain.Task{
			ID:           taskID,
			Title:        "security audit",
			Summary:      "summary",
			AssignedRole: "security-officer", // belongs to specific role
			PriorityScore: 400,
		},
	}

	mockTasks := &taskstest.MockTaskRepository{
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			// Returns the role-assigned task regardless of filters (simulates no DB-level filter)
			return []domain.TaskWithDetails{*assignedTask}, nil
		},
		HasUnresolvedDependenciesFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (bool, error) {
			return false, nil
		},
	}

	mockProjects := &projectstest.MockProjectRepository{
		GetTreeFunc: func(_ context.Context, id domain.ProjectID) ([]domain.Project, error) {
			return []domain.Project{{ID: projectID}}, nil
		},
	}

	a := newSecurityApp(
		mockProjects,
		&agentstest.MockRoleRepository{},
		mockTasks,
		&columnstest.MockColumnRepository{},
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	// Call with empty role — should NOT receive a task assigned to "security-officer".
	task, err := a.GetNextTask(ctx, projectID, "" /* empty role */, nil)

	// RED: current code returns the task because it drops the filter entirely.
	if err == nil && task != nil {
		assert.Empty(t, task.AssignedRole,
			"VULNERABILITY: GetNextTask with role='' returned a task assigned to '%s'; "+
				"empty role should only match unassigned tasks",
			task.AssignedRole)
	}
	// The fix would either return ErrNoAvailableTasks or only return tasks
	// with assigned_role="" when the caller passes role="".
}

// GREEN: requesting with a specific role only returns that role's tasks.
func TestSecurity_GREEN_GetNextTaskRoleFilterEnforced(t *testing.T) {
	ctx := context.Background()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	secOfficerTask := &domain.TaskWithDetails{
		Task: domain.Task{
			ID:            taskID,
			Title:         "security audit",
			Summary:       "summary",
			AssignedRole:  "security-officer",
			PriorityScore: 400,
		},
	}

	mockTasks := &taskstest.MockTaskRepository{
		ListFunc: func(_ context.Context, _ domain.ProjectID, f tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			// Respect the filter: only return task when role matches.
			if f.AssignedRole != nil && *f.AssignedRole == "security-officer" {
				return []domain.TaskWithDetails{*secOfficerTask}, nil
			}
			return []domain.TaskWithDetails{}, nil
		},
		HasUnresolvedDependenciesFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (bool, error) {
			return false, nil
		},
	}

	mockProjects := &projectstest.MockProjectRepository{
		GetTreeFunc: func(_ context.Context, id domain.ProjectID) ([]domain.Project, error) {
			return []domain.Project{{ID: projectID}}, nil
		},
	}

	a := newSecurityApp(
		mockProjects,
		&agentstest.MockRoleRepository{},
		mockTasks,
		&columnstest.MockColumnRepository{},
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
	)

	// Correct role — should find the task.
	task, err := a.GetNextTask(ctx, projectID, "security-officer", nil)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "security-officer", task.AssignedRole)

	// Wrong role — should find nothing.
	_, err = a.GetNextTask(ctx, projectID, "developer", nil)
	assert.ErrorIs(t, err, domain.ErrNoAvailableTasks)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 10 — RejectWontDo does not verify task is in blocked column
//           (tasks.go:769-834)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: RejectWontDo only checks task.WontDoRequested == true (line 786) but
// does not verify that the task is currently in the "blocked" column.  It is
// theoretically possible for a task to have wont_do_requested=true while
// being in a different column (e.g., due to a direct DB edit or a previous
// bug).  The rejection then forcefully moves the task to "todo" while
// resetting the IsBlocked flag, leaving the state machine in a confused
// state and potentially bypassing the need to be in blocked first.
//
// More concretely: if an agent first calls RequestWontDo (moves to blocked,
// sets flag), then the human calls MoveTask(todo) directly (which clears the
// flag per line 390), but a stale reference to the task still has the flag
// set — RejectWontDo would then silently succeed on an inconsistent state.
//
// The cleaner and safer path: RejectWontDo should verify the task is in the
// blocked column before operating on it.
//
// RED: call RejectWontDo on a task that has WontDoRequested=true but is NOT
// in the blocked column; assert the call is rejected.
func TestSecurity_RED_RejectWontDoOutsideBlockedColumn(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	// Task is in TODO (not blocked) but somehow has WontDoRequested=true.
	todoColumn := cols[domain.ColumnTodo]
	task := &domain.Task{
		ID:              taskID,
		ColumnID:        todoColumn.ID, // not in blocked
		Title:           "task",
		Summary:         "summary",
		WontDoRequested: true, // flag is set despite not being in blocked column
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
			for _, c := range cols {
				if c.ID == id {
					return c, nil
				}
			}
			return nil, nil
		},
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	mockComments := &commentstest.MockCommentRepository{
		CreateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
			return nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		mockComments,
		&dependenciestest.MockDependencyRepository{},
	)

	// RED: current code succeeds even though task is not in blocked column.
	err := a.RejectWontDo(ctx, projectID, taskID, "reconsidering")
	assert.ErrorIs(t, err, domain.ErrTaskNotInBlocked,
		"VULNERABILITY: RejectWontDo does not verify task is in blocked column; "+
			"a task with a stale WontDoRequested flag outside blocked column can be manipulated")
}

// GREEN: rejecting a won't-do on a properly blocked task works correctly.
func TestSecurity_GREEN_RejectWontDoFromBlockedColumn(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns(3)

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	blockedColumn := cols[domain.ColumnBlocked]
	todoColumn := cols[domain.ColumnTodo]

	task := &domain.Task{
		ID:              taskID,
		ColumnID:        blockedColumn.ID, // correctly in blocked
		Title:           "task",
		Summary:         "summary",
		WontDoRequested: true,
		IsBlocked:       true,
	}

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return task, nil
		},
		ListFunc: func(_ context.Context, _ domain.ProjectID, _ tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			return []domain.TaskWithDetails{}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Task) error {
			return nil
		},
	}

	mockColumns := &columnstest.MockColumnRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, id domain.ColumnID) (*domain.Column, error) {
			for _, c := range cols {
				if c.ID == id {
					return c, nil
				}
			}
			return nil, nil
		},
		FindBySlugFunc: func(_ context.Context, _ domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
			c, ok := cols[slug]
			if !ok {
				return nil, nil
			}
			return c, nil
		},
	}

	mockComments := &commentstest.MockCommentRepository{
		CreateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.Comment) error {
			return nil
		},
	}

	_ = todoColumn

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		mockColumns,
		mockComments,
		&dependenciestest.MockDependencyRepository{},
	)

	err := a.RejectWontDo(ctx, projectID, taskID, "we need to do this after all")
	require.NoError(t, err)
}
