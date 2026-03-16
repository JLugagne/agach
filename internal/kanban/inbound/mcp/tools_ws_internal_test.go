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
	updateProjectFn   func(context.Context, domain.ProjectID, string, string) error
	deleteProjectFn   func(context.Context, domain.ProjectID) error
	createTaskFn      func(context.Context, domain.ProjectID, string, string, string, domain.Priority, string, string, string, []string, []string, string) (domain.Task, error)
	updateTaskFn      func(context.Context, domain.ProjectID, domain.TaskID, *string, *string, *string, *string, *string, *domain.Priority, *[]string, *[]string, *domain.TokenUsage) error
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
func (m *internalMockCommands) UpdateProject(ctx context.Context, projectID domain.ProjectID, name, description string) error {
	return m.updateProjectFn(ctx, projectID, name, description)
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
func (m *internalMockCommands) CreateTask(ctx context.Context, projectID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string) (domain.Task, error) {
	return m.createTaskFn(ctx, projectID, title, summary, description, priority, createdByRole, createdByAgent, assignedRole, contextFiles, tags, estimatedEffort)
}
func (m *internalMockCommands) UpdateTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage) error {
	return m.updateTaskFn(ctx, projectID, taskID, title, description, assignedRole, estimatedEffort, resolution, priority, contextFiles, tags, tokenUsage)
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

// Verify internalMockCommands implements service.Commands
var _ service.Commands = (*internalMockCommands)(nil)

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
		createTaskFn: func(_ context.Context, pID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string) (domain.Task, error) {
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
	require.Len(t, events, 1, "exactly one event should be broadcast after completeTask")

	event := events[0]
	assert.Equal(t, "task_completed", event.Type)
	assert.Equal(t, string(projectID), event.ProjectID)

	data, ok := event.Data.(map[string]interface{})
	require.True(t, ok, "event.Data must be map[string]interface{}")
	assert.Equal(t, string(taskID), data["task_id"])
	assert.Equal(t, "Done: implemented all the tests", data["completion_summary"])
	assert.Equal(t, "agent-go-1", data["completed_by_agent"])
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
	require.Len(t, events, 1, "exactly one event should be broadcast after blockTask")

	event := events[0]
	assert.Equal(t, "task_blocked", event.Type)
	assert.Equal(t, string(projectID), event.ProjectID)

	data, ok := event.Data.(map[string]string)
	require.True(t, ok, "event.Data must be map[string]string")
	assert.Equal(t, string(taskID), data["task_id"])
	assert.Equal(t, "Waiting for design approval", data["blocked_reason"])
	assert.Equal(t, "agent-pm-1", data["blocked_by_agent"])
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
		updateProjectFn: func(_ context.Context, pID domain.ProjectID, name, description string) error {
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
		updateTaskFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage) error {
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

// TestUpdateTaskFiles_EmitsBroadcast tests that the updateTaskFiles handler
// emits a "task_updated" event.
func TestUpdateTaskFiles_EmitsBroadcast(t *testing.T) {
	ctx := context.Background()
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	cmds := &internalMockCommands{
		updateTaskFilesFn: func(_ context.Context, pID domain.ProjectID, tID domain.TaskID, filesModified, contextFiles *[]string) error {
			return nil
		},
	}

	recorder := &internalRecorder{}
	handler := newInternalToolHandler(cmds, recorder)

	_, err := handler.updateTaskFiles(ctx, map[string]interface{}{
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
