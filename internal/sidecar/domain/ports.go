package domain

import "context"

// ServerAPI defines the operations the sidecar can perform against the agach server.
// Implemented by the outbound proxy client.
type ServerAPI interface {
	CreateTask(ctx context.Context, req CreateTaskRequest) (CreateTaskResponse, error)
	AddDependency(ctx context.Context, taskID, dependsOnTaskID string) error
	CompleteTask(ctx context.Context, taskID string, req CompleteTaskRequest) error
	MoveTask(ctx context.Context, taskID, targetColumn string) error
	BlockTask(ctx context.Context, taskID string, req BlockTaskRequest) error
	RequestWontDo(ctx context.Context, taskID string, req WontDoRequest) error
	UpdateFeatureChangelogs(ctx context.Context, req FeatureChangelogsRequest) error
}

// CreateTaskRequest mirrors the server's CreateTaskRequest for the fields the sidecar uses.
type CreateTaskRequest struct {
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	Description     string   `json:"description"`
	Priority        string   `json:"priority"`
	AssignedRole    string   `json:"assigned_role"`
	ContextFiles    []string `json:"context_files"`
	Tags            []string `json:"tags"`
	EstimatedEffort string   `json:"estimated_effort"`
}

// CreateTaskResponse is the minimal response from task creation.
type CreateTaskResponse struct {
	ID string `json:"id"`
}

// CompleteTaskRequest mirrors the server's CompleteTaskRequest.
type CompleteTaskRequest struct {
	CompletionSummary string   `json:"completion_summary"`
	FilesModified     []string `json:"files_modified"`
	CompletedByAgent  string   `json:"completed_by_agent"`
}

// BlockTaskRequest mirrors the server's BlockTaskRequest.
type BlockTaskRequest struct {
	BlockedReason  string `json:"blocked_reason"`
	BlockedByAgent string `json:"blocked_by_agent"`
}

// WontDoRequest mirrors the server's RequestWontDoRequest.
type WontDoRequest struct {
	WontDoReason      string `json:"wont_do_reason"`
	WontDoRequestedBy string `json:"wont_do_requested_by"`
}

// FeatureChangelogsRequest updates the feature's changelogs.
type FeatureChangelogsRequest struct {
	UserChangelog *string `json:"user_changelog,omitempty"`
	TechChangelog *string `json:"tech_changelog,omitempty"`
}
