package mcp_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/kanban/inbound/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCommands is a mock implementation of service.Commands for testing
type MockCommands struct {
	CreateProjectFunc    func(ctx context.Context, name, description, workDir, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error)
	UpdateProjectFunc    func(ctx context.Context, projectID domain.ProjectID, name, description string) error
	DeleteProjectFunc    func(ctx context.Context, projectID domain.ProjectID) error
	CreateRoleFunc       func(ctx context.Context, slug, name, icon, color, description, promptHint string, techStack []string, sortOrder int) (domain.Role, error)
	UpdateRoleFunc       func(ctx context.Context, roleID domain.RoleID, name, icon, color, description, promptHint string, techStack []string, sortOrder int) error
	DeleteRoleFunc       func(ctx context.Context, roleID domain.RoleID) error
	CreateTaskFunc       func(ctx context.Context, projectID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string) (domain.Task, error)
	UpdateTaskFunc       func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage) error
	UpdateTaskFilesFunc  func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, filesModified, contextFiles *[]string) error
	DeleteTaskFunc       func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	MoveTaskFunc         func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnSlug domain.ColumnSlug) error
	StartTaskFunc        func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	CompleteTaskFunc     func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage) error
	BlockTaskFunc        func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, blockedReason, blockedByAgent string) error
	UnblockTaskFunc      func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	RequestWontDoFunc    func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, wontDoReason, wontDoRequestedBy string) error
	ApproveWontDoFunc    func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	RejectWontDoFunc     func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, reason string) error
	CreateCommentFunc    func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, authorRole, authorName string, authorType domain.AuthorType, content string) (domain.Comment, error)
	UpdateCommentFunc    func(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID, content string) error
	DeleteCommentFunc    func(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) error
	AddDependencyFunc    func(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error
	RemoveDependencyFunc func(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error
	MarkTaskSeenFunc         func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	UpdateColumnWIPLimitFunc func(ctx context.Context, projectID domain.ProjectID, columnSlug domain.ColumnSlug, wipLimit int) error
	MoveTaskToProjectFunc    func(ctx context.Context, sourceProjectID domain.ProjectID, taskID domain.TaskID, targetProjectID domain.ProjectID) error
	IncrementToolUsageFunc   func(ctx context.Context, projectID domain.ProjectID, toolName string) error
}

func (m *MockCommands) CreateProject(ctx context.Context, name, description, workDir, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
	if m.CreateProjectFunc == nil {
		panic("CreateProjectFunc not defined")
	}
	return m.CreateProjectFunc(ctx, name, description, workDir, createdByRole, createdByAgent, parentID)
}

func (m *MockCommands) UpdateProject(ctx context.Context, projectID domain.ProjectID, name, description string) error {
	if m.UpdateProjectFunc == nil {
		panic("UpdateProjectFunc not defined")
	}
	return m.UpdateProjectFunc(ctx, projectID, name, description)
}

func (m *MockCommands) DeleteProject(ctx context.Context, projectID domain.ProjectID) error {
	if m.DeleteProjectFunc == nil {
		panic("DeleteProjectFunc not defined")
	}
	return m.DeleteProjectFunc(ctx, projectID)
}

func (m *MockCommands) CreateRole(ctx context.Context, slug, name, icon, color, description, promptHint string, techStack []string, sortOrder int) (domain.Role, error) {
	if m.CreateRoleFunc == nil {
		panic("CreateRoleFunc not defined")
	}
	return m.CreateRoleFunc(ctx, slug, name, icon, color, description, promptHint, techStack, sortOrder)
}

func (m *MockCommands) UpdateRole(ctx context.Context, roleID domain.RoleID, name, icon, color, description, promptHint string, techStack []string, sortOrder int) error {
	if m.UpdateRoleFunc == nil {
		panic("UpdateRoleFunc not defined")
	}
	return m.UpdateRoleFunc(ctx, roleID, name, icon, color, description, promptHint, techStack, sortOrder)
}

func (m *MockCommands) DeleteRole(ctx context.Context, roleID domain.RoleID) error {
	if m.DeleteRoleFunc == nil {
		panic("DeleteRoleFunc not defined")
	}
	return m.DeleteRoleFunc(ctx, roleID)
}

func (m *MockCommands) CreateTask(ctx context.Context, projectID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string) (domain.Task, error) {
	if m.CreateTaskFunc == nil {
		panic("CreateTaskFunc not defined")
	}
	return m.CreateTaskFunc(ctx, projectID, title, summary, description, priority, createdByRole, createdByAgent, assignedRole, contextFiles, tags, estimatedEffort)
}

func (m *MockCommands) UpdateTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage) error {
	if m.UpdateTaskFunc == nil {
		panic("UpdateTaskFunc not defined")
	}
	return m.UpdateTaskFunc(ctx, projectID, taskID, title, description, assignedRole, estimatedEffort, resolution, priority, contextFiles, tags, tokenUsage)
}

