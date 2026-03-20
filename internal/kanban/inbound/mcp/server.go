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
type ToolFunction func(ctx context.Context, args map[string]any) (any, error)

// Server wraps the official MCP SDK server
type Server struct {
	inner       *mcpsdk.Server
	toolHandler *ToolHandler
	commands    service.Commands
	logger      *logrus.Logger
}

// NewServer creates a new MCP server using the official SDK
func NewServer(commands service.Commands, queries service.Queries, hub any, logger *logrus.Logger) (*Server, error) {
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
// It unmarshals CallToolRequest arguments into map[string]any and
// serializes the result as JSON text content.
func (s *Server) wrapHandler(toolName string, fn ToolFunction) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		start := time.Now()

		var args map[string]any
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
			args = make(map[string]any)
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

func (s *Server) addTool(name, description string, schema map[string]any, required []string, handler ToolFunction) {
	inputSchema := map[string]any{
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
	// Project management
	s.addTool("list_projects",
		"Lists all root projects or direct children of a parent project. Optionally filter by work_dir.",
		map[string]any{
			"parent_id": map[string]any{"type": "string", "description": "Optional parent project ID. If omitted, lists root projects."},
			"work_dir":  map[string]any{"type": "string", "description": "Optional absolute path to filter projects by working directory."},
		},
		nil,
		s.toolHandler.listProjects,
	)

	s.addTool("get_project_info",
		"Returns detailed information about a project including stats and breadcrumb",
		map[string]any{
			"project_id": map[string]any{"type": "string", "description": "The project ID"},
		},
		[]string{"project_id"},
		s.toolHandler.getProjectInfo,
	)

	s.addTool("create_project",
		"Creates a new project (root or sub-project) with its own SQLite database",
		map[string]any{
			"name":             map[string]any{"type": "string", "description": "Project name"},
			"description":      map[string]any{"type": "string", "description": "Optional project description"},
			"work_dir":         map[string]any{"type": "string", "description": "Absolute path to the project's working directory on the filesystem"},
			"parent_id":        map[string]any{"type": "string", "description": "Optional parent project ID (creates sub-project if provided)"},
			"created_by_role":  map[string]any{"type": "string", "description": "Role slug of creator (e.g. 'architect', 'tech_lead')"},
			"created_by_agent": map[string]any{"type": "string", "description": "Optional agent identifier"},
		},
		[]string{"name", "work_dir", "created_by_role"},
		s.toolHandler.createProject,
	)

	s.addTool("update_project",
		"Updates project name, description, or default role",
		map[string]any{
			"project_id":   map[string]any{"type": "string", "description": "The project ID to update"},
			"name":         map[string]any{"type": "string", "description": "New project name"},
			"description":  map[string]any{"type": "string", "description": "New project description"},
			"default_role": map[string]any{"type": "string", "description": "Default role slug auto-assigned to new tasks; empty string to clear"},
		},
		[]string{"project_id"},
		s.toolHandler.updateProject,
	)

	s.addTool("delete_project",
		"Deletes a project, all sub-projects, and their databases",
		map[string]any{
			"project_id": map[string]any{"type": "string", "description": "The project ID to delete"},
		},
		[]string{"project_id"},
		s.toolHandler.deleteProject,
	)

	// Role management
	s.addTool("list_roles",
		"Lists all configured roles for a project",
		map[string]any{
			"project_id": map[string]any{"type": "string", "description": "The project ID"},
		},
		[]string{"project_id"},
		s.toolHandler.listRoles,
	)

	s.addTool("get_role",
		"Returns details for a specific role",
		map[string]any{
			"slug": map[string]any{"type": "string", "description": "Role slug (e.g. 'backend_go', 'frontend_react')"},
		},
		[]string{"slug"},
		s.toolHandler.getRole,
	)

	s.addTool("update_role",
		"Updates a role's description, prompt hint, icon, color, or tech stack",
		map[string]any{
			"slug":        map[string]any{"type": "string", "description": "Role slug to update (e.g. 'backend', 'frontend')"},
			"name":        map[string]any{"type": "string", "description": "New role display name"},
			"description": map[string]any{"type": "string", "description": "New role description"},
			"prompt_hint": map[string]any{"type": "string", "description": "New prompt hint for AI agents"},
			"icon":        map[string]any{"type": "string", "description": "New icon emoji"},
			"color":       map[string]any{"type": "string", "description": "New hex color"},
			"tech_stack":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Updated tech stack list"},
		},
		[]string{"slug"},
		s.toolHandler.updateRole,
	)

	// Task management
	s.addTool("create_tasks",
		"Creates one or more tasks. Tasks start in 'backlog' by default. Use start_in='todo' only when the task has no dependencies and is immediately ready. Returns id and title of each created task.",
		map[string]any{
			"project_id": map[string]any{"type": "string", "description": "The project ID"},
			"tasks": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title":            map[string]any{"type": "string"},
						"summary":          map[string]any{"type": "string"},
						"description":      map[string]any{"type": "string"},
						"priority":         map[string]any{"type": "string", "enum": []string{"critical", "high", "medium", "low"}},
						"created_by_role":  map[string]any{"type": "string"},
						"created_by_agent": map[string]any{"type": "string"},
						"assigned_role":    map[string]any{"type": "string"},
						"context_files":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"tags":             map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"estimated_effort": map[string]any{"type": "string", "enum": []string{"XS", "S", "M", "L", "XL"}},
						"depends_on":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Task IDs this task depends on (can reference IDs from earlier tasks in this batch)"},
						"start_in":         map[string]any{"type": "string", "enum": []string{"backlog", "todo"}, "description": "Which column the task starts in. Default 'backlog'."},
					},
					"required": []string{"title", "summary", "created_by_role"},
				},
			},
		},
		[]string{"project_id", "tasks"},
		s.toolHandler.bulkCreateTasks,
	)

	s.addTool("update_task",
		"Updates task fields including metadata and file lists. Pass only the fields you want to change.",
		map[string]any{
			"project_id":       map[string]any{"type": "string", "description": "The project ID"},
			"task_id":          map[string]any{"type": "string", "description": "The task ID"},
			"title":            map[string]any{"type": "string"},
			"description":      map[string]any{"type": "string"},
			"resolution":       map[string]any{"type": "string"},
			"assigned_role":    map[string]any{"type": "string"},
			"priority":         map[string]any{"type": "string", "enum": []string{"critical", "high", "medium", "low"}},
			"context_files":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Files relevant for understanding the task"},
			"files_modified":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Files the agent has modified"},
			"tags":             map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"estimated_effort": map[string]any{"type": "string", "enum": []string{"XS", "S", "M", "L", "XL"}},
		},
		[]string{"project_id", "task_id"},
		s.toolHandler.updateTask,
	)

	s.addTool("get_task",
		"Returns the task description and whether it has comments. Use list_comments to fetch the actual comments. Set include_resolution to also get resolution and completion_summary (useful for reviewing parent/dependency tasks).",
		map[string]any{
			"project_id":         map[string]any{"type": "string"},
			"task_id":            map[string]any{"type": "string"},
			"include_resolution": map[string]any{"type": "boolean", "description": "If true, also returns resolution and completion_summary fields"},
		},
		[]string{"project_id", "task_id"},
		s.toolHandler.getTask,
	)

	s.addTool("list_tasks",
		"Lists tasks with optional filters. Returns paginated results (default 50). To find the next task to work on, use get_next_task instead — it is faster and returns only the highest-priority ready task.",
		map[string]any{
			"project_id":        map[string]any{"type": "string"},
			"column":            map[string]any{"type": "string"},
			"assigned_role":     map[string]any{"type": "string"},
			"tag":               map[string]any{"type": "string"},
			"priority":          map[string]any{"type": "string"},
			"search":            map[string]any{"type": "string", "description": "Full-text search query (matches title, summary, description, tags)"},
			"is_blocked":        map[string]any{"type": "boolean"},
			"wont_do_requested": map[string]any{"type": "boolean"},
			"ready_only":        map[string]any{"type": "boolean", "description": "If true, returns only tasks that are ready to be worked on (in todo, not blocked, no unresolved deps)"},
			"limit":             map[string]any{"type": "integer", "description": "Max results to return (default 50)"},
			"offset":            map[string]any{"type": "integer", "description": "Number of results to skip (default 0)"},
		},
		[]string{"project_id"},
		s.toolHandler.listTasks,
	)

	s.addTool("get_next_task",
		"Returns the highest-priority ready task in the 'todo' column. A task is ready when it is not blocked, has no unresolved dependencies, and (optionally) matches the requested role. Returns null task if none available.",
		map[string]any{
			"project_id":     map[string]any{"type": "string", "description": "The project ID"},
			"role":           map[string]any{"type": "string", "description": "Optional role slug to filter tasks by assigned role"},
			"sub_project_id": map[string]any{"type": "string", "description": "Optional sub-project ID to scope the search to that sub-project and its descendants"},
		},
		[]string{"project_id"},
		s.toolHandler.getNextTask,
	)

	s.addTool("move_task",
		"Moves a task to 'todo', 'in_progress', or back to 'backlog'",
		map[string]any{
			"project_id":    map[string]any{"type": "string"},
			"task_id":       map[string]any{"type": "string"},
			"target_column": map[string]any{"type": "string", "enum": []string{"backlog", "todo", "in_progress"}},
		},
		[]string{"project_id", "task_id", "target_column"},
		s.toolHandler.moveTask,
	)

	s.addTool("complete_task",
		"Marks a task as done",
		map[string]any{
			"project_id":         map[string]any{"type": "string"},
			"task_id":            map[string]any{"type": "string"},
			"completion_summary": map[string]any{"type": "string"},
			"files_modified":     map[string]any{"type": "array"},
			"completed_by_agent": map[string]any{"type": "string"},
		},
		[]string{"project_id", "task_id", "completion_summary", "completed_by_agent"},
		s.toolHandler.completeTask,
	)

	s.addTool("block_task",
		"Marks a task as blocked",
		map[string]any{
			"project_id":       map[string]any{"type": "string"},
			"task_id":          map[string]any{"type": "string"},
			"blocked_reason":   map[string]any{"type": "string"},
			"blocked_by_agent": map[string]any{"type": "string"},
		},
		[]string{"project_id", "task_id", "blocked_reason", "blocked_by_agent"},
		s.toolHandler.blockTask,
	)

	s.addTool("request_wont_do",
		"Requests a task be marked as Won't Do",
		map[string]any{
			"project_id":     map[string]any{"type": "string"},
			"task_id":        map[string]any{"type": "string"},
			"wont_do_reason": map[string]any{"type": "string"},
			"requested_by":   map[string]any{"type": "string"},
		},
		[]string{"project_id", "task_id", "wont_do_reason", "requested_by"},
		s.toolHandler.requestWontDo,
	)

	s.addTool("reorder_task",
		"Changes the position of a task within its current column (manual reordering within the same priority level)",
		map[string]any{
			"project_id": map[string]any{"type": "string", "description": "The project ID"},
			"task_id":    map[string]any{"type": "string", "description": "The task ID to reorder"},
			"position":   map[string]any{"type": "integer", "description": "The new 0-based position within the column"},
		},
		[]string{"project_id", "task_id", "position"},
		s.toolHandler.reorderTask,
	)

	s.addTool("move_task_to_project",
		"Moves a task from one sub-project to another. The task lands in the 'todo' column of the target project. Comments and dependencies are NOT moved. Blocking/won't-do flags are reset. Source and target projects must share the same parent (siblings) or have a direct parent-child relationship.",
		map[string]any{
			"project_id":        map[string]any{"type": "string", "description": "The source project ID"},
			"task_id":           map[string]any{"type": "string", "description": "The task ID to move"},
			"target_project_id": map[string]any{"type": "string", "description": "The destination project ID"},
		},
		[]string{"project_id", "task_id", "target_project_id"},
		s.toolHandler.moveTaskToProject,
	)

	// Dependencies
	s.addTool("add_dependencies",
		"Adds one or more dependencies between tasks. Use move_to_todo=true on the last entry for a task to signal it is ready to be worked on (moves from backlog to todo).",
		map[string]any{
			"project_id": map[string]any{"type": "string"},
			"dependencies": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"task_id":            map[string]any{"type": "string"},
						"depends_on_task_id": map[string]any{"type": "string"},
						"move_to_todo":       map[string]any{"type": "boolean", "description": "If true, moves the task from backlog to todo after adding this dependency."},
					},
					"required": []string{"task_id", "depends_on_task_id"},
				},
			},
		},
		[]string{"project_id", "dependencies"},
		s.toolHandler.bulkAddDependencies,
	)

	s.addTool("remove_dependency",
		"Removes a dependency",
		map[string]any{
			"project_id":         map[string]any{"type": "string"},
			"task_id":            map[string]any{"type": "string"},
			"depends_on_task_id": map[string]any{"type": "string"},
		},
		[]string{"project_id", "task_id", "depends_on_task_id"},
		s.toolHandler.removeDependency,
	)

	s.addTool("list_dependencies",
		"Lists dependencies for a task",
		map[string]any{
			"project_id": map[string]any{"type": "string"},
			"task_id":    map[string]any{"type": "string"},
		},
		[]string{"project_id", "task_id"},
		s.toolHandler.listDependencies,
	)

	// Comments
	s.addTool("add_comment",
		"Adds a comment to a task",
		map[string]any{
			"project_id":  map[string]any{"type": "string"},
			"task_id":     map[string]any{"type": "string"},
			"author_role": map[string]any{"type": "string"},
			"author_name": map[string]any{"type": "string"},
			"content":     map[string]any{"type": "string"},
		},
		[]string{"project_id", "task_id", "author_role", "content"},
		s.toolHandler.addComment,
	)

	s.addTool("list_comments",
		"Lists comments for a task. Returns paginated results (default 50).",
		map[string]any{
			"project_id": map[string]any{"type": "string"},
			"task_id":    map[string]any{"type": "string"},
			"limit":      map[string]any{"type": "integer", "description": "Max results to return (default 50)"},
			"offset":     map[string]any{"type": "integer", "description": "Number of results to skip (default 0)"},
		},
		[]string{"project_id", "task_id"},
		s.toolHandler.listComments,
	)

	// Board
	s.addTool("get_board",
		"Returns a lightweight board overview: task counts per column and sub-projects with their summaries. Use list_tasks or get_next_task for actual task data.",
		map[string]any{
			"project_id": map[string]any{"type": "string"},
		},
		[]string{"project_id"},
		s.toolHandler.getBoard,
	)
}
