package domain

import (
	"encoding/json"
	"time"
)

// RunScope defines which projects/subprojects to include in a run
type RunScope string

const (
	RunScopeMain    RunScope = "main"
	RunScopeAll     RunScope = "all"
	RunScopeSpecific RunScope = "specific"
)

// RunConfig holds the configuration for a TUI execution run
type RunConfig struct {
	ProjectID    string
	ProjectName  string
	Scope        RunScope
	SubProjectID string // only if Scope == RunScopeSpecific
	MaxWorkers   int
	RoleSlug     string // agent role slug to use
	ServerURL    string
	AutoStart    bool // automatically pick up tasks arriving via SSE when a worker slot is free
}

// WorkerStatus represents the current state of a worker goroutine
type WorkerStatus string

const (
	WorkerIdle    WorkerStatus = "idle"
	WorkerRunning WorkerStatus = "running"
	WorkerDone    WorkerStatus = "done"
	WorkerError   WorkerStatus = "error"
)

// TaskRun holds information about a task execution
type TaskRun struct {
	TaskID      string
	TaskTitle   string
	ProjectID   string
	ProjectName string
	IsSubProject bool
	ParentName  string

	// Claude session info
	SessionID string
	AgentRole string // --agent flag value (role slug)

	// Token usage (updated in real-time via stream-json)
	InputTokens              int
	OutputTokens             int
	CacheReadInputTokens     int
	CacheCreationInputTokens int
	TotalTokens              int

	// Cold start cost (first assistant exchange)
	ColdStartCaptured                bool
	ColdStartInputTokens             int
	ColdStartOutputTokens            int
	ColdStartCacheReadInputTokens    int
	ColdStartCacheCreationInputTokens int
	Exchanges        int
	Model            string

	StartedAt   time.Time
	CompletedAt *time.Time
	Status      WorkerStatus
	Error       string
}

// WorkerState holds the current state of a worker
type WorkerState struct {
	ID      int
	Status  WorkerStatus
	Current *TaskRun
	Past    []TaskRun
}

// RunState is the global state of an active run
type RunState struct {
	Config  RunConfig
	Workers []WorkerState
	Started time.Time
	Stopped bool
}

// ColumnCounts holds the number of tasks in each kanban column
type ColumnCounts struct {
	Todo       int
	InProgress int
	Done       int
	Blocked    int
}

// MessageKind identifies the type of a live message from the Claude stream.
type MessageKind string

const (
	MessageKindAssistant  MessageKind = "assistant"
	MessageKindToolUse    MessageKind = "tool_use"
	MessageKindToolResult MessageKind = "tool_result"
	MessageKindSystem     MessageKind = "system"
	MessageKindResult     MessageKind = "result"
)

// LiveMessage represents a single parsed message from a Claude stream event.
type LiveMessage struct {
	Kind     MessageKind
	Content  string
	WorkerID int
	At       time.Time
}

// StreamEvent represents a parsed event from claude --output-format stream-json
type StreamEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	Message   *struct {
		Usage *struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
		Model   string `json:"model"`
		Content []struct {
			Type    string          `json:"type"`
			Text    string          `json:"text"`
			Name    string          `json:"name"`
			Input   json.RawMessage `json:"input"`
			Content string          `json:"content"`
		} `json:"content"`
	} `json:"message"`
	Messages []LiveMessage `json:"-"` // parsed from this event, WorkerID left as 0 (filled by caller)
}
