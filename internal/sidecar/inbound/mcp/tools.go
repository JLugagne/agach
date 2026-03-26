package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/sidecar/app"
	"github.com/JLugagne/agach-mcp/internal/sidecar/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- bulk_create_tasks ---

type bulkCreateTasksArgs struct {
	Tasks []domain.BulkTaskInput `json:"tasks"`
}

func registerBulkCreateTasks(s *mcp.Server, application *app.App) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "bulk_create_tasks",
		Description: "Create multiple tasks at once. Each task can have a 'ref' field for intra-batch dependency references. Dependencies in 'depends_on' can reference other tasks by their 'ref' or by existing task IDs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args bulkCreateTasksArgs) (*mcp.CallToolResult, any, error) {
		created, err := application.BulkCreateTasks(ctx, args.Tasks)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return jsonResult(created)
	})
}

// --- bulk_add_dependencies ---

type bulkAddDependenciesArgs struct {
	Dependencies []domain.BulkDependencyInput `json:"dependencies"`
}

func registerBulkAddDependencies(s *mcp.Server, application *app.App) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "bulk_add_dependencies",
		Description: "Add multiple task dependencies at once. Each entry specifies a task_id and the depends_on_task_id it should depend on.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args bulkAddDependenciesArgs) (*mcp.CallToolResult, any, error) {
		if err := application.BulkAddDependencies(ctx, args.Dependencies); err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("dependencies added")
	})
}

// --- complete_task ---

type completeTaskArgs struct {
	TaskID            string   `json:"task_id"`
	CompletionSummary string   `json:"completion_summary"`
	FilesModified     []string `json:"files_modified"`
	CompletedByAgent  string   `json:"completed_by_agent"`
}

func registerCompleteTask(s *mcp.Server, application *app.App) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "complete_task",
		Description: "Mark a task as completed with a summary of what was done.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args completeTaskArgs) (*mcp.CallToolResult, any, error) {
		err := application.CompleteTask(ctx, args.TaskID, domain.CompleteTaskRequest{
			CompletionSummary: args.CompletionSummary,
			FilesModified:     args.FilesModified,
			CompletedByAgent:  args.CompletedByAgent,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("task completed")
	})
}

// --- run_task ---

type runTaskArgs struct {
	TaskID string `json:"task_id"`
}

func registerRunTask(s *mcp.Server, application *app.App) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "run_task",
		Description: "Start working on a task by moving it to in_progress.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args runTaskArgs) (*mcp.CallToolResult, any, error) {
		if err := application.RunTask(ctx, args.TaskID); err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("task started")
	})
}

// --- block_task ---

type blockTaskArgs struct {
	TaskID         string `json:"task_id"`
	BlockedReason  string `json:"blocked_reason"`
	BlockedByAgent string `json:"blocked_by_agent"`
}

func registerBlockTask(s *mcp.Server, application *app.App) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "block_task",
		Description: "Block a task with a reason explaining why it cannot proceed.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args blockTaskArgs) (*mcp.CallToolResult, any, error) {
		err := application.BlockTask(ctx, args.TaskID, domain.BlockTaskRequest{
			BlockedReason:  args.BlockedReason,
			BlockedByAgent: args.BlockedByAgent,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("task blocked")
	})
}

// --- wont_do_task ---

type wontDoTaskArgs struct {
	TaskID            string `json:"task_id"`
	WontDoReason      string `json:"wont_do_reason"`
	WontDoRequestedBy string `json:"wont_do_requested_by"`
}

func registerWontDoTask(s *mcp.Server, application *app.App) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "wont_do_task",
		Description: "Request that a task be marked as won't do, with a reason.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args wontDoTaskArgs) (*mcp.CallToolResult, any, error) {
		err := application.WontDoTask(ctx, args.TaskID, domain.WontDoRequest{
			WontDoReason:      args.WontDoReason,
			WontDoRequestedBy: args.WontDoRequestedBy,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("won't do requested")
	})
}

// --- feature_changelogs ---

type featureChangelogsArgs struct {
	UserChangelog *string `json:"user_changelog,omitempty"`
	TechChangelog *string `json:"tech_changelog,omitempty"`
}

func registerFeatureChangelogs(s *mcp.Server, application *app.App) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "feature_changelogs",
		Description: "Update the feature's changelogs. Provide user_changelog for user-facing changes and/or tech_changelog for technical implementation notes. The feature ID is automatically injected.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args featureChangelogsArgs) (*mcp.CallToolResult, any, error) {
		err := application.UpdateFeatureChangelogs(ctx, domain.FeatureChangelogsRequest{
			UserChangelog: args.UserChangelog,
			TechChangelog: args.TechChangelog,
		})
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult("feature changelogs updated")
	})
}

// --- helpers ---

func textResult(text string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil, nil
}

func jsonResult(data any) (*mcp.CallToolResult, any, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return errorResult(fmt.Errorf("marshal result: %w", err)), nil, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(b)},
		},
	}, nil, nil
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: err.Error()},
		},
	}
}
