package mcp

// This file contains internal integration tests for WebSocket event broadcasting
// from ToolHandler methods. Because the handler methods are unexported, these
// tests live in the same package (package mcp) to access them directly.
//
// Tests verify that each handler method calls hub.Broadcast with the correct
// event type, project_id, and data fields when the underlying command/query
// succeeds.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	tasksrepo "github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// internalRecorder captures Broadcast calls from within the mcp package.
type internalRecorder struct {
	mu     sync.Mutex
	events []websocket.Event
}

func (r *internalRecorder) Broadcast(event websocket.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *internalRecorder) recorded() []websocket.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]websocket.Event, len(r.events))
	copy(out, r.events)
	return out
}

func (r *internalRecorder) last() websocket.Event {
	evts := r.recorded()
	if len(evts) == 0 {
		panic("no events recorded")
	}
	return evts[len(evts)-1]
}

func (r *internalRecorder) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

// internalMockCommands is a minimal mock of service.Commands for internal tests.
type internalMockCommands struct {
	createProjectFn   func(context.Context, string, string, string, string, string, *domain.ProjectID) (domain.Project, error)
	updateProjectFn   func(context.Context, domain.ProjectID, string, string, *string) error
	deleteProjectFn   func(context.Context, domain.ProjectID) error
	createTaskFn      func(context.Context, domain.ProjectID, string, string, string, domain.Priority, string, string, string, []string, []string, string, bool) (domain.Task, error)
	updateTaskFn      func(context.Context, domain.ProjectID, domain.TaskID, *string, *string, *string, *string, *string, *domain.Priority, *[]string, *[]string, *domain.TokenUsage, *int) error
	updateTaskFilesFn func(context.Context, domain.ProjectID, domain.TaskID, *[]string, *[]string) error
	deleteTaskFn      func(context.Context, domain.ProjectID, domain.TaskID) error
	moveTaskFn        func(context.Context, domain.ProjectID, domain.TaskID, domain.ColumnSlug) error
	completeTaskFn    func(context.Context, domain.ProjectID, domain.TaskID, string, []string, string, *domain.TokenUsage) error
	blockTaskFn       func(context.Context, domain.ProjectID, domain.TaskID, string, string) error
	requestWontDoFn   func(context.Context, domain.ProjectID, domain.TaskID, string, string) error
	createCommentFn   func(context.Context, domain.ProjectID, domain.TaskID, string, string, domain.AuthorType, string) (domain.Comment, error)
}