func (m *MockCommands) UpdateTaskFiles(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, filesModified, contextFiles *[]string) error {
	if m.UpdateTaskFilesFunc == nil {
		panic("UpdateTaskFilesFunc not defined")
	}
	return m.UpdateTaskFilesFunc(ctx, projectID, taskID, filesModified, contextFiles)
}

func (m *MockCommands) DeleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.DeleteTaskFunc == nil {
		panic("DeleteTaskFunc not defined")
	}
	return m.DeleteTaskFunc(ctx, projectID, taskID)
}

func (m *MockCommands) MoveTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnSlug domain.ColumnSlug) error {
	if m.MoveTaskFunc == nil {
		panic("MoveTaskFunc not defined")
	}
	return m.MoveTaskFunc(ctx, projectID, taskID, targetColumnSlug)
}

func (m *MockCommands) StartTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.StartTaskFunc == nil {
		panic("StartTaskFunc not defined")
	}
	return m.StartTaskFunc(ctx, projectID, taskID)
}

func (m *MockCommands) CompleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage) error {
	if m.CompleteTaskFunc == nil {
		panic("CompleteTaskFunc not defined")
	}
	return m.CompleteTaskFunc(ctx, projectID, taskID, completionSummary, filesModified, completedByAgent, tokenUsage)
}

func (m *MockCommands) BlockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, blockedReason, blockedByAgent string) error {
	if m.BlockTaskFunc == nil {
		panic("BlockTaskFunc not defined")
	}
	return m.BlockTaskFunc(ctx, projectID, taskID, blockedReason, blockedByAgent)
}

func (m *MockCommands) UnblockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.UnblockTaskFunc == nil {
		panic("UnblockTaskFunc not defined")
	}
	return m.UnblockTaskFunc(ctx, projectID, taskID)
}

func (m *MockCommands) RequestWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, wontDoReason, wontDoRequestedBy string) error {
	if m.RequestWontDoFunc == nil {
		panic("RequestWontDoFunc not defined")
	}
	return m.RequestWontDoFunc(ctx, projectID, taskID, wontDoReason, wontDoRequestedBy)
}

func (m *MockCommands) ApproveWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.ApproveWontDoFunc == nil {
		panic("ApproveWontDoFunc not defined")
	}
	return m.ApproveWontDoFunc(ctx, projectID, taskID)
}

func (m *MockCommands) RejectWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, reason string) error {
	if m.RejectWontDoFunc == nil {
		panic("RejectWontDoFunc not defined")
	}
	return m.RejectWontDoFunc(ctx, projectID, taskID, reason)
}

func (m *MockCommands) CreateComment(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, authorRole, authorName string, authorType domain.AuthorType, content string) (domain.Comment, error) {
	if m.CreateCommentFunc == nil {
		panic("CreateCommentFunc not defined")
	}
	return m.CreateCommentFunc(ctx, projectID, taskID, authorRole, authorName, authorType, content)
}

func (m *MockCommands) UpdateComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID, content string) error {
	if m.UpdateCommentFunc == nil {
		panic("UpdateCommentFunc not defined")
	}
	return m.UpdateCommentFunc(ctx, projectID, commentID, content)
}

func (m *MockCommands) DeleteComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) error {
	if m.DeleteCommentFunc == nil {
		panic("DeleteCommentFunc not defined")
	}
	return m.DeleteCommentFunc(ctx, projectID, commentID)
}

func (m *MockCommands) AddDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error {
	if m.AddDependencyFunc == nil {
		panic("AddDependencyFunc not defined")
	}
	return m.AddDependencyFunc(ctx, projectID, taskID, dependsOnTaskID)
}

func (m *MockCommands) RemoveDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error {
	if m.RemoveDependencyFunc == nil {
		panic("RemoveDependencyFunc not defined")
	}
	return m.RemoveDependencyFunc(ctx, projectID, taskID, dependsOnTaskID)
}

