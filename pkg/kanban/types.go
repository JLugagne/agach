package kanban

import (
	"time"

	"github.com/JLugagne/agach-mcp/pkg/apierror"
)

// CreateProjectRequest represents a request to create a project
type CreateProjectRequest struct {
	Name           string  `json:"name" validate:"required,min=1,max=200"`
	Description    string  `json:"description" validate:"max=5000"`
	GitURL         string  `json:"git_url" validate:"omitempty,max=500"`
	ParentID       *string `json:"parent_id" validate:"omitempty,entity_id"`
	CreatedByRole  string  `json:"created_by_role" validate:"max=100"`
	CreatedByAgent string  `json:"created_by_agent" validate:"max=100"`
}

// UpdateProjectRequest represents a request to update a project
type UpdateProjectRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=200"`
	Description *string `json:"description" validate:"omitempty,max=5000"`
	GitURL      *string `json:"git_url" validate:"omitempty,max=500"`
	DefaultRole *string `json:"default_role" validate:"omitempty,max=100"`
}

// ProjectResponse represents a project in API responses
type ProjectResponse struct {
	ID             string    `json:"id"`
	ParentID       *string   `json:"parent_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	GitURL         string    `json:"git_url"`
	CreatedByRole  string    `json:"created_by_role"`
	CreatedByAgent string    `json:"created_by_agent"`
	DefaultRole    string    `json:"default_role"`
	DockerfileID   *string   `json:"dockerfile_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ProjectSummaryResponse represents project task summary
type ProjectSummaryResponse struct {
	BacklogCount    int `json:"backlog_count"`
	TodoCount       int `json:"todo_count"`
	InProgressCount int `json:"in_progress_count"`
	DoneCount       int `json:"done_count"`
	BlockedCount    int `json:"blocked_count"`
}

// CreateAgentRequest represents a request to create an agent
type CreateAgentRequest struct {
	Slug           string   `json:"slug" validate:"required,min=1,max=50,slug"`
	Name           string   `json:"name" validate:"required,min=1,max=100"`
	Icon           string   `json:"icon" validate:"max=10"`
	Color          string   `json:"color" validate:"omitempty,hexcolor"`
	Description    string   `json:"description" validate:"max=1000"`
	TechStack      []string `json:"tech_stack" validate:"max=100,dive,max=50"`
	PromptHint     string   `json:"prompt_hint" validate:"max=5000"`
	PromptTemplate string   `json:"prompt_template" validate:"omitempty,max=50000"`
	SkillSlugs     []string `json:"skill_slugs" validate:"max=100,dive,max=50"`
	SortOrder      int      `json:"sort_order"`
}

// UpdateAgentRequest represents a request to update an agent
type UpdateAgentRequest struct {
	Name           *string   `json:"name" validate:"omitempty,min=1,max=100"`
	Icon           *string   `json:"icon" validate:"omitempty,max=10"`
	Color          *string   `json:"color" validate:"omitempty,hexcolor"`
	Description    *string   `json:"description" validate:"omitempty,max=1000"`
	TechStack      *[]string `json:"tech_stack" validate:"omitempty,max=100,dive,max=50"`
	PromptHint     *string   `json:"prompt_hint" validate:"omitempty,max=5000"`
	PromptTemplate *string   `json:"prompt_template" validate:"omitempty,max=50000"`
	SkillSlugs     *[]string `json:"skill_slugs" validate:"omitempty,max=100,dive,max=50"`
	SortOrder      *int      `json:"sort_order"`
}

// AgentResponse represents an agent in API responses
type AgentResponse struct {
	ID             string    `json:"id"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	Icon           string    `json:"icon"`
	Color          string    `json:"color"`
	Description    string    `json:"description"`
	TechStack      []string  `json:"tech_stack"`
	PromptHint     string    `json:"prompt_hint"`
	PromptTemplate string    `json:"prompt_template"`
	Content        string    `json:"content"`
	SkillCount     int       `json:"skill_count"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
}

// Backward compatibility aliases
type CreateRoleRequest = CreateAgentRequest
type UpdateRoleRequest = UpdateAgentRequest
type RoleResponse = AgentResponse

// CreateSkillRequest represents a request to create a skill
type CreateSkillRequest struct {
	Slug        string `json:"slug"        validate:"required,min=1,max=50,slug"`
	Name        string `json:"name"        validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"max=2000"`
	Content     string `json:"content"     validate:"max=100000"`
	Icon        string `json:"icon"        validate:"max=10"`
	Color       string `json:"color"       validate:"omitempty,hexcolor"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateSkillRequest represents a request to update a skill
type UpdateSkillRequest struct {
	Name        *string `json:"name"        validate:"omitempty,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	Content     *string `json:"content"     validate:"omitempty,max=100000"`
	Icon        *string `json:"icon"        validate:"omitempty,max=10"`
	Color       *string `json:"color"       validate:"omitempty,hexcolor"`
	SortOrder   *int    `json:"sort_order"`
}

// SkillResponse represents a skill in API responses
type SkillResponse struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Icon        string    `json:"icon"`
	Color       string    `json:"color"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AddSkillToAgentRequest represents a request to assign a skill to an agent
type AddSkillToAgentRequest struct {
	SkillSlug string `json:"skill_slug" validate:"required,min=1,max=50"`
}

// CreateTaskRequest represents a request to create a task
type CreateTaskRequest struct {
	Title           string   `json:"title" validate:"required,min=1,max=500"`
	Summary         string   `json:"summary" validate:"required,min=1,max=1000"`
	Description     string   `json:"description" validate:"max=10000"`
	Priority        string   `json:"priority" validate:"omitempty,oneof=critical high medium low"`
	CreatedByRole   string   `json:"created_by_role" validate:"max=100"`
	CreatedByAgent  string   `json:"created_by_agent" validate:"max=100"`
	AssignedRole    string   `json:"assigned_role" validate:"max=100"`
	ContextFiles    []string `json:"context_files" validate:"max=100,dive,max=500"`
	Tags            []string `json:"tags" validate:"max=100,dive,max=50"`
	EstimatedEffort string   `json:"estimated_effort" validate:"omitempty,oneof=XS S M L XL"`
	DependsOn       []string `json:"depends_on" validate:"max=100,dive,entity_id"`
	StartInBacklog  bool     `json:"start_in_backlog"`
	FeatureID       *string  `json:"feature_id" validate:"omitempty,entity_id"`
}

// UpdateTaskRequest represents a request to update a task
type UpdateTaskRequest struct {
	Title                *string   `json:"title" validate:"omitempty,min=1,max=500"`
	Description          *string   `json:"description" validate:"omitempty,max=10000"`
	Priority             *string   `json:"priority" validate:"omitempty,oneof=critical high medium low"`
	AssignedRole         *string   `json:"assigned_role" validate:"omitempty,max=100"`
	ContextFiles         *[]string `json:"context_files" validate:"omitempty,max=100,dive,max=500"`
	Tags                 *[]string `json:"tags" validate:"omitempty,max=100,dive,max=50"`
	EstimatedEffort      *string   `json:"estimated_effort" validate:"omitempty,oneof=XS S M L XL"`
	Resolution           *string   `json:"resolution" validate:"omitempty,max=10000"`
	InputTokens          *int      `json:"input_tokens,omitempty" validate:"omitempty,min=0"`
	OutputTokens         *int      `json:"output_tokens,omitempty" validate:"omitempty,min=0"`
	CacheReadTokens      *int      `json:"cache_read_tokens,omitempty" validate:"omitempty,min=0"`
	CacheWriteTokens     *int      `json:"cache_write_tokens,omitempty" validate:"omitempty,min=0"`
	Model                     *string   `json:"model,omitempty" validate:"omitempty,max=200"`
	ColdStartInputTokens      *int      `json:"cold_start_input_tokens,omitempty" validate:"omitempty,min=0"`
	ColdStartOutputTokens     *int      `json:"cold_start_output_tokens,omitempty" validate:"omitempty,min=0"`
	ColdStartCacheReadTokens  *int      `json:"cold_start_cache_read_tokens,omitempty" validate:"omitempty,min=0"`
	ColdStartCacheWriteTokens *int      `json:"cold_start_cache_write_tokens,omitempty" validate:"omitempty,min=0"`
	HumanEstimateSeconds      *int      `json:"human_estimate_seconds,omitempty" validate:"omitempty,min=0"`
	FeatureID            *string   `json:"feature_id" validate:"omitempty,entity_id"`
}

// MoveTaskRequest represents a request to move a task
type MoveTaskRequest struct {
	TargetColumn string `json:"target_column" validate:"required,oneof=backlog todo in_progress done blocked"`
	Reason       string `json:"reason" validate:"max=1000"`
}

// CompleteTaskRequest represents a request to complete a task
type CompleteTaskRequest struct {
	CompletionSummary    string   `json:"completion_summary" validate:"required,min=100,max=10000"`
	FilesModified        []string `json:"files_modified" validate:"dive,max=500"`
	CompletedByAgent     string   `json:"completed_by_agent" validate:"required,max=100"`
	InputTokens          int      `json:"input_tokens,omitempty"`
	OutputTokens         int      `json:"output_tokens,omitempty"`
	CacheReadTokens      int      `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens     int      `json:"cache_write_tokens,omitempty"`
	Model                string   `json:"model,omitempty"`
	HumanEstimateSeconds int      `json:"human_estimate_seconds,omitempty"`
}

