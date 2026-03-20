package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/repositories/tasks"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	"github.com/JLugagne/agach-mcp/pkg/websocket"
)

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

	return map[string]any{"id": string(project.ID), "success": true}, nil
}

func (h *ToolHandler) updateProject(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	name, _ := args["name"].(string)
	description, _ := args["description"].(string)
	var defaultRole *string
	if v, ok := args["default_role"].(string); ok {
		defaultRole = &v
	}

	err := h.commands.UpdateProject(ctx, projectID, name, description, defaultRole)
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
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)

	roles, err := h.queries.ListProjectRoles(ctx, projectID)
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

	return map[string]any{"success": true}, nil
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

	// Determine start column: backlog (default) or todo
	startInBacklog := true
	if startIn, ok := args["start_in"].(string); ok && startIn == "todo" {
		startInBacklog = false
	}

	task, err := h.commands.CreateTask(ctx, projectID, title, summary, description, priority, createdByRole, createdByAgent, assignedRole, contextFiles, tags, estimatedEffort, startInBacklog)
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

	return map[string]any{"id": string(task.ID)}, nil
}

func (h *ToolHandler) bulkCreateTasks(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)

	tasksRaw, ok := args["tasks"].([]any)
	if !ok || len(tasksRaw) == 0 {
		return nil, fmt.Errorf("tasks is required and must be a non-empty array")
	}

	inputs := make([]service.BulkTaskInput, 0, len(tasksRaw))
	for i, raw := range tasksRaw {
		t, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("tasks[%d] must be an object", i)
		}

		title, _ := t["title"].(string)
		if title == "" {
			return nil, fmt.Errorf("tasks[%d].title is required", i)
		}
		summary, _ := t["summary"].(string)
		if summary == "" {
			return nil, fmt.Errorf("tasks[%d].summary is required", i)
		}
		createdByRole, _ := t["created_by_role"].(string)
		if createdByRole == "" {
			return nil, fmt.Errorf("tasks[%d].created_by_role is required", i)
		}

		description, _ := t["description"].(string)
		createdByAgent, _ := t["created_by_agent"].(string)
		assignedRole, _ := t["assigned_role"].(string)
		estimatedEffort, _ := t["estimated_effort"].(string)

		priorityStr, _ := t["priority"].(string)
		priority := domain.PriorityMedium
		if priorityStr != "" {
			priority = domain.Priority(priorityStr)
		}

		contextFiles := parseStringArray(t["context_files"])
		tags := parseStringArray(t["tags"])
		dependsOnStrs := parseStringArray(t["depends_on"])

		dependsOn := make([]domain.TaskID, len(dependsOnStrs))
		for j, id := range dependsOnStrs {
			dependsOn[j] = domain.TaskID(id)
		}

		// Determine start column: backlog (default) or todo
		startInBacklog := true
		if startIn, ok := t["start_in"].(string); ok && startIn == "todo" {
			startInBacklog = false
		}

		inputs = append(inputs, service.BulkTaskInput{
			Title:           title,
			Summary:         summary,
			Description:     description,
			Priority:        priority,
			CreatedByRole:   createdByRole,
			CreatedByAgent:  createdByAgent,
			AssignedRole:    assignedRole,
			ContextFiles:    contextFiles,
			Tags:            tags,
			EstimatedEffort: estimatedEffort,
			StartInBacklog:  startInBacklog,
			DependsOn:       dependsOn,
		})
	}

	createdTasks, err := h.commands.BulkCreateTasks(ctx, projectID, inputs)
	if err != nil {
		return nil, err
	}

	type createdTask struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}

	results := make([]createdTask, 0, len(createdTasks))
	for _, task := range createdTasks {
		if h.hub != nil {
			h.hub.Broadcast(websocket.Event{
				Type:      "task_created",
				ProjectID: string(projectID),
				Data:      task,
			})
		}
		results = append(results, createdTask{ID: string(task.ID), Title: task.Title})
	}

	return results, nil
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
	var filesModified *[]string

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
	if args["files_modified"] != nil {
		fm := parseStringArray(args["files_modified"])
		filesModified = &fm
	}

	err := h.commands.UpdateTask(ctx, projectID, taskID, title, description, assignedRole, estimatedEffort, resolution, priority, contextFiles, tags, nil, nil)
	if err != nil {
		return nil, err
	}

	// Handle files_modified update if provided
	if filesModified != nil {
		err = h.commands.UpdateTaskFiles(ctx, projectID, taskID, filesModified, nil)
	}
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
		if targetColumn != domain.ColumnInProgress {
			h.hub.Broadcast(websocket.Event{
				Type:      "wip_slot_available",
				ProjectID: string(projectID),
				Data:      map[string]string{"project_id": string(projectID)},
			})
		}
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

	err := h.commands.CompleteTask(ctx, projectID, taskID, completionSummary, filesModified, completedByAgent, nil)
	if err != nil {
		return nil, err
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
		h.hub.Broadcast(websocket.Event{
			Type:      "wip_slot_available",
			ProjectID: string(projectID),
			Data:      map[string]string{"project_id": string(projectID)},
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

	commentCount, err := h.queries.CountComments(ctx, projectID, taskID)
	if err != nil {
		return nil, err
	}

	deps, err := h.queries.ListDependencies(ctx, projectID, taskID)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"description":  task.Description,
		"has_comments": commentCount > 0,
	}
	depIDs := make([]string, 0, len(deps))
	for _, d := range deps {
		depIDs = append(depIDs, string(d.DependsOnTaskID))
	}
	result["depends_on"] = depIDs
	if len(task.ContextFiles) > 0 {
		result["context_files"] = task.ContextFiles
	}
	if len(task.FilesModified) > 0 {
		result["files_modified"] = task.FilesModified
	}
	if includeResolution, _ := args["include_resolution"].(bool); includeResolution {
		if task.Resolution != "" {
			result["resolution"] = task.Resolution
		}
		if task.CompletionSummary != "" {
			result["completion_summary"] = task.CompletionSummary
		}
	}
	return result, nil
}

func (h *ToolHandler) getNextTask(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	role, _ := args["role"].(string)

	var subProjectID *domain.ProjectID
	if spID, ok := args["sub_project_id"].(string); ok && spID != "" {
		pid := domain.ProjectID(spID)
		subProjectID = &pid
	}

	task, err := h.queries.GetNextTask(ctx, projectID, role, subProjectID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return map[string]any{"task": nil, "message": "no task available"}, nil
	}
	result := map[string]any{
		"id":            string(task.ID),
		"title":         task.Title,
		"summary":       task.Summary,
		"priority":      string(task.Priority),
		"assigned_role": task.AssignedRole,
		"project_id":    string(projectID),
		"session_id":    task.SessionID,
	}
	if len(task.ContextFiles) > 0 {
		result["context_files"] = task.ContextFiles
	}
	return result, nil
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
		h.hub.Broadcast(websocket.Event{
			Type:      "wip_slot_available",
			ProjectID: string(projectID),
			Data:      map[string]string{"project_id": string(projectID)},
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

	// If move_to_todo is true, move the task from backlog to todo
	if moveToTodo, _ := args["move_to_todo"].(bool); moveToTodo {
		if err := h.commands.MoveTask(ctx, projectID, taskID, domain.ColumnTodo); err != nil {
			return nil, fmt.Errorf("dependency added but failed to move task to todo: %w", err)
		}
	}

	return map[string]any{"success": true}, nil
}

func (h *ToolHandler) bulkAddDependencies(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)

	depsRaw, ok := args["dependencies"].([]any)
	if !ok || len(depsRaw) == 0 {
		return nil, fmt.Errorf("dependencies is required and must be a non-empty array")
	}

	added := 0
	for i, raw := range depsRaw {
		d, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("dependencies[%d] must be an object", i)
		}
		taskID, _ := d["task_id"].(string)
		if taskID == "" {
			return nil, fmt.Errorf("dependencies[%d].task_id is required", i)
		}
		dependsOn, _ := d["depends_on_task_id"].(string)
		if dependsOn == "" {
			return nil, fmt.Errorf("dependencies[%d].depends_on_task_id is required", i)
		}

		if err := h.commands.AddDependency(ctx, projectID, domain.TaskID(taskID), domain.TaskID(dependsOn)); err != nil {
			return nil, fmt.Errorf("dependencies[%d]: %w", i, err)
		}

		// If move_to_todo is true, move the task from backlog to todo
		if moveToTodo, _ := d["move_to_todo"].(bool); moveToTodo {
			if err := h.commands.MoveTask(ctx, projectID, domain.TaskID(taskID), domain.ColumnTodo); err != nil {
				return nil, fmt.Errorf("dependencies[%d] move_to_todo: %w", i, err)
			}
		}

		added++
	}

	return map[string]any{"added": added}, nil
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

func (h *ToolHandler) reportTokens(ctx context.Context, args map[string]any) (any, error) {
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

	usage := &domain.TokenUsage{
		InputTokens:      intArg(args, "input", 0),
		OutputTokens:     intArg(args, "output", 0),
		CacheReadTokens:  intArg(args, "cache_read", 0),
		CacheWriteTokens: intArg(args, "cache_write", 0),
	}
	if model, ok := args["model"].(string); ok {
		usage.Model = model
	}

	err := h.commands.UpdateTask(ctx, projectID, taskID, nil, nil, nil, nil, nil, nil, nil, nil, usage, nil)
	if err != nil {
		return nil, err
	}

	if h.hub != nil {
		h.hub.Broadcast(websocket.Event{
			Type:      "task_updated",
			ProjectID: string(projectID),
			Data:      map[string]string{"task_id": string(taskID)},
		})
	}

	return map[string]any{"success": true}, nil
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

func (h *ToolHandler) getWIPSlots(ctx context.Context, args map[string]any) (any, error) {
	projectIDVal, ok := args["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("project_id is required and must be a string")
	}
	projectID := domain.ProjectID(projectIDVal)
	info, err := h.queries.GetWIPSlots(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return info, nil
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
		{Slug: "backlog", Name: "Backlog", Count: info.TaskSummary.BacklogCount},
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
