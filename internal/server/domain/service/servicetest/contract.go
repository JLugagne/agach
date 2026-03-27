package servicetest

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
	"github.com/JLugagne/agach-mcp/internal/server/domain/service"
)

// MockCommands is a function-based mock implementation of the service.Commands interface.
// It allows flexible testing by injecting custom behavior for each method.
//
// Example usage:
//
//	mock := &MockCommands{
//		CreateProjectFunc: func(ctx context.Context, name, description, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
//			return domain.Project{ID: domain.NewProjectID(), Name: name}, nil
//		},
//	}
type MockCommands struct {
	CreateProjectFunc            func(ctx context.Context, name, description, gitURL, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error)
	UpdateProjectFunc            func(ctx context.Context, projectID domain.ProjectID, name, description string, gitURL, defaultRole *string) error
	DeleteProjectFunc            func(ctx context.Context, projectID domain.ProjectID) error
	CreateAgentFunc              func(ctx context.Context, slug, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) (domain.Agent, error)
	UpdateAgentFunc              func(ctx context.Context, roleID domain.AgentID, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) error
	DeleteAgentFunc              func(ctx context.Context, roleID domain.AgentID) error
	CreateProjectAgentFunc       func(ctx context.Context, projectID domain.ProjectID, slug, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) (domain.Agent, error)
	UpdateProjectAgentFunc       func(ctx context.Context, projectID domain.ProjectID, roleID domain.AgentID, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) error
	DeleteProjectAgentFunc       func(ctx context.Context, projectID domain.ProjectID, roleID domain.AgentID) error
	CreateTaskFunc               func(ctx context.Context, projectID domain.ProjectID, input service.CreateTaskInput) (domain.Task, error)
	BulkCreateTasksFunc          func(ctx context.Context, projectID domain.ProjectID, inputs []service.BulkTaskInput) ([]domain.Task, error)
	UpdateTaskFunc               func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.FeatureID, clearFeature bool) error
	UpdateTaskFilesFunc          func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, filesModified, contextFiles *[]string) error
	DeleteTaskFunc               func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	MoveTaskFunc                 func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnSlug domain.ColumnSlug, nodeID string) error
	ReorderTaskFunc              func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, newPosition int) error
	StartTaskFunc                func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, nodeID string) error
	CompleteTaskFunc             func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage, nodeID string) error
	BlockTaskFunc                func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, blockedReason, blockedByAgent, nodeID string) error
	UnblockTaskFunc              func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, nodeID string) error
	RequestWontDoFunc            func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, wontDoReason, wontDoRequestedBy, nodeID string) error
	ApproveWontDoFunc            func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	RejectWontDoFunc             func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, reason string) error
	CreateCommentFunc            func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, authorRole, authorName string, authorType domain.AuthorType, content string) (domain.Comment, error)
	UpdateCommentFunc            func(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID, content string) error
	DeleteCommentFunc            func(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) error
	AddDependencyFunc            func(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error
	RemoveDependencyFunc         func(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error
	MarkTaskSeenFunc             func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	MoveTaskToProjectFunc        func(ctx context.Context, sourceProjectID domain.ProjectID, taskID domain.TaskID, targetProjectID domain.ProjectID) error
	IncrementToolUsageFunc       func(ctx context.Context, projectID domain.ProjectID, toolName string) error
	UpdateTaskSessionIDFunc      func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error
	CloneAgentFunc               func(ctx context.Context, sourceSlug, newSlug, newName string) (domain.Agent, error)
	AssignAgentToProjectFunc     func(ctx context.Context, projectID domain.ProjectID, agentSlug string) error
	RemoveAgentFromProjectFunc   func(ctx context.Context, projectID domain.ProjectID, agentSlug string, reassignTo *string, clearAssignment bool) error
	BulkReassignTasksFunc        func(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error)
	CreateSkillFunc              func(ctx context.Context, slug, name, description, content, icon, color string, sortOrder int) (domain.Skill, error)
	UpdateSkillFunc              func(ctx context.Context, skillID domain.SkillID, name, description, content, icon, color string, sortOrder int) error
	DeleteSkillFunc              func(ctx context.Context, skillID domain.SkillID) error
	AddSkillToAgentFunc          func(ctx context.Context, agentSlug, skillSlug string) error
	RemoveSkillFromAgentFunc     func(ctx context.Context, agentSlug, skillSlug string) error
	CreateDockerfileFunc         func(ctx context.Context, slug, name, description, version, content string, isLatest bool, sortOrder int) (domain.Dockerfile, error)
	UpdateDockerfileFunc         func(ctx context.Context, dockerfileID domain.DockerfileID, name, description, content *string, isLatest *bool, sortOrder *int) error
	DeleteDockerfileFunc         func(ctx context.Context, dockerfileID domain.DockerfileID) error
	SetProjectDockerfileFunc     func(ctx context.Context, projectID domain.ProjectID, dockerfileID domain.DockerfileID) error
	ClearProjectDockerfileFunc   func(ctx context.Context, projectID domain.ProjectID) error
	CreateFeatureFunc            func(ctx context.Context, projectID domain.ProjectID, name, description, createdByRole, createdByAgent string) (domain.Feature, error)
	UpdateFeatureFunc            func(ctx context.Context, featureID domain.FeatureID, name, description string) error
	UpdateFeatureStatusFunc      func(ctx context.Context, featureID domain.FeatureID, status domain.FeatureStatus, nodeID string) error
	DeleteFeatureFunc            func(ctx context.Context, featureID domain.FeatureID) error
	UpdateFeatureChangelogsFunc  func(ctx context.Context, featureID domain.FeatureID, userChangelog, techChangelog *string) error
	CreateNotificationFunc       func(ctx context.Context, projectID *domain.ProjectID, scope domain.NotificationScope, agentSlug string, severity domain.NotificationSeverity, title, text, linkURL, linkText, linkStyle string) (domain.Notification, error)
	MarkNotificationReadFunc     func(ctx context.Context, notificationID domain.NotificationID) error
	MarkAllNotificationsReadFunc func(ctx context.Context, projectID *domain.ProjectID) error
	DeleteNotificationFunc       func(ctx context.Context, notificationID domain.NotificationID) error
	CreateSpecializedAgentFunc   func(ctx context.Context, parentSlug, slug, name string, skillSlugs []string, sortOrder int) (domain.SpecializedAgent, error)
	UpdateSpecializedAgentFunc   func(ctx context.Context, id domain.SpecializedAgentID, name string, skillSlugs []string, sortOrder int) error
	DeleteSpecializedAgentFunc   func(ctx context.Context, id domain.SpecializedAgentID) error
	GrantUserAccessFunc          func(ctx context.Context, projectID domain.ProjectID, userID, role string) error
	RevokeUserAccessFunc         func(ctx context.Context, projectID domain.ProjectID, userID string) error
	UpdateUserAccessRoleFunc     func(ctx context.Context, projectID domain.ProjectID, userID, role string) error
	GrantTeamAccessFunc          func(ctx context.Context, projectID domain.ProjectID, teamID string) error
	RevokeTeamAccessFunc         func(ctx context.Context, projectID domain.ProjectID, teamID string) error
}

func (m *MockCommands) CreateProject(ctx context.Context, name, description, gitURL, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error) {
	if m.CreateProjectFunc == nil {
		panic("called not defined CreateProjectFunc")
	}
	return m.CreateProjectFunc(ctx, name, description, gitURL, createdByRole, createdByAgent, parentID)
}

func (m *MockCommands) UpdateProject(ctx context.Context, projectID domain.ProjectID, name, description string, gitURL, defaultRole *string) error {
	if m.UpdateProjectFunc == nil {
		panic("called not defined UpdateProjectFunc")
	}
	return m.UpdateProjectFunc(ctx, projectID, name, description, gitURL, defaultRole)
}

func (m *MockCommands) DeleteProject(ctx context.Context, projectID domain.ProjectID) error {
	if m.DeleteProjectFunc == nil {
		panic("called not defined DeleteProjectFunc")
	}
	return m.DeleteProjectFunc(ctx, projectID)
}

func (m *MockCommands) CreateAgent(ctx context.Context, slug, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) (domain.Agent, error) {
	if m.CreateAgentFunc == nil {
		panic("called not defined CreateAgentFunc")
	}
	return m.CreateAgentFunc(ctx, slug, name, icon, color, description, promptHint, promptTemplate, model, thinking, techStack, sortOrder)
}

func (m *MockCommands) UpdateAgent(ctx context.Context, roleID domain.AgentID, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) error {
	if m.UpdateAgentFunc == nil {
		panic("called not defined UpdateAgentFunc")
	}
	return m.UpdateAgentFunc(ctx, roleID, name, icon, color, description, promptHint, promptTemplate, model, thinking, techStack, sortOrder)
}

func (m *MockCommands) DeleteAgent(ctx context.Context, roleID domain.AgentID) error {
	if m.DeleteAgentFunc == nil {
		panic("called not defined DeleteAgentFunc")
	}
	return m.DeleteAgentFunc(ctx, roleID)
}

func (m *MockCommands) CreateProjectAgent(ctx context.Context, projectID domain.ProjectID, slug, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) (domain.Agent, error) {
	if m.CreateProjectAgentFunc == nil {
		panic("called not defined CreateProjectAgentFunc")
	}
	return m.CreateProjectAgentFunc(ctx, projectID, slug, name, icon, color, description, promptHint, promptTemplate, model, thinking, techStack, sortOrder)
}

func (m *MockCommands) UpdateProjectAgent(ctx context.Context, projectID domain.ProjectID, roleID domain.AgentID, name, icon, color, description, promptHint, promptTemplate, model, thinking string, techStack []string, sortOrder int) error {
	if m.UpdateProjectAgentFunc == nil {
		panic("called not defined UpdateProjectAgentFunc")
	}
	return m.UpdateProjectAgentFunc(ctx, projectID, roleID, name, icon, color, description, promptHint, promptTemplate, model, thinking, techStack, sortOrder)
}

func (m *MockCommands) DeleteProjectAgent(ctx context.Context, projectID domain.ProjectID, roleID domain.AgentID) error {
	if m.DeleteProjectAgentFunc == nil {
		panic("called not defined DeleteProjectAgentFunc")
	}
	return m.DeleteProjectAgentFunc(ctx, projectID, roleID)
}

func (m *MockCommands) CreateTask(ctx context.Context, projectID domain.ProjectID, input service.CreateTaskInput) (domain.Task, error) {
	if m.CreateTaskFunc == nil {
		panic("called not defined CreateTaskFunc")
	}
	return m.CreateTaskFunc(ctx, projectID, input)
}

func (m *MockCommands) BulkCreateTasks(ctx context.Context, projectID domain.ProjectID, inputs []service.BulkTaskInput) ([]domain.Task, error) {
	if m.BulkCreateTasksFunc == nil {
		panic("called not defined BulkCreateTasksFunc")
	}
	return m.BulkCreateTasksFunc(ctx, projectID, inputs)
}

func (m *MockCommands) UpdateTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.FeatureID, clearFeature bool) error {
	if m.UpdateTaskFunc == nil {
		panic("called not defined UpdateTaskFunc")
	}
	return m.UpdateTaskFunc(ctx, projectID, taskID, title, description, assignedRole, estimatedEffort, resolution, priority, contextFiles, tags, tokenUsage, humanEstimateSeconds, featureID, clearFeature)
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

func (m *MockCommands) MoveTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnSlug domain.ColumnSlug, nodeID string) error {
	if m.MoveTaskFunc == nil {
		panic("called not defined MoveTaskFunc")
	}
	return m.MoveTaskFunc(ctx, projectID, taskID, targetColumnSlug, nodeID)
}

