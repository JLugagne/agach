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

// TestSecurity_RED_CompleteTask_EmptyCompletionSummary documents that CompleteTask
// at the app layer does not validate the completionSummary parameter. The domain
// defines ErrCompletionSummaryRequired with "minimum 100 characters" but the app
// layer never enforces it. An agent can complete a task with an empty summary,
// making post-completion review impossible.
//
// Vulnerability: task_service.go:577 — completionSummary is assigned to
// task.CompletionSummary without any length or emptiness check.
// TODO(security): Add "if completionSummary == \"\" { return domain.ErrCompletionSummaryRequired }"
// at the top of CompleteTask.
func TestSecurity_RED_CompleteTask_EmptyCompletionSummary(t *testing.T) {
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
		"RED: CompleteTask accepts an empty completion summary; "+
			"domain.ErrCompletionSummaryRequired is declared but never checked in app layer")
	t.Log("RED: CompleteTask does not validate completionSummary at the app layer")
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. CompleteTask does not validate negative token values
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_CompleteTask_NegativeTokenValues documents that CompleteTask
// accumulates token values with raw += (line 589-592) without the negative-value
// guard that UpdateTask has (line 303). A malicious or buggy agent can pass
// negative token counts at completion time to corrupt the counter.
//
// Vulnerability: task_service.go:589 — "task.InputTokens += tokenUsage.InputTokens"
// without checking tokenUsage.InputTokens >= 0.
// TODO(security): Add the same negative-value check as UpdateTask or use addClamped.
func TestSecurity_RED_CompleteTask_NegativeTokenValues(t *testing.T) {
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
			"RED: CompleteTask accepted negative token values and corrupted counter to %d; "+
				"UpdateTask has this guard but CompleteTask does not",
			savedTask.InputTokens)
	}
	t.Log("RED: CompleteTask does not validate negative token values (unlike UpdateTask)")
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. UpdateTask can set title to empty string
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_UpdateTask_EmptyTitleAccepted documents that UpdateTask
// allows setting the task title to an empty string via the title pointer
// parameter. CreateTask enforces ErrTaskTitleRequired but UpdateTask bypasses
// this check — at line 277: "if title != nil { task.Title = *title }" with
// no validation of the value.
//
// Vulnerability: task_service.go:277-278 — title is assigned without checking
// for empty string.
// TODO(security): Add "if *title == \"\" { return domain.ErrTaskTitleRequired }"
// before assigning.
func TestSecurity_RED_UpdateTask_EmptyTitleAccepted(t *testing.T) {
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
		"RED: UpdateTask accepts an empty title via pointer; "+
			"CreateTask validates this but UpdateTask does not; "+
			"fix: add empty check before assigning *title")
	t.Log("RED: UpdateTask allows clearing the title to an empty string")
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. UpdateFeature allows empty name
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_UpdateFeature_EmptyNameAccepted documents that UpdateFeature
// sets feature.Name = name without checking for empty string. CreateFeature
// enforces ErrFeatureNameRequired but UpdateFeature does not.
//
// Vulnerability: feature_service.go:65 — "feature.Name = name" with no empty check.
// TODO(security): Add "if name == \"\" { return domain.ErrFeatureNameRequired }"
func TestSecurity_RED_UpdateFeature_EmptyNameAccepted(t *testing.T) {
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
		"RED: UpdateFeature accepts an empty name; "+
			"CreateFeature validates this but UpdateFeature does not; "+
			"fix: add empty name check at top of UpdateFeature")
	t.Log("RED: UpdateFeature allows clearing the feature name to an empty string")
}

// ─────────────────────────────────────────────────────────────────────────────
// 5. AddDependency does not check self-dependency
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_AddDependency_SelfDependency documents that AddDependency
// does not check if taskID == dependsOnTaskID before proceeding with the cycle
// check and creation. The domain defines ErrCannotDependOnSelf but it is never
// used. Self-dependencies create a trivial cycle that the WouldCreateCycle check
// may not detect (depending on the implementation), and even if detected, the
// error message would be misleading (ErrCircularDependency instead of
// ErrCannotDependOnSelf).
//
// Vulnerability: dependency_service.go:28 — no check for taskID == dependsOnTaskID.
// TODO(security): Add "if taskID == dependsOnTaskID { return domain.ErrCannotDependOnSelf }"
func TestSecurity_RED_AddDependency_SelfDependency(t *testing.T) {
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
		"RED: AddDependency does not check self-dependency; "+
			"domain.ErrCannotDependOnSelf is declared but never used; "+
			"fix: add 'if taskID == dependsOnTaskID' guard at top of AddDependency")
	t.Log("RED: AddDependency allows a task to depend on itself")
}