func (m *internalMockCommands) CreateProject(ctx context.Context, name, description, workDir, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
	return m.createProjectFn(ctx, name, description, workDir, createdByRole, createdByAgent, parentID)
}
func (m *internalMockCommands) UpdateProject(ctx context.Context, projectID domain.ProjectID, name, description string, defaultRole *string) error {
	return m.updateProjectFn(ctx, projectID, name, description, defaultRole)
}
func (m *internalMockCommands) DeleteProject(ctx context.Context, projectID domain.ProjectID) error {
	return m.deleteProjectFn(ctx, projectID)
}
func (m *internalMockCommands) CreateRole(ctx context.Context, slug, name, icon, color, description, promptHint string, techStack []string, sortOrder int) (domain.Role, error) {
	panic("not used in this test")
}
func (m *internalMockCommands) UpdateRole(ctx context.Context, roleID domain.RoleID, name, icon, color, description, promptHint string, techStack []string, sortOrder int) error {
	panic("not used in this test")
}
func (m *internalMockCommands) DeleteRole(ctx context.Context, roleID domain.RoleID) error {
	panic("not used in this test")
}
func (m *internalMockCommands) CreateTask(ctx context.Context, projectID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string, startInBacklog bool) (domain.Task, error) {
	return m.createTaskFn(ctx, projectID, title, summary, description, priority, createdByRole, createdByAgent, assignedRole, contextFiles, tags, estimatedEffort, startInBacklog)
}
func (m *internalMockCommands) UpdateTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int) error {
	return m.updateTaskFn(ctx, projectID, taskID, title, description, assignedRole, estimatedEffort, resolution, priority, contextFiles, tags, tokenUsage, humanEstimateSeconds)
}
func (m *internalMockCommands) UpdateTaskFiles(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, filesModified, contextFiles *[]string) error {
	return m.updateTaskFilesFn(ctx, projectID, taskID, filesModified, contextFiles)
}
func (m *internalMockCommands) DeleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	return m.deleteTaskFn(ctx, projectID, taskID)
}
func (m *internalMockCommands) MoveTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnSlug domain.ColumnSlug) error {
	return m.moveTaskFn(ctx, projectID, taskID, targetColumnSlug)
}
func (m *internalMockCommands) ReorderTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, newPosition int) error {
	panic("not used in this test")
}
func (m *internalMockCommands) StartTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	panic("not used in this test")
}
func (m *internalMockCommands) CompleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage) error {
	return m.completeTaskFn(ctx, projectID, taskID, completionSummary, filesModified, completedByAgent, tokenUsage)
}
func (m *internalMockCommands) BlockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, blockedReason, blockedByAgent string) error {
	return m.blockTaskFn(ctx, projectID, taskID, blockedReason, blockedByAgent)
}
func (m *internalMockCommands) UnblockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	panic("not used in this test")
}
func (m *internalMockCommands) RequestWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, wontDoReason, wontDoRequestedBy string) error {
	return m.requestWontDoFn(ctx, projectID, taskID, wontDoReason, wontDoRequestedBy)
}
func (m *internalMockCommands) ApproveWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	panic("not used in this test")
}
func (m *internalMockCommands) RejectWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, reason string) error {
	panic("not used in this test")
}
func (m *internalMockCommands) CreateComment(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, authorRole, authorName string, authorType domain.AuthorType, content string) (domain.Comment, error) {
	return m.createCommentFn(ctx, projectID, taskID, authorRole, authorName, authorType, content)
}
func (m *internalMockCommands) UpdateComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID, content string) error {
	panic("not used in this test")
}
func (m *internalMockCommands) DeleteComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) error {
	panic("not used in this test")
}
func (m *internalMockCommands) AddDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error {
	panic("not used in this test")
}
func (m *internalMockCommands) RemoveDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error {
	panic("not used in this test")
}
func (m *internalMockCommands) MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	panic("not used in this test")
}
func (m *internalMockCommands) UpdateColumnWIPLimit(ctx context.Context, projectID domain.ProjectID, columnSlug domain.ColumnSlug, wipLimit int) error {
	panic("not used in this test")
}
func (m *internalMockCommands) MoveTaskToProject(ctx context.Context, sourceProjectID domain.ProjectID, taskID domain.TaskID, targetProjectID domain.ProjectID) error {
	panic("not used in this test")
}