func (m *MockCommands) ReorderTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, newPosition int) error {
	if m.ReorderTaskFunc == nil {
		panic("called not defined ReorderTaskFunc")
	}
	return m.ReorderTaskFunc(ctx, projectID, taskID, newPosition)
}

func (m *MockCommands) StartTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, nodeID string) error {
	if m.StartTaskFunc == nil {
		panic("called not defined StartTaskFunc")
	}
	return m.StartTaskFunc(ctx, projectID, taskID, nodeID)
}

func (m *MockCommands) CompleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage, nodeID string) error {
	if m.CompleteTaskFunc == nil {
		panic("called not defined CompleteTaskFunc")
	}
	return m.CompleteTaskFunc(ctx, projectID, taskID, completionSummary, filesModified, completedByAgent, tokenUsage, nodeID)
}

func (m *MockCommands) BlockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, blockedReason, blockedByAgent, nodeID string) error {
	if m.BlockTaskFunc == nil {
		panic("called not defined BlockTaskFunc")
	}
	return m.BlockTaskFunc(ctx, projectID, taskID, blockedReason, blockedByAgent, nodeID)
}

func (m *MockCommands) UnblockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, nodeID string) error {
	if m.UnblockTaskFunc == nil {
		panic("called not defined UnblockTaskFunc")
	}
	return m.UnblockTaskFunc(ctx, projectID, taskID, nodeID)
}