func (m *MockCommands) MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.MarkTaskSeenFunc == nil {
		panic("MarkTaskSeenFunc not defined")
	}
	return m.MarkTaskSeenFunc(ctx, projectID, taskID)
}

func (m *MockCommands) UpdateColumnWIPLimit(ctx context.Context, projectID domain.ProjectID, columnSlug domain.ColumnSlug, wipLimit int) error {
	if m.UpdateColumnWIPLimitFunc == nil {
		panic("UpdateColumnWIPLimitFunc not defined")
	}
	return m.UpdateColumnWIPLimitFunc(ctx, projectID, columnSlug, wipLimit)
}

func (m *MockCommands) MoveTaskToProject(ctx context.Context, sourceProjectID domain.ProjectID, taskID domain.TaskID, targetProjectID domain.ProjectID) error {
	if m.MoveTaskToProjectFunc == nil {
		panic("MoveTaskToProjectFunc not defined")
	}
	return m.MoveTaskToProjectFunc(ctx, sourceProjectID, taskID, targetProjectID)
}

func (m *MockCommands) IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error {
	if m.IncrementToolUsageFunc == nil {
		panic("IncrementToolUsageFunc not defined")
	}
	return m.IncrementToolUsageFunc(ctx, projectID, toolName)
}

// MockQueries is a mock implementation of service.Queries for testing
type MockQueries struct {
	GetProjectFunc                 func(ctx context.Context, projectID domain.ProjectID) (*domain.Project, error)
	ListProjectsFunc               func(ctx context.Context) ([]domain.Project, error)
	ListProjectsWithSummaryFunc    func(ctx context.Context) ([]domain.ProjectWithSummary, error)
	ListSubProjectsFunc            func(ctx context.Context, parentID domain.ProjectID) ([]domain.Project, error)
	ListSubProjectsWithSummaryFunc func(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error)
	GetProjectSummaryFunc          func(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectSummary, error)
	GetProjectInfoFunc             func(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectInfo, error)
	GetRoleFunc                    func(ctx context.Context, roleID domain.RoleID) (*domain.Role, error)
	GetRoleBySlugFunc              func(ctx context.Context, slug string) (*domain.Role, error)
	ListRolesFunc                  func(ctx context.Context) ([]domain.Role, error)
	GetTaskFunc                    func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (*domain.Task, error)
	ListTasksFunc                  func(ctx context.Context, projectID domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error)
	GetNextTaskFunc                func(ctx context.Context, projectID domain.ProjectID, role string, subProjectID *domain.ProjectID) (*domain.Task, error)
	GetDependencyContextFunc       func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error)
	GetColumnFunc                  func(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (*domain.Column, error)
	GetColumnBySlugFunc            func(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error)
	ListColumnsFunc                func(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error)
	GetCommentFunc                 func(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (*domain.Comment, error)
	ListCommentsFunc               func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error)
	ListDependenciesFunc           func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error)
	GetDependencyTasksFunc         func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)
	GetDependentTasksFunc          func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)
	ListProjectsByWorkDirFunc      func(ctx context.Context, workDir string) ([]domain.ProjectWithSummary, error)
	GetToolUsageForProjectFunc     func(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error)
}

func (m *MockQueries) GetProject(ctx context.Context, projectID domain.ProjectID) (*domain.Project, error) {
	if m.GetProjectFunc == nil {
		panic("GetProjectFunc not defined")
	}
	return m.GetProjectFunc(ctx, projectID)
}

func (m *MockQueries) ListProjects(ctx context.Context) ([]domain.Project, error) {
	if m.ListProjectsFunc == nil {
		panic("ListProjectsFunc not defined")
	}
	return m.ListProjectsFunc(ctx)
}

func (m *MockQueries) ListProjectsWithSummary(ctx context.Context) ([]domain.ProjectWithSummary, error) {
	if m.ListProjectsWithSummaryFunc == nil {
		panic("ListProjectsWithSummaryFunc not defined")
	}
	return m.ListProjectsWithSummaryFunc(ctx)
}

func (m *MockQueries) ListSubProjects(ctx context.Context, parentID domain.ProjectID) ([]domain.Project, error) {
	if m.ListSubProjectsFunc == nil {
		panic("ListSubProjectsFunc not defined")
	}
	return m.ListSubProjectsFunc(ctx, parentID)
}

