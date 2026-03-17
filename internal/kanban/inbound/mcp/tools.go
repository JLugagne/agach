package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
)

// taskForWork contains only the fields an agent needs to start working on a task.
// Irrelevant fields (blocked_reason, wont_do_reason, completion_summary, etc.) are omitted.
type taskForWork struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	Description     string   `json:"description"`
	Priority        string   `json:"priority"`
	AssignedRole    string   `json:"assigned_role,omitempty"`
	CreatedByRole   string   `json:"created_by_role,omitempty"`
	ContextFiles    []string `json:"context_files,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	EstimatedEffort string   `json:"estimated_effort,omitempty"`
	Resolution      string   `json:"resolution,omitempty"`
}

func toTaskForWork(task *domain.Task) taskForWork {
	return taskForWork{
		ID:              string(task.ID),
		Title:           task.Title,
		Summary:         task.Summary,
		Description:     task.Description,
		Priority:        string(task.Priority),
		AssignedRole:    task.AssignedRole,
		CreatedByRole:   task.CreatedByRole,
		ContextFiles:    task.ContextFiles,
		Tags:            task.Tags,
		EstimatedEffort: task.EstimatedEffort,
		Resolution:      task.Resolution,
	}
}

// taskSummary is a lightweight representation of a task for MCP list responses.
// Use get_task for full details.
type taskSummary struct {
	ID                string `json:"id"`
	ColumnID          string `json:"column_id"`
	Title             string `json:"title"`
	Summary           string `json:"summary"`
	Priority          string `json:"priority"`
	AssignedRole      string `json:"assigned_role,omitempty"`
	IsBlocked         bool   `json:"is_blocked,omitempty"`
	WontDoRequested   bool   `json:"wont_do_requested,omitempty"`
	HasUnresolvedDeps bool   `json:"has_unresolved_deps,omitempty"`
	Ready             bool   `json:"ready"`
	CommentCount      int    `json:"comment_count,omitempty"`
	CreatedAt         string `json:"created_at"`
}

func toTaskSummaries(taskList []domain.TaskWithDetails) []taskSummary {
	summaries := make([]taskSummary, len(taskList))
	for i, t := range taskList {
		ready := string(t.ColumnID) == "col_todo" && !t.IsBlocked && !t.WontDoRequested && !t.HasUnresolvedDeps
		summaries[i] = taskSummary{
			ID:                string(t.ID),
			ColumnID:          string(t.ColumnID),
			Title:             t.Title,
			Summary:           t.Summary,
			Priority:          string(t.Priority),
			AssignedRole:      t.AssignedRole,
			IsBlocked:         t.IsBlocked,
			WontDoRequested:   t.WontDoRequested,
			HasUnresolvedDeps: t.HasUnresolvedDeps,
			Ready:             ready,
			CommentCount:      t.CommentCount,
			CreatedAt:         t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}
	return summaries
}

// taskDetail is a full representation of a task with omitempty on fields that are commonly empty.
// Used by get_task to avoid sending zero-value fields that waste tokens.
type taskDetail struct {
	ID                string     `json:"id"`
	ColumnID          string     `json:"column_id"`
	Title             string     `json:"title"`
	Summary           string     `json:"summary"`
	Description       string     `json:"description,omitempty"`
	Priority          string     `json:"priority"`
	PriorityScore     int        `json:"priority_score"`
	Position          int        `json:"position"`
	CreatedByRole     string     `json:"created_by_role,omitempty"`
	CreatedByAgent    string     `json:"created_by_agent,omitempty"`
	AssignedRole      string     `json:"assigned_role,omitempty"`
	IsBlocked         bool       `json:"is_blocked,omitempty"`
	BlockedReason     string     `json:"blocked_reason,omitempty"`
	BlockedAt         *time.Time `json:"blocked_at,omitempty"`
	BlockedByAgent    string     `json:"blocked_by_agent,omitempty"`
	WontDoRequested   bool       `json:"wont_do_requested,omitempty"`
	WontDoReason      string     `json:"wont_do_reason,omitempty"`
	WontDoRequestedBy string     `json:"wont_do_requested_by,omitempty"`
	WontDoRequestedAt *time.Time `json:"wont_do_requested_at,omitempty"`
	CompletionSummary string     `json:"completion_summary,omitempty"`
	CompletedByAgent  string     `json:"completed_by_agent,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	FilesModified     []string   `json:"files_modified,omitempty"`
	Resolution        string     `json:"resolution,omitempty"`
	ContextFiles      []string   `json:"context_files,omitempty"`
	Tags              []string   `json:"tags,omitempty"`
	EstimatedEffort   string     `json:"estimated_effort,omitempty"`
	InputTokens       int        `json:"input_tokens,omitempty"`
	OutputTokens      int        `json:"output_tokens,omitempty"`
	CacheReadTokens      int        `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens     int        `json:"cache_write_tokens,omitempty"`
	Model                string     `json:"model,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	DurationSeconds      int        `json:"duration_seconds,omitempty"`
	HumanEstimateSeconds int        `json:"human_estimate_seconds,omitempty"`
	CreatedAt            string     `json:"created_at"`
	UpdatedAt            string     `json:"updated_at"`
}