func (m *MockCommands) RequestWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, wontDoReason, wontDoRequestedBy, nodeID string) error {
	if m.RequestWontDoFunc == nil {
		panic("called not defined RequestWontDoFunc")
	}
	return m.RequestWontDoFunc(ctx, projectID, taskID, wontDoReason, wontDoRequestedBy, nodeID)
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

func (m *MockCommands) UpdateTaskSessionID(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error {
	if m.UpdateTaskSessionIDFunc == nil {
		panic("called not defined UpdateTaskSessionIDFunc")
	}
	return m.UpdateTaskSessionIDFunc(ctx, projectID, taskID, sessionID)
}

func (m *MockCommands) CloneAgent(ctx context.Context, sourceSlug, newSlug, newName string) (domain.Agent, error) {
	if m.CloneAgentFunc == nil {
		panic("called not defined CloneAgentFunc")
	}
	return m.CloneAgentFunc(ctx, sourceSlug, newSlug, newName)
}

func (m *MockCommands) AssignAgentToProject(ctx context.Context, projectID domain.ProjectID, agentSlug string) error {
	if m.AssignAgentToProjectFunc == nil {
		panic("called not defined AssignAgentToProjectFunc")
	}
	return m.AssignAgentToProjectFunc(ctx, projectID, agentSlug)
}

func (m *MockCommands) RemoveAgentFromProject(ctx context.Context, projectID domain.ProjectID, agentSlug string, reassignTo *string, clearAssignment bool) error {
	if m.RemoveAgentFromProjectFunc == nil {
		panic("called not defined RemoveAgentFromProjectFunc")
	}
	return m.RemoveAgentFromProjectFunc(ctx, projectID, agentSlug, reassignTo, clearAssignment)
}

func (m *MockCommands) BulkReassignTasks(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error) {
	if m.BulkReassignTasksFunc == nil {
		panic("called not defined BulkReassignTasksFunc")
	}
	return m.BulkReassignTasksFunc(ctx, projectID, oldSlug, newSlug)
}

func (m *MockCommands) CreateSkill(ctx context.Context, slug, name, description, content, icon, color string, sortOrder int) (domain.Skill, error) {
	if m.CreateSkillFunc == nil {
		panic("called not defined CreateSkillFunc")
	}
	return m.CreateSkillFunc(ctx, slug, name, description, content, icon, color, sortOrder)
}

func (m *MockCommands) UpdateSkill(ctx context.Context, skillID domain.SkillID, name, description, content, icon, color string, sortOrder int) error {
	if m.UpdateSkillFunc == nil {
		panic("called not defined UpdateSkillFunc")
	}
	return m.UpdateSkillFunc(ctx, skillID, name, description, content, icon, color, sortOrder)
}

func (m *MockCommands) DeleteSkill(ctx context.Context, skillID domain.SkillID) error {
	if m.DeleteSkillFunc == nil {
		panic("called not defined DeleteSkillFunc")
	}
	return m.DeleteSkillFunc(ctx, skillID)
}

func (m *MockCommands) AddSkillToAgent(ctx context.Context, agentSlug, skillSlug string) error {
	if m.AddSkillToAgentFunc == nil {
		panic("called not defined AddSkillToAgentFunc")
	}
	return m.AddSkillToAgentFunc(ctx, agentSlug, skillSlug)
}

func (m *MockCommands) RemoveSkillFromAgent(ctx context.Context, agentSlug, skillSlug string) error {
	if m.RemoveSkillFromAgentFunc == nil {
		panic("called not defined RemoveSkillFromAgentFunc")
	}
	return m.RemoveSkillFromAgentFunc(ctx, agentSlug, skillSlug)
}

func (m *MockCommands) CreateDockerfile(ctx context.Context, slug, name, description, version, content string, isLatest bool, sortOrder int) (domain.Dockerfile, error) {
	if m.CreateDockerfileFunc == nil {
		panic("called not defined CreateDockerfileFunc")
	}
	return m.CreateDockerfileFunc(ctx, slug, name, description, version, content, isLatest, sortOrder)
}

func (m *MockCommands) UpdateDockerfile(ctx context.Context, dockerfileID domain.DockerfileID, name, description, content *string, isLatest *bool, sortOrder *int) error {
	if m.UpdateDockerfileFunc == nil {
		panic("called not defined UpdateDockerfileFunc")
	}
	return m.UpdateDockerfileFunc(ctx, dockerfileID, name, description, content, isLatest, sortOrder)
}

func (m *MockCommands) DeleteDockerfile(ctx context.Context, dockerfileID domain.DockerfileID) error {
	if m.DeleteDockerfileFunc == nil {
		panic("called not defined DeleteDockerfileFunc")
	}
	return m.DeleteDockerfileFunc(ctx, dockerfileID)
}

func (m *MockCommands) SetProjectDockerfile(ctx context.Context, projectID domain.ProjectID, dockerfileID domain.DockerfileID) error {
	if m.SetProjectDockerfileFunc == nil {
		panic("called not defined SetProjectDockerfileFunc")
	}
	return m.SetProjectDockerfileFunc(ctx, projectID, dockerfileID)
}

func (m *MockCommands) ClearProjectDockerfile(ctx context.Context, projectID domain.ProjectID) error {
	if m.ClearProjectDockerfileFunc == nil {
		panic("called not defined ClearProjectDockerfileFunc")
	}
	return m.ClearProjectDockerfileFunc(ctx, projectID)
}

func (m *MockCommands) CreateFeature(ctx context.Context, projectID domain.ProjectID, name, description, createdByRole, createdByAgent string) (domain.Feature, error) {
	if m.CreateFeatureFunc == nil {
		panic("called not defined CreateFeatureFunc")
	}
	return m.CreateFeatureFunc(ctx, projectID, name, description, createdByRole, createdByAgent)
}

func (m *MockCommands) UpdateFeature(ctx context.Context, featureID domain.FeatureID, name, description string) error {
	if m.UpdateFeatureFunc == nil {
		panic("called not defined UpdateFeatureFunc")
	}
	return m.UpdateFeatureFunc(ctx, featureID, name, description)
}

func (m *MockCommands) UpdateFeatureStatus(ctx context.Context, featureID domain.FeatureID, status domain.FeatureStatus, nodeID string) error {
	if m.UpdateFeatureStatusFunc == nil {
		panic("called not defined UpdateFeatureStatusFunc")
	}
	return m.UpdateFeatureStatusFunc(ctx, featureID, status, nodeID)
}

func (m *MockCommands) DeleteFeature(ctx context.Context, featureID domain.FeatureID) error {
	if m.DeleteFeatureFunc == nil {
		panic("called not defined DeleteFeatureFunc")
	}
	return m.DeleteFeatureFunc(ctx, featureID)
}

func (m *MockCommands) UpdateFeatureChangelogs(ctx context.Context, featureID domain.FeatureID, userChangelog, techChangelog *string) error {
	if m.UpdateFeatureChangelogsFunc == nil {
		panic("called not defined UpdateFeatureChangelogsFunc")
	}
	return m.UpdateFeatureChangelogsFunc(ctx, featureID, userChangelog, techChangelog)
}

func (m *MockCommands) CreateNotification(ctx context.Context, projectID *domain.ProjectID, scope domain.NotificationScope, agentSlug string, severity domain.NotificationSeverity, title, text, linkURL, linkText, linkStyle string) (domain.Notification, error) {
	if m.CreateNotificationFunc == nil {
		panic("called not defined CreateNotificationFunc")
	}
	return m.CreateNotificationFunc(ctx, projectID, scope, agentSlug, severity, title, text, linkURL, linkText, linkStyle)
}

func (m *MockCommands) MarkNotificationRead(ctx context.Context, notificationID domain.NotificationID) error {
	if m.MarkNotificationReadFunc == nil {
		panic("called not defined MarkNotificationReadFunc")
	}
	return m.MarkNotificationReadFunc(ctx, notificationID)
}

func (m *MockCommands) MarkAllNotificationsRead(ctx context.Context, projectID *domain.ProjectID) error {
	if m.MarkAllNotificationsReadFunc == nil {
		panic("called not defined MarkAllNotificationsReadFunc")
	}
	return m.MarkAllNotificationsReadFunc(ctx, projectID)
}

func (m *MockCommands) DeleteNotification(ctx context.Context, notificationID domain.NotificationID) error {
	if m.DeleteNotificationFunc == nil {
		panic("called not defined DeleteNotificationFunc")
	}
	return m.DeleteNotificationFunc(ctx, notificationID)
}

func (m *MockCommands) CreateSpecializedAgent(ctx context.Context, parentSlug, slug, name string, skillSlugs []string, sortOrder int) (domain.SpecializedAgent, error) {
	if m.CreateSpecializedAgentFunc == nil {
		panic("called not defined CreateSpecializedAgentFunc")
	}
	return m.CreateSpecializedAgentFunc(ctx, parentSlug, slug, name, skillSlugs, sortOrder)
}

func (m *MockCommands) UpdateSpecializedAgent(ctx context.Context, id domain.SpecializedAgentID, name string, skillSlugs []string, sortOrder int) error {
	if m.UpdateSpecializedAgentFunc == nil {
		panic("called not defined UpdateSpecializedAgentFunc")
	}
	return m.UpdateSpecializedAgentFunc(ctx, id, name, skillSlugs, sortOrder)
}

func (m *MockCommands) DeleteSpecializedAgent(ctx context.Context, id domain.SpecializedAgentID) error {
	if m.DeleteSpecializedAgentFunc == nil {
		panic("called not defined DeleteSpecializedAgentFunc")
	}
	return m.DeleteSpecializedAgentFunc(ctx, id)
}

func (m *MockCommands) GrantUserAccess(ctx context.Context, projectID domain.ProjectID, userID, role string) error {
	if m.GrantUserAccessFunc == nil {
		return nil
	}
	return m.GrantUserAccessFunc(ctx, projectID, userID, role)
}

func (m *MockCommands) RevokeUserAccess(ctx context.Context, projectID domain.ProjectID, userID string) error {
	if m.RevokeUserAccessFunc == nil {
		return nil
	}
	return m.RevokeUserAccessFunc(ctx, projectID, userID)
}

func (m *MockCommands) UpdateUserAccessRole(ctx context.Context, projectID domain.ProjectID, userID, role string) error {
	if m.UpdateUserAccessRoleFunc == nil {
		return nil
	}
	return m.UpdateUserAccessRoleFunc(ctx, projectID, userID, role)
}

func (m *MockCommands) GrantTeamAccess(ctx context.Context, projectID domain.ProjectID, teamID string) error {
	if m.GrantTeamAccessFunc == nil {
		return nil
	}
	return m.GrantTeamAccessFunc(ctx, projectID, teamID)
}

func (m *MockCommands) RevokeTeamAccess(ctx context.Context, projectID domain.ProjectID, teamID string) error {
	if m.RevokeTeamAccessFunc == nil {
		return nil
	}
	return m.RevokeTeamAccessFunc(ctx, projectID, teamID)
}

// MockQueries is a function-based mock implementation of the service.Queries interface.
type MockQueries struct {
	GetProjectFunc                    func(ctx context.Context, projectID domain.ProjectID) (*domain.Project, error)
	ListProjectsFunc                  func(ctx context.Context) ([]domain.Project, error)
	ListProjectsWithSummaryFunc       func(ctx context.Context) ([]domain.ProjectWithSummary, error)
	ListSubProjectsFunc               func(ctx context.Context, parentID domain.ProjectID) ([]domain.Project, error)
	ListSubProjectsWithSummaryFunc    func(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error)
	GetProjectSummaryFunc             func(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectSummary, error)
	GetProjectInfoFunc                func(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectInfo, error)
	GetAgentFunc                      func(ctx context.Context, roleID domain.AgentID) (*domain.Agent, error)
	GetAgentBySlugFunc                func(ctx context.Context, slug string) (*domain.Agent, error)
	ListAgentsFunc                    func(ctx context.Context) ([]domain.Agent, error)
	ListProjectAgentsFunc             func(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error)
	GetProjectAgentBySlugFunc         func(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Agent, error)
	GetTaskFunc                       func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (*domain.Task, error)
	ListTasksFunc                     func(ctx context.Context, projectID domain.ProjectID, filters service.TaskFilters) ([]domain.TaskWithDetails, error)
	GetNextTaskFunc                   func(ctx context.Context, projectID domain.ProjectID, role string, featureID *domain.ProjectID) (*domain.Task, error)
	GetNextTasksFunc                  func(ctx context.Context, projectID domain.ProjectID, role string, count int, featureID *domain.ProjectID) ([]domain.Task, error)
	GetDependencyContextFunc          func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error)
	GetColumnFunc                     func(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (*domain.Column, error)
	GetColumnBySlugFunc               func(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error)
	ListColumnsFunc                   func(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error)
	GetCommentFunc                    func(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (*domain.Comment, error)
	ListCommentsFunc                  func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error)
	CountCommentsFunc                 func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error)
	ListDependenciesFunc              func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error)
	GetDependencyTasksFunc            func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)
	GetDependentTasksFunc             func(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error)
	GetToolUsageForProjectFunc        func(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error)
	GetTimelineFunc                   func(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error)
	GetColdStartStatsFunc             func(ctx context.Context, projectID domain.ProjectID) ([]domain.AgentColdStartStat, error)
	GetSkillFunc                      func(ctx context.Context, skillID domain.SkillID) (*domain.Skill, error)
	GetSkillBySlugFunc                func(ctx context.Context, slug string) (*domain.Skill, error)
	ListSkillsFunc                    func(ctx context.Context) ([]domain.Skill, error)
	ListAgentSkillsFunc               func(ctx context.Context, agentSlug string) ([]domain.Skill, error)
	GetProjectTasksByAgentFunc        func(ctx context.Context, projectID domain.ProjectID, agentSlug string) ([]domain.Task, error)
	GetDockerfileFunc                 func(ctx context.Context, dockerfileID domain.DockerfileID) (*domain.Dockerfile, error)
	GetDockerfileBySlugFunc           func(ctx context.Context, slug string) (*domain.Dockerfile, error)
	GetDockerfileBySlugAndVersionFunc func(ctx context.Context, slug, version string) (*domain.Dockerfile, error)
	ListDockerfilesFunc               func(ctx context.Context) ([]domain.Dockerfile, error)
	GetProjectDockerfileFunc          func(ctx context.Context, projectID domain.ProjectID) (*domain.Dockerfile, error)
	GetModelTokenStatsFunc            func(ctx context.Context, projectID domain.ProjectID) ([]domain.ModelTokenStat, error)
	ListModelPricingFunc              func(ctx context.Context) ([]domain.ModelPricing, error)
	GetFeatureFunc                    func(ctx context.Context, featureID domain.FeatureID) (*domain.Feature, error)
	ListFeaturesFunc                  func(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error)
	GetFeatureStatsFunc               func(ctx context.Context, projectID domain.ProjectID) (*domain.FeatureStats, error)
	ListFeatureTaskSummariesFunc      func(ctx context.Context, featureID domain.FeatureID) ([]domain.FeatureTaskSummary, error)
	ListNotificationsFunc             func(ctx context.Context, projectID *domain.ProjectID, scope *domain.NotificationScope, agentSlug string, unreadOnly bool, limit, offset int) ([]domain.Notification, error)
	GetNotificationUnreadCountFunc    func(ctx context.Context, projectID *domain.ProjectID, scope *domain.NotificationScope, agentSlug string) (int, error)
	ListSpecializedAgentsFunc         func(ctx context.Context, parentSlug string) ([]domain.SpecializedAgent, error)
	GetSpecializedAgentFunc           func(ctx context.Context, slug string) (*domain.SpecializedAgent, error)
	ListSpecializedAgentSkillsFunc    func(ctx context.Context, slug string) ([]domain.Skill, error)
	CountSpecializedByParentFunc      func(ctx context.Context, parentSlug string) (int, error)
	ListProjectUserAccessFunc         func(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectUserAccess, error)
	ListProjectTeamAccessFunc         func(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectTeamAccess, error)
	HasProjectAccessFunc              func(ctx context.Context, projectID domain.ProjectID, userID string, teamIDs []string) (bool, error)
	ListAccessibleProjectIDsFunc      func(ctx context.Context, userID string, teamIDs []string) ([]domain.ProjectID, error)
}