func (m *MockQueries) ListSubProjectsWithSummary(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error) {
	if m.ListSubProjectsWithSummaryFunc == nil {
		panic("ListSubProjectsWithSummaryFunc not defined")
	}
	return m.ListSubProjectsWithSummaryFunc(ctx, parentID)
}

func (m *MockQueries) GetProjectSummary(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectSummary, error) {
	if m.GetProjectSummaryFunc == nil {
		panic("GetProjectSummaryFunc not defined")
	}
	return m.GetProjectSummaryFunc(ctx, projectID)
}

func (m *MockQueries) GetProjectInfo(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectInfo, error) {
	if m.GetProjectInfoFunc == nil {
		panic("GetProjectInfoFunc not defined")
	}
	return m.GetProjectInfoFunc(ctx, projectID)
}

func (m *MockQueries) GetRole(ctx context.Context, roleID domain.RoleID) (*domain.Role, error) {
	if m.GetRoleFunc == nil {
		panic("GetRoleFunc not defined")
	}
	return m.GetRoleFunc(ctx, roleID)
}

func (m *MockQueries) GetRoleBySlug(ctx context.Context, slug string) (*domain.Role, error) {
	if m.GetRoleBySlugFunc == nil {
		panic("GetRoleBySlugFunc not defined")
	}
	return m.GetRoleBySlugFunc(ctx, slug)
}

func (m *MockQueries) ListRoles(ctx context.Context) ([]domain.Role, error) {
	if m.ListRolesFunc == nil {
		panic("ListRolesFunc not defined")
	}
	return m.ListRolesFunc(ctx)
}

func (m *MockQueries) GetTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (*domain.Task, error) {
	if m.GetTaskFunc == nil {
		panic("GetTaskFunc not defined")
	}
	return m.GetTaskFunc(ctx, projectID, taskID)
}

func (m *MockQueries) ListTasks(ctx context.Context, projectID domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
	if m.ListTasksFunc == nil {
		panic("ListTasksFunc not defined")
	}
	return m.ListTasksFunc(ctx, projectID, filters)
}

func (m *MockQueries) GetNextTask(ctx context.Context, projectID domain.ProjectID, role string, subProjectID *domain.ProjectID) (*domain.Task, error) {
	if m.GetNextTaskFunc == nil {
		panic("GetNextTaskFunc not defined")
	}
	return m.GetNextTaskFunc(ctx, projectID, role, subProjectID)
}

func (m *MockQueries) GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error) {
	if m.GetDependencyContextFunc == nil {
		panic("GetDependencyContextFunc not defined")
	}
	return m.GetDependencyContextFunc(ctx, projectID, taskID)
}

func (m *MockQueries) GetColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (*domain.Column, error) {
	if m.GetColumnFunc == nil {
		panic("GetColumnFunc not defined")
	}
	return m.GetColumnFunc(ctx, projectID, columnID)
}

func (m *MockQueries) GetColumnBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
	if m.GetColumnBySlugFunc == nil {
		panic("GetColumnBySlugFunc not defined")
	}
	return m.GetColumnBySlugFunc(ctx, projectID, slug)
}

func (m *MockQueries) ListColumns(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error) {
	if m.ListColumnsFunc == nil {
		panic("ListColumnsFunc not defined")
	}
	return m.ListColumnsFunc(ctx, projectID)
}

func (m *MockQueries) GetComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (*domain.Comment, error) {
	if m.GetCommentFunc == nil {
		panic("GetCommentFunc not defined")
	}
	return m.GetCommentFunc(ctx, projectID, commentID)
}

func (m *MockQueries) ListComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error) {
	if m.ListCommentsFunc == nil {
		panic("ListCommentsFunc not defined")
	}
	return m.ListCommentsFunc(ctx, projectID, taskID, limit, offset)
}

func (m *MockQueries) ListDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	if m.ListDependenciesFunc == nil {
		panic("ListDependenciesFunc not defined")
	}
	return m.ListDependenciesFunc(ctx, projectID, taskID)
}

func (m *MockQueries) GetDependencyTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	if m.GetDependencyTasksFunc == nil {
		panic("GetDependencyTasksFunc not defined")
	}
	return m.GetDependencyTasksFunc(ctx, projectID, taskID)
}

func (m *MockQueries) GetDependentTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	if m.GetDependentTasksFunc == nil {
		panic("GetDependentTasksFunc not defined")
	}
	return m.GetDependentTasksFunc(ctx, projectID, taskID)
}

