package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/JLugagne/agach-mcp/internal/kanban/domain"
	"github.com/JLugagne/agach-mcp/internal/kanban/domain/service"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

// ToolFunction is a function that handles a tool invocation
type ToolFunction func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// Server wraps the official MCP SDK server
type Server struct {
	inner       *mcpsdk.Server
	toolHandler *ToolHandler
	commands    service.Commands
	logger      *logrus.Logger
}

// NewServer creates a new MCP server using the official SDK
func NewServer(commands service.Commands, queries service.Queries, hub interface{}, logger *logrus.Logger) (*Server, error) {
	inner := mcpsdk.NewServer(
		&mcpsdk.Implementation{
			Name:    "agach-kanban",
			Version: "1.0.0",
		},
		nil,
	)

	srv := &Server{
		inner:    inner,
		commands: commands,
		logger:   logger,
	}

	srv.toolHandler = NewToolHandler(commands, queries, hub)
	srv.registerAllTools()

	return srv, nil
}

// Run starts the MCP server with stdio transport (blocking)
func (s *Server) Run(ctx context.Context) error {
	return s.inner.Run(ctx, &mcpsdk.StdioTransport{})
}

// HTTPHandler returns an http.Handler that serves MCP over Streamable HTTP transport (2025-03-26 spec).
func (s *Server) HTTPHandler() http.Handler {
	return mcpsdk.NewStreamableHTTPHandler(func(r *http.Request) *mcpsdk.Server {
		return s.inner
	}, nil)
}

// wrapHandler converts a ToolFunction into an mcpsdk.ToolHandler.
// It unmarshals CallToolRequest arguments into map[string]interface{} and
// serializes the result as JSON text content.
func (s *Server) wrapHandler(toolName string, fn ToolFunction) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		start := time.Now()

		var args map[string]interface{}
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				s.logger.WithFields(logrus.Fields{
					"tool":     toolName,
					"duration": time.Since(start).String(),
					"error":    true,
				}).Debug("mcp request completed")
				return &mcpsdk.CallToolResult{
					Content: []mcpsdk.Content{
						&mcpsdk.TextContent{Text: fmt.Sprintf("Invalid arguments: %v", err)},
					},
					IsError: true,
				}, nil
			}
		}
		if args == nil {
			args = make(map[string]interface{})
		}

		result, err := fn(ctx, args)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"tool":     toolName,
				"duration": time.Since(start).String(),
				"error":    true,
			}).Debug("mcp request completed")
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: err.Error()},
				},
				IsError: true,
			}, nil
		}

		jsonBytes, err := json.Marshal(result)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"tool":     toolName,
				"duration": time.Since(start).String(),
				"error":    true,
			}).Debug("mcp request completed")
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{Text: fmt.Sprintf("Failed to serialize result: %v", err)},
				},
				IsError: true,
			}, nil
		}

		s.logger.WithFields(logrus.Fields{
			"tool":     toolName,
			"duration": time.Since(start).String(),
		}).Debug("mcp request completed")

		// Track tool usage (fire-and-forget)
		if projectIDStr, ok := args["project_id"].(string); ok && projectIDStr != "" {
			pid := domain.ProjectID(projectIDStr)
			go func() {
				if err := s.commands.IncrementToolUsage(context.Background(), pid, toolName); err != nil {
					s.logger.WithError(err).WithField("tool", toolName).Warn("failed to track tool usage")
				}
			}()
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: string(jsonBytes)},
			},
		}, nil
	}
}

func (s *Server) addTool(name, description string, schema map[string]interface{}, required []string, handler ToolFunction) {
	inputSchema := map[string]interface{}{
		"type":       "object",
		"properties": schema,
	}
	if len(required) > 0 {
		inputSchema["required"] = required
	}

	s.inner.AddTool(&mcpsdk.Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}, s.wrapHandler(name, handler))
}

