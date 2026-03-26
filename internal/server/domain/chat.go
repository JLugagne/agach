package domain

import "time"

// ChatSessionState represents the state of a chat session
type ChatSessionState string

const (
	ChatStateActive  ChatSessionState = "active"
	ChatStateEnded   ChatSessionState = "ended"
	ChatStateTimeout ChatSessionState = "timeout"
)

// ValidChatSessionStates is the set of all valid chat session states
var ValidChatSessionStates = map[ChatSessionState]bool{
	ChatStateActive:  true,
	ChatStateEnded:   true,
	ChatStateTimeout: true,
}

// ChatSession represents a chat session between a user and Claude for a feature
type ChatSession struct {
	ID               ChatSessionID    `json:"id"`
	FeatureID        FeatureID        `json:"feature_id"`
	ProjectID        ProjectID        `json:"project_id"`
	NodeID           string           `json:"node_id,omitempty"`
	State            ChatSessionState `json:"state"`
	ClaudeSessionID  string           `json:"claude_session_id,omitempty"`
	JSONLPath        string           `json:"jsonl_path,omitempty"`
	InputTokens      int              `json:"input_tokens"`
	OutputTokens     int              `json:"output_tokens"`
	CacheReadTokens  int              `json:"cache_read_tokens"`
	CacheWriteTokens int              `json:"cache_write_tokens"`
	Model            string           `json:"model,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	EndedAt          *time.Time       `json:"ended_at,omitempty"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// ChatSessionWithCost includes computed cost information
type ChatSessionWithCost struct {
	ChatSession
	TotalCost float64 `json:"total_cost"`
}
