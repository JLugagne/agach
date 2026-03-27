package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// newID generates a UUIDv7 string.
func newID() string {
	id, _ := uuid.NewV7()
	return id.String()
}

// ProjectID represents a unique project identifier
type ProjectID string

// NewProjectID generates a new project ID
func NewProjectID() ProjectID {
	return ProjectID(newID())
}

// String returns the string representation of a ProjectID
func (id ProjectID) String() string {
	return string(id)
}

// ParseProjectID validates and returns a ProjectID.
// A valid ID is a standard UUID string.
func ParseProjectID(s string) (ProjectID, error) {
	if _, err := uuid.Parse(s); err != nil {
		return "", fmt.Errorf("invalid project ID %q: must be a valid UUID", s)
	}
	return ProjectID(s), nil
}

// AgentID represents a unique agent identifier
type AgentID string

// NewAgentID generates a new agent ID
func NewAgentID() AgentID {
	return AgentID(newID())
}

// String returns the string representation of an AgentID
func (id AgentID) String() string {
	return string(id)
}

// RoleID is an alias for AgentID — kept for test compatibility until _test.go files are updated.
type RoleID = AgentID

// NewRoleID is an alias for NewAgentID — kept for test compatibility until _test.go files are updated.
var NewRoleID = NewAgentID

// TaskID represents a unique task identifier
type TaskID string

// NewTaskID generates a new task ID as a UUID string.
func NewTaskID() TaskID {
	return TaskID(newID())
}

// String returns the string representation of a TaskID
func (id TaskID) String() string {
	return string(id)
}

// ColumnID represents a unique column identifier
type ColumnID string

// NewColumnID generates a new column ID
func NewColumnID() ColumnID {
	return ColumnID(newID())
}

// String returns the string representation of a ColumnID
func (id ColumnID) String() string {
	return string(id)
}

// CommentID represents a unique comment identifier
type CommentID string

// NewCommentID generates a new comment ID
func NewCommentID() CommentID {
	return CommentID(newID())
}

// String returns the string representation of a CommentID
func (id CommentID) String() string {
	return string(id)
}

// FeatureID represents a unique feature identifier
type FeatureID string

// NewFeatureID generates a new feature ID
func NewFeatureID() FeatureID {
	return FeatureID(newID())
}

// String returns the string representation of a FeatureID
func (id FeatureID) String() string {
	return string(id)
}

// ParseFeatureID validates and returns a FeatureID.
func ParseFeatureID(s string) (FeatureID, error) {
	if _, err := uuid.Parse(s); err != nil {
		return "", fmt.Errorf("invalid feature ID %q: must be a valid UUID", s)
	}
	return FeatureID(s), nil
}

// DependencyID represents a unique dependency identifier
type DependencyID string

// NewDependencyID generates a new dependency ID
func NewDependencyID() DependencyID {
	return DependencyID(newID())
}

// String returns the string representation of a DependencyID
func (id DependencyID) String() string {
	return string(id)
}

// SkillID represents a unique skill identifier
type SkillID string

// NewSkillID generates a new skill ID
func NewSkillID() SkillID {
	return SkillID(newID())
}

// String returns the string representation of a SkillID
func (id SkillID) String() string {
	return string(id)
}

// SpecializedAgentID represents a unique specialized agent identifier
type SpecializedAgentID string

// NewSpecializedAgentID generates a new specialized agent ID
func NewSpecializedAgentID() SpecializedAgentID {
	return SpecializedAgentID(newID())
}

// String returns the string representation of a SpecializedAgentID
func (id SpecializedAgentID) String() string {
	return string(id)
}

// DockerfileID represents a unique dockerfile identifier
type DockerfileID string

// NewDockerfileID generates a new dockerfile ID
func NewDockerfileID() DockerfileID {
	return DockerfileID(newID())
}

// String returns the string representation of a DockerfileID
func (id DockerfileID) String() string {
	return string(id)
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
		return 0
	}
}

// ColumnSlug represents the fixed column slugs
type ColumnSlug string

const (
	ColumnBacklog    ColumnSlug = "backlog"
	ColumnTodo       ColumnSlug = "todo"
	ColumnInProgress ColumnSlug = "in_progress"
	ColumnDone       ColumnSlug = "done"
	ColumnBlocked    ColumnSlug = "blocked"
)

