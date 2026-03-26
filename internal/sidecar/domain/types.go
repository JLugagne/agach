package domain

// Mode controls which MCP tools are available.
type Mode string

const (
	ModePM      Mode = "pm"
	ModeDefault Mode = ""
)

// BulkTaskInput is a single task within a bulk_create_tasks call.
type BulkTaskInput struct {
	Ref             string   `json:"ref"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	Description     string   `json:"description"`
	Priority        string   `json:"priority"`
	AssignedRole    string   `json:"assigned_role"`
	ContextFiles    []string `json:"context_files"`
	Tags            []string `json:"tags"`
	EstimatedEffort string   `json:"estimated_effort"`
	DependsOn       []string `json:"depends_on"`
}

// BulkDependencyInput is a single dependency within a bulk_add_dependencies call.
type BulkDependencyInput struct {
	TaskID          string `json:"task_id"`
	DependsOnTaskID string `json:"depends_on_task_id"`
}

// CreatedTask is the result of creating a single task.
type CreatedTask struct {
	Ref string `json:"ref,omitempty"`
	ID  string `json:"id"`
}
