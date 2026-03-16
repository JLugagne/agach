package kanban

import (
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
)

// CreateProjectRequest represents a request to create a project
type CreateProjectRequest struct {
	Name           string  `json:"name" validate:"required,min=1,max=200"`
	Description    string  `json:"description" validate:"max=5000"`
	WorkDir        string  `json:"work_dir" validate:"max=1000"`
	ParentID       *string `json:"parent_id" validate:"omitempty,entity_id"`
	CreatedByRole  string  `json:"created_by_role" validate:"max=100"`
	CreatedByAgent string  `json:"created_by_agent" validate:"max=100"`
}

// UpdateProjectRequest represents a request to update a project
type UpdateProjectRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=200"`
	Description *string `json:"description" validate:"omitempty,max=5000"`
}

// ProjectResponse represents a project in API responses
type ProjectResponse struct {
	ID             string    `json:"id"`
	ParentID       *string   `json:"parent_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	WorkDir        string    `json:"work_dir"`
	CreatedByRole  string    `json:"created_by_role"`
	CreatedByAgent string    `json:"created_by_agent"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ProjectSummaryResponse represents project task summary
type ProjectSummaryResponse struct {
	TodoCount       int `json:"todo_count"`
	InProgressCount int `json:"in_progress_count"`
	DoneCount       int `json:"done_count"`
	BlockedCount    int `json:"blocked_count"`
}

// CreateRoleRequest represents a request to create a role
type CreateRoleRequest struct {
	Slug        string   `json:"slug" validate:"required,min=1,max=50,alphanum"`
	Name        string   `json:"name" validate:"required,min=1,max=100"`
	Icon        string   `json:"icon" validate:"max=10"`
	Color       string   `json:"color" validate:"omitempty,hexcolor"`
	Description string   `json:"description" validate:"max=1000"`
	TechStack   []string `json:"tech_stack" validate:"dive,max=50"`
	PromptHint  string   `json:"prompt_hint" validate:"max=5000"`
	SortOrder   int      `json:"sort_order"`
}

// UpdateRoleRequest represents a request to update a role
type UpdateRoleRequest struct {
	Name        *string   `json:"name" validate:"omitempty,min=1,max=100"`
	Icon        *string   `json:"icon" validate:"omitempty,max=10"`
	Color       *string   `json:"color" validate:"omitempty,hexcolor"`
	Description *string   `json:"description" validate:"omitempty,max=1000"`
	TechStack   *[]string `json:"tech_stack" validate:"omitempty,dive,max=50"`
	PromptHint  *string   `json:"prompt_hint" validate:"omitempty,max=5000"`
	SortOrder   *int      `json:"sort_order"`
}

// RoleResponse represents a role in API responses
type RoleResponse struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Icon        string    `json:"icon"`
	Color       string    `json:"color"`
	Description string    `json:"description"`
	TechStack   []string  `json:"tech_stack"`
	PromptHint  string    `json:"prompt_hint"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
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
	ContextFiles    []string `json:"context_files" validate:"dive,max=500"`
	Tags            []string `json:"tags" validate:"dive,max=50"`
	EstimatedEffort string   `json:"estimated_effort" validate:"omitempty,oneof=XS S M L XL"`
	DependsOn       []string `json:"depends_on" validate:"dive,entity_id"`
}

// UpdateTaskRequest represents a request to update a task
type UpdateTaskRequest struct {
	Title           *string   `json:"title" validate:"omitempty,min=1,max=500"`
	Description     *string   `json:"description" validate:"omitempty,max=10000"`
	Priority        *string   `json:"priority" validate:"omitempty,oneof=critical high medium low"`
	AssignedRole    *string   `json:"assigned_role" validate:"omitempty,max=100"`
	ContextFiles    *[]string `json:"context_files" validate:"omitempty,dive,max=500"`
	Tags            *[]string `json:"tags" validate:"omitempty,dive,max=50"`
	EstimatedEffort  *string `json:"estimated_effort" validate:"omitempty,oneof=XS S M L XL"`
	Resolution       *string `json:"resolution" validate:"omitempty,max=10000"`
	InputTokens      *int    `json:"input_tokens,omitempty"`
	OutputTokens     *int    `json:"output_tokens,omitempty"`
	CacheReadTokens  *int    `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens *int    `json:"cache_write_tokens,omitempty"`
	Model            *string `json:"model,omitempty"`
}

// MoveTaskRequest represents a request to move a task
type MoveTaskRequest struct {
	TargetColumn string `json:"target_column" validate:"required,oneof=todo in_progress done blocked"`
	Reason       string `json:"reason" validate:"max=1000"`
}

// CompleteTaskRequest represents a request to complete a task
type CompleteTaskRequest struct {
	CompletionSummary string   `json:"completion_summary" validate:"required,min=100,max=10000"`
	FilesModified     []string `json:"files_modified" validate:"dive,max=500"`
	CompletedByAgent  string   `json:"completed_by_agent" validate:"required,max=100"`
	InputTokens       int      `json:"input_tokens,omitempty"`
	OutputTokens      int      `json:"output_tokens,omitempty"`
	CacheReadTokens   int      `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens  int      `json:"cache_write_tokens,omitempty"`
	Model             string   `json:"model,omitempty"`
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

// TaskResponse represents a task in API responses
type TaskResponse struct {
	ID                string     `json:"id"`
	ColumnID          string     `json:"column_id"`
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
	SeenAt            *time.Time `json:"seen_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
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
	WIPLimit  int       `json:"wip_limit"`
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

// Validation errors
var (
	ErrInvalidProjectRequest = &domain.Error{
		Code:    "INVALID_PROJECT_REQUEST",
		Message: "invalid project request data",
	}
	ErrInvalidRoleRequest = &domain.Error{
		Code:    "INVALID_ROLE_REQUEST",
		Message: "invalid role request data",
	}
	ErrInvalidTaskRequest = &domain.Error{
		Code:    "INVALID_TASK_REQUEST",
		Message: "invalid task request data",
	}
	ErrInvalidCommentRequest = &domain.Error{
		Code:    "INVALID_COMMENT_REQUEST",
		Message: "invalid comment request data",
	}
	ErrInvalidDependencyRequest = &domain.Error{
		Code:    "INVALID_DEPENDENCY_REQUEST",
		Message: "invalid dependency request data",
	}
	ErrInvalidImageRequest = &domain.Error{
		Code:    "INVALID_IMAGE_REQUEST",
		Message: "invalid image request",
	}
)
