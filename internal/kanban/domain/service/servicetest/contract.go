package servicetest

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// MockCommands is a function-based mock implementation of the service.Commands interface.
// It allows flexible testing by injecting custom behavior for each method.
//
// Example usage:
//
//	mock := &MockCommands{
//		CreateProjectFunc: func(ctx context.Context, name, description, workDir, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
//			return domain.Project{ID: domain.NewProjectID(), Name: name}, nil
//		},
//	}
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
		panic("called not defined CreateProjectFunc")
	}
	return m.CreateProjectFunc(ctx, name, description, workDir, createdByRole, createdByAgent, parentID)
}

func (m *MockCommands) UpdateProject(ctx context.Context, projectID domain.ProjectID, name, description string) error {
	if m.UpdateProjectFunc == nil {
		panic("called not defined UpdateProjectFunc")
	}
	return m.UpdateProjectFunc(ctx, projectID, name, description)
}

func (m *MockCommands) DeleteProject(ctx context.Context, projectID domain.ProjectID) error {
	if m.DeleteProjectFunc == nil {
		panic("called not defined DeleteProjectFunc")
	}
	return m.DeleteProjectFunc(ctx, projectID)
}

func (m *MockCommands) CreateRole(ctx context.Context, slug, name, icon, color, description, promptHint string, techStack []string, sortOrder int) (domain.Role, error) {
	if m.CreateRoleFunc == nil {
		panic("called not defined CreateRoleFunc")
	}
	return m.CreateRoleFunc(ctx, slug, name, icon, color, description, promptHint, techStack, sortOrder)
}

func (m *MockCommands) UpdateRole(ctx context.Context, roleID domain.RoleID, name, icon, color, description, promptHint string, techStack []string, sortOrder int) error {
	if m.UpdateRoleFunc == nil {
		panic("called not defined UpdateRoleFunc")
	}
	return m.UpdateRoleFunc(ctx, roleID, name, icon, color, description, promptHint, techStack, sortOrder)
}

func (m *MockCommands) DeleteRole(ctx context.Context, roleID domain.RoleID) error {
	if m.DeleteRoleFunc == nil {
		panic("called not defined DeleteRoleFunc")
	}
	return m.DeleteRoleFunc(ctx, roleID)
}

func (m *MockCommands) CreateTask(ctx context.Context, projectID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string) (domain.Task, error) {
	if m.CreateTaskFunc == nil {
		panic("called not defined CreateTaskFunc")
	}
	return m.CreateTaskFunc(ctx, projectID, title, summary, description, priority, createdByRole, createdByAgent, assignedRole, contextFiles, tags, estimatedEffort)
}

func (m *MockCommands) UpdateTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage) error {
	if m.UpdateTaskFunc == nil {
		panic("called not defined UpdateTaskFunc")
	}
	return m.UpdateTaskFunc(ctx, projectID, taskID, title, description, assignedRole, estimatedEffort, resolution, priority, contextFiles, tags, tokenUsage)
}

func (m *MockCommands) UpdateTaskFiles(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, filesModified, contextFiles *[]string) error {
	if m.UpdateTaskFilesFunc == nil {
		panic("called not defined UpdateTaskFilesFunc")
	}
	return m.UpdateTaskFilesFunc(ctx, projectID, taskID, filesModified, contextFiles)
}

func (m *MockCommands) DeleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.DeleteTaskFunc == nil {
		panic("called not defined DeleteTaskFunc")
	}
	return m.DeleteTaskFunc(ctx, projectID, taskID)
}

func (m *MockCommands) MoveTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnSlug domain.ColumnSlug) error {
	if m.MoveTaskFunc == nil {
		panic("called not defined MoveTaskFunc")
	}
	return m.MoveTaskFunc(ctx, projectID, taskID, targetColumnSlug)
}

func (m *MockCommands) StartTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.StartTaskFunc == nil {
		panic("called not defined StartTaskFunc")
	}
	return m.StartTaskFunc(ctx, projectID, taskID)
}