func (m *MockQueries) GetProject(ctx context.Context, projectID domain.ProjectID) (*domain.Project, error) {
	if m.GetProjectFunc == nil {
		panic("called not defined GetProjectFunc")
	}
	return m.GetProjectFunc(ctx, projectID)
}

func (m *MockQueries) ListProjects(ctx context.Context) ([]domain.Project, error) {
	if m.ListProjectsFunc == nil {
		panic("called not defined ListProjectsFunc")
	}
	return m.ListProjectsFunc(ctx)
}

func (m *MockQueries) ListProjectsWithSummary(ctx context.Context) ([]domain.ProjectWithSummary, error) {
	if m.ListProjectsWithSummaryFunc == nil {
		panic("called not defined ListProjectsWithSummaryFunc")
	}
	return m.ListProjectsWithSummaryFunc(ctx)
}

func (m *MockQueries) ListSubProjects(ctx context.Context, parentID domain.ProjectID) ([]domain.Project, error) {
	if m.ListSubProjectsFunc == nil {
		panic("called not defined ListSubProjectsFunc")
	}
	return m.ListSubProjectsFunc(ctx, parentID)
}

func (m *MockQueries) ListSubProjectsWithSummary(ctx context.Context, parentID domain.ProjectID) ([]domain.ProjectWithSummary, error) {
	if m.ListSubProjectsWithSummaryFunc == nil {
		panic("called not defined ListSubProjectsWithSummaryFunc")
	}
	return m.ListSubProjectsWithSummaryFunc(ctx, parentID)
}