func toTaskDetail(task *domain.Task) taskDetail {
	return taskDetail{
		ID:                string(task.ID),
		ColumnID:          string(task.ColumnID),
		Title:             task.Title,
		Summary:           task.Summary,
		Description:       task.Description,
		Priority:          string(task.Priority),
		PriorityScore:     task.PriorityScore,
		Position:          task.Position,
		CreatedByRole:     task.CreatedByRole,
		CreatedByAgent:    task.CreatedByAgent,
		AssignedRole:      task.AssignedRole,
		IsBlocked:         task.IsBlocked,
		BlockedReason:     task.BlockedReason,
		BlockedAt:         task.BlockedAt,
		BlockedByAgent:    task.BlockedByAgent,
		WontDoRequested:   task.WontDoRequested,
		WontDoReason:      task.WontDoReason,
		WontDoRequestedBy: task.WontDoRequestedBy,
		WontDoRequestedAt: task.WontDoRequestedAt,
		CompletionSummary: task.CompletionSummary,
		CompletedByAgent:  task.CompletedByAgent,
		CompletedAt:       task.CompletedAt,
		FilesModified:     task.FilesModified,
		Resolution:        task.Resolution,
		ContextFiles:      task.ContextFiles,
		Tags:              task.Tags,
		EstimatedEffort:   task.EstimatedEffort,
		InputTokens:       task.InputTokens,
		OutputTokens:      task.OutputTokens,
		CacheReadTokens:      task.CacheReadTokens,
		CacheWriteTokens:     task.CacheWriteTokens,
		Model:                task.Model,
		StartedAt:            task.StartedAt,
		DurationSeconds:      task.DurationSeconds,
		HumanEstimateSeconds: task.HumanEstimateSeconds,
		CreatedAt:            task.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:            task.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// roleSummary is a lightweight representation of a role for MCP list responses.
// Use get_role for the full role including prompt_hint and tech_stack.
type roleSummary struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
}

func toRoleSummaries(roles []domain.Role) []roleSummary {
	summaries := make([]roleSummary, len(roles))
	for i, r := range roles {
		summaries[i] = roleSummary{
			Slug:        r.Slug,
			Name:        r.Name,
			Icon:        r.Icon,
			Color:       r.Color,
			Description: r.Description,
		}
	}
	return summaries
}

// Broadcaster is an interface for broadcasting WebSocket events
type Broadcaster interface {
	Broadcast(event websocket.Event)
}

// ToolHandler handles MCP tool calls and delegates to service layer
type ToolHandler struct {
	commands service.Commands
	queries  service.Queries
	hub      Broadcaster
}

// NewToolHandler creates a new MCP tool handler
func NewToolHandler(commands service.Commands, queries service.Queries, hub any) *ToolHandler {
	var broadcaster Broadcaster
	if h, ok := hub.(Broadcaster); ok {
		broadcaster = h
	}

	return &ToolHandler{
		commands: commands,
		queries:  queries,
		hub:      broadcaster,
	}
}

// Tool handler implementations

func (h *ToolHandler) listProjects(ctx context.Context, args map[string]any) (any, error) {
	parentIDStr, hasParent := args["parent_id"].(string)
	workDir, hasWorkDir := args["work_dir"].(string)

	if hasParent && parentIDStr != "" {
		parentID := domain.ProjectID(parentIDStr)
		projects, err := h.queries.ListSubProjectsWithSummary(ctx, parentID)
		if err != nil {
			return nil, err
		}
		return projects, nil
	}

	if hasWorkDir && workDir != "" {
		projects, err := h.queries.ListProjectsByWorkDir(ctx, workDir)
		if err != nil {
			return nil, err
		}
		return projects, nil
	}

	projects, err := h.queries.ListProjectsWithSummary(ctx)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (h *ToolHandler) getProjectInfo(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)

	info, err := h.queries.GetProjectInfo(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Slim breadcrumb to just id + name
	type breadcrumbEntry struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	breadcrumb := make([]breadcrumbEntry, len(info.Breadcrumb))
	for i, p := range info.Breadcrumb {
		breadcrumb[i] = breadcrumbEntry{ID: string(p.ID), Name: p.Name}
	}

	// Slim children to essential fields
	type childSummary struct {
		ID          string                `json:"id"`
		Name        string                `json:"name"`
		TaskSummary domain.ProjectSummary `json:"task_summary"`
	}
	children := make([]childSummary, len(info.Children))
	for i, c := range info.Children {
		children[i] = childSummary{
			ID:          string(c.ID),
			Name:        c.Name,
			TaskSummary: c.TaskSummary,
		}
	}

	return map[string]any{
		"project": map[string]any{
			"id":          string(info.Project.ID),
			"name":        info.Project.Name,
			"description": info.Project.Description,
			"parent_id":   info.Project.ParentID,
			"work_dir":    info.Project.WorkDir,
		},
		"task_summary": info.TaskSummary,
		"children":     children,
		"breadcrumb":   breadcrumb,
	}, nil
}

func (h *ToolHandler) createProject(ctx context.Context, args map[string]any) (any, error) {
	nameVal, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required and must be a string")
	}
	description, _ := args["description"].(string)
	workDirVal, ok := args["work_dir"].(string)
	if !ok {
		return nil, fmt.Errorf("work_dir is required and must be a string")
	}
	createdByRoleVal, ok := args["created_by_role"].(string)
	if !ok {
		return nil, fmt.Errorf("created_by_role is required and must be a string")
	}
	name := nameVal
	workDir := workDirVal
	createdByRole := createdByRoleVal
	createdByAgent, _ := args["created_by_agent"].(string)

	var parentID *domain.ProjectID
	if parentIDStr, ok := args["parent_id"].(string); ok && parentIDStr != "" {
		pid := domain.ProjectID(parentIDStr)
		parentID = &pid
	}

	project, err := h.commands.CreateProject(ctx, name, description, workDir, createdByRole, createdByAgent, parentID)
	if err != nil {
		return nil, err
	}

	// Broadcast project_created event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type: "project_created",
			Data: map[string]any{
				"id":        string(project.ID),
				"name":      project.Name,
				"parent_id": project.ParentID,
			},
		})
	}

	return project, nil
}

