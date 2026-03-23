package app

import (
	"github.com/JLugagne/agach-mcp/internal/agach/domain"
	"github.com/JLugagne/agach-mcp/pkg/server/client"
)

var validUUID = "550e8400-e29b-41d4-a716-446655440000"
var validUUID2 = "660e8400-e29b-41d4-a716-446655440001"

func validTask() client.NextTaskResult {
	return client.NextTaskResult{
		ID:        validUUID,
		Title:     "Test Task",
		Role:      "go-test",
		ProjectID: validUUID2,
		SessionID: "",
	}
}

func validCfg() domain.RunConfig {
	return domain.RunConfig{
		ProjectID: validUUID,
	}
}