func (m *internalMockCommands) IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error {
	panic("not used in this test")
}
func (m *internalMockCommands) CreateProjectRole(ctx context.Context, projectID domain.ProjectID, slug, name, icon, color, description, promptHint string, techStack []string, sortOrder int) (domain.Role, error) {
	panic("not used in this test")
}
func (m *internalMockCommands) UpdateProjectRole(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID, name, icon, color, description, promptHint string, techStack []string, sortOrder int) error {
	panic("not used in this test")
}
func (m *internalMockCommands) DeleteProjectRole(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error {
	panic("not used in this test")
}
func (m *internalMockCommands) UpdateTaskSessionID(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error {
	panic("not used in this test")
}
func (m *internalMockCommands) BulkCreateTasks(ctx context.Context, projectID domain.ProjectID, inputs []service.BulkTaskInput) ([]domain.Task, error) {
	panic("not used in this test")
}

// Verify internalMockCommands implements service.Commands
var _ service.Commands = (*internalMockCommands)(nil)

// internalMockQueries is a minimal mock of service.Queries for internal tests.
type internalMockQueries struct {
	getNextTaskFn    func(context.Context, domain.ProjectID, string, *domain.ProjectID) (*domain.Task, error)
	listTasksFn      func(context.Context, domain.ProjectID, tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error)
	getTaskFn        func(context.Context, domain.ProjectID, domain.TaskID) (*domain.Task, error)
	listCommentsFn   func(context.Context, domain.ProjectID, domain.TaskID, int, int) ([]domain.Comment, error)
	countCommentsFn  func(context.Context, domain.ProjectID, domain.TaskID) (int, error)
	listDepsFn       func(context.Context, domain.ProjectID, domain.TaskID) ([]domain.TaskDependency, error)
}

func (m *internalMockQueries) GetNextTask(ctx context.Context, projectID domain.ProjectID, role string, subProjectID *domain.ProjectID) (*domain.Task, error) {
	return m.getNextTaskFn(ctx, projectID, role, subProjectID)
}
func (m *internalMockQueries) ListTasks(ctx context.Context, projectID domain.ProjectID, filters tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
	return m.listTasksFn(ctx, projectID, filters)
}
func (m *internalMockQueries) GetTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (*domain.Task, error) {
	return m.getTaskFn(ctx, projectID, taskID)
}
func (m *internalMockQueries) ListComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error) {
	return m.listCommentsFn(ctx, projectID, taskID, limit, offset)
}
func (m *internalMockQueries) CountComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error) {
	if m.countCommentsFn != nil {
		return m.countCommentsFn(ctx, projectID, taskID)
	}
	return 0, nil
}
func (m *internalMockQueries) ListDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	if m.listDepsFn != nil {
		return m.listDepsFn(ctx, projectID, taskID)
	}
	return []domain.TaskDependency{}, nil
}
func (m *internalMockQueries) GetProject(ctx context.Context, projectID domain.ProjectID) (*domain.Project, error) {
	panic("not used")
}
func (m *internalMockQueries) ListProjects(ctx context.Context) ([]domain.Project, error) { panic("not used") }
func (m *internalMockQueries) ListProjectsWithSummary(ctx context.Context) ([]domain.ProjectWithSummary, error) {
	panic("not used")
}
func (m *internalMockQueries) ListSubProjects(ctx context.Context, parentID domain.ProjectID) ([]domain.Project, error) {
	panic("not used")
}
func (m *internalMockQueries) ListSubProjectsWithSummary(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error) {
	panic("not used")
}
func (m *internalMockQueries) GetProjectSummary(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectSummary, error) {
	panic("not used")
}
func (m *internalMockQueries) GetProjectInfo(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectInfo, error) {
	panic("not used")
}
func (m *internalMockQueries) ListProjectsByWorkDir(ctx context.Context, workDir string) ([]domain.ProjectWithSummary, error) {
	panic("not used")
}
func (m *internalMockQueries) GetRole(ctx context.Context, roleID domain.RoleID) (*domain.Role, error) {
	panic("not used")
}
func (m *internalMockQueries) GetRoleBySlug(ctx context.Context, slug string) (*domain.Role, error) {
	panic("not used")
}
func (m *internalMockQueries) ListRoles(ctx context.Context) ([]domain.Role, error) { panic("not used") }
func (m *internalMockQueries) ListProjectRoles(ctx context.Context, projectID domain.ProjectID) ([]domain.Role, error) {
	panic("not used")
}
func (m *internalMockQueries) GetProjectRoleBySlug(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Role, error) {
	panic("not used")
}
func (m *internalMockQueries) GetNextTasks(ctx context.Context, projectID domain.ProjectID, role string, count int, subProjectID *domain.ProjectID) ([]domain.Task, error) {
	panic("not used")
}
func (m *internalMockQueries) GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error) {
	return []domain.DependencyContext{}, nil
}
func (m *internalMockQueries) GetColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (*domain.Column, error) {
	panic("not used")
}
func (m *internalMockQueries) GetColumnBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
	panic("not used")
}
func (m *internalMockQueries) ListColumns(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error) {
	panic("not used")
}
func (m *internalMockQueries) GetComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (*domain.Comment, error) {
	panic("not used")
}
func (m *internalMockQueries) GetDependencyTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	panic("not used")
}
func (m *internalMockQueries) GetDependentTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	panic("not used")
}
func (m *internalMockQueries) GetToolUsageForProject(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error) {
	panic("not used")
}
func (m *internalMockQueries) GetTimeline(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error) {
	panic("not used")
}
func (m *internalMockQueries) GetColdStartStats(ctx context.Context, projectID domain.ProjectID) ([]domain.RoleColdStartStat, error) {
	panic("not used")
}
func (m *internalMockQueries) GetWIPSlots(ctx context.Context, projectID domain.ProjectID) (*domain.WIPSlotsInfo, error) {
	return nil, nil
}