func (h *ToolHandler) updateProject(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	name, _ := args["name"].(string)
	description, _ := args["description"].(string)

	err := h.commands.UpdateProject(ctx, projectID, name, description)
	if err != nil {
		return nil, err
	}

	// Broadcast project_updated event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type: "project_updated",
			Data: map[string]any{
				"id":          string(projectID),
				"name":        name,
				"description": description,
			},
		})
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) deleteProject(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)

	err := h.commands.DeleteProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Broadcast project_deleted event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type: "project_deleted",
			Data: map[string]any{
				"id": string(projectID),
			},
		})
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) listRoles(ctx context.Context, args map[string]any) (any, error) {
	roles, err := h.queries.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	return toRoleSummaries(roles), nil
}

func (h *ToolHandler) getRole(ctx context.Context, args map[string]any) (any, error) {
	slugVal, ok := args["slug"].(string)
	if !ok {
		return nil, fmt.Errorf("slug is required and must be a string")
	}
	slug := slugVal

	role, err := h.queries.GetRoleBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (h *ToolHandler) updateRole(ctx context.Context, args map[string]any) (any, error) {
	slugVal, ok := args["slug"].(string)
	if !ok {
		return nil, fmt.Errorf("slug is required and must be a string")
	}
	slug := slugVal

	// Look up role by slug to get the ID
	role, err := h.queries.GetRoleBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	name, _ := args["name"].(string)
	icon, _ := args["icon"].(string)
	color, _ := args["color"].(string)
	description, _ := args["description"].(string)
	promptHint, _ := args["prompt_hint"].(string)
	sortOrder := 0

	var techStack []string
	if ts, ok := args["tech_stack"].([]any); ok {
		for _, v := range ts {
			if s, ok := v.(string); ok {
				techStack = append(techStack, s)
			}
		}
	}

	err = h.commands.UpdateRole(ctx, role.ID, name, icon, color, description, promptHint, techStack, sortOrder)
	if err != nil {
		return nil, err
	}

	// Broadcast role_updated event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type: "role_updated",
			Data: map[string]any{
				"slug": slug,
			},
		})
	}

	// Return updated role
	updated, err := h.queries.GetRoleBySlug(ctx, slug)
	if err != nil {
		return map[string]any{"success": true}, nil
	}
	return updated, nil
}