func (m *MockQueries) GetProjectSummary(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectSummary, error) {
	if m.GetProjectSummaryFunc == nil {
		panic("called not defined GetProjectSummaryFunc")
	}
	return m.GetProjectSummaryFunc(ctx, projectID)
}

func (m *MockQueries) GetProjectInfo(ctx context.Context, projectID domain.ProjectID) (*domain.ProjectInfo, error) {
	if m.GetProjectInfoFunc == nil {
		panic("called not defined GetProjectInfoFunc")
	}
	return m.GetProjectInfoFunc(ctx, projectID)
}

func (m *MockQueries) GetAgent(ctx context.Context, roleID domain.AgentID) (*domain.Agent, error) {
	if m.GetAgentFunc == nil {
		panic("called not defined GetAgentFunc")
	}
	return m.GetAgentFunc(ctx, roleID)
}

func (m *MockQueries) GetAgentBySlug(ctx context.Context, slug string) (*domain.Agent, error) {
	if m.GetAgentBySlugFunc == nil {
		panic("called not defined GetAgentBySlugFunc")
	}
	return m.GetAgentBySlugFunc(ctx, slug)
}

func (m *MockQueries) ListAgents(ctx context.Context) ([]domain.Agent, error) {
	if m.ListAgentsFunc == nil {
		panic("called not defined ListAgentsFunc")
	}
	return m.ListAgentsFunc(ctx)
}

