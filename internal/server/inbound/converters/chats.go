package converters

import (
	"github.com/JLugagne/agach-mcp/internal/server/domain"
	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

// ToPublicChatSession converts domain.ChatSession to pkgserver.ChatSessionResponse
func ToPublicChatSession(cs domain.ChatSession) pkgserver.ChatSessionResponse {
	return pkgserver.ChatSessionResponse{
		ID:               cs.ID.String(),
		FeatureID:        cs.FeatureID.String(),
		ProjectID:        cs.ProjectID.String(),
		NodeID:           cs.NodeID,
		State:            string(cs.State),
		ClaudeSessionID:  cs.ClaudeSessionID,
		JSONLPath:        cs.JSONLPath,
		InputTokens:      cs.InputTokens,
		OutputTokens:     cs.OutputTokens,
		CacheReadTokens:  cs.CacheReadTokens,
		CacheWriteTokens: cs.CacheWriteTokens,
		Model:            cs.Model,
		CreatedAt:        cs.CreatedAt,
		EndedAt:          cs.EndedAt,
		UpdatedAt:        cs.UpdatedAt,
	}
}

// ToPublicChatSessions converts []domain.ChatSession to []pkgserver.ChatSessionResponse
func ToPublicChatSessions(css []domain.ChatSession) []pkgserver.ChatSessionResponse {
	result := make([]pkgserver.ChatSessionResponse, len(css))
	for i, cs := range css {
		result[i] = ToPublicChatSession(cs)
	}
	return result
}
