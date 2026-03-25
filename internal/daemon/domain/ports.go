package domain

import (
	"context"
	"encoding/json"
)

// WSEvent is the event received from the server over a WebSocket connection.
type WSEvent struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	ProjectID string          `json:"project_id,omitempty"`
	Data      json.RawMessage `json:"data"`
}

// WSEventHandler is the callback invoked for each incoming WebSocket event.
type WSEventHandler func(event WSEvent)

// ServerAuth is the port for daemon authentication with the server.
type ServerAuth interface {
	RefreshDaemonToken(ctx context.Context, nodeID, refreshToken string) (string, error)
}

// ServerOnboarding is the port for the daemon onboarding flow.
type ServerOnboarding interface {
	CompleteOnboarding(ctx context.Context, code, nodeName string) (*OnboardingResult, error)
}

// ServerConnection is the port for the persistent WebSocket connection to the server.
type ServerConnection interface {
	RunWithReconnect(ctx context.Context) error
	Send(msg interface{}) error
}

// ProjectFetcher is the port for fetching project metadata from the server.
type ProjectFetcher interface {
	GetProject(ctx context.Context, token, projectID string) (*ProjectInfo, error)
}

// ChatUploader is the port for uploading chat JSONL files to the server.
type ChatUploader interface {
	UploadJSONL(ctx context.Context, token, projectID, featureID, sessionID, filePath string) error
}
