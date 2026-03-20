package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	pkgkanban "github.com/JLugagne/agach-mcp/pkg/kanban"
)

// Client is an HTTP client for the Kanban REST API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

type NextTaskResult struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Role      string `json:"role"`
	ProjectID string `json:"project_id"`
	SessionID string `json:"session_id"`
}

type ListTasksParams struct {
	Column      string
	AssignedRole string
	Tag         string
	Priority    string
	Search      string
	Limit       int
	Offset      int
}

type apiResponse[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

func decodeResponse[T any](resp *http.Response) (T, error) {
	defer resp.Body.Close()
	var result apiResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		var zero T
		return zero, err
	}
	if result.Error != nil {
		var zero T
		return zero, fmt.Errorf("%s: %s", result.Error.Code, result.Error.Message)
	}
	return result.Data, nil
}

// Projects

func (c *Client) ListProjects() ([]pkgkanban.ProjectResponse, error) {
	resp, err := c.do(http.MethodGet, "/api/projects", nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[[]pkgkanban.ProjectResponse](resp)
}

func (c *Client) GetProject(id string) (*pkgkanban.ProjectResponse, error) {
	resp, err := c.do(http.MethodGet, "/api/projects/"+id, nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[pkgkanban.ProjectResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateProject(req pkgkanban.CreateProjectRequest) (*pkgkanban.ProjectResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/projects", req)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[pkgkanban.ProjectResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Per-project roles

func (c *Client) ListProjectRoles(projectID string) ([]pkgkanban.RoleResponse, error) {
	resp, err := c.do(http.MethodGet, "/api/projects/"+projectID+"/roles", nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[[]pkgkanban.RoleResponse](resp)
}

func (c *Client) CreateProjectRole(projectID string, req pkgkanban.CreateRoleRequest) (*pkgkanban.RoleResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/projects/"+projectID+"/roles", req)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[pkgkanban.RoleResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateProjectRole(projectID, slug string, req pkgkanban.UpdateRoleRequest) error {
	resp, err := c.do(http.MethodPatch, "/api/projects/"+projectID+"/roles/"+slug, req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

func (c *Client) DeleteProjectRole(projectID, slug string) error {
	resp, err := c.do(http.MethodDelete, "/api/projects/"+projectID+"/roles/"+slug, nil)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// Tasks

func (c *Client) GetNextTasks(projectID string, count int, role string, subProjectID *string, includeSubProjects bool) ([]NextTaskResult, error) {
	u := fmt.Sprintf("/api/projects/%s/next-tasks?count=%d", projectID, count)
	if role != "" {
		u += "&role=" + url.QueryEscape(role)
	}
	if subProjectID != nil {
		u += "&sub_project_id=" + url.QueryEscape(*subProjectID)
	}
	if includeSubProjects {
		u += "&include_subprojects=true"
	}
	resp, err := c.do(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[[]NextTaskResult](resp)
}

// WaitForNextTask blocks until the SSE stream emits an event for the given project
// (meaning a new task is ready) or the context is cancelled.
func (c *Client) WaitForNextTask(ctx context.Context, projectID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/projects/"+url.PathEscape(projectID)+"/sse", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			// Any event received — a task is ready
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// UpdateTaskSessionID saves the claude session_id on a task (best-effort)
func (c *Client) UpdateTaskSessionID(projectID, taskID, sessionID string) error {
	req := map[string]string{"session_id": sessionID}
	resp, err := c.do(http.MethodPatch, fmt.Sprintf("/api/projects/%s/tasks/%s/session", projectID, taskID), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// UpdateTask updates a task via PATCH
func (c *Client) UpdateTask(projectID, taskID string, req pkgkanban.UpdateTaskRequest) error {
	resp, err := c.do(http.MethodPatch, fmt.Sprintf("/api/projects/%s/tasks/%s", projectID, taskID), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

func (c *Client) ListTasks(projectID string, params ListTasksParams) ([]pkgkanban.TaskWithDetailsResponse, error) {
	u := fmt.Sprintf("/api/projects/%s/tasks", projectID)
	q := url.Values{}
	if params.Column != "" {
		q.Set("column", params.Column)
	}
	if params.AssignedRole != "" {
		q.Set("assigned_role", params.AssignedRole)
	}
	if params.Tag != "" {
		q.Set("tag", params.Tag)
	}
	if params.Priority != "" {
		q.Set("priority", params.Priority)
	}
	if params.Search != "" {
		q.Set("search", params.Search)
	}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	if params.Offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", params.Offset))
	}
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	resp, err := c.do(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[[]pkgkanban.TaskWithDetailsResponse](resp)
}

func (c *Client) CreateTask(projectID string, req pkgkanban.CreateTaskRequest) (string, error) {
	resp, err := c.do(http.MethodPost, "/api/projects/"+projectID+"/tasks", req)
	if err != nil {
		return "", err
	}
	result, err := decodeResponse[pkgkanban.TaskResponse](resp)
	if err != nil {
		return "", err
	}
	return result.ID, nil
}

func (c *Client) CompleteTask(projectID, taskID string, req pkgkanban.CompleteTaskRequest) error {
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/complete", projectID, taskID), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

func (c *Client) BlockTask(projectID, taskID string, req pkgkanban.BlockTaskRequest) error {
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/block", projectID, taskID), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

func (c *Client) MoveTask(projectID, taskID, targetColumn string) error {
	req := pkgkanban.MoveTaskRequest{TargetColumn: targetColumn}
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/move", projectID, taskID), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// Comments

func (c *Client) AddComment(projectID, taskID string, req pkgkanban.CreateCommentRequest) (string, error) {
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/comments", projectID, taskID), req)
	if err != nil {
		return "", err
	}
	result, err := decodeResponse[pkgkanban.CommentResponse](resp)
	if err != nil {
		return "", err
	}
	return result.ID, nil
}

func (c *Client) ListComments(projectID, taskID string, limit, offset int) ([]pkgkanban.CommentResponse, error) {
	u := fmt.Sprintf("/api/projects/%s/tasks/%s/comments?limit=%d&offset=%d", projectID, taskID, limit, offset)
	resp, err := c.do(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[[]pkgkanban.CommentResponse](resp)
}

// ColumnCounts holds the number of tasks in each kanban column
type ColumnCounts struct {
	Todo       int
	InProgress int
	Done       int
	Blocked    int
}

func (c *Client) GetColumnCounts(projectID string) (ColumnCounts, error) {
	columns := []string{"todo", "in_progress", "done", "blocked"}
	var counts ColumnCounts
	for _, col := range columns {
		tasks, err := c.ListTasks(projectID, ListTasksParams{Column: col, Limit: 9999})
		if err != nil {
			return counts, err
		}
		n := len(tasks)
		switch col {
		case "todo":
			counts.Todo = n
		case "in_progress":
			counts.InProgress = n
		case "done":
			counts.Done = n
		case "blocked":
			counts.Blocked = n
		}
	}
	return counts, nil
}

// Columns

func (c *Client) GetColumns(projectID string) ([]pkgkanban.ColumnResponse, error) {
	resp, err := c.do(http.MethodGet, "/api/projects/"+projectID+"/columns", nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[[]pkgkanban.ColumnResponse](resp)
}

// Dependencies

func (c *Client) AddDependency(projectID, taskID, dependsOnTaskID string) error {
	req := pkgkanban.AddDependencyRequest{DependsOnTaskID: dependsOnTaskID}
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/dependencies", projectID, taskID), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// WIPSlotsResult holds WIP slot info returned by the server
type WIPSlotsResult struct {
	WIPLimit   int `json:"wip_limit"`
	InProgress int `json:"in_progress"`
	FreeSlots  int `json:"free_slots"`
}

func (c *Client) GetWIPSlots(projectID string) (*WIPSlotsResult, error) {
	resp, err := c.do(http.MethodGet, "/api/projects/"+projectID+"/wip-slots", nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[WIPSlotsResult](resp)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// WaitForWIPSlot blocks until the SSE stream for the given project emits any event
// (task_moved out of in_progress, task_completed, task_blocked) or the context is cancelled.
// Callers should re-check GetWIPSlots after this returns.
func (c *Client) WaitForWIPSlot(ctx context.Context, projectID string) error {
	return c.WaitForNextTask(ctx, projectID)
}
