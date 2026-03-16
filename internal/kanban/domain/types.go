package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// shortID generates an 8-character random hex string (4 bytes of entropy).
// Sufficient for a single-instance system; SQLite PRIMARY KEY catches any collision.
func shortID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random ID: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// ProjectID represents a unique project identifier
type ProjectID string

// NewProjectID generates a new project ID
func NewProjectID() ProjectID {
	return ProjectID(shortID())
}

// RoleID represents a unique role identifier
type RoleID string

// NewRoleID generates a new role ID
func NewRoleID() RoleID {
	return RoleID(shortID())
}

// TaskID represents a unique task identifier
type TaskID string

// NewTaskID generates a new task ID
func NewTaskID() TaskID {
	return TaskID(shortID())
}

// ColumnID represents a unique column identifier
type ColumnID string

// NewColumnID generates a new column ID
func NewColumnID() ColumnID {
	return ColumnID(shortID())
}

// CommentID represents a unique comment identifier
type CommentID string

// NewCommentID generates a new comment ID
func NewCommentID() CommentID {
	return CommentID(shortID())
}

// DependencyID represents a unique dependency identifier
type DependencyID string

// NewDependencyID generates a new dependency ID
func NewDependencyID() DependencyID {
	return DependencyID(shortID())
}

// Priority represents task priority levels
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
)

// PriorityScore returns the numeric score for a priority
func (p Priority) Score() int {
	switch p {
	case PriorityCritical:
		return 400
	case PriorityHigh:
		return 300
	case PriorityMedium:
		return 200
	case PriorityLow:
		return 100
	default:
		return 200 // default to medium
	}
}

// ColumnSlug represents the fixed column slugs
type ColumnSlug string

const (
	ColumnTodo       ColumnSlug = "todo"
	ColumnInProgress ColumnSlug = "in_progress"
	ColumnDone       ColumnSlug = "done"
	ColumnBlocked    ColumnSlug = "blocked"
)

// AuthorType represents the type of comment author
type AuthorType string

const (
	AuthorTypeAgent AuthorType = "agent"
	AuthorTypeHuman AuthorType = "human"
)

// Project represents a project or sub-project
type Project struct {
	ID             ProjectID  `json:"id"`
	ParentID       *ProjectID `json:"parent_id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	WorkDir        string     `json:"work_dir"`
	CreatedByRole  string     `json:"created_by_role"`
	CreatedByAgent string     `json:"created_by_agent"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Role represents an agent role in the system
type Role struct {
	ID          RoleID    `json:"id"`
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

// Column represents a kanban column
type Column struct {
	ID        ColumnID   `json:"id"`
	Slug      ColumnSlug `json:"slug"`
	Name      string     `json:"name"`
	Position  int        `json:"position"`
	WIPLimit  int        `json:"wip_limit"`
	CreatedAt time.Time  `json:"created_at"`
}

// TokenUsage represents cumulative token usage for a task
type TokenUsage struct {
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	CacheReadTokens  int    `json:"cache_read_tokens"`
	CacheWriteTokens int    `json:"cache_write_tokens"`
	Model            string `json:"model"`
}

// Task represents a task in the kanban board
type Task struct {
	ID                TaskID     `json:"id"`
	ColumnID          ColumnID   `json:"column_id"`
	Title             string     `json:"title"`
	Summary           string     `json:"summary"` // Brief description (required at creation)
	Description       string     `json:"description"`
	Priority          Priority   `json:"priority"`
	PriorityScore     int        `json:"priority_score"`
	Position          int        `json:"position"`
	CreatedByRole     string     `json:"created_by_role"`
	CreatedByAgent    string     `json:"created_by_agent"`
	AssignedRole      string     `json:"assigned_role"`
	IsBlocked         bool       `json:"is_blocked"` // 1 when in "blocked" column
	BlockedReason     string     `json:"blocked_reason"`
	BlockedAt         *time.Time `json:"blocked_at"`
	BlockedByAgent    string     `json:"blocked_by_agent"`
	WontDoRequested   bool       `json:"wont_do_requested"` // 1 when agent requests won't-do (task in "blocked")
	WontDoReason      string     `json:"wont_do_reason"`
	WontDoRequestedBy string     `json:"wont_do_requested_by"`
	WontDoRequestedAt *time.Time `json:"wont_do_requested_at"`
	CompletionSummary string     `json:"completion_summary"`
	CompletedByAgent  string     `json:"completed_by_agent"`
	CompletedAt       *time.Time `json:"completed_at"`
	FilesModified     []string   `json:"files_modified"`
	Resolution        string     `json:"resolution"` // Filled when agent stops work or human moves back
	ContextFiles      []string   `json:"context_files"`
	Tags              []string   `json:"tags"`
	EstimatedEffort   string     `json:"estimated_effort"`
	InputTokens       int        `json:"input_tokens"`
	OutputTokens      int        `json:"output_tokens"`
	CacheReadTokens   int        `json:"cache_read_tokens"`
	CacheWriteTokens  int        `json:"cache_write_tokens"`
	Model             string     `json:"model"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	SeenAt            *time.Time `json:"seen_at"` // NULL = unseen, non-NULL = timestamp of first view
}

// Comment represents a comment on a task
type Comment struct {
	ID         CommentID  `json:"id"`
	TaskID     TaskID     `json:"task_id"`
	AuthorRole string     `json:"author_role"`
	AuthorName string     `json:"author_name"`
	AuthorType AuthorType `json:"author_type"`
	Content    string     `json:"content"`
	EditedAt   *time.Time `json:"edited_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// TaskDependency represents a dependency between tasks
type TaskDependency struct {
	ID              DependencyID `json:"id"`
	TaskID          TaskID       `json:"task_id"`
	DependsOnTaskID TaskID       `json:"depends_on_task_id"`
	CreatedAt       time.Time    `json:"created_at"`
}

// ProjectSummary contains task counts per column for a project
type ProjectSummary struct {
	TodoCount       int `json:"todo_count"`
	InProgressCount int `json:"in_progress_count"`
	DoneCount       int `json:"done_count"`
	BlockedCount    int `json:"blocked_count"`
}

// ProjectWithSummary contains a project with its task summary and children count
type ProjectWithSummary struct {
	Project
	ChildrenCount int            `json:"children_count"`
	TaskSummary   ProjectSummary `json:"task_summary"`
}

// ProjectInfo contains complete project information for agents
type ProjectInfo struct {
	Project     Project              `json:"project"`      // Full project metadata
	TaskSummary ProjectSummary       `json:"task_summary"` // Task counts per column
	Children    []ProjectWithSummary `json:"children"`     // Direct sub-projects with their summaries
	Breadcrumb  []Project            `json:"breadcrumb"`   // Path from root to this project
}

// TaskWithDetails contains a task with additional metadata
type TaskWithDetails struct {
	Task
	HasUnresolvedDeps bool `json:"has_unresolved_deps"`
	CommentCount      int  `json:"comment_count"`
}

// ToolUsageStat represents the execution count of a single MCP tool
type ToolUsageStat struct {
	ToolName       string     `json:"tool_name"`
	ExecutionCount int        `json:"execution_count"`
	LastExecutedAt *time.Time `json:"last_executed_at"`
}

// DependencyContext provides context about a completed dependency
type DependencyContext struct {
	TaskID            TaskID   `json:"task_id"`
	Title             string   `json:"title"`
	CompletionSummary string   `json:"completion_summary"`
	FilesModified     []string `json:"files_modified"`
}