func (h *ToolHandler) createTask(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	titleVal, ok := args["title"].(string)
	if !ok {
		return nil, fmt.Errorf("title is required and must be a string")
	}
	summaryVal, ok := args["summary"].(string)
	if !ok {
		return nil, fmt.Errorf("summary is required and must be a string")
	}
	description, _ := args["description"].(string)
	createdByRoleVal, ok := args["created_by_role"].(string)
	if !ok {
		return nil, fmt.Errorf("created_by_role is required and must be a string")
	}
	title := titleVal
	summary := summaryVal
	createdByRole := createdByRoleVal
	createdByAgent, _ := args["created_by_agent"].(string)
	assignedRole, _ := args["assigned_role"].(string)
	estimatedEffort, _ := args["estimated_effort"].(string)

	priorityStr, _ := args["priority"].(string)
	priority := domain.PriorityMedium
	if priorityStr != "" {
		priority = domain.Priority(priorityStr)
	}

	contextFiles := parseStringArray(args["context_files"])
	tags := parseStringArray(args["tags"])
	dependsOn := parseStringArray(args["depends_on"])

	task, err := h.commands.CreateTask(ctx, projectID, title, summary, description, priority, createdByRole, createdByAgent, assignedRole, contextFiles, tags, estimatedEffort)
	if err != nil {
		return nil, err
	}

	// Add dependencies if provided
	for _, depID := range dependsOn {
		if err := h.commands.AddDependency(ctx, projectID, task.ID, domain.TaskID(depID)); err != nil {
			return nil, fmt.Errorf("failed to add dependency: %w", err)
		}
	}

	// Broadcast task_created event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "task_created",
			ProjectID: string(projectID),
			Data:      task,
		})
	}

	return map[string]any{"id": string(task.ID), "title": task.Title}, nil
}