func (m *MockQueries) ListProjectAgents(ctx context.Context, projectID domain.ProjectID) ([]domain.Agent, error) {
	if m.ListProjectAgentsFunc == nil {
		panic("called not defined ListProjectAgentsFunc")
	}
	return m.ListProjectAgentsFunc(ctx, projectID)
}

func (m *MockQueries) GetProjectAgentBySlug(ctx context.Context, projectID domain.ProjectID, slug string) (*domain.Agent, error) {
	if m.GetProjectAgentBySlugFunc == nil {
		panic("called not defined GetProjectAgentBySlugFunc")
	}
	return m.GetProjectAgentBySlugFunc(ctx, projectID, slug)
}

func (m *MockQueries) GetTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (*domain.Task, error) {
	if m.GetTaskFunc == nil {
		panic("called not defined GetTaskFunc")
	}
	return m.GetTaskFunc(ctx, projectID, taskID)
}

func (m *MockQueries) ListTasks(ctx context.Context, projectID domain.ProjectID, filters service.TaskFilters) ([]domain.TaskWithDetails, error) {
	if m.ListTasksFunc == nil {
		panic("called not defined ListTasksFunc")
	}
	return m.ListTasksFunc(ctx, projectID, filters)
}

func (m *MockQueries) GetNextTask(ctx context.Context, projectID domain.ProjectID, role string, featureID *domain.ProjectID) (*domain.Task, error) {
	if m.GetNextTaskFunc == nil {
		panic("called not defined GetNextTaskFunc")
	}
	return m.GetNextTaskFunc(ctx, projectID, role, featureID)
}

func (m *MockQueries) GetNextTasks(ctx context.Context, projectID domain.ProjectID, role string, count int, featureID *domain.ProjectID) ([]domain.Task, error) {
	if m.GetNextTasksFunc == nil {
		panic("called not defined GetNextTasksFunc")
	}
	return m.GetNextTasksFunc(ctx, projectID, role, count, featureID)
}

func (m *MockQueries) GetDependencyContext(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.DependencyContext, error) {
	if m.GetDependencyContextFunc == nil {
		panic("called not defined GetDependencyContextFunc")
	}
	return m.GetDependencyContextFunc(ctx, projectID, taskID)
}