// BlockTaskRequest represents a request to block a task
type BlockTaskRequest struct {
	BlockedReason  string `json:"blocked_reason" validate:"required,min=50,max=5000"`
	BlockedByAgent string `json:"blocked_by_agent" validate:"required,max=100"`
}

// RequestWontDoRequest represents a request to mark a task as won't do
type RequestWontDoRequest struct {
	WontDoReason      string `json:"wont_do_reason" validate:"required,min=50,max=5000"`
	WontDoRequestedBy string `json:"wont_do_requested_by" validate:"required,max=100"`
}

// RejectWontDoRequest represents a request to reject won't do
type RejectWontDoRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=5000"`
}

// MoveTaskToProjectRequest represents a request to move a task to another project
type MoveTaskToProjectRequest struct {
	TargetProjectID string `json:"target_project_id" validate:"required,entity_id"`
}

// ReorderTaskRequest represents a request to reorder a task within its column
type ReorderTaskRequest struct {
	Position int `json:"position" validate:"min=0"`
}

// TaskResponse represents a task in API responses
type TaskResponse struct {
	ID                string     `json:"id"`
	ColumnID          string     `json:"column_id"`
	FeatureID         *string    `json:"feature_id"`
	Title             string     `json:"title"`
	Summary           string     `json:"summary"`
	Description       string     `json:"description"`
	Priority          string     `json:"priority"`
	PriorityScore     int        `json:"priority_score"`
	Position          int        `json:"position"`
	CreatedByRole     string     `json:"created_by_role"`
	CreatedByAgent    string     `json:"created_by_agent"`
	AssignedRole      string     `json:"assigned_role"`
	IsBlocked         bool       `json:"is_blocked"`
	BlockedReason     string     `json:"blocked_reason"`
	BlockedAt         *time.Time `json:"blocked_at"`
	BlockedByAgent    string     `json:"blocked_by_agent"`
	WontDoRequested   bool       `json:"wont_do_requested"`
	WontDoReason      string     `json:"wont_do_reason"`
	WontDoRequestedBy string     `json:"wont_do_requested_by"`
	WontDoRequestedAt *time.Time `json:"wont_do_requested_at"`
	CompletionSummary string     `json:"completion_summary"`
	CompletedByAgent  string     `json:"completed_by_agent"`
	CompletedAt       *time.Time `json:"completed_at"`
	FilesModified     []string   `json:"files_modified"`
	Resolution        string     `json:"resolution"`
	ContextFiles      []string   `json:"context_files"`
	Tags              []string   `json:"tags"`
	EstimatedEffort   string     `json:"estimated_effort"`
	InputTokens       int        `json:"input_tokens"`
	OutputTokens      int        `json:"output_tokens"`
	CacheReadTokens   int        `json:"cache_read_tokens"`
	CacheWriteTokens  int        `json:"cache_write_tokens"`
	Model             string     `json:"model"`
	SessionID         string     `json:"session_id"`
	SeenAt               *time.Time `json:"seen_at"`
	StartedAt            *time.Time `json:"started_at"`
	DurationSeconds      int        `json:"duration_seconds"`
	HumanEstimateSeconds int        `json:"human_estimate_seconds"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// TaskWithDetailsResponse represents a task with additional metadata
type TaskWithDetailsResponse struct {
	TaskResponse
	HasUnresolvedDeps bool   `json:"has_unresolved_deps"`
	CommentCount      int    `json:"comment_count"`
	ProjectID         string `json:"project_id,omitempty"`
	ProjectName       string `json:"project_name,omitempty"`
}

// CreateCommentRequest represents a request to create a comment
type CreateCommentRequest struct {
	AuthorRole   string `json:"author_role" validate:"required,max=100"`
	AuthorName   string `json:"author_name" validate:"max=100"`
	Content      string `json:"content" validate:"required,min=1,max=10000"`
	MarkAsWontDo bool   `json:"mark_as_wont_do"`
}

// UpdateCommentRequest represents a request to update a comment
type UpdateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=10000"`
}