// FeatureStatus represents the status of a feature
type FeatureStatus string

const (
	FeatureStatusDraft      FeatureStatus = "draft"
	FeatureStatusReady      FeatureStatus = "ready"
	FeatureStatusInProgress FeatureStatus = "in_progress"
	FeatureStatusDone       FeatureStatus = "done"
	FeatureStatusBlocked    FeatureStatus = "blocked"
)

// ValidFeatureStatuses is the set of all valid feature statuses.
var ValidFeatureStatuses = map[FeatureStatus]bool{
	FeatureStatusDraft:      true,
	FeatureStatusReady:      true,
	FeatureStatusInProgress: true,
	FeatureStatusDone:       true,
	FeatureStatusBlocked:    true,
}

// AuthorType represents the type of comment author
type AuthorType string

const (
	AuthorTypeAgent AuthorType = "agent"
	AuthorTypeHuman AuthorType = "human"
)

// Project represents a project or sub-project
type Project struct {
	ID             ProjectID    `json:"id"`
	ParentID       *ProjectID   `json:"parent_id"`
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	GitURL         string       `json:"git_url"`
	DefaultRole    string       `json:"default_role"`
	CreatedByRole  string       `json:"created_by_role"`
	CreatedByAgent string       `json:"created_by_agent"`
	DockerfileID   *DockerfileID `json:"dockerfile_id"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// Feature represents a feature (story/epic) within a project
type Feature struct {
	ID             FeatureID     `json:"id"`
	ProjectID      ProjectID     `json:"project_id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	UserChangelog  string        `json:"user_changelog"`
	TechChangelog  string        `json:"tech_changelog"`
	Status         FeatureStatus `json:"status"`
	CreatedByRole  string        `json:"created_by_role"`
	CreatedByAgent string        `json:"created_by_agent"`
	NodeID         string        `json:"node_id,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// Agent represents an agent in the system
type Agent struct {
	ID             AgentID   `json:"id"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	Icon           string    `json:"icon"`
	Color          string    `json:"color"`
	Description    string    `json:"description"`
	TechStack      []string  `json:"tech_stack"`
	PromptHint     string    `json:"prompt_hint"`
	PromptTemplate string    `json:"prompt_template"`
	Content        string    `json:"content"`
	Model          string    `json:"model"`
	Thinking       string    `json:"thinking"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
}

// Role is an alias for Agent — kept for test compatibility until _test.go files are updated.
type Role = Agent

// Skill represents a reusable capability that can be assigned to an agent
type Skill struct {
	ID          SkillID   `json:"id"`
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

// SpecializedAgent represents a specialization of an existing agent (role)
type SpecializedAgent struct {
	ID            SpecializedAgentID `json:"id"`
	ParentAgentID AgentID            `json:"parent_agent_id"`
	Slug          string             `json:"slug"`
	Name          string             `json:"name"`
	SortOrder     int                `json:"sort_order"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

// Dockerfile represents a Docker Compose service definition with versioning
type Dockerfile struct {
	ID          DockerfileID `json:"id"`
	Slug        string       `json:"slug"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Version     string       `json:"version"`
	Content     string       `json:"content"`
	IsLatest    bool         `json:"is_latest"`
	SortOrder   int          `json:"sort_order"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Column represents a kanban column
type Column struct {
	ID        ColumnID   `json:"id"`
	Slug      ColumnSlug `json:"slug"`
	Name      string     `json:"name"`
	Position  int        `json:"position"`
	CreatedAt time.Time  `json:"created_at"`
}

// TokenUsage represents cumulative token usage for a task
type TokenUsage struct {
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	CacheReadTokens  int    `json:"cache_read_tokens"`
	CacheWriteTokens int    `json:"cache_write_tokens"`
	Model            string `json:"model"`

	// Cold start (first exchange) — SET semantics, not accumulated
	ColdStartInputTokens      int `json:"cold_start_input_tokens,omitempty"`
	ColdStartOutputTokens     int `json:"cold_start_output_tokens,omitempty"`
	ColdStartCacheReadTokens  int `json:"cold_start_cache_read_tokens,omitempty"`
	ColdStartCacheWriteTokens int `json:"cold_start_cache_write_tokens,omitempty"`
}

// Task represents a task in the kanban board
type Task struct {
	ID        TaskID     `json:"id"`
	ColumnID  ColumnID   `json:"column_id"`
	FeatureID *FeatureID `json:"feature_id"`
	Title     string     `json:"title"`
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
	Model                     string     `json:"model"`
	ColdStartInputTokens      int        `json:"cold_start_input_tokens"`
	ColdStartOutputTokens     int        `json:"cold_start_output_tokens"`
	ColdStartCacheReadTokens  int        `json:"cold_start_cache_read_tokens"`
	ColdStartCacheWriteTokens int        `json:"cold_start_cache_write_tokens"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	SeenAt               *time.Time `json:"seen_at"` // NULL = unseen, non-NULL = timestamp of first view
	StartedAt            *time.Time `json:"started_at"`
	DurationSeconds      int        `json:"duration_seconds"`
	HumanEstimateSeconds int        `json:"human_estimate_seconds"`
	SessionID            string     `json:"session_id"` // Claude Code session ID for resuming
	NodeID               string     `json:"node_id,omitempty"`
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
	BacklogCount    int `json:"backlog_count"`
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

// FeatureWithTaskSummary contains a feature with its task counts
type FeatureWithTaskSummary struct {
	Feature
	TaskSummary ProjectSummary `json:"task_summary"`
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

// AgentColdStartStat holds aggregated cold-start token stats per agent
type AgentColdStartStat struct {
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

// RoleColdStartStat is an alias for AgentColdStartStat — kept for test compatibility until _test.go files are updated.
type RoleColdStartStat = AgentColdStartStat

// TimelineEntry represents task counts for a single day
type TimelineEntry struct {
	Date           string `json:"date"`            // "2026-03-17"
	TasksCreated   int    `json:"tasks_created"`
	TasksCompleted int    `json:"tasks_completed"`
}

// ModelTokenStat holds aggregated token usage for a single model
type ModelTokenStat struct {
	Model           string `json:"model"`
	TaskCount       int    `json:"task_count"`
	InputTokens     int    `json:"input_tokens"`
	OutputTokens    int    `json:"output_tokens"`
	CacheReadTokens int    `json:"cache_read_tokens"`
	CacheWriteTokens int   `json:"cache_write_tokens"`
}

// FeatureTaskSummary represents a completed task's summary for a feature changelog view
type FeatureTaskSummary struct {
	ID                TaskID     `json:"id"`
	Title             string     `json:"title"`
	CompletionSummary string     `json:"completion_summary"`
	CompletedByAgent  string     `json:"completed_by_agent"`
	CompletedAt       time.Time  `json:"completed_at"`
	FilesModified     []string   `json:"files_modified"`
	DurationSeconds   int        `json:"duration_seconds"`
	InputTokens       int        `json:"input_tokens"`
	OutputTokens      int        `json:"output_tokens"`
	CacheReadTokens   int        `json:"cache_read_tokens"`
	CacheWriteTokens  int        `json:"cache_write_tokens"`
	Model             string     `json:"model"`
}

// ModelPricing holds per-model pricing rates (per million tokens)
type ModelPricing struct {
	ID               string    `json:"id"`
	ModelID          string    `json:"model_id"`
	InputPricePer1M  float64   `json:"input_price_per_1m"`
	OutputPricePer1M float64   `json:"output_price_per_1m"`
	CacheReadPricePer1M  float64 `json:"cache_read_price_per_1m"`
	CacheWritePricePer1M float64 `json:"cache_write_price_per_1m"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ProjectUserAccess represents a user's access grant to a project.
type ProjectUserAccess struct {
	ID        string    `json:"id"`
	ProjectID ProjectID `json:"project_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"` // "admin" or "member"
	CreatedAt time.Time `json:"created_at"`
}

// ProjectTeamAccess represents a team's access grant to a project.
type ProjectTeamAccess struct {
	ID        string    `json:"id"`
	ProjectID ProjectID `json:"project_id"`
	TeamID    string    `json:"team_id"`
	CreatedAt time.Time `json:"created_at"`
}

// FeatureStats holds summary stats about features in a project
type FeatureStats struct {
	TotalCount      int `json:"total_count"`
	NotReadyCount   int `json:"not_ready_count"`
	ReadyCount      int `json:"ready_count"`
	InProgressCount int `json:"in_progress_count"`
	DoneCount       int `json:"done_count"`
	BlockedCount    int `json:"blocked_count"`
}

// NotificationID represents a unique notification identifier
type NotificationID string

// NewNotificationID generates a new notification ID
func NewNotificationID() NotificationID {
	return NotificationID(newID())
}

// String returns the string representation of a NotificationID
func (id NotificationID) String() string {
	return string(id)
}

// NotificationSeverity represents notification severity levels
type NotificationSeverity string

const (
	SeverityInfo    NotificationSeverity = "info"
	SeveritySuccess NotificationSeverity = "success"
	SeverityWarning NotificationSeverity = "warning"
	SeverityError   NotificationSeverity = "error"
)

// ValidNotificationSeverities is the set of all valid notification severities.
var ValidNotificationSeverities = map[NotificationSeverity]bool{
	SeverityInfo:    true,
	SeveritySuccess: true,
	SeverityWarning: true,
	SeverityError:   true,
}

// NotificationScope represents the scope of a notification
type NotificationScope string

const (
	NotificationScopeProject NotificationScope = "project"
	NotificationScopeAgent   NotificationScope = "agent"
	NotificationScopeGlobal  NotificationScope = "global"
)

// ValidNotificationScopes is the set of all valid notification scopes.
var ValidNotificationScopes = map[NotificationScope]bool{
	NotificationScopeProject: true,
	NotificationScopeAgent:   true,
	NotificationScopeGlobal:  true,
}

// ChatSessionID represents a unique chat session identifier
type ChatSessionID string

// NewChatSessionID generates a new chat session ID
func NewChatSessionID() ChatSessionID {
	return ChatSessionID(newID())
}

// String returns the string representation of a ChatSessionID
func (id ChatSessionID) String() string {
	return string(id)
}

// ParseChatSessionID validates and returns a ChatSessionID.
func ParseChatSessionID(s string) (ChatSessionID, error) {
	if _, err := uuid.Parse(s); err != nil {
		return "", fmt.Errorf("invalid chat session ID %q: must be a valid UUID", s)
	}
	return ChatSessionID(s), nil
}

// Notification represents a user-facing notification
type Notification struct {
	ID        NotificationID       `json:"id"`
	ProjectID *ProjectID           `json:"project_id,omitempty"`
	Scope     NotificationScope    `json:"scope"`
	AgentSlug string               `json:"agent_slug,omitempty"`
	Severity  NotificationSeverity `json:"severity"`
	Title     string               `json:"title"`
	Text      string               `json:"text"`
	LinkURL   string               `json:"link_url,omitempty"`
	LinkText  string               `json:"link_text,omitempty"`
	LinkStyle string               `json:"link_style,omitempty"`
	ReadAt    *time.Time           `json:"read_at"`
	CreatedAt time.Time            `json:"created_at"`
}

// Domain validation constants.
const (
	MaxTaskTitleLength       = 500
	MaxCommentContentLength  = 10_000
	MaxPromptTemplateLength  = 100_000
)

// ValidateTitle checks that the task title is within bounds.
func (t *Task) ValidateTitle() error {
	if len(t.Title) > MaxTaskTitleLength {
		return fmt.Errorf("task title exceeds maximum length of %d characters", MaxTaskTitleLength)
	}
	return nil
}

// ValidateContent checks that the comment content is within bounds.
func (c *Comment) ValidateContent() error {
	if len(c.Content) > MaxCommentContentLength {
		return fmt.Errorf("comment content exceeds maximum length of %d characters", MaxCommentContentLength)
	}
	return nil
}

// ValidateAuthorType checks that the author type is valid.
func (c *Comment) ValidateAuthorType() error {
	if c.AuthorType != AuthorTypeAgent && c.AuthorType != AuthorTypeHuman {
		return fmt.Errorf("invalid author type %q: must be %q or %q", c.AuthorType, AuthorTypeAgent, AuthorTypeHuman)
	}
	return nil
}

// ValidatePromptTemplate checks that the prompt template is within bounds.
func (a *Agent) ValidatePromptTemplate() error {
	if len(a.PromptTemplate) > MaxPromptTemplateLength {
		return fmt.Errorf("prompt template exceeds maximum length of %d characters", MaxPromptTemplateLength)
	}
	return nil
}