func (m *MockQueries) GetColumn(ctx context.Context, projectID domain.ProjectID, columnID domain.ColumnID) (*domain.Column, error) {
	if m.GetColumnFunc == nil {
		panic("called not defined GetColumnFunc")
	}
	return m.GetColumnFunc(ctx, projectID, columnID)
}

func (m *MockQueries) GetColumnBySlug(ctx context.Context, projectID domain.ProjectID, slug domain.ColumnSlug) (*domain.Column, error) {
	if m.GetColumnBySlugFunc == nil {
		panic("called not defined GetColumnBySlugFunc")
	}
	return m.GetColumnBySlugFunc(ctx, projectID, slug)
}

func (m *MockQueries) ListColumns(ctx context.Context, projectID domain.ProjectID) ([]domain.Column, error) {
	if m.ListColumnsFunc == nil {
		panic("called not defined ListColumnsFunc")
	}
	return m.ListColumnsFunc(ctx, projectID)
}

func (m *MockQueries) GetComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) (*domain.Comment, error) {
	if m.GetCommentFunc == nil {
		return nil, domain.ErrCommentNotFound
	}
	return m.GetCommentFunc(ctx, projectID, commentID)
}

func (m *MockQueries) ListComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, limit, offset int) ([]domain.Comment, error) {
	if m.ListCommentsFunc == nil {
		panic("called not defined ListCommentsFunc")
	}
	return m.ListCommentsFunc(ctx, projectID, taskID, limit, offset)
}

func (m *MockQueries) CountComments(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) (int, error) {
	if m.CountCommentsFunc == nil {
		panic("called not defined CountCommentsFunc")
	}
	return m.CountCommentsFunc(ctx, projectID, taskID)
}

func (m *MockQueries) ListDependencies(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.TaskDependency, error) {
	if m.ListDependenciesFunc == nil {
		panic("called not defined ListDependenciesFunc")
	}
	return m.ListDependenciesFunc(ctx, projectID, taskID)
}

func (m *MockQueries) GetDependencyTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	if m.GetDependencyTasksFunc == nil {
		panic("called not defined GetDependencyTasksFunc")
	}
	return m.GetDependencyTasksFunc(ctx, projectID, taskID)
}

func (m *MockQueries) GetDependentTasks(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) ([]domain.Task, error) {
	if m.GetDependentTasksFunc == nil {
		panic("called not defined GetDependentTasksFunc")
	}
	return m.GetDependentTasksFunc(ctx, projectID, taskID)
}

func (m *MockQueries) GetToolUsageForProject(ctx context.Context, projectID domain.ProjectID) ([]domain.ToolUsageStat, error) {
	if m.GetToolUsageForProjectFunc == nil {
		panic("called not defined GetToolUsageForProjectFunc")
	}
	return m.GetToolUsageForProjectFunc(ctx, projectID)
}

func (m *MockQueries) GetTimeline(ctx context.Context, projectID domain.ProjectID, days int) ([]domain.TimelineEntry, error) {
	if m.GetTimelineFunc == nil {
		panic("called not defined GetTimelineFunc")
	}
	return m.GetTimelineFunc(ctx, projectID, days)
}

func (m *MockQueries) GetColdStartStats(ctx context.Context, projectID domain.ProjectID) ([]domain.AgentColdStartStat, error) {
	if m.GetColdStartStatsFunc == nil {
		panic("called not defined GetColdStartStatsFunc")
	}
	return m.GetColdStartStatsFunc(ctx, projectID)
}

func (m *MockQueries) GetSkill(ctx context.Context, skillID domain.SkillID) (*domain.Skill, error) {
	if m.GetSkillFunc == nil {
		panic("called not defined GetSkillFunc")
	}
	return m.GetSkillFunc(ctx, skillID)
}

func (m *MockQueries) GetSkillBySlug(ctx context.Context, slug string) (*domain.Skill, error) {
	if m.GetSkillBySlugFunc == nil {
		panic("called not defined GetSkillBySlugFunc")
	}
	return m.GetSkillBySlugFunc(ctx, slug)
}

func (m *MockQueries) ListSkills(ctx context.Context) ([]domain.Skill, error) {
	if m.ListSkillsFunc == nil {
		panic("called not defined ListSkillsFunc")
	}
	return m.ListSkillsFunc(ctx)
}

func (m *MockQueries) ListAgentSkills(ctx context.Context, agentSlug string) ([]domain.Skill, error) {
	if m.ListAgentSkillsFunc == nil {
		panic("called not defined ListAgentSkillsFunc")
	}
	return m.ListAgentSkillsFunc(ctx, agentSlug)
}

func (m *MockQueries) GetProjectTasksByAgent(ctx context.Context, projectID domain.ProjectID, agentSlug string) ([]domain.Task, error) {
	if m.GetProjectTasksByAgentFunc == nil {
		panic("called not defined GetProjectTasksByAgentFunc")
	}
	return m.GetProjectTasksByAgentFunc(ctx, projectID, agentSlug)
}

func (m *MockQueries) GetDockerfile(ctx context.Context, dockerfileID domain.DockerfileID) (*domain.Dockerfile, error) {
	if m.GetDockerfileFunc == nil {
		panic("called not defined GetDockerfileFunc")
	}
	return m.GetDockerfileFunc(ctx, dockerfileID)
}

func (m *MockQueries) GetDockerfileBySlug(ctx context.Context, slug string) (*domain.Dockerfile, error) {
	if m.GetDockerfileBySlugFunc == nil {
		panic("called not defined GetDockerfileBySlugFunc")
	}
	return m.GetDockerfileBySlugFunc(ctx, slug)
}

func (m *MockQueries) GetDockerfileBySlugAndVersion(ctx context.Context, slug, version string) (*domain.Dockerfile, error) {
	if m.GetDockerfileBySlugAndVersionFunc == nil {
		panic("called not defined GetDockerfileBySlugAndVersionFunc")
	}
	return m.GetDockerfileBySlugAndVersionFunc(ctx, slug, version)
}