func (h *ToolHandler) updateTask(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)

	var title, description, assignedRole, estimatedEffort, resolution *string
	var priority *domain.Priority
	var contextFiles, tags *[]string

	if v, ok := args["title"].(string); ok {
		title = &v
	}
	if v, ok := args["description"].(string); ok {
		description = &v
	}
	if v, ok := args["assigned_role"].(string); ok {
		assignedRole = &v
	}
	if v, ok := args["estimated_effort"].(string); ok {
		estimatedEffort = &v
	}
	if v, ok := args["resolution"].(string); ok {
		resolution = &v
	}
	if v, ok := args["priority"].(string); ok {
		p := domain.Priority(v)
		priority = &p
	}
	if args["context_files"] != nil {
		cf := parseStringArray(args["context_files"])
		contextFiles = &cf
	}
	if args["tags"] != nil {
		t := parseStringArray(args["tags"])
		tags = &t
	}

	var tokenUsage *domain.TokenUsage
	inputTokens := intArg(args, "input_tokens", -1)
	outputTokens := intArg(args, "output_tokens", -1)
	cacheReadTokens := intArg(args, "cache_read_tokens", -1)
	cacheWriteTokens := intArg(args, "cache_write_tokens", -1)
	model, hasModel := args["model"].(string)
	if inputTokens >= 0 || outputTokens >= 0 || cacheReadTokens >= 0 || cacheWriteTokens >= 0 || hasModel {
		tokenUsage = &domain.TokenUsage{}
		if inputTokens >= 0 {
			tokenUsage.InputTokens = inputTokens
		}
		if outputTokens >= 0 {
			tokenUsage.OutputTokens = outputTokens
		}
		if cacheReadTokens >= 0 {
			tokenUsage.CacheReadTokens = cacheReadTokens
		}
		if cacheWriteTokens >= 0 {
			tokenUsage.CacheWriteTokens = cacheWriteTokens
		}
		tokenUsage.Model = model
	}

	var humanEstimateSeconds *int
	if v := intArg(args, "human_estimate_seconds", -1); v >= 0 {
		humanEstimateSeconds = &v
	}

	err := h.commands.UpdateTask(ctx, projectID, taskID, title, description, assignedRole, estimatedEffort, resolution, priority, contextFiles, tags, tokenUsage, humanEstimateSeconds)
	if err != nil {
		return nil, err
	}

	// Broadcast task_updated event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "task_updated",
			ProjectID: string(projectID),
			Data:      map[string]string{"task_id": string(taskID)},
		})
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) updateTaskFiles(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)

	var filesModified, contextFiles *[]string

	if args["files_modified"] != nil {
		fm := parseStringArray(args["files_modified"])
		filesModified = &fm
	}
	if args["context_files"] != nil {
		cf := parseStringArray(args["context_files"])
		contextFiles = &cf
	}

	err := h.commands.UpdateTaskFiles(ctx, projectID, taskID, filesModified, contextFiles)
	if err != nil {
		return nil, err
	}

	// Broadcast task_updated event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "task_updated",
			ProjectID: string(projectID),
			Data:      map[string]string{"task_id": string(taskID)},
		})
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) moveTask(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)
	targetColumnVal, ok := args["target_column"].(string)
	if !ok {
		return nil, fmt.Errorf("target_column is required and must be a string")
	}
	targetColumn := domain.ColumnSlug(targetColumnVal)

	err := h.commands.MoveTask(ctx, projectID, taskID, targetColumn)
	if err != nil {
		return nil, err
	}

	// Broadcast task_moved event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "task_moved",
			ProjectID: string(projectID),
			Data: map[string]string{
				"task_id":       string(taskID),
				"target_column": string(targetColumn),
			},
		})
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) completeTask(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)
	completionSummaryVal, ok := args["completion_summary"].(string)
	if !ok {
		return nil, fmt.Errorf("completion_summary is required and must be a string")
	}
	completedByAgentVal, ok := args["completed_by_agent"].(string)
	if !ok {
		return nil, fmt.Errorf("completed_by_agent is required and must be a string")
	}
	completionSummary := completionSummaryVal
	completedByAgent := completedByAgentVal
	filesModified := parseStringArray(args["files_modified"])

	var tokenUsage *domain.TokenUsage
	inputTokens := intArg(args, "input_tokens", -1)
	outputTokens := intArg(args, "output_tokens", -1)
	cacheReadTokens := intArg(args, "cache_read_tokens", -1)
	cacheWriteTokens := intArg(args, "cache_write_tokens", -1)
	model, hasModel := args["model"].(string)
	if inputTokens >= 0 || outputTokens >= 0 || cacheReadTokens >= 0 || cacheWriteTokens >= 0 || hasModel {
		tokenUsage = &domain.TokenUsage{}
		if inputTokens >= 0 {
			tokenUsage.InputTokens = inputTokens
		}
		if outputTokens >= 0 {
			tokenUsage.OutputTokens = outputTokens
		}
		if cacheReadTokens >= 0 {
			tokenUsage.CacheReadTokens = cacheReadTokens
		}
		if cacheWriteTokens >= 0 {
			tokenUsage.CacheWriteTokens = cacheWriteTokens
		}
		tokenUsage.Model = model
	}

	err := h.commands.CompleteTask(ctx, projectID, taskID, completionSummary, filesModified, completedByAgent, tokenUsage)
	if err != nil {
		return nil, err
	}

	// If a human estimate was provided, persist it via UpdateTask
	if humanEst := intArg(args, "human_estimate_seconds", -1); humanEst >= 0 {
		_ = h.commands.UpdateTask(ctx, projectID, taskID, nil, nil, nil, nil, nil, nil, nil, nil, nil, &humanEst)
	}

	// Broadcast task_completed event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "task_completed",
			ProjectID: string(projectID),
			Data: map[string]any{
				"task_id":            string(taskID),
				"completion_summary": completionSummary,
				"files_modified":     filesModified,
				"completed_by_agent": completedByAgent,
			},
		})
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) getTask(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)

	task, err := h.queries.GetTask(ctx, projectID, taskID)
	if err != nil {
		return nil, err
	}
	return toTaskDetail(task), nil
}

