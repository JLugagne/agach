package daemonws

import "encoding/json"

// Message type constants
const (
	TypeDockerList    = "docker.list"
	TypeDockerRebuild = "docker.rebuild"
	TypeDockerLogs    = "docker.logs"
	TypeDockerPrune   = "docker.prune"
	TypeBuildEvent    = "docker.build_event"
	TypePruneEvent    = "docker.prune_event"
	TypeError         = "error"
)

// Message represents a WebSocket message exchanged between daemon and server.
type Message struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// BuildEvent represents a build status event.
type BuildEvent struct {
	DockerfileSlug string `json:"dockerfile_slug"`
	BuildID        string `json:"build_id"`
	Status         string `json:"status"`
	Log            string `json:"log,omitempty"`
}

// PruneEvent represents a prune progress event.
type PruneEvent struct {
	DockerfileSlug string `json:"dockerfile_slug"`
	BuildID        string `json:"build_id,omitempty"`
	Status         string `json:"status"`
	Removed        int    `json:"removed"`
	Total          int    `json:"total"`
}