func (m *MockQueries) ListDockerfiles(ctx context.Context) ([]domain.Dockerfile, error) {
	if m.ListDockerfilesFunc == nil {
		panic("called not defined ListDockerfilesFunc")
	}
	return m.ListDockerfilesFunc(ctx)
}

func (m *MockQueries) GetProjectDockerfile(ctx context.Context, projectID domain.ProjectID) (*domain.Dockerfile, error) {
	if m.GetProjectDockerfileFunc == nil {
		panic("called not defined GetProjectDockerfileFunc")
	}
	return m.GetProjectDockerfileFunc(ctx, projectID)
}

func (m *MockQueries) GetModelTokenStats(ctx context.Context, projectID domain.ProjectID) ([]domain.ModelTokenStat, error) {
	if m.GetModelTokenStatsFunc == nil {
		return nil, nil
	}
	return m.GetModelTokenStatsFunc(ctx, projectID)
}

func (m *MockQueries) ListModelPricing(ctx context.Context) ([]domain.ModelPricing, error) {
	if m.ListModelPricingFunc == nil {
		return nil, nil
	}
	return m.ListModelPricingFunc(ctx)
}

func (m *MockQueries) GetFeature(ctx context.Context, featureID domain.FeatureID) (*domain.Feature, error) {
	if m.GetFeatureFunc == nil {
		return &domain.Feature{ID: featureID}, nil
	}
	return m.GetFeatureFunc(ctx, featureID)
}

func (m *MockQueries) ListFeatures(ctx context.Context, projectID domain.ProjectID, statusFilter []domain.FeatureStatus) ([]domain.FeatureWithTaskSummary, error) {
	if m.ListFeaturesFunc == nil {
		return nil, nil
	}
	return m.ListFeaturesFunc(ctx, projectID, statusFilter)
}

func (m *MockQueries) GetFeatureStats(ctx context.Context, projectID domain.ProjectID) (*domain.FeatureStats, error) {
	if m.GetFeatureStatsFunc == nil {
		return &domain.FeatureStats{}, nil
	}
	return m.GetFeatureStatsFunc(ctx, projectID)
}

func (m *MockQueries) ListFeatureTaskSummaries(ctx context.Context, featureID domain.FeatureID) ([]domain.FeatureTaskSummary, error) {
	if m.ListFeatureTaskSummariesFunc == nil {
		return nil, nil
	}
	return m.ListFeatureTaskSummariesFunc(ctx, featureID)
}

func (m *MockQueries) ListNotifications(ctx context.Context, projectID *domain.ProjectID, scope *domain.NotificationScope, agentSlug string, unreadOnly bool, limit, offset int) ([]domain.Notification, error) {
	if m.ListNotificationsFunc == nil {
		panic("called not defined ListNotificationsFunc")
	}
	return m.ListNotificationsFunc(ctx, projectID, scope, agentSlug, unreadOnly, limit, offset)
}

func (m *MockQueries) GetNotificationUnreadCount(ctx context.Context, projectID *domain.ProjectID, scope *domain.NotificationScope, agentSlug string) (int, error) {
	if m.GetNotificationUnreadCountFunc == nil {
		panic("called not defined GetNotificationUnreadCountFunc")
	}
	return m.GetNotificationUnreadCountFunc(ctx, projectID, scope, agentSlug)
}

func (m *MockQueries) ListSpecializedAgents(ctx context.Context, parentSlug string) ([]domain.SpecializedAgent, error) {
	if m.ListSpecializedAgentsFunc == nil {
		return []domain.SpecializedAgent{}, nil
	}
	return m.ListSpecializedAgentsFunc(ctx, parentSlug)
}

func (m *MockQueries) GetSpecializedAgent(ctx context.Context, slug string) (*domain.SpecializedAgent, error) {
	if m.GetSpecializedAgentFunc == nil {
		panic("called not defined GetSpecializedAgentFunc")
	}
	return m.GetSpecializedAgentFunc(ctx, slug)
}

func (m *MockQueries) ListSpecializedAgentSkills(ctx context.Context, slug string) ([]domain.Skill, error) {
	if m.ListSpecializedAgentSkillsFunc == nil {
		return []domain.Skill{}, nil
	}
	return m.ListSpecializedAgentSkillsFunc(ctx, slug)
}

func (m *MockQueries) CountSpecializedByParent(ctx context.Context, parentSlug string) (int, error) {
	if m.CountSpecializedByParentFunc == nil {
		return 0, nil
	}
	return m.CountSpecializedByParentFunc(ctx, parentSlug)
}

func (m *MockQueries) ListProjectUserAccess(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectUserAccess, error) {
	if m.ListProjectUserAccessFunc == nil {
		return []domain.ProjectUserAccess{}, nil
	}
	return m.ListProjectUserAccessFunc(ctx, projectID)
}

func (m *MockQueries) ListProjectTeamAccess(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectTeamAccess, error) {
	if m.ListProjectTeamAccessFunc == nil {
		return []domain.ProjectTeamAccess{}, nil
	}
	return m.ListProjectTeamAccessFunc(ctx, projectID)
}

func (m *MockQueries) HasProjectAccess(ctx context.Context, projectID domain.ProjectID, userID string, teamIDs []string) (bool, error) {
	if m.HasProjectAccessFunc == nil {
		return true, nil
	}
	return m.HasProjectAccessFunc(ctx, projectID, userID, teamIDs)
}

func (m *MockQueries) ListAccessibleProjectIDs(ctx context.Context, userID string, teamIDs []string) ([]domain.ProjectID, error) {
	if m.ListAccessibleProjectIDsFunc == nil {
		return []domain.ProjectID{}, nil
	}
	return m.ListAccessibleProjectIDsFunc(ctx, userID, teamIDs)
}
