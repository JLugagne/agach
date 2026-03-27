package security_test

// deep_security_test.go — Deep security analysis of the kanban app layer.
//
// Each vulnerability is documented with:
//   - Issue description and file:line reference
//   - A test that asserts the correct (fixed) behaviour
//   - A GREEN test that confirms adjacent safe behaviour
//
// Vulnerabilities covered:
//   1. (Removed — WIP limits no longer enforced)
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
//   8. (Removed — WIP limits no longer enforced)
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

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents/agentstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/columns/columnstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/comments/commentstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dependencies/dependenciestest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects/projectstest"
	tasksrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks/taskstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 1 — WIP limit TOCTOU (tasks.go:424-443)
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
// Asserts that moving directly to blocked column without a reason is rejected.
func TestSecurity_MoveToBlockedColumnWithoutReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

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

	// MoveTask to blocked without a reason must be rejected.
	err := a.MoveTask(ctx, projectID, taskID, domain.ColumnBlocked, "")
	assert.Error(t, err,
		"MoveTask to blocked column must require a reason or be disallowed in favour of BlockTask")
	_ = savedTask
}

// GREEN: using the dedicated BlockTask path correctly requires a reason.
func TestSecurity_GREEN_BlockTaskRequiresReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

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
	err := a.BlockTask(ctx, projectID, taskID, "waiting for external API credentials", "agent-1", "")
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
// Asserts that completing a task in "todo" is rejected.
func TestSecurity_CompleteTaskFromTodo(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	todoColumn := cols[domain.ColumnTodo]
	doneColumn := cols[domain.ColumnDone]

	task := &domain.Task{
		ID:        taskID,
		ColumnID:  todoColumn.ID, // task is in TODO, not in_progress
		Title:     "task",
		Summary:   "summary",
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

	err := a.CompleteTask(ctx, projectID, taskID, "completed it quickly", nil, "agent-1", nil, "")
	assert.Error(t, err,
		"CompleteTask must reject a task that is not in the 'in_progress' column")
}

// GREEN: completing a task that is properly in_progress succeeds.
func TestSecurity_GREEN_CompleteTaskFromInProgress(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	inProgressColumn := cols[domain.ColumnInProgress]
	doneColumn := cols[domain.ColumnDone]

	task := &domain.Task{
		ID:        taskID,
		ColumnID:  inProgressColumn.ID, // correctly in_progress
		Title:     "task",
		Summary:   "summary",
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

	err := a.CompleteTask(ctx, projectID, taskID, "done", nil, "agent-1", nil, "")
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
// Asserts that BlockTask rejects an empty blocked reason.
func TestSecurity_BlockTaskEmptyReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

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

	err := a.BlockTask(ctx, projectID, taskID, "", "agent-1", "")
	assert.ErrorIs(t, err, domain.ErrBlockedReasonRequired,
		"BlockTask must reject an empty blocked reason with ErrBlockedReasonRequired")
}

// GREEN: blocking with a sufficient reason is accepted.
func TestSecurity_GREEN_BlockTaskWithSufficientReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

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
	err := a.BlockTask(ctx, projectID, taskID, reason, "agent-1", "")
	require.NoError(t, err)
}

// ─────────────────────────────────────────────────────────────────────────────
// Vuln 5 — RequestWontDo accepts empty reason (tasks.go:647-703)
// ─────────────────────────────────────────────────────────────────────────────
//
// Issue: Same pattern as BlockTask — ErrWontDoReasonRequired is declared in
// errors.go but RequestWontDo never validates the wontDoReason argument.
//
// Asserts that RequestWontDo rejects an empty reason.
func TestSecurity_RequestWontDoEmptyReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

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

	err := a.RequestWontDo(ctx, projectID, taskID, "", "agent-1", "")
	assert.ErrorIs(t, err, domain.ErrWontDoReasonRequired,
		"RequestWontDo must reject an empty reason with ErrWontDoReasonRequired")
}

// GREEN: providing a sufficient reason is accepted.
func TestSecurity_GREEN_RequestWontDoWithSufficientReason(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

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
	err := a.RequestWontDo(ctx, projectID, taskID, reason, "agent-1", "")
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
// Simulates a delete failure and asserts that the task is NOT left in the target
// project (the operation is rolled back).
func TestSecurity_MoveTaskToProjectNonAtomic(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

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

	assert.Error(t, err, "delete failure must propagate")
	assert.False(t, createdInTarget,
		"MoveTaskToProject must not leave a task in the target project when the source delete fails")
}

// GREEN: when both create and delete succeed, the move completes cleanly.
func TestSecurity_GREEN_MoveTaskToProjectSucceedsAtomically(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

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
// Passes a tokenUsage value whose addition would overflow int and asserts
// that the call is rejected or the result is clamped.
func TestSecurity_TokenCounterOverflow(t *testing.T) {
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

	// The fix either returns an error or clamps to MaxInt.
	if err == nil {
		assert.True(t, savedTask.InputTokens >= 0,
			"token count must not overflow to negative (got %d)",
			savedTask.InputTokens)
		assert.True(t, savedTask.InputTokens >= task.InputTokens,
			"token count after addition (%d) must not be less than original (%d)",
			savedTask.InputTokens, task.InputTokens)
	}
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
// Asserts that when role is empty, GetNextTask does NOT return a task that has a
// non-empty assigned_role (i.e., the task belongs to someone else).
func TestSecurity_GetNextTaskRoleBypass(t *testing.T) {
	ctx := context.Background()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	// A task explicitly assigned to "security-officer".
	assignedTask := &domain.TaskWithDetails{
		Task: domain.Task{
			ID:            taskID,
			Title:         "security audit",
			Summary:       "summary",
			AssignedRole:  "security-officer", // belongs to specific role
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

	// Call with empty role — must NOT receive a task assigned to "security-officer".
	task, err := a.GetNextTask(ctx, projectID, "" /* empty role */, nil)

	if err == nil && task != nil {
		assert.Empty(t, task.AssignedRole,
			"GetNextTask with role='' must not return a task assigned to '%s'; "+
				"empty role should only match unassigned tasks",
			task.AssignedRole)
	}
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
//
//	(tasks.go:769-834)
//
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
// Calls RejectWontDo on a task that has WontDoRequested=true but is NOT
// in the blocked column; asserts the call is rejected.
func TestSecurity_RejectWontDoOutsideBlockedColumn(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

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

	err := a.RejectWontDo(ctx, projectID, taskID, "reconsidering")
	assert.ErrorIs(t, err, domain.ErrTaskNotInBlocked,
		"RejectWontDo must reject a task that is not in the blocked column")
}

// GREEN: rejecting a won't-do on a properly blocked task works correctly.
func TestSecurity_GREEN_RejectWontDoFromBlockedColumn(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

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