// newInternalToolHandler creates a ToolHandler with the given recorder wired as
// the broadcaster, using the provided commands mock and a no-op queries stub.
func newInternalToolHandler(cmds *internalMockCommands, recorder *internalRecorder) *ToolHandler {
	return &ToolHandler{
		commands: cmds,
		queries:  nil, // not needed for command-focused tests
		hub:      recorder,
	}
}

// TestCreateProject_EmitsBroadcast tests that the createProject handler method
// calls hub.Broadcast with the correct "project_created" event payload.
func TestCreateProject_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()

	cmds := &internalMockCommands{
		createProjectFn: func(_ context.Context, name, description, workDir, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
			return domain.Project{
				ID:             projectID,
				Name:           name,
				Description:    description,
				CreatedByRole:  createdByRole,
				CreatedByAgent: createdByAgent,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}, nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.createProject(ctx, map[string]interface{}{
		"name":             "My Project",
		"description":      "A test project",
		"work_dir":         "/workspace",
		"created_by_role":  "architect",
		"created_by_agent": "agent-1",
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 1, "exactly one event should be broadcast after createProject")

	event := events[0]
	assert.Equal(t, "project_created", event.Type, "event type must be 'project_created'")
	assert.Empty(t, event.ProjectID, "project_created event must not carry a project_id field")

	data, ok := event.Data.(map[string]interface{})
	require.True(t, ok, "event.Data must be map[string]interface{}")
	assert.Equal(t, string(projectID), data["id"])
	assert.Equal(t, "My Project", data["name"])
}

// TestMoveTask_EmitsBroadcast tests that the moveTask handler emits a
// "task_moved" event with project_id, task_id, and target_column.
// This is the primary bug-suspect scenario from the issue description.
func TestMoveTask_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		moveTaskFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, slug domain.ColumnSlug) error {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, domain.ColumnSlug("in_progress"), slug)
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.moveTask(ctx, map[string]interface{}{
		"project_id":    string(projectID),
		"task_id":       string(taskID),
		"target_column": "in_progress",
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 1, "exactly one event should be broadcast after moveTask")

	event := events[0]
	assert.Equal(t, "task_moved", event.Type, "event type must be 'task_moved'")
	assert.Equal(t, string(projectID), event.ProjectID, "task_moved event must carry project_id for UI routing")

	data, ok := event.Data.(map[string]string)
	require.True(t, ok, "event.Data must be map[string]string")
	assert.Equal(t, string(taskID), data["task_id"])
	assert.Equal(t, "in_progress", data["target_column"])
}

// TestCreateTask_EmitsBroadcast tests that the createTask handler emits a
// "task_created" event carrying the full task object and the project_id.
func TestCreateTask_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		createTaskFn: func(_ context.Context, pID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string, startInBacklog bool) (domain.Task, error) {
			return domain.Task{
				ID:       taskID,
				Title:    title,
				Summary:  summary,
				Priority: priority,
			}, nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.createTask(ctx, map[string]interface{}{
		"project_id":      string(projectID),
		"title":           "Write tests",
		"summary":         "Write integration tests for WS broadcasting",
		"created_by_role": "backend_go",
		"priority":        "high",
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 1, "exactly one event should be broadcast after createTask")

	event := events[0]
	assert.Equal(t, "task_created", event.Type)
	assert.Equal(t, string(projectID), event.ProjectID)

	// The task object is embedded in Data
	task, ok := event.Data.(domain.Task)
	require.True(t, ok, "event.Data must be domain.Task")
	assert.Equal(t, taskID, task.ID)
	assert.Equal(t, "Write tests", task.Title)
}

// TestCompleteTask_EmitsBroadcast tests that the completeTask handler emits a
// "task_completed" event with all required fields.
func TestCompleteTask_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		completeTaskFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage) error {
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.completeTask(ctx, map[string]interface{}{
		"project_id":         string(projectID),
		"task_id":            string(taskID),
		"completion_summary": "Done: implemented all the tests",
		"completed_by_agent": "agent-go-1",
		"files_modified":     []interface{}{"pkg/ws/hub.go"},
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 2, "exactly two events should be broadcast after completeTask")

	event := events[0]
	assert.Equal(t, "task_completed", event.Type)
	assert.Equal(t, string(projectID), event.ProjectID)

	data, ok := event.Data.(map[string]interface{})
	require.True(t, ok, "event.Data must be map[string]interface{}")
	assert.Equal(t, string(taskID), data["task_id"])
	assert.Equal(t, "Done: implemented all the tests", data["completion_summary"])
	assert.Equal(t, "agent-go-1", data["completed_by_agent"])

	event2 := events[1]
	assert.Equal(t, "wip_slot_available", event2.Type)
	assert.Equal(t, string(projectID), event2.ProjectID)
}

// TestBlockTask_EmitsBroadcast tests that the blockTask handler emits a
// "task_blocked" event with the correct fields.
func TestBlockTask_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		blockTaskFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, blockedReason, blockedByAgent string) error {
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.blockTask(ctx, map[string]interface{}{
		"project_id":       string(projectID),
		"task_id":          string(taskID),
		"blocked_reason":   "Waiting for design approval",
		"blocked_by_agent": "agent-pm-1",
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 2, "exactly two events should be broadcast after blockTask")

	event := events[0]
	assert.Equal(t, "task_blocked", event.Type)
	assert.Equal(t, string(projectID), event.ProjectID)

	data, ok := event.Data.(map[string]string)
	require.True(t, ok, "event.Data must be map[string]string")
	assert.Equal(t, string(taskID), data["task_id"])
	assert.Equal(t, "Waiting for design approval", data["blocked_reason"])
	assert.Equal(t, "agent-pm-1", data["blocked_by_agent"])

	event2 := events[1]
	assert.Equal(t, "wip_slot_available", event2.Type)
	assert.Equal(t, string(projectID), event2.ProjectID)
}

// TestMoveTask_ToBacklog_EmitsWIPSlotAvailable verifies that moving a task to
// a column other than in_progress broadcasts both task_moved and
// wip_slot_available events.
func TestMoveTask_ToBacklog_EmitsWIPSlotAvailable(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		moveTaskFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, slug domain.ColumnSlug) error {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, domain.ColumnSlug("backlog"), slug)
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.moveTask(ctx, map[string]interface{}{
		"project_id":    string(projectID),
		"task_id":       string(taskID),
		"target_column": "backlog",
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 2, "expected task_moved + wip_slot_available when moving out of in_progress")

	assert.Equal(t, "task_moved", events[0].Type)
	assert.Equal(t, string(projectID), events[0].ProjectID)

	assert.Equal(t, "wip_slot_available", events[1].Type)
	assert.Equal(t, string(projectID), events[1].ProjectID)
}

// TestMoveTask_ToInProgress_DoesNotEmitWIPSlotAvailable verifies that moving a
// task INTO in_progress emits only task_moved — no wip_slot_available.
func TestMoveTask_ToInProgress_DoesNotEmitWIPSlotAvailable(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		moveTaskFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, slug domain.ColumnSlug) error {
			assert.Equal(t, domain.ColumnSlug("in_progress"), slug)
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.moveTask(ctx, map[string]interface{}{
		"project_id":    string(projectID),
		"task_id":       string(taskID),
		"target_column": "in_progress",
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 1, "expected only task_moved when moving into in_progress")

	assert.Equal(t, "task_moved", events[0].Type)
	assert.Equal(t, string(projectID), events[0].ProjectID)

	for _, e := range events {
		assert.NotEqual(t, "wip_slot_available", e.Type,
			"must NOT emit wip_slot_available when target is in_progress")
	}
}

// TestRequestWontDo_EmitsBroadcast tests that the requestWontDo handler emits
// a "wont_do_requested" event.
func TestRequestWontDo_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		requestWontDoFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, wontDoReason, requestedBy string) error {
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.requestWontDo(ctx, map[string]interface{}{
		"project_id":     string(projectID),
		"task_id":        string(taskID),
		"wont_do_reason": "Feature scope removed from Q1",
		"requested_by":   "agent-architect-1",
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 1, "exactly one event should be broadcast after requestWontDo")

	event := events[0]
	assert.Equal(t, "wont_do_requested", event.Type)
	assert.Equal(t, string(projectID), event.ProjectID)

	data, ok := event.Data.(map[string]string)
	require.True(t, ok, "event.Data must be map[string]string")
	assert.Equal(t, string(taskID), data["task_id"])
	assert.Equal(t, "Feature scope removed from Q1", data["wont_do_reason"])
	assert.Equal(t, "agent-architect-1", data["requested_by"])
}

// TestUpdateProject_EmitsBroadcast tests that the updateProject handler emits
// a "project_updated" event.
func TestUpdateProject_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()

	cmds := &internalMockCommands{
		updateProjectFn: func(_ context.Context, pID domain.ProjectID, name, description string, defaultRole *string) error {
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.updateProject(ctx, map[string]interface{}{
		"project_id":  string(projectID),
		"name":        "Renamed Project",
		"description": "Updated description",
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, "project_updated", event.Type)
	assert.Empty(t, event.ProjectID, "project_updated must not carry project_id")

	data, ok := event.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, string(projectID), data["id"])
	assert.Equal(t, "Renamed Project", data["name"])
}

// TestDeleteProject_EmitsBroadcast tests that the deleteProject handler emits
// a "project_deleted" event.
func TestDeleteProject_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()

	cmds := &internalMockCommands{
		deleteProjectFn: func(_ context.Context, pID domain.ProjectID) error {
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.deleteProject(ctx, map[string]interface{}{
		"project_id": string(projectID),
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, "project_deleted", event.Type)
	assert.Empty(t, event.ProjectID)

	data, ok := event.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, string(projectID), data["id"])
}

// TestUpdateTask_EmitsBroadcast tests that the updateTask handler emits a
// "task_updated" event with the task_id.
func TestUpdateTask_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		updateTaskFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int) error {
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	newTitle := "Updated Title"
	_, err := handler.updateTask(ctx, map[string]interface{}{
		"project_id": string(projectID),
		"task_id":    string(taskID),
		"title":      newTitle,
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, "task_updated", event.Type)
	assert.Equal(t, string(projectID), event.ProjectID)

	data, ok := event.Data.(map[string]string)
	require.True(t, ok, "event.Data must be map[string]string")
	assert.Equal(t, string(taskID), data["task_id"])
}

// TestUpdateTask_FilesModified_EmitsBroadcast tests that the updateTask handler
// with files_modified emits a "task_updated" event.
func TestUpdateTask_FilesModified_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		updateTaskFn: func(_ context.Context, _ domain.ProjectID, _ domain.TaskID, _, _, _, _, _ *string, _ *domain.Priority, _, _ *[]string, _ *domain.TokenUsage, _ *int) error {
			return nil
		},
		updateTaskFilesFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, filesModified, contextFiles *[]string) error {
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.updateTask(ctx, map[string]interface{}{
		"project_id":     string(projectID),
		"task_id":        string(taskID),
		"files_modified": []interface{}{"main.go"},
	})
	require.NoError(t, err)

	events := recorder.recorded()
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, "task_updated", event.Type)
	assert.Equal(t, string(projectID), event.ProjectID)
}

// TestMoveTask_NoEventOnError tests that when MoveTask returns an error, no
// broadcast event is emitted.
func TestMoveTask_NoEventOnError(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		moveTaskFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, slug domain.ColumnSlug) error {
			return domain.ErrTaskNotFound
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.moveTask(ctx, map[string]interface{}{
		"project_id":    string(projectID),
		"task_id":       string(taskID),
		"target_column": "in_progress",
	})
	require.Error(t, err, "moveTask should return the command error")
	assert.Equal(t, 0, recorder.Count(), "no event should be broadcast when moveTask fails")
}

// TestCreateProject_NoEventOnError tests that when CreateProject returns an
// error, no broadcast event is emitted.
func TestCreateProject_NoEventOnError(t *testing.T) {
	ctx := context.Background()

	cmds := &internalMockCommands{
		createProjectFn: func(_ context.Context, name, description, workDir, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
			return domain.Project{}, domain.ErrProjectNameRequired
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.createProject(ctx, map[string]interface{}{
		"name":             "",
		"work_dir":         "/workspace",
		"created_by_role":  "architect",
		"created_by_agent": "",
	})
	require.Error(t, err, "createProject should return the command error")
	assert.Equal(t, 0, recorder.Count(), "no event should be broadcast when createProject fails")
}

// TestGetNextTask_ReturnsTask tests that getNextTask returns task data when
// a task is available.
func TestGetNextTask_ReturnsTask(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	queries := &internalMockQueries{
		getNextTaskFn: func(_ context.Context, pID domain.ProjectID, role string, subProjectID *domain.ProjectID) (*domain.Task, error) {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, "backend_go", role)
			return &domain.Task{
				ID:           taskID,
				Title:        "Next task",
				Summary:      "Do some work",
				Priority:     domain.PriorityHigh,
				AssignedRole: role,
			}, nil
		},
	}

	handler := &ToolHandler{commands: nil, queries: queries, hub: nil}

	result, err := handler.getNextTask(ctx, map[string]interface{}{
		"project_id": string(projectID),
		"role":       "backend_go",
	})
	require.NoError(t, err)

	data, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, string(taskID), data["id"])
	assert.Equal(t, "Next task", data["title"])
	assert.Equal(t, "backend_go", data["assigned_role"])
}

// TestGetNextTask_NoTaskAvailable tests that getNextTask returns a nil task
// response when no task is available.
func TestGetNextTask_NoTaskAvailable(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()

	queries := &internalMockQueries{
		getNextTaskFn: func(_ context.Context, pID domain.ProjectID, role string, subProjectID *domain.ProjectID) (*domain.Task, error) {
			return nil, domain.ErrNoAvailableTasks
		},
	}

	handler := &ToolHandler{commands: nil, queries: queries, hub: nil}

	_, err := handler.getNextTask(ctx, map[string]interface{}{
		"project_id": string(projectID),
		"role":       "backend_go",
	})
	require.Error(t, err)
}

// TestGetNextTask_MissingProjectID tests that getNextTask returns an error
// when project_id is missing.
func TestGetNextTask_MissingProjectID(t *testing.T) {
	ctx := context.Background()
	handler := &ToolHandler{commands: nil, queries: nil, hub: nil}

	_, err := handler.getNextTask(ctx, map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project_id")
}

// TestListTasks_ReturnsTaskSummaries tests that listTasks returns a slice of
// task summaries from the queries service.
func TestListTasks_ReturnsTaskSummaries(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	queries := &internalMockQueries{
		listTasksFn: func(_ context.Context, pID domain.ProjectID, filters tasksrepo.TaskFilters) ([]domain.TaskWithDetails, error) {
			assert.Equal(t, projectID, pID)
			return []domain.TaskWithDetails{
				{
					Task: domain.Task{
						ID:      taskID,
						Title:   "Task 1",
						Summary: "Summary 1",
					},
				},
			}, nil
		},
	}

	handler := &ToolHandler{commands: nil, queries: queries, hub: nil}

	result, err := handler.listTasks(ctx, map[string]interface{}{
		"project_id": string(projectID),
	})
	require.NoError(t, err)

	summaries, ok := result.([]taskSummary)
	require.True(t, ok)
	require.Len(t, summaries, 1)
	assert.Equal(t, string(taskID), summaries[0].ID)
	assert.Equal(t, "Task 1", summaries[0].Title)
}

// TestListTasks_MissingProjectID tests that listTasks returns an error when
// project_id is not provided.
func TestListTasks_MissingProjectID(t *testing.T) {
	ctx := context.Background()
	handler := &ToolHandler{commands: nil, queries: nil, hub: nil}

	_, err := handler.listTasks(ctx, map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project_id")
}

// TestGetTask_ReturnsTaskDetails tests that getTask returns description,
// has_comments and depends_on from the queries service.
func TestGetTask_ReturnsTaskDetails(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	queries := &internalMockQueries{
		getTaskFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID) (*domain.Task, error) {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			return &domain.Task{
				ID:          taskID,
				Title:       "Test task",
				Description: "Task description",
				ContextFiles: []string{"file1.go"},
			}, nil
		},
		countCommentsFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID) (int, error) {
			return 2, nil
		},
		listDepsFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID) ([]domain.TaskDependency, error) {
			return []domain.TaskDependency{}, nil
		},
	}

	handler := &ToolHandler{commands: nil, queries: queries, hub: nil}

	result, err := handler.getTask(ctx, map[string]interface{}{
		"project_id": string(projectID),
		"task_id":    string(taskID),
	})
	require.NoError(t, err)

	data, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Task description", data["description"])
	assert.Equal(t, true, data["has_comments"])
	assert.NotNil(t, data["context_files"])
}

// TestListComments_ReturnsComments tests that listComments returns comments
// from the queries service.
func TestListComments_ReturnsComments(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	commentID := domain.NewCommentID()

	queries := &internalMockQueries{
		listCommentsFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, limit, offset int) ([]domain.Comment, error) {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			return []domain.Comment{
				{
					ID:         commentID,
					TaskID:     tID,
					AuthorRole: "developer",
					AuthorType: domain.AuthorTypeAgent,
					Content:    "Progress update",
				},
			}, nil
		},
	}

	handler := &ToolHandler{commands: nil, queries: queries, hub: nil}

	result, err := handler.listComments(ctx, map[string]interface{}{
		"project_id": string(projectID),
		"task_id":    string(taskID),
	})
	require.NoError(t, err)

	comments, ok := result.([]domain.Comment)
	require.True(t, ok)
	require.Len(t, comments, 1)
	assert.Equal(t, commentID, comments[0].ID)
	assert.Equal(t, "Progress update", comments[0].Content)
}

// TestNilHub_DoesNotPanic tests that ToolHandler with a nil hub does not panic
// when any operation that would broadcast is called.
func TestNilHub_DoesNotPanic(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		moveTaskFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, slug domain.ColumnSlug) error {
			return nil
		},
	}

	// Create handler with nil hub — the ToolHandler code checks hub != nil before calling Broadcast
	handler := &ToolHandler{
		commands: cmds,
		queries:  nil,
		hub:      nil,
	}

	require.NotPanics(t, func() {
		_, err := handler.moveTask(ctx, map[string]interface{}{
			"project_id":    string(projectID),
			"task_id":       string(taskID),
			"target_column": "in_progress",
		})
		require.NoError(t, err)
	})
}