// CommentResponse represents a comment in API responses
type CommentResponse struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"task_id"`
	AuthorRole string     `json:"author_role"`
	AuthorName string     `json:"author_name"`
	AuthorType string     `json:"author_type"`
	Content    string     `json:"content"`
	EditedAt   *time.Time `json:"edited_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// AddDependencyRequest represents a request to add a dependency
type AddDependencyRequest struct {
	DependsOnTaskID string `json:"depends_on_task_id" validate:"required,entity_id"`
}

// ColumnResponse represents a column in API responses
type ColumnResponse struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

// BoardResponse represents the full kanban board
type BoardResponse struct {
	Columns []ColumnWithTasksResponse `json:"columns"`
}

// ColumnWithTasksResponse represents a column with its tasks
type ColumnWithTasksResponse struct {
	ColumnResponse
	Tasks []TaskWithDetailsResponse `json:"tasks"`
}

// DependencyContextResponse represents dependency context
type DependencyContextResponse struct {
	TaskID            string   `json:"task_id"`
	Title             string   `json:"title"`
	CompletionSummary string   `json:"completion_summary"`
	FilesModified     []string `json:"files_modified"`
}

// ToolUsageStatResponse represents a tool usage statistic in API responses
type ToolUsageStatResponse struct {
	ToolName       string     `json:"tool_name"`
	ExecutionCount int        `json:"execution_count"`
	LastExecutedAt *time.Time `json:"last_executed_at"`
}

