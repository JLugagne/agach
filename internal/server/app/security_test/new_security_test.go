package security_test

// new_security_test.go — Additional security tests for the app (service) layer.
//
// Vulnerabilities covered (NOT already in existing security tests):
//  1. CompleteTask accepts empty completion summary (task_service.go:577)
//  2. CompleteTask does not validate negative token values (task_service.go:589)
//  3. UpdateTask can set title to empty string (task_service.go:277)
//  4. UpdateFeature allows empty name (feature_service.go:65)
//  5. AddDependency does not check self-dependency (dependency_service.go:28)
//  6. GrantUserAccess no role validation (project_access.go:21)
//  7. ReorderTask accepts negative position (task_service.go:505)
//  8. FeatureStatus transition not validated — any transition allowed (feature_service.go:72)
//  9. ApproveWontDo does not set CompletionSummary (task_service.go:815-823)

import (
	"context"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/server/app"
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/agents/agentstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/columns/columnstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/comments/commentstest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/dependencies/dependenciestest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/features/featurestest"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/projects/projectstest"
	tasksrepo "github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/server/domain/repositories/tasks/taskstest"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProjectAccessRepository is a minimal mock for ProjectAccessRepository.
type mockProjectAccessRepository struct {
	grantUserFunc func(ctx context.Context, projectID domain.ProjectID, userID, role string) error
}

func (m *mockProjectAccessRepository) GrantUser(ctx context.Context, projectID domain.ProjectID, userID, role string) error {
	if m.grantUserFunc != nil {
		return m.grantUserFunc(ctx, projectID, userID, role)
	}
	return nil
}
func (m *mockProjectAccessRepository) RevokeUser(_ context.Context, _ domain.ProjectID, _ string) error {
	return nil
}
func (m *mockProjectAccessRepository) UpdateUserRole(_ context.Context, _ domain.ProjectID, _, _ string) error {
	return nil
}
func (m *mockProjectAccessRepository) GrantTeam(_ context.Context, _ domain.ProjectID, _ string) error {
	return nil
}
func (m *mockProjectAccessRepository) RevokeTeam(_ context.Context, _ domain.ProjectID, _ string) error {
	return nil
}
func (m *mockProjectAccessRepository) ListUserAccess(_ context.Context, _ domain.ProjectID) ([]domain.ProjectUserAccess, error) {
	return nil, nil
}
func (m *mockProjectAccessRepository) ListTeamAccess(_ context.Context, _ domain.ProjectID) ([]domain.ProjectTeamAccess, error) {
	return nil, nil
}
func (m *mockProjectAccessRepository) HasAccess(_ context.Context, _ domain.ProjectID, _ string, _ []string) (bool, error) {
	return false, nil
}
func (m *mockProjectAccessRepository) ListAccessibleProjectIDs(_ context.Context, _ string, _ []string) ([]domain.ProjectID, error) {
	return nil, nil
}

// newAppWithFeatures creates a test app with features mock included.
func newAppWithFeatures(
	projects *projectstest.MockProjectRepository,
	roles *agentstest.MockRoleRepository,
	tasks *taskstest.MockTaskRepository,
	columns *columnstest.MockColumnRepository,
	comments *commentstest.MockCommentRepository,
	deps *dependenciestest.MockDependencyRepository,
	features *featurestest.MockFeature,
) *app.App {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	return app.NewApp(app.Config{
		Projects:     projects,
		Agents:       roles,
		Tasks:        tasks,
		Columns:      columns,
		Comments:     comments,
		Dependencies: deps,
		Features:     features,
		Logger:       logger,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. CompleteTask accepts empty completion summary
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_CompleteTask_EmptyCompletionSummary verifies that CompleteTask
// rejects an empty completionSummary. The domain defines ErrCompletionSummaryRequired
// and the fix enforces it at the app layer.
//
// Vulnerability that was fixed: task_service.go — completionSummary was assigned to
// task.CompletionSummary without any length or emptiness check.
func TestSecurity_CompleteTask_EmptyCompletionSummary(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	inProgressColumn := cols[domain.ColumnInProgress]

	task := &domain.Task{
		ID:        taskID,
		ColumnID:  inProgressColumn.ID,
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

	// Empty completion summary — should be rejected.
	err := a.CompleteTask(ctx, projectID, taskID, "" /* empty */, nil, "agent-1", nil, "")

	assert.ErrorIs(t, err, domain.ErrCompletionSummaryRequired,
		"CompleteTask must reject an empty completion summary with ErrCompletionSummaryRequired")
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. CompleteTask does not validate negative token values
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_CompleteTask_NegativeTokenValues verifies that CompleteTask
// rejects negative token values. The same guard present in UpdateTask is now
// also enforced in CompleteTask.
//
// Vulnerability that was fixed: task_service.go — "task.InputTokens += tokenUsage.InputTokens"
// was applied without checking tokenUsage.InputTokens >= 0.
func TestSecurity_CompleteTask_NegativeTokenValues(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	inProgressColumn := cols[domain.ColumnInProgress]

	task := &domain.Task{
		ID:          taskID,
		ColumnID:    inProgressColumn.ID,
		Title:       "task",
		Summary:     "summary",
		IsBlocked:   false,
		InputTokens: 1000,
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

	negativeUsage := &domain.TokenUsage{
		InputTokens:  -5000,
		OutputTokens: -100,
	}

	err := a.CompleteTask(ctx, projectID, taskID, "completed the task with a proper summary", nil, "agent-1", negativeUsage, "")

	if err == nil {
		assert.GreaterOrEqual(t, savedTask.InputTokens, 0,
			"CompleteTask must not let negative token values corrupt the counter (got %d)",
			savedTask.InputTokens)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. UpdateTask can set title to empty string
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_UpdateTask_EmptyTitleAccepted verifies that UpdateTask rejects
// an empty title string passed via the title pointer parameter. CreateTask
// enforces ErrTaskTitleRequired and UpdateTask now does too.
//
// Vulnerability that was fixed: task_service.go — title was assigned without
// checking for empty string.
func TestSecurity_UpdateTask_EmptyTitleAccepted(t *testing.T) {
	ctx := context.Background()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return &domain.Task{
				ID:       taskID,
				ColumnID: columnID,
				Title:    "Original title",
				Summary:  "summary",
			}, nil
		},
		UpdateFunc: func(_ context.Context, _ domain.ProjectID, t domain.Task) error {
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

	emptyTitle := ""
	err := a.UpdateTask(ctx, projectID, taskID, &emptyTitle, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, false)

	assert.ErrorIs(t, err, domain.ErrTaskTitleRequired,
		"UpdateTask must reject an empty title with ErrTaskTitleRequired")
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. UpdateFeature allows empty name
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_UpdateFeature_EmptyNameAccepted verifies that UpdateFeature rejects
// an empty feature name. CreateFeature enforces ErrFeatureNameRequired and
// UpdateFeature now does too.
//
// Vulnerability that was fixed: feature_service.go — "feature.Name = name" was
// set without an empty check.
func TestSecurity_UpdateFeature_EmptyNameAccepted(t *testing.T) {
	ctx := context.Background()

	featureID := domain.NewFeatureID()
	projectID := domain.NewProjectID()

	mockFeatures := &featurestest.MockFeature{
		FindByIDFunc: func(_ context.Context, _ domain.FeatureID) (*domain.Feature, error) {
			return &domain.Feature{
				ID:        featureID,
				ProjectID: projectID,
				Name:      "Original Feature Name",
				Status:    domain.FeatureStatusDraft,
			}, nil
		},
		UpdateFunc: func(_ context.Context, f domain.Feature) error {
			return nil
		},
	}

	a := newAppWithFeatures(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		&taskstest.MockTaskRepository{},
		&columnstest.MockColumnRepository{},
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
		mockFeatures,
	)

	err := a.UpdateFeature(ctx, featureID, "" /* empty name */, "description")

	assert.ErrorIs(t, err, domain.ErrFeatureNameRequired,
		"UpdateFeature must reject an empty name with ErrFeatureNameRequired")
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. AddDependency does not check self-dependency
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_AddDependency_SelfDependency verifies that AddDependency rejects
// a task depending on itself with ErrCannotDependOnSelf. The domain declares
// ErrCannotDependOnSelf and the fix enforces it before the cycle check.
//
// Vulnerability that was fixed: dependency_service.go — no check for
// taskID == dependsOnTaskID before proceeding.
func TestSecurity_AddDependency_SelfDependency(t *testing.T) {
	ctx := context.Background()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, id domain.TaskID) (*domain.Task, error) {
			return &domain.Task{
				ID:       taskID,
				ColumnID: columnID,
				Title:    "Task",
				Summary:  "summary",
			}, nil
		},
	}

	mockDeps := &dependenciestest.MockDependencyRepository{
		WouldCreateCycleFunc: func(_ context.Context, _ domain.ProjectID, _, _ domain.TaskID) (bool, error) {
			return false, nil // cycle check does not catch self-dep
		},
		CreateFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskDependency) error {
			return nil
		},
	}

	a := newSecurityApp(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		mockTasks,
		&columnstest.MockColumnRepository{},
		&commentstest.MockCommentRepository{},
		mockDeps,
	)

	// Task depends on itself — should be rejected with ErrCannotDependOnSelf.
	err := a.AddDependency(ctx, projectID, taskID, taskID)

	assert.ErrorIs(t, err, domain.ErrCannotDependOnSelf,
		"AddDependency must reject a self-dependency with ErrCannotDependOnSelf")
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. GrantUserAccess no role validation
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_GrantUserAccess_InvalidRole verifies that GrantUserAccess rejects
// invalid role strings. The domain defines ProjectUserAccess.Role as "admin" or
// "member" and the fix enforces this at the app layer.
//
// Vulnerability that was fixed: project_access.go — role was passed through to
// the repository without validation.
func TestSecurity_GrantUserAccess_InvalidRole(t *testing.T) {
	ctx := context.Background()

	var storedRole string
	mockAccess := &mockProjectAccessRepository{
		grantUserFunc: func(_ context.Context, _ domain.ProjectID, _, role string) error {
			storedRole = role
			return nil
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	a := app.NewApp(app.Config{
		Projects:      &projectstest.MockProjectRepository{},
		Agents:        &agentstest.MockRoleRepository{},
		Tasks:         &taskstest.MockTaskRepository{},
		Columns:       &columnstest.MockColumnRepository{},
		Comments:      &commentstest.MockCommentRepository{},
		Dependencies:  &dependenciestest.MockDependencyRepository{},
		ProjectAccess: mockAccess,
		Logger:        logger,
	})

	projectID := domain.NewProjectID()

	// Attempt to grant access with an invalid role
	err := a.GrantUserAccess(ctx, projectID, "user-123", "superadmin")

	if err == nil {
		assert.NotEqual(t, "superadmin", storedRole,
			"GrantUserAccess must not store an invalid role; role should be validated against {\"admin\", \"member\"}")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 7. ReorderTask accepts negative position
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_ReorderTask_NegativePosition verifies that ReorderTask rejects
// negative position values. A negative position can corrupt task ordering in a
// column and the fix adds the bounds check.
//
// Vulnerability that was fixed: task_service.go — newPosition was passed to
// tasks.ReorderTask without checking >= 0.
func TestSecurity_ReorderTask_NegativePosition(t *testing.T) {
	ctx := context.Background()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	columnID := domain.NewColumnID()

	mockTasks := &taskstest.MockTaskRepository{
		FindByIDFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID) (*domain.Task, error) {
			return &domain.Task{
				ID:       taskID,
				ColumnID: columnID,
				Title:    "Task",
				Summary:  "summary",
				Position: 2,
			}, nil
		},
		ReorderTaskFunc: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID, pos int) error {
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

	err := a.ReorderTask(ctx, projectID, taskID, -5)

	assert.Error(t, err,
		"ReorderTask must reject a negative position value")
}

// ─────────────────────────────────────────────────────────────────────────────
// 8. FeatureStatus transition not validated
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_UpdateFeatureStatus_InvalidTransition verifies that
// UpdateFeatureStatus rejects invalid transitions (e.g., done -> draft).
// The fix adds transition validation so completed features cannot be silently
// reverted.
//
// Vulnerability that was fixed: feature_service.go — only the target status
// was validated, not the transition from the current status.
func TestSecurity_UpdateFeatureStatus_InvalidTransition(t *testing.T) {
	ctx := context.Background()

	featureID := domain.NewFeatureID()
	projectID := domain.NewProjectID()

	mockFeatures := &featurestest.MockFeature{
		FindByIDFunc: func(_ context.Context, _ domain.FeatureID) (*domain.Feature, error) {
			return &domain.Feature{
				ID:        featureID,
				ProjectID: projectID,
				Name:      "Feature X",
				Status:    domain.FeatureStatusDone, // completed feature
			}, nil
		},
		UpdateStatusFunc: func(_ context.Context, _ domain.FeatureID, _ domain.FeatureStatus, _ string) error {
			return nil
		},
	}

	a := newAppWithFeatures(
		&projectstest.MockProjectRepository{},
		&agentstest.MockRoleRepository{},
		&taskstest.MockTaskRepository{},
		&columnstest.MockColumnRepository{},
		&commentstest.MockCommentRepository{},
		&dependenciestest.MockDependencyRepository{},
		mockFeatures,
	)

	// Transition from done -> draft — should not be allowed.
	err := a.UpdateFeatureStatus(ctx, featureID, domain.FeatureStatusDraft, "")

	assert.Error(t, err,
		"UpdateFeatureStatus must reject an invalid transition (done -> draft)")
}

// ─────────────────────────────────────────────────────────────────────────────
// 9. ApproveWontDo does not set CompletionSummary
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_ApproveWontDo_NoCompletionSummary verifies that ApproveWontDo
// populates CompletionSummary when moving a task to Done. This keeps done tasks
// consistent with those completed via CompleteTask.
//
// Vulnerability that was fixed: task_service.go — task was moved to done with
// CompletedAt set but CompletionSummary left empty.
func TestSecurity_ApproveWontDo_NoCompletionSummary(t *testing.T) {
	ctx := context.Background()
	cols := makeColumns()

	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	blockedColumn := cols[domain.ColumnBlocked]

	task := &domain.Task{
		ID:              taskID,
		ColumnID:        blockedColumn.ID,
		Title:           "task to won't-do",
		Summary:         "summary",
		IsBlocked:       true,
		WontDoRequested: true,
		WontDoReason:    "out of scope for MVP",
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

	err := a.ApproveWontDo(ctx, projectID, taskID)
	require.NoError(t, err)

	// Task is now in done and CompletionSummary must be set.
	assert.NotEmpty(t, savedTask.CompletionSummary,
		"ApproveWontDo must populate CompletionSummary when moving a task to Done")
}
