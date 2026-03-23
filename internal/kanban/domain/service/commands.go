package service

import (
	"context"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
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
	FeatureID       *domain.ProjectID
}

// Commands defines write operations for the Kanban system
type Commands interface {
	// Project commands
	CreateProject(ctx context.Context, name, description, gitURL, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error)
	UpdateProject(ctx context.Context, projectID domain.ProjectID, name, description string, gitURL, defaultRole *string) error
	DeleteProject(ctx context.Context, projectID domain.ProjectID) error

	// Agent commands (global)
	CreateAgent(ctx context.Context, slug, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) (domain.Agent, error)
	UpdateAgent(ctx context.Context, agentID domain.AgentID, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) error
	DeleteAgent(ctx context.Context, agentID domain.AgentID) error

	// Agent commands (per-project)
	CreateProjectAgent(ctx context.Context, projectID domain.ProjectID, slug, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) (domain.Agent, error)
	UpdateProjectAgent(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) error
	DeleteProjectAgent(ctx context.Context, projectID domain.ProjectID, agentID domain.AgentID) error

	// Task commands
	CreateTask(ctx context.Context, projectID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string, startInBacklog bool, featureID *domain.ProjectID) (domain.Task, error)
	BulkCreateTasks(ctx context.Context, projectID domain.ProjectID, inputs []BulkTaskInput) ([]domain.Task, error)
	UpdateTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int, featureID *domain.ProjectID, clearFeature bool) error
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

	// Column commands
	UpdateColumnWIPLimit(ctx context.Context, projectID domain.ProjectID, columnSlug domain.ColumnSlug, wipLimit int) error

	// Comment commands
	CreateComment(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, authorRole, authorName string, authorType domain.AuthorType, content string) (domain.Comment, error)
	UpdateComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID, content string) error
	DeleteComment(ctx context.Context, projectID domain.ProjectID, commentID domain.CommentID) error

	// Dependency commands
	AddDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error
	RemoveDependency(ctx context.Context, projectID domain.ProjectID, taskID, dependsOnTaskID domain.TaskID) error

	// Seen commands
	MarkTaskSeen(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID) error

	// Cross-project task commands
	MoveTaskToProject(ctx context.Context, sourceProjectID domain.ProjectID, taskID domain.TaskID, targetProjectID domain.ProjectID) error

	// Tool usage commands
	IncrementToolUsage(ctx context.Context, projectID domain.ProjectID, toolName string) error

	// Session commands
	UpdateTaskSessionID(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, sessionID string) error

	// Agent management commands

	// CloneAgent creates a copy of an existing agent (global) with a new slug and name.
	CloneAgent(ctx context.Context, sourceSlug, newSlug, newName string) (domain.Agent, error)

	// AssignAgentToProject assigns a global agent (by slug) to a project.
	AssignAgentToProject(ctx context.Context, projectID domain.ProjectID, agentSlug string) error

	// RemoveAgentFromProject removes an agent from a project.
	RemoveAgentFromProject(ctx context.Context, projectID domain.ProjectID, agentSlug string, reassignTo *string, clearAssignment bool) error

	// BulkReassignTasks sets assigned_role from oldSlug to newSlug for all tasks in a project.
	BulkReassignTasks(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error)

	// Skill commands

	CreateSkill(ctx context.Context, slug, name, description, content, icon, color string, sortOrder int) (domain.Skill, error)
	UpdateSkill(ctx context.Context, skillID domain.SkillID, name, description, content, icon, color string, sortOrder int) error
	DeleteSkill(ctx context.Context, skillID domain.SkillID) error
	AddSkillToAgent(ctx context.Context, agentSlug, skillSlug string) error
	RemoveSkillFromAgent(ctx context.Context, agentSlug, skillSlug string) error

	// Dockerfile commands

	CreateDockerfile(ctx context.Context, slug, name, description, version, content string, isLatest bool, sortOrder int) (domain.Dockerfile, error)
	UpdateDockerfile(ctx context.Context, dockerfileID domain.DockerfileID, name, description, content *string, isLatest *bool, sortOrder *int) error
	DeleteDockerfile(ctx context.Context, dockerfileID domain.DockerfileID) error
	SetProjectDockerfile(ctx context.Context, projectID domain.ProjectID, dockerfileID domain.DockerfileID) error
	ClearProjectDockerfile(ctx context.Context, projectID domain.ProjectID) error
}
