package domain

import "time"

// OnboardingResult holds the tokens and node information received after
// successful daemon onboarding.
type OnboardingResult struct {
	AccessToken  string
	RefreshToken string
	NodeID       string
	NodeName     string
	Mode         string
}

// ProjectInfo holds project metadata returned from the server.
type ProjectInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	GitURL string `json:"git_url"`
}

// DaemonState represents the current state of the daemon.
type DaemonState string

const (
	StateDisconnected DaemonState = "disconnected"
	StateOnboarding   DaemonState = "onboarding"
	StateConnected    DaemonState = "connected"
)

// BuildID represents a unique docker build identifier.
type BuildID string

func (id BuildID) String() string { return string(id) }

// BuildStatus represents the status of a docker build.
type BuildStatus string

const (
	BuildPending  BuildStatus = "pending"
	BuildBuilding BuildStatus = "building"
	BuildSuccess  BuildStatus = "success"
	BuildFailed   BuildStatus = "failed"
)

// DockerBuild represents a docker image build record.
type DockerBuild struct {
	ID             BuildID
	DockerfileSlug string
	Version        string
	ImageHash      string
	ImageSize      int64
	Status         BuildStatus
	BuildLog       string
	CreatedAt      time.Time
	CompletedAt    *time.Time
}
