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
}

// Commands defines write operations for the Kanban system
type Commands interface {
	// Project commands
	CreateProject(ctx context.Context, name, description, workDir, createdByRole, createdByAgent string, parentID *domain.ProjectID) (domain.Project, error)
	UpdateProject(ctx context.Context, projectID domain.ProjectID, name, description string, defaultRole *string) error
	DeleteProject(ctx context.Context, projectID domain.ProjectID) error

	// Role commands (global)
	CreateRole(ctx context.Context, slug, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) (domain.Role, error)
	UpdateRole(ctx context.Context, roleID domain.RoleID, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) error
	DeleteRole(ctx context.Context, roleID domain.RoleID) error

	// Role commands (per-project)
	CreateProjectRole(ctx context.Context, projectID domain.ProjectID, slug, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) (domain.Role, error)
	UpdateProjectRole(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID, name, icon, color, description, promptHint, promptTemplate string, techStack []string, sortOrder int) error
	DeleteProjectRole(ctx context.Context, projectID domain.ProjectID, roleID domain.RoleID) error

	// Task commands
	CreateTask(ctx context.Context, projectID domain.ProjectID, title, summary, description string, priority domain.Priority, createdByRole, createdByAgent, assignedRole string, contextFiles, tags []string, estimatedEffort string, startInBacklog bool) (domain.Task, error)
	BulkCreateTasks(ctx context.Context, projectID domain.ProjectID, inputs []BulkTaskInput) ([]domain.Task, error)
	UpdateTask(ctx context.Context, projectID domain.ProjectID, taskID domain.TaskID, title, description, assignedRole, estimatedEffort, resolution *string, priority *domain.Priority, contextFiles, tags *[]string, tokenUsage *domain.TokenUsage, humanEstimateSeconds *int) error
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

	// CloneRole creates a copy of an existing role (global) with a new slug and name.
	// All fields (description, tech_stack, prompt_hint, prompt_template, content) are copied.
	// Returns ErrRoleNotFound if sourceSlug does not exist.
	// Returns ErrRoleAlreadyExists if newSlug is already taken.
	CloneRole(ctx context.Context, sourceSlug, newSlug, newName string) (domain.Role, error)

	// AssignAgentToProject assigns a global agent (by slug) to a project.
	// Returns ErrRoleNotFound if slug does not match any global role.
	// Returns ErrAgentAlreadyInProject if already assigned.
	AssignAgentToProject(ctx context.Context, projectID domain.ProjectID, agentSlug string) error

	// RemoveAgentFromProject removes an agent from a project.
	// Returns ErrAgentNotInProject if not assigned.
	// Returns ErrAgentHasTasks if there are tasks still assigned to this agent in the
	// project AND reassignTo is nil AND clearAssignment is false.
	// If reassignTo is non-nil, calls BulkReassignTasks first.
	// If clearAssignment is true, sets assigned_role to "" for all affected tasks first.
	RemoveAgentFromProject(ctx context.Context, projectID domain.ProjectID, agentSlug string, reassignTo *string, clearAssignment bool) error

	// BulkReassignTasks sets assigned_role from oldSlug to newSlug for all tasks in a project.
	// If newSlug is empty string, clears assigned_role.
	// Returns the count of updated tasks.
	BulkReassignTasks(ctx context.Context, projectID domain.ProjectID, oldSlug, newSlug string) (int, error)

	// Skill commands

	// CreateSkill creates a new global skill.
	CreateSkill(ctx context.Context, slug, name, description, content, icon, color string, sortOrder int) (domain.Skill, error)

	// UpdateSkill updates a skill's mutable fields.
	UpdateSkill(ctx context.Context, skillID domain.SkillID, name, description, content, icon, color string, sortOrder int) error

	// DeleteSkill deletes a skill.
	// Returns ErrSkillInUse if the skill is assigned to any agent.
	DeleteSkill(ctx context.Context, skillID domain.SkillID) error

	// AddSkillToAgent assigns a skill to an agent (global role) by slug.
	// Returns ErrRoleNotFound or ErrSkillNotFound if either does not exist.
	// Returns ErrSkillAlreadyExists if already assigned.
	AddSkillToAgent(ctx context.Context, agentSlug, skillSlug string) error

	// RemoveSkillFromAgent removes a skill assignment from an agent.
	// Returns ErrSkillNotFound if the assignment does not exist.
	RemoveSkillFromAgent(ctx context.Context, agentSlug, skillSlug string) error
}