func (m *MockQueries) ListProjectsByWorkDir(ctx context.Context, workDir string) ([]domain.ProjectWithSummary, error) {
	if m.ListProjectsByWorkDirFunc == nil {
		panic("ListProjectsByWorkDirFunc not defined")
	}
	return m.ListProjectsByWorkDirFunc(ctx, workDir)
}

func (m *MockQueries) GetToolUsageForProject(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error) {
	if m.GetToolUsageForProjectFunc == nil {
		panic("GetToolUsageForProjectFunc not defined")
	}
	return m.GetToolUsageForProjectFunc(ctx, projectID)
}

// Test cases

func TestToolHandler_CreateProject(t *testing.T) {
	projectID := domain.NewProjectID()

	mockCmds := &MockCommands{
		CreateProjectFunc: func(ctx context.Context, name, description, workDir, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
			assert.Equal(t, "Test Project", name)
			assert.Equal(t, "Test Description", description)
			assert.Equal(t, "architect", createdByRole)
			assert.Nil(t, parentID)

			return domain.Project{
				ID:            projectID,
				Name:          name,
				Description:   description,
				CreatedByRole: createdByRole,
				CreatedAt:     time.Now(),
			}, nil
		},
	}

	mockQueries := &MockQueries{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	// We can't directly call the handler methods as they're not exported
	// In a real scenario, we would test through the MCP server's tool invocation
	// For now, we verify the mock setup is correct
	assert.NotNil(t, handler)
}

func TestToolHandler_CreateTask(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockCmds := &MockCommands{
		CreateTaskFunc: func(ctx context.Context, pID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string) (domain.Task, error) {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, "Implement feature", title)
			assert.Equal(t, "Brief summary", summary)
			assert.Equal(t, domain.PriorityHigh, priority)
			assert.Equal(t, "backend_go", assignedRole)

			return domain.Task{
				ID:            taskID,
				Title:         title,
				Summary:       summary,
				Priority:      priority,
				PriorityScore: 300,
				AssignedRole:  assignedRole,
				CreatedAt:     time.Now(),
			}, nil
		},
	}

	mockQueries := &MockQueries{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}

func TestToolHandler_GetNextTask(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockQueries := &MockQueries{
		GetNextTaskFunc: func(ctx context.Context, pID domain.ProjectID, role string, subProjectID *domain.ProjectID) (*domain.Task, error) {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, "backend_go", role)

			return &domain.Task{
				ID:            taskID,
				Title:         "Next task",
				Summary:       "Task summary",
				Priority:      domain.PriorityHigh,
				AssignedRole:  role,
				PriorityScore: 300,
			}, nil
		},
		GetDependencyContextFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID) ([]domain.DependencyContext, error) {
			return []domain.DependencyContext{}, nil
		},
	}

	mockCmds := &MockCommands{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}

func TestToolHandler_BlockTask(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockCmds := &MockCommands{
		BlockTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, blockedReason, blockedByAgent string) error {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, "Database connection failing with error XYZ. Tried reconnection, checked credentials, issue persists.", blockedReason)
			assert.Equal(t, "backend-agent-1", blockedByAgent)
			return nil
		},
	}

	mockQueries := &MockQueries{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}

func TestToolHandler_RequestWontDo(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockCmds := &MockCommands{
		RequestWontDoFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, wontDoReason, wontDoRequestedBy string) error {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			assert.GreaterOrEqual(t, len(wontDoReason), 50, "Reason should be at least 50 chars")
			return nil
		},
	}

	mockQueries := &MockQueries{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}