// registerAllTools registers all MCP tools with the SDK server
func (s *Server) registerAllTools() {
	// Project management tools
	s.addTool("list_projects",
		"Lists all root projects or direct children of a parent project. Optionally filter by work_dir.",
		map[string]interface{}{
			"parent_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional parent project ID. If omitted, lists root projects.",
			},
			"work_dir": map[string]interface{}{
				"type":        "string",
				"description": "Optional absolute path to filter projects by working directory.",
			},
		},
		nil,
		s.toolHandler.listProjects,
	)

	s.addTool("get_project_info",
		"Returns detailed information about a project including stats and breadcrumb",
		map[string]interface{}{
			"project_id": map[string]interface{}{
				"type":        "string",
				"description": "The project ID",
			},
		},
		[]string{"project_id"},
		s.toolHandler.getProjectInfo,
	)

	s.addTool("create_project",
		"Creates a new project (root or sub-project) with its own SQLite database",
		map[string]interface{}{
			"name":             map[string]interface{}{"type": "string", "description": "Project name"},
			"description":      map[string]interface{}{"type": "string", "description": "Optional project description"},
			"work_dir":         map[string]interface{}{"type": "string", "description": "Absolute path to the project's working directory on the filesystem"},
			"parent_id":        map[string]interface{}{"type": "string", "description": "Optional parent project ID (creates sub-project if provided)"},
			"created_by_role":  map[string]interface{}{"type": "string", "description": "Role slug of creator (e.g. 'architect', 'tech_lead')"},
			"created_by_agent": map[string]interface{}{"type": "string", "description": "Optional agent identifier"},
		},
		[]string{"name", "work_dir", "created_by_role"},
		s.toolHandler.createProject,
	)

	s.addTool("update_project",
		"Updates project name or description",
		map[string]interface{}{
			"project_id":  map[string]interface{}{"type": "string", "description": "The project ID to update"},
			"name":        map[string]interface{}{"type": "string", "description": "New project name"},
			"description": map[string]interface{}{"type": "string", "description": "New project description"},
		},
		[]string{"project_id"},
		s.toolHandler.updateProject,
	)

	s.addTool("delete_project",
		"Deletes a project, all sub-projects, and their databases",
		map[string]interface{}{
			"project_id": map[string]interface{}{"type": "string", "description": "The project ID to delete"},
		},
		[]string{"project_id"},
		s.toolHandler.deleteProject,
	)

	// Role management
	s.addTool("list_roles",
		"Lists all configured roles in the system",
		map[string]interface{}{},
		nil,
		s.toolHandler.listRoles,
	)

	s.addTool("get_role",
		"Returns details for a specific role",
		map[string]interface{}{
			"slug": map[string]interface{}{"type": "string", "description": "Role slug (e.g. 'backend_go', 'frontend_react')"},
		},
		[]string{"slug"},
		s.toolHandler.getRole,
	)

	s.addTool("update_role",
		"Updates a role's description, prompt hint, icon, color, or tech stack",
		map[string]interface{}{
			"slug":        map[string]interface{}{"type": "string", "description": "Role slug to update (e.g. 'backend', 'frontend')"},
			"name":        map[string]interface{}{"type": "string", "description": "New role display name"},
			"description": map[string]interface{}{"type": "string", "description": "New role description"},
			"prompt_hint": map[string]interface{}{"type": "string", "description": "New prompt hint for AI agents"},
			"icon":        map[string]interface{}{"type": "string", "description": "New icon emoji"},
			"color":       map[string]interface{}{"type": "string", "description": "New hex color"},
			"tech_stack":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Updated tech stack list"},
		},
		[]string{"slug"},
		s.toolHandler.updateRole,
	)

	// Task management
	s.addTool("create_task",
		"Creates a new task in the 'todo' column",
		map[string]interface{}{
			"project_id":       map[string]interface{}{"type": "string", "description": "The project ID"},
			"title":            map[string]interface{}{"type": "string", "description": "Task title"},
			"summary":          map[string]interface{}{"type": "string", "description": "Brief summary (required)"},
			"description":      map[string]interface{}{"type": "string", "description": "Detailed description"},
			"priority":         map[string]interface{}{"type": "string", "enum": []string{"critical", "high", "medium", "low"}},
			"created_by_role":  map[string]interface{}{"type": "string"},
			"created_by_agent": map[string]interface{}{"type": "string"},
			"assigned_role":    map[string]interface{}{"type": "string"},
			"context_files":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"tags":             map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"estimated_effort": map[string]interface{}{"type": "string", "enum": []string{"XS", "S", "M", "L", "XL"}},
			"depends_on":       map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
		},
		[]string{"project_id", "title", "summary", "created_by_role"},
		s.toolHandler.createTask,
	)

	s.addTool("update_task",
		"Updates task fields",
		map[string]interface{}{
			"project_id":         map[string]interface{}{"type": "string"},
			"task_id":            map[string]interface{}{"type": "string"},
			"title":              map[string]interface{}{"type": "string"},
			"description":        map[string]interface{}{"type": "string"},
			"resolution":         map[string]interface{}{"type": "string"},
			"assigned_role":      map[string]interface{}{"type": "string"},
			"priority":           map[string]interface{}{"type": "string", "enum": []string{"critical", "high", "medium", "low"}},
			"context_files":      map[string]interface{}{"type": "array"},
			"tags":               map[string]interface{}{"type": "array"},
			"estimated_effort":   map[string]interface{}{"type": "string"},
			"input_tokens":       map[string]interface{}{"type": "integer", "description": "Number of input tokens consumed"},
			"output_tokens":      map[string]interface{}{"type": "integer", "description": "Number of output tokens produced"},
			"cache_read_tokens":  map[string]interface{}{"type": "integer", "description": "Number of cache read tokens"},
			"cache_write_tokens": map[string]interface{}{"type": "integer", "description": "Number of cache write tokens"},
			"model":              map[string]interface{}{"type": "string", "description": "Model name used"},
		},
		[]string{"project_id", "task_id"},
		s.toolHandler.updateTask,
	)

	s.addTool("update_task_files",
		"Updates the list of files linked to a task. Accepts files_modified (files the agent has changed) and context_files (files relevant for understanding the task). Can be called independently during work or alongside move/block/complete operations.",
		map[string]interface{}{
			"project_id":     map[string]interface{}{"type": "string", "description": "The project ID"},
			"task_id":        map[string]interface{}{"type": "string", "description": "The task ID"},
			"files_modified": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "List of file paths the agent has modified"},
			"context_files":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "List of file paths relevant to understanding the task"},
		},
		[]string{"project_id", "task_id"},
		s.toolHandler.updateTaskFiles,
	)

	s.addTool("move_task",
		"Moves a task to 'todo' or 'in_progress'",
		map[string]interface{}{
			"project_id":    map[string]interface{}{"type": "string"},
			"task_id":       map[string]interface{}{"type": "string"},
			"target_column": map[string]interface{}{"type": "string", "enum": []string{"todo", "in_progress"}},
		},
		[]string{"project_id", "task_id", "target_column"},
		s.toolHandler.moveTask,
	)

	s.addTool("complete_task",
		"Marks a task as done",
		map[string]interface{}{
			"project_id":         map[string]interface{}{"type": "string"},
			"task_id":            map[string]interface{}{"type": "string"},
			"completion_summary": map[string]interface{}{"type": "string"},
			"files_modified":     map[string]interface{}{"type": "array"},
			"completed_by_agent": map[string]interface{}{"type": "string"},
			"input_tokens":       map[string]interface{}{"type": "integer", "description": "Number of input tokens consumed"},
			"output_tokens":      map[string]interface{}{"type": "integer", "description": "Number of output tokens produced"},
			"cache_read_tokens":  map[string]interface{}{"type": "integer", "description": "Number of cache read tokens"},
			"cache_write_tokens": map[string]interface{}{"type": "integer", "description": "Number of cache write tokens"},
			"model":              map[string]interface{}{"type": "string", "description": "Model name used"},
		},
		[]string{"project_id", "task_id", "completion_summary", "completed_by_agent"},
		s.toolHandler.completeTask,
	)

	s.addTool("get_task",
		"Returns task details",
		map[string]interface{}{
			"project_id": map[string]interface{}{"type": "string"},
			"task_id":    map[string]interface{}{"type": "string"},
		},
		[]string{"project_id", "task_id"},
		s.toolHandler.getTask,
	)

	s.addTool("list_tasks",
		"Lists tasks with optional filters. Returns paginated results (default 50). To find the next task to work on, use get_next_task instead — it is faster and returns only the highest-priority ready task.",
		map[string]interface{}{
			"project_id":        map[string]interface{}{"type": "string"},
			"column":            map[string]interface{}{"type": "string"},
			"assigned_role":     map[string]interface{}{"type": "string"},
			"tag":               map[string]interface{}{"type": "string"},
			"priority":          map[string]interface{}{"type": "string"},
			"search":            map[string]interface{}{"type": "string", "description": "Full-text search query (matches title, summary, description, tags)"},
			"is_blocked":        map[string]interface{}{"type": "boolean"},
			"wont_do_requested": map[string]interface{}{"type": "boolean"},
			"ready_only":        map[string]interface{}{"type": "boolean", "description": "If true, returns only tasks that are ready to be worked on (in todo, not blocked, no unresolved deps)"},
			"limit":             map[string]interface{}{"type": "integer", "description": "Max results to return (default 50)"},
			"offset":            map[string]interface{}{"type": "integer", "description": "Number of results to skip (default 0)"},
		},
		[]string{"project_id"},
		s.toolHandler.listTasks,
	)

	s.addTool("get_next_task",
		"Returns the highest-priority task ready to be processed (in todo, all dependencies done). Optionally scoped to a sub-project tree.",
		map[string]interface{}{
			"project_id":     map[string]interface{}{"type": "string", "description": "The root project ID"},
			"role":           map[string]interface{}{"type": "string", "description": "Role slug to filter tasks by"},
			"sub_project_id": map[string]interface{}{"type": "string", "description": "Optional sub-project ID to scope the search to that sub-project and its descendants"},
		},
		[]string{"project_id", "role"},
		s.toolHandler.getNextTask,
	)

	s.addTool("block_task",
		"Marks a task as blocked",
		map[string]interface{}{
			"project_id":       map[string]interface{}{"type": "string"},
			"task_id":          map[string]interface{}{"type": "string"},
			"blocked_reason":   map[string]interface{}{"type": "string"},
			"blocked_by_agent": map[string]interface{}{"type": "string"},
		},
		[]string{"project_id", "task_id", "blocked_reason", "blocked_by_agent"},
		s.toolHandler.blockTask,
	)

	s.addTool("request_wont_do",
		"Requests a task be marked as Won't Do",
		map[string]interface{}{
			"project_id":     map[string]interface{}{"type": "string"},
			"task_id":        map[string]interface{}{"type": "string"},
			"wont_do_reason": map[string]interface{}{"type": "string"},
			"requested_by":   map[string]interface{}{"type": "string"},
		},
		[]string{"project_id", "task_id", "wont_do_reason", "requested_by"},
		s.toolHandler.requestWontDo,
	)

	// Dependencies
	s.addTool("add_dependency",
		"Adds a dependency between tasks",
		map[string]interface{}{
			"project_id":         map[string]interface{}{"type": "string"},
			"task_id":            map[string]interface{}{"type": "string"},
			"depends_on_task_id": map[string]interface{}{"type": "string"},
		},
		[]string{"project_id", "task_id", "depends_on_task_id"},
		s.toolHandler.addDependency,
	)

	s.addTool("remove_dependency",
		"Removes a dependency",
		map[string]interface{}{
			"project_id":         map[string]interface{}{"type": "string"},
			"task_id":            map[string]interface{}{"type": "string"},
			"depends_on_task_id": map[string]interface{}{"type": "string"},
		},
		[]string{"project_id", "task_id", "depends_on_task_id"},
		s.toolHandler.removeDependency,
	)

	s.addTool("list_dependencies",
		"Lists dependencies for a task",
		map[string]interface{}{
			"project_id": map[string]interface{}{"type": "string"},
			"task_id":    map[string]interface{}{"type": "string"},
		},
		[]string{"project_id", "task_id"},
		s.toolHandler.listDependencies,
	)

	// Comments
	s.addTool("add_comment",
		"Adds a comment to a task",
		map[string]interface{}{
			"project_id":  map[string]interface{}{"type": "string"},
			"task_id":     map[string]interface{}{"type": "string"},
			"author_role": map[string]interface{}{"type": "string"},
			"author_name": map[string]interface{}{"type": "string"},
			"content":     map[string]interface{}{"type": "string"},
		},
		[]string{"project_id", "task_id", "author_role", "content"},
		s.toolHandler.addComment,
	)

	s.addTool("list_comments",
		"Lists comments for a task. Returns paginated results (default 50).",
		map[string]interface{}{
			"project_id": map[string]interface{}{"type": "string"},
			"task_id":    map[string]interface{}{"type": "string"},
			"limit":      map[string]interface{}{"type": "integer", "description": "Max results to return (default 50)"},
			"offset":     map[string]interface{}{"type": "integer", "description": "Number of results to skip (default 0)"},
		},
		[]string{"project_id", "task_id"},
		s.toolHandler.listComments,
	)

	s.addTool("move_task_to_project",
		"Moves a task from one sub-project to another. The task lands in the 'todo' column of the target project. Comments and dependencies are NOT moved. Blocking/won't-do flags are reset. Source and target projects must share the same parent (siblings) or have a direct parent-child relationship.",
		map[string]interface{}{
			"project_id":        map[string]interface{}{"type": "string", "description": "The source project ID"},
			"task_id":           map[string]interface{}{"type": "string", "description": "The task ID to move"},
			"target_project_id": map[string]interface{}{"type": "string", "description": "The destination project ID"},
		},
		[]string{"project_id", "task_id", "target_project_id"},
		s.toolHandler.moveTaskToProject,
	)

	// Board
	s.addTool("get_board",
		"Returns a lightweight board overview: task counts per column and sub-projects with their summaries. Use list_tasks or get_next_task for actual task data.",
		map[string]interface{}{
			"project_id": map[string]interface{}{"type": "string"},
		},
		[]string{"project_id"},
		s.toolHandler.getBoard,
	)
}
