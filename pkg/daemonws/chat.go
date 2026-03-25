package daemonws

import "encoding/json"

const (
	TypeChatStart      = "chat.start"
	TypeChatMessage    = "chat.message"
	TypeChatUserMsg    = "chat.user_message"
	TypeChatEnd        = "chat.end"
	TypeChatError      = "chat.error"
	TypeChatStats      = "chat.stats"
	TypeChatPing       = "chat.ping"
	TypeChatTTLWarning = "chat.ttl_warning"
)

type ChatStartRequest struct {
	SessionID       string `json:"session_id"`
	FeatureID       string `json:"feature_id"`
	ProjectID       string `json:"project_id"`
	NodeID          string `json:"node_id,omitempty"`
	ResumeSessionID string `json:"resume_session_id,omitempty"`
}

type ChatStartResponse struct {
	SessionID       string `json:"session_id"`
	ClaudeSessionID string `json:"claude_session_id"`
	WorktreePath    string `json:"worktree_path"`
}

type ChatUserMessage struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}

type ChatMessageEvent struct {
	SessionID   string          `json:"session_id"`
	MessageType string          `json:"message_type"`
	Content     json.RawMessage `json:"content"`
	IsFinal     bool            `json:"is_final"`
}

type ChatStatsEvent struct {
	SessionID        string  `json:"session_id"`
	MessageCount     int     `json:"message_count"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	CacheReadTokens  int     `json:"cache_read_tokens"`
	CacheWriteTokens int     `json:"cache_write_tokens"`
	TotalCost        float64 `json:"total_cost"`
	DurationSeconds  int     `json:"duration_seconds"`
	Model            string  `json:"model"`
}

type ChatEndEvent struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
	JSONLPath string `json:"jsonl_path,omitempty"`
}

type ChatErrorEvent struct {
	SessionID string `json:"session_id"`
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`
}

type ChatTTLWarningEvent struct {
	SessionID        string `json:"session_id"`
	SecondsRemaining int    `json:"seconds_remaining"`
}