func (h *ToolHandler) listTasks(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)

	filters := tasks.TaskFilters{}

	if columnStr, ok := args["column"].(string); ok {
		slug := domain.ColumnSlug(columnStr)
		filters.ColumnSlug = &slug
	}
	if assignedRole, ok := args["assigned_role"].(string); ok {
		filters.AssignedRole = &assignedRole
	}
	if tag, ok := args["tag"].(string); ok {
		filters.Tag = &tag
	}
	if priorityStr, ok := args["priority"].(string); ok {
		priority := domain.Priority(priorityStr)
		filters.Priority = &priority
	}
	if isBlocked, ok := args["is_blocked"].(bool); ok {
		filters.IsBlocked = &isBlocked
	}
	if wontDoRequested, ok := args["wont_do_requested"].(bool); ok {
		filters.WontDoRequested = &wontDoRequested
	}
	if search, ok := args["search"].(string); ok {
		filters.Search = search
	}

	filters.Limit = intArg(args, "limit", 50)
	filters.Offset = intArg(args, "offset", 0)

	taskList, err := h.queries.ListTasks(ctx, projectID, filters)
	if err != nil {
		return nil, err
	}

	readyOnly, _ := args["ready_only"].(bool)
	if readyOnly {
		ready := make([]domain.TaskWithDetails, 0, len(taskList))
		for _, t := range taskList {
			if string(t.ColumnID) == "col_todo" && !t.IsBlocked && !t.WontDoRequested && !t.HasUnresolvedDeps {
				ready = append(ready, t)
			}
		}
		taskList = ready
	}

	return toTaskSummaries(taskList), nil
}

func (h *ToolHandler) getNextTask(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)

	role := ""
	if roleVal, ok := args["role"].(string); ok {
		role = roleVal
	}

	var subProjectID *domain.ProjectID
	if spID, ok := args["sub_project_id"].(string); ok && spID != "" {
		id := domain.ProjectID(spID)
		subProjectID = &id
	}

	task, err := h.queries.GetNextTask(ctx, projectID, role, subProjectID)
	if err != nil {
		if domain.IsDomainError(err) {
			return map[string]any{"task": nil, "message": err.Error()}, nil
		}
		return nil, err
	}

	if task == nil {
		return map[string]any{"task": nil, "message": "No tasks available for this role"}, nil
	}

	// Get dependency context
	depContext, err := h.queries.GetDependencyContext(ctx, projectID, task.ID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"task":               toTaskForWork(task),
		"dependency_context": depContext,
	}, nil
}

func (h *ToolHandler) blockTask(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)
	blockedReasonVal, ok := args["blocked_reason"].(string)
	if !ok {
		return nil, fmt.Errorf("blocked_reason is required and must be a string")
	}
	blockedByAgentVal, ok := args["blocked_by_agent"].(string)
	if !ok {
		return nil, fmt.Errorf("blocked_by_agent is required and must be a string")
	}
	blockedReason := blockedReasonVal
	blockedByAgent := blockedByAgentVal

	err := h.commands.BlockTask(ctx, projectID, taskID, blockedReason, blockedByAgent)
	if err != nil {
		return nil, err
	}

	// Broadcast task_blocked event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "task_blocked",
			ProjectID: string(projectID),
			Data: map[string]string{
				"task_id":          string(taskID),
				"blocked_reason":   blockedReason,
				"blocked_by_agent": blockedByAgent,
			},
		})
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) requestWontDo(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)
	wontDoReasonVal, ok := args["wont_do_reason"].(string)
	if !ok {
		return nil, fmt.Errorf("wont_do_reason is required and must be a string")
	}
	requestedByVal, ok := args["requested_by"].(string)
	if !ok {
		return nil, fmt.Errorf("requested_by is required and must be a string")
	}
	wontDoReason := wontDoReasonVal
	requestedBy := requestedByVal

	err := h.commands.RequestWontDo(ctx, projectID, taskID, wontDoReason, requestedBy)
	if err != nil {
		return nil, err
	}

	// Broadcast wont_do_requested event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "wont_do_requested",
			ProjectID: string(projectID),
			Data: map[string]string{
				"task_id":        string(taskID),
				"wont_do_reason": wontDoReason,
				"requested_by":   requestedBy,
			},
		})
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) addDependency(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)
	dependsOnTaskIDVal, ok := args["depends_on_task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("depends_on_task_id is required and must be a string")
	}
	dependsOnTaskID := domain.TaskID(dependsOnTaskIDVal)

	err := h.commands.AddDependency(ctx, projectID, taskID, dependsOnTaskID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) removeDependency(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)
	dependsOnTaskIDVal, ok := args["depends_on_task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("depends_on_task_id is required and must be a string")
	}
	dependsOnTaskID := domain.TaskID(dependsOnTaskIDVal)

	err := h.commands.RemoveDependency(ctx, projectID, taskID, dependsOnTaskID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) listDependencies(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)

	deps, err := h.queries.ListDependencies(ctx, projectID, taskID)
	if err != nil {
		return nil, err
	}
	return deps, nil
}

