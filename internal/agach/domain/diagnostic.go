package domain

import "time"

// DiagnosticStatus represents the state of a single agent diagnostic run
type DiagnosticStatus string

const (
	DiagnosticPending DiagnosticStatus = "pending"
	DiagnosticRunning DiagnosticStatus = "running"
	DiagnosticDone    DiagnosticStatus = "done"
	DiagnosticError   DiagnosticStatus = "error"
)

// DiagnosticResult holds token measurements for one agent cold-start probe
type DiagnosticResult struct {
	AgentSlug string // "" for baseline (no --agent flag)
	Status    DiagnosticStatus
	Error     string

	// Token usage (from assistant event)
	InputTokens              int // total: input + cache_read + cache_creation
	OutputTokens             int
	CacheReadInputTokens     int
	CacheCreationInputTokens int
	Duration                 time.Duration

	// From result event
	CostUSD float64
	Model   string

	// From system init event
	SystemToolCount int
	MCPToolCount    int
	MCPTools        []string
	AgentCount      int
	Agents          []string
	SkillCount      int
	Skills          []string

	// From /context command
	ContextRaw string
}

// DiagnosticUpdate is sent as results come in from RunDiagnostic
type DiagnosticUpdate struct {
	Results []DiagnosticResult // snapshot of all results (baseline first)
	Done    bool               // true when all probes are finished
}