// TimelineEntryResponse represents daily task counts in API responses
type TimelineEntryResponse struct {
	Date           string `json:"date"`
	TasksCreated   int    `json:"tasks_created"`
	TasksCompleted int    `json:"tasks_completed"`
}

// ColdStartStatResponse represents cold-start token statistics per role in API responses
type ColdStartStatResponse struct {
	AssignedRole       string  `json:"assigned_role"`
	Count              int     `json:"count"`
	MinInputTokens     int     `json:"min_input_tokens"`
	MaxInputTokens     int     `json:"max_input_tokens"`
	AvgInputTokens     float64 `json:"avg_input_tokens"`
	MinOutputTokens    int     `json:"min_output_tokens"`
	MaxOutputTokens    int     `json:"max_output_tokens"`
	AvgOutputTokens    float64 `json:"avg_output_tokens"`
	MinCacheReadTokens int     `json:"min_cache_read_tokens"`
	MaxCacheReadTokens int     `json:"max_cache_read_tokens"`
	AvgCacheReadTokens float64 `json:"avg_cache_read_tokens"`
}

// TasksByAgentResponse is returned by the ListTasksByAgent endpoint
type TasksByAgentResponse struct {
	AgentSlug string         `json:"agent_slug"`
	TaskCount int            `json:"task_count"`
	Tasks     []TaskResponse `json:"tasks"`
}

// CloneAgentRequest represents a request to clone an agent
type CloneAgentRequest struct {
	NewSlug string `json:"new_slug" validate:"required,min=1,max=50,slug"`
	NewName string `json:"new_name" validate:"omitempty,max=100"`
}

// CloneRoleRequest is an alias for backward compatibility
type CloneRoleRequest = CloneAgentRequest

// AssignAgentToProjectRequest represents a request to assign an agent to a project
type AssignAgentToProjectRequest struct {
	AgentSlug string `json:"agent_slug" validate:"required,min=1,max=50"`
}

// RemoveAgentFromProjectRequest represents the body for removing an agent from a project.
// ReassignTo is the target agent slug to reassign tasks to, or empty to clear assignments.
// If both ReassignTo is empty and ClearAssignment is false, returns ErrAgentHasTasks when tasks exist.
type RemoveAgentFromProjectRequest struct {
	ReassignTo      *string `json:"reassign_to"       validate:"omitempty,max=50"`
	ClearAssignment bool    `json:"clear_assignment"`
}