func (h *ToolHandler) addComment(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)
	authorRoleVal, ok := args["author_role"].(string)
	if !ok {
		return nil, fmt.Errorf("author_role is required and must be a string")
	}
	authorRole := authorRoleVal
	authorName, _ := args["author_name"].(string)
	contentVal, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content is required and must be a string")
	}
	content := contentVal

	comment, err := h.commands.CreateComment(ctx, projectID, taskID, authorRole, authorName, domain.AuthorTypeAgent, content)
	if err != nil {
		return nil, err
	}

	// Broadcast comment_added event
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "comment_added",
			ProjectID: string(projectID),
			Data:      comment,
		})
	}

	return map[string]any{"id": string(comment.ID), "success": true}, nil
}

func (h *ToolHandler) listComments(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)

	limit := intArg(args, "limit", 50)
	offset := intArg(args, "offset", 0)

	comments, err := h.queries.ListComments(ctx, projectID, taskID, limit, offset)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

func (h *ToolHandler) reorderTask(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)
	newPosition := intArg(args, "position", 0)

	err := h.commands.ReorderTask(ctx, projectID, taskID, newPosition)
	if err != nil {
		return nil, err
	}

	// Broadcast task_updated event so UI clients refresh the board
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "task_updated",
			ProjectID: string(projectID),
			Data: map[string]any{
				"task_id":  string(taskID),
				"position": newPosition,
			},
		})
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) moveTaskToProject(ctx context.Context, args map[string]any) (any, error) {
	sourceProjectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	sourceProjectID := domain.ProjectID(sourceProjectIDVal)
	taskIDVal, ok := args["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id is required and must be a string")
	}
	taskID := domain.TaskID(taskIDVal)
	targetProjectIDVal, ok := args["target_project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("target_project_id is required and must be a string")
	}
	targetProjectID := domain.ProjectID(targetProjectIDVal)

	err := h.commands.MoveTaskToProject(ctx, sourceProjectID, taskID, targetProjectID)
	if err != nil {
		return nil, err
	}

	// Broadcast task_deleted on the source project so the UI removes it
	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "task_deleted",
			ProjectID: string(sourceProjectID),
			Data:      map[string]string{"task_id": string(taskID)},
		})
		// Broadcast task_created on the target project so the UI fetches the new task
		h.hub.Broadcast(websocket.Event{
			Type:      "task_created",
			ProjectID: string(targetProjectID),
			Data: map[string]string{
				"source_project_id": string(sourceProjectID),
				"source_task_id":    string(taskID),
			},
		})
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) getBoard(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)

	// Return a lightweight board overview: column counts + sub-projects with summaries.
	// Agents should use get_next_task or list_tasks (with filters) for actual task data.
	info, err := h.queries.GetProjectInfo(ctx, projectID)
	if err != nil {
		return nil, err
	}

	type columnOverview struct {
		Slug  string `json:"slug"`
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	columns := []columnOverview{
		{Slug: "todo", Name: "To Do", Count: info.TaskSummary.TodoCount},
		{Slug: "in_progress", Name: "In Progress", Count: info.TaskSummary.InProgressCount},
		{Slug: "done", Name: "Done", Count: info.TaskSummary.DoneCount},
		{Slug: "blocked", Name: "Blocked", Count: info.TaskSummary.BlockedCount},
	}

	return map[string]any{
		"project":      info.Project,
		"columns":      columns,
		"sub_projects": info.Children,
	}, nil
}

// Helper functions

func intArg(args map[string]any, key string, defaultVal int) int {
	v, ok := args[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return defaultVal
		}
		return int(i)
	default:
		return defaultVal
	}
}

func parseStringArray(v any) []string {
	if v == nil {
		return nil
	}

	switch arr := v.(type) {
	case []any:
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case []string:
		return arr
	default:
		// Try JSON unmarshaling as fallback
		if bytes, err := json.Marshal(v); err == nil {
			var result []string
			if err := json.Unmarshal(bytes, &result); err == nil {
				return result
			}
		}
		return nil
	}
}