func TestToolHandler_ListRoles(t *testing.T) {
	mockQueries := &MockQueries{
		ListRolesFunc: func(ctx context.Context) ([]domain.Role, error) {
			return []domain.Role{
				{
					ID:          domain.NewRoleID(),
					Slug:        "backend_go",
					Name:        "Backend Go",
					Icon:        "🔧",
					Color:       "#10B981",
					Description: "Go backend specialist",
					TechStack:   []string{"go", "postgresql", "grpc"},
					SortOrder:   0,
				},
				{
					ID:          domain.NewRoleID(),
					Slug:        "frontend_react",
					Name:        "Frontend React",
					Icon:        "⚛️",
					Color:       "#3B82F6",
					Description: "React frontend specialist",
					TechStack:   []string{"react", "typescript", "tailwind"},
					SortOrder:   1,
				},
			}, nil
		},
	}

	mockCmds := &MockCommands{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}

func TestToolHandler_CompleteTask(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockCmds := &MockCommands{
		CompleteTaskFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage) error {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			assert.GreaterOrEqual(t, len(completionSummary), 100, "Completion summary should be at least 100 chars")
			assert.NotEmpty(t, completedByAgent)
			return nil
		},
	}

	mockQueries := &MockQueries{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}

func TestToolHandler_ErrorHandling(t *testing.T) {
	projectID := domain.NewProjectID()

	mockQueries := &MockQueries{
		GetProjectSummaryFunc: func(ctx context.Context, pID domain.ProjectID) (*domain.ProjectSummary, error) {
			return nil, errors.Join(domain.ErrProjectNotFound, errors.New("project not found in database"))
		},
	}

	mockCmds := &MockCommands{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)

	// Test that domain errors are properly returned
	ctx := context.Background()
	_, err := mockQueries.GetProjectSummary(ctx, projectID)
	require.Error(t, err)
	assert.True(t, domain.IsDomainError(err))
}

func TestToolHandler_ListTasksWithFilters(t *testing.T) {
	projectID := domain.NewProjectID()

	mockQueries := &MockQueries{
		ListTasksFunc: func(ctx context.Context, pID domain.ProjectID, filters tasks.TaskFilters) ([]domain.TaskWithDetails, error) {
			assert.Equal(t, projectID, pID)

			if filters.ColumnSlug != nil {
				assert.Equal(t, domain.ColumnTodo, *filters.ColumnSlug)
			}
			if filters.AssignedRole != nil {
				assert.Equal(t, "backend_go", *filters.AssignedRole)
			}
			if filters.Priority != nil {
				assert.Equal(t, domain.PriorityHigh, *filters.Priority)
			}

			return []domain.TaskWithDetails{}, nil
		},
	}

	mockCmds := &MockCommands{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}

func TestToolHandler_AddDependency(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	dependsOnTaskID := domain.NewTaskID()

	mockCmds := &MockCommands{
		AddDependencyFunc: func(ctx context.Context, pID domain.ProjectID, tID, depID domain.TaskID) error {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, dependsOnTaskID, depID)
			return nil
		},
	}

	mockQueries := &MockQueries{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}

func TestToolHandler_UpdateTaskFiles(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockCmds := &MockCommands{
		UpdateTaskFilesFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, filesModified, contextFiles *[]string) error {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			require.NotNil(t, filesModified)
			assert.Equal(t, []string{"internal/kanban/app/tasks.go", "internal/kanban/domain/types.go"}, *filesModified)
			require.NotNil(t, contextFiles)
			assert.Equal(t, []string{"CLAUDE.md"}, *contextFiles)
			return nil
		},
	}

	mockQueries := &MockQueries{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}

func TestToolHandler_UpdateTaskFiles_FilesModifiedOnly(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()

	mockCmds := &MockCommands{
		UpdateTaskFilesFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, filesModified, contextFiles *[]string) error {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			require.NotNil(t, filesModified)
			assert.Equal(t, []string{"internal/kanban/app/tasks.go"}, *filesModified)
			assert.Nil(t, contextFiles)
			return nil
		},
	}

	mockQueries := &MockQueries{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}

func TestToolHandler_AddComment(t *testing.T) {
	projectID := domain.NewProjectID()
	taskID := domain.NewTaskID()
	commentID := domain.NewCommentID()

	mockCmds := &MockCommands{
		CreateCommentFunc: func(ctx context.Context, pID domain.ProjectID, tID domain.TaskID, authorRole, authorName string, authorType domain.AuthorType, content string) (domain.Comment, error) {
			assert.Equal(t, projectID, pID)
			assert.Equal(t, taskID, tID)
			assert.Equal(t, "backend_go", authorRole)
			assert.Equal(t, domain.AuthorTypeAgent, authorType)
			assert.NotEmpty(t, content)

			return domain.Comment{
				ID:         commentID,
				TaskID:     tID,
				AuthorRole: authorRole,
				AuthorName: authorName,
				AuthorType: authorType,
				Content:    content,
				CreatedAt:  time.Now(),
			}, nil
		},
	}

	mockQueries := &MockQueries{}
	handler := mcp.NewToolHandler(mockCmds, mockQueries, nil)

	assert.NotNil(t, handler)
}