// ─────────────────────────────────────────────────────────────────────────────
// 6. GrantUserAccess no role validation
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_GrantUserAccess_InvalidRole documents that GrantUserAccess
// passes the role parameter directly to the repository without any validation.
// The domain defines ProjectUserAccess.Role as "admin" or "member" but the app
// layer does not enforce this. An attacker can inject arbitrary role strings
// (e.g., "superadmin", "root") that may bypass role-based checks.
//
// Vulnerability: project_access.go:21 — role is passed through without validation.
// TODO(security): Add validation that role is one of {"admin", "member"} before
// calling access.GrantUser.
func TestSecurity_RED_GrantUserAccess_InvalidRole(t *testing.T) {
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
			"RED: GrantUserAccess accepted and stored the invalid role 'superadmin'; "+
				"the role should be validated against a known set {\"admin\", \"member\"}")
	} else {
		// If error, the fix is already in place
		t.Log("GrantUserAccess correctly rejected the invalid role")
	}
	t.Log("RED: GrantUserAccess does not validate the role parameter")
}

// ─────────────────────────────────────────────────────────────────────────────
// 7. ReorderTask accepts negative position
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_ReorderTask_NegativePosition documents that ReorderTask
// passes the newPosition parameter directly to the repository without bounds
// checking. A negative position can corrupt the position-based ordering of tasks
// in a column, potentially making them invisible to the UI or causing sorting
// anomalies.
//
// Vulnerability: task_service.go:521 — newPosition is passed to
// tasks.ReorderTask without checking >= 0.
// TODO(security): Add "if newPosition < 0 { return domain.ErrInvalidTaskData }"
func TestSecurity_RED_ReorderTask_NegativePosition(t *testing.T) {
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
		"RED: ReorderTask accepts negative position values; "+
			"a negative position can corrupt task ordering; "+
			"fix: add bounds check 'if newPosition < 0 { return domain.ErrInvalidTaskData }'")
	t.Log("RED: ReorderTask does not validate negative position values")
}

// ─────────────────────────────────────────────────────────────────────────────
// 8. FeatureStatus transition not validated
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_UpdateFeatureStatus_InvalidTransition documents that
// UpdateFeatureStatus validates that the target status is a valid FeatureStatus
// value, but does NOT validate the transition. Any status can transition to any
// other status (e.g., done -> draft), bypassing the intended workflow and
// potentially discarding changelogs or resurrecting completed features.
//
// Vulnerability: feature_service.go:72-86 — only validates status is valid,
// not the transition from current status.
// TODO(security): Add transition validation — at minimum block done -> draft
// and draft -> done without going through ready/in_progress.
func TestSecurity_RED_UpdateFeatureStatus_InvalidTransition(t *testing.T) {
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
		"RED: UpdateFeatureStatus allows any status transition including done -> draft; "+
			"completed features can be silently reverted; "+
			"fix: add transition validation in UpdateFeatureStatus")
	t.Log("RED: UpdateFeatureStatus does not validate status transitions")
}

// ─────────────────────────────────────────────────────────────────────────────
// 9. ApproveWontDo does not set CompletionSummary
// ─────────────────────────────────────────────────────────────────────────────

// TestSecurity_RED_ApproveWontDo_NoCompletionSummary documents that ApproveWontDo
// moves a task to Done but does not set CompletionSummary. The resulting task in
// the done column has an empty completion_summary, making it inconsistent with
// tasks completed via CompleteTask. Downstream analytics, changelogs, and feature
// summaries that rely on CompletionSummary being populated for done tasks will
// produce incomplete or missing data.
//
// Vulnerability: task_service.go:815-823 — task is moved to done column with
// CompletedAt set but CompletionSummary left as its original value (typically empty).
// TODO(security): Set task.CompletionSummary to a descriptive value like
// "Won't do (approved): <wont_do_reason>".
func TestSecurity_RED_ApproveWontDo_NoCompletionSummary(t *testing.T) {
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

	// Task is now in done but CompletionSummary is empty.
	assert.NotEmpty(t, savedTask.CompletionSummary,
		"RED: ApproveWontDo moves task to Done but leaves CompletionSummary empty; "+
			"this breaks feature changelog generation and completion analytics; "+
			"fix: set CompletionSummary to a descriptive string when approving won't-do")
	t.Log("RED: ApproveWontDo does not populate CompletionSummary when moving to done")
}