func (m *MockCommands) CompleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage) error {
	if m.CompleteTaskFunc == nil {
		panic("called not defined CompleteTaskFunc")
	}
	return m.CompleteTaskFunc(ctx, projectID, taskID, completionSummary, filesModified, completedByAgent, tokenUsage)
}

func (m *MockCommands) BlockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, blockedReason, blockedByAgent string) error {
	if m.BlockTaskFunc == nil {
		panic("called not defined BlockTaskFunc")
	}
	return m.BlockTaskFunc(ctx, projectID, taskID, blockedReason, blockedByAgent)
}

func (m *MockCommands) UnblockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.UnblockTaskFunc == nil {
		panic("called not defined UnblockTaskFunc")
	}
	return m.UnblockTaskFunc(ctx, projectID, taskID)
}

func (m *MockCommands) RequestWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, wontDoReason, wontDoRequestedBy string) error {
	if m.RequestWontDoFunc == nil {
		panic("called not defined RequestWontDoFunc")
	}
	return m.RequestWontDoFunc(ctx, projectID, taskID, wontDoReason, wontDoRequestedBy)
}

func (m *MockCommands) ApproveWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.ApproveWontDoFunc == nil {
		panic("called not defined ApproveWontDoFunc")
	}
	return m.ApproveWontDoFunc(ctx, projectID, taskID)
}

func (m *MockCommands) RejectWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, reason string) error {
	if m.RejectWontDoFunc == nil {
		panic("called not defined RejectWontDoFunc")
	}
	return m.RejectWontDoFunc(ctx, projectID, taskID, reason)
}

func (m *MockCommands) CreateComment(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, authorRole, authorName string, authorType domain.AuthorType, content string) (domain.Comment, error) {
	if m.CreateCommentFunc == nil {
		panic("called not defined CreateCommentFunc")
	}
	return m.CreateCommentFunc(ctx, projectID, taskID, authorRole, authorName, authorType, content)
}

func (m *MockCommands) UpdateComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID, content string) error {
	if m.UpdateCommentFunc == nil {
		panic("called not defined UpdateCommentFunc")
	}
	return m.UpdateCommentFunc(ctx, projectID, commentID, content)
}

func (m *MockCommands) DeleteComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) error {
	if m.DeleteCommentFunc == nil {
		panic("called not defined DeleteCommentFunc")
	}
	return m.DeleteCommentFunc(ctx, projectID, commentID)
}

func (m *MockCommands) AddDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error {
	if m.AddDependencyFunc == nil {
		panic("called not defined AddDependencyFunc")
	}
	return m.AddDependencyFunc(ctx, projectID, taskID, dependsOnTaskID)
}

func (m *MockCommands) RemoveDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error {
	if m.RemoveDependencyFunc == nil {
		panic("called not defined RemoveDependencyFunc")
	}
	return m.RemoveDependencyFunc(ctx, projectID, taskID, dependsOnTaskID)
}

func (m *MockCommands) MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error {
	if m.MarkTaskSeenFunc == nil {
		panic("called not defined MarkTaskSeenFunc")
	}
	return m.MarkTaskSeenFunc(ctx, projectID, taskID)
}

func (m *MockCommands) UpdateColumnWIPLimit(ctx context.Context, projectID domain.ProjectID, columnSlug domain.ColumnSlug, wipLimit int) error {
	if m.UpdateColumnWIPLimitFunc == nil {
		panic("called not defined UpdateColumnWIPLimitFunc")
	}
	return m.UpdateColumnWIPLimitFunc(ctx, projectID, columnSlug, wipLimit)
}

func (m *MockCommands) MoveTaskToProject(ctx context.Context, sourceProjectID domain.ProjectID, taskID domain.TaskID, targetProjectID domain.ProjectID) error {
	if m.MoveTaskToProjectFunc == nil {
		panic("called not defined MoveTaskToProjectFunc")
	}
	return m.MoveTaskToProjectFunc(ctx, sourceProjectID, taskID, targetProjectID)
}

func (m *MockCommands) IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error {
	if m.IncrementToolUsageFunc == nil {
		panic("called not defined IncrementToolUsageFunc")
	}
	return m.IncrementToolUsageFunc(ctx, projectID, toolName)
}
