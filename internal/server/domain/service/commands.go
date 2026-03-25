package service

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/server/domain"
)

// BulkTaskInput holds per-task input for a bulk create operation.
type BulkTaskInput struct {
	Title           string
	Summary         string
	Description     string
	Priority        domain.Priority
	CreatedByRole   string
	CreatedByAgent  string
	AssignedRole    string
	ContextFiles    []string
	Tags            []string
	EstimatedEffort string
	StartInBacklog  bool
	DependsOn       []domain.TaskID
	FeatureID       *domain.FeatureID
}

type ProjectCommands interface {
	CreateProject(ctx context.Context, name, description, gitURL, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error)
	UpdateProject(ctx context.Context, projectID domain.ProjectID, name, description string, gitURL, defaultRole *string) error
	DeleteProject(ctx context.Context, projectID domain.ProjectID) error
}

type TaskCommands interface {
	CreateTask(ctx context.Context, projectID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string, startInBacklog bool, featureID *domain.FeatureID) (domain.Task, error)
	BulkCreateTasks(ctx context.Context, projectID domain.ProjectID, inputs []BulkTaskInput) ([]domain.Task, error)
	UpdateTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.FeatureID, clearFeature bool) error
	UpdateTaskFiles(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, filesModified, contextFiles *[]string) error
	DeleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	MoveTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, targetColumnSlug domain.ColumnSlug) error
	ReorderTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, newPosition int) error
	StartTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	CompleteTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, completionSummary string, filesModified []string, completedByAgent string, tokenUsage *domain.TokenUsage) error
	BlockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, blockedReason, blockedByAgent string) error
	UnblockTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	RequestWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, wontDoReason, wontDoRequestedBy string) error
	ApproveWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	RejectWontDo(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, reason string) error
	UpdateTaskSessionID(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error
}

type AgentCommands interface {
	CreateAgent(ctx context.Context, slug, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) (domain.Agent, error)
	UpdateAgent(ctx context.Context, agentID domain.AgentID, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) error
	DeleteAgent(ctx context.Context, agentID domain.AgentID) error
	CreateProjectAgent(ctx context.Context, projectID domain.ProjectID, slug, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) (domain.Agent, error)
	UpdateProjectAgent(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) error
	DeleteProjectAgent(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error
	CloneAgent(ctx context.Context, sourceSlug, newSlug, newName string) (domain.Agent, error)
	AssignAgentToProject(ctx context.Context, projectID domain.ProjectID, agentSlug string) error
	RemoveAgentFromProject(ctx context.Context, projectID domain.ProjectID, agentSlug string, reassignTo *string, clearAssignment bool) error
	BulkReassignTasks(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error)
}

type CommentCommands interface {
	CreateComment(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, authorRole, authorName string, authorType domain.AuthorType, content string) (domain.Comment, error)
	UpdateComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID, content string) error
	DeleteComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) error
}

type DependencyCommands interface {
	AddDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error
	RemoveDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error
}

type SkillCommands interface {
	CreateSkill(ctx context.Context, slug, name, description, content, icon, color string, sortOrder int) (domain.Skill, error)
	UpdateSkill(ctx context.Context, skillID domain.SkillID, name, description, content, icon, color string, sortOrder int) error
	DeleteSkill(ctx context.Context, skillID domain.SkillID) error
	AddSkillToAgent(ctx context.Context, agentSlug, skillSlug string) error
	RemoveSkillFromAgent(ctx context.Context, agentSlug, skillSlug string) error
}

type FeatureCommands interface {
	CreateFeature(ctx context.Context, projectID domain.ProjectID, name, description, createdByRole, createdByAgent string) (domain.Feature, error)
	UpdateFeature(ctx context.Context, featureID domain.FeatureID, name, description string) error
	UpdateFeatureStatus(ctx context.Context, featureID domain.FeatureID, status domain.FeatureStatus) error
	DeleteFeature(ctx context.Context, featureID domain.FeatureID) error
}

type DockerfileCommands interface {
	CreateDockerfile(ctx context.Context, slug, name, description, version, content string, isLatest bool, sortOrder int) (domain.Dockerfile, error)
	UpdateDockerfile(ctx context.Context, dockerfileID domain.DockerfileID, name, description, content *string, isLatest *bool, sortOrder *int) error
	DeleteDockerfile(ctx context.Context, dockerfileID domain.DockerfileID) error
	SetProjectDockerfile(ctx context.Context, projectID domain.ProjectID, dockerfileID domain.DockerfileID) error
	ClearProjectDockerfile(ctx context.Context, projectID domain.ProjectID) error
}

type NotificationCommands interface {
	CreateNotification(ctx context.Context, projectID *domain.ProjectID, scope domain.NotificationScope, agentSlug string, severity domain.NotificationSeverity, title, text, linkURL, linkText, linkStyle string) (domain.Notification, error)
	MarkNotificationRead(ctx context.Context, notificationID domain.NotificationID) error
	MarkAllNotificationsRead(ctx context.Context, projectID *domain.ProjectID) error
	DeleteNotification(ctx context.Context, notificationID domain.NotificationID) error
}

type SpecializedAgentCommands interface {
	CreateSpecializedAgent(ctx context.Context, parentSlug, slug, name string, skillSlugs []string, sortOrder int) (domain.SpecializedAgent, error)
	UpdateSpecializedAgent(ctx context.Context, id domain.SpecializedAgentID, name string, skillSlugs []string, sortOrder int) error
	DeleteSpecializedAgent(ctx context.Context, id domain.SpecializedAgentID) error
}

// Commands defines write operations for the Kanban system.
type Commands interface {
	ProjectCommands
	TaskCommands
	AgentCommands
	CommentCommands
	DependencyCommands
	SkillCommands
	FeatureCommands
	DockerfileCommands
	NotificationCommands
	SpecializedAgentCommands
	MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error
	MoveTaskToProject(ctx context.Context, sourceProjectID domain.ProjectID, taskID domain.TaskID, targetProjectID domain.ProjectID) error
	IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error
}