// BulkReassignTasksRequest represents a request to reassign tasks between agents
type BulkReassignTasksRequest struct {
	OldSlug string `json:"old_slug" validate:"required,min=1,max=50"`
	NewSlug string `json:"new_slug" validate:"omitempty,max=50"`
}

// BulkReassignTasksResponse contains the count of updated tasks
type BulkReassignTasksResponse struct {
	UpdatedCount int `json:"updated_count"`
}

// CreateDockerfileRequest represents a request to create a dockerfile
type CreateDockerfileRequest struct {
	Slug        string `json:"slug" validate:"required,min=1,max=50,slug"`
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"max=1000"`
	Version     string `json:"version" validate:"required,min=1,max=50"`
	Content     string `json:"content" validate:"max=100000"`
	IsLatest    bool   `json:"is_latest"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateDockerfileRequest represents a request to update a dockerfile
type UpdateDockerfileRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,max=1000"`
	Content     *string `json:"content" validate:"omitempty,max=100000"`
	IsLatest    *bool   `json:"is_latest"`
	SortOrder   *int    `json:"sort_order"`
}

// DockerfileResponse represents a dockerfile in API responses
type DockerfileResponse struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Content     string    `json:"content"`
	IsLatest    bool      `json:"is_latest"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SetProjectDockerfileRequest represents a request to assign a dockerfile to a project
type SetProjectDockerfileRequest struct {
	DockerfileID string `json:"dockerfile_id" validate:"required,min=1,max=50"`
}

// Features

type CreateFeatureRequest struct {
	Name           string `json:"name" validate:"required,min=1,max=200"`
	Description    string `json:"description" validate:"max=5000"`
	CreatedByRole  string `json:"created_by_role" validate:"max=100"`
	CreatedByAgent string `json:"created_by_agent" validate:"max=100"`
}

type UpdateFeatureRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=200"`
	Description *string `json:"description" validate:"omitempty,max=5000"`
}

type UpdateFeatureStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=draft ready in_progress done blocked"`
}

type FeatureResponse struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	CreatedByRole  string `json:"created_by_role"`
	CreatedByAgent string `json:"created_by_agent"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type FeatureWithSummaryResponse struct {
	FeatureResponse
	TaskSummary ProjectSummaryResponse `json:"task_summary"`
}

// Validation errors
var (
	ErrInvalidProjectRequest = &apierror.Error{
		Code:    "INVALID_PROJECT_REQUEST",
		Message: "invalid project request data",
	}
	ErrInvalidAgentRequest = &apierror.Error{
		Code:    "INVALID_AGENT_REQUEST",
		Message: "invalid agent request data",
	}
	// Backward compatibility
	ErrInvalidRoleRequest = ErrInvalidAgentRequest
	ErrInvalidTaskRequest = &apierror.Error{
		Code:    "INVALID_TASK_REQUEST",
		Message: "invalid task request data",
	}
	ErrInvalidCommentRequest = &apierror.Error{
		Code:    "INVALID_COMMENT_REQUEST",
		Message: "invalid comment request data",
	}
	ErrInvalidDependencyRequest = &apierror.Error{
		Code:    "INVALID_DEPENDENCY_REQUEST",
		Message: "invalid dependency request data",
	}
	ErrInvalidImageRequest = &apierror.Error{
		Code:    "INVALID_IMAGE_REQUEST",
		Message: "invalid image request",
	}
	ErrInvalidSkillRequest = &apierror.Error{
		Code:    "INVALID_SKILL_REQUEST",
		Message: "invalid skill request",
	}
	ErrInvalidAgentAssignmentRequest = &apierror.Error{
		Code:    "INVALID_AGENT_ASSIGNMENT_REQUEST",
		Message: "invalid agent assignment request",
	}
	ErrInvalidDockerfileRequest = &apierror.Error{
		Code:    "INVALID_DOCKERFILE_REQUEST",
		Message: "invalid dockerfile request",
	}
	ErrInvalidFeatureRequest = &apierror.Error{
		Code:    "INVALID_FEATURE_REQUEST",
		Message: "invalid feature request data",
	}
)
