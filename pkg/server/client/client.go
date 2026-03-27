package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	pkgserver "github.com/JLugagne/agach-mcp/pkg/server"
)

const (
	maxResponseBytes = 1 * 1024 * 1024 // 1 MB
	maxSessionIDLen  = 512
	defaultTimeout   = 2 * time.Second
)

var (
	blockedHosts = []string{
		"169.254.169.254",
		"metadata.google.internal",
		"169.254.170.2",
	}

	privateIPv4Ranges = []net.IPNet{
		mustParseCIDR("10.0.0.0/8"),
		mustParseCIDR("172.16.0.0/12"),
		mustParseCIDR("192.168.0.0/16"),
	}

	reInternalError = regexp.MustCompile(`(?i)(pq:|SQLSTATE\s*\w+)`)
)

func mustParseCIDR(s string) net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return *network
}

// Client is an HTTP client for the Kanban REST API
type Client struct {
	baseURL    string
	httpClient *http.Client
	err        error
}

func New(baseURL string) *Client {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil
	}

	if u.User != nil {
		return nil
	}

	host := u.Hostname()
	for _, blocked := range blockedHosts {
		if strings.EqualFold(host, blocked) {
			return nil
		}
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLinkLocalUnicast() {
			c := &Client{
				baseURL:    baseURL,
				httpClient: &http.Client{Timeout: defaultTimeout},
			}
			c.err = fmt.Errorf("base URL host %q is blocked: link-local address not allowed", host)
			return c
		}
		if ip.IsLoopback() && ip.To4() == nil {
			c := &Client{
				baseURL:    baseURL,
				httpClient: &http.Client{Timeout: defaultTimeout},
			}
			c.err = fmt.Errorf("base URL host %q is blocked: IPv6 loopback not allowed", host)
			return c
		}
		for _, network := range privateIPv4Ranges {
			if network.Contains(ip) {
				c := &Client{
					baseURL:    baseURL,
					httpClient: &http.Client{Timeout: defaultTimeout},
				}
				c.err = fmt.Errorf("base URL host %q is blocked: private network address not allowed", host)
				return c
			}
		}
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// escapePath encodes a path segment so that special characters including slashes
// are preserved as percent-encoded sequences in the server's r.URL.Path.
func escapePath(s string) string {
	escaped := url.PathEscape(s)
	return strings.ReplaceAll(escaped, "%", "%25")
}

type NextTaskResult struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Role      string `json:"role"`
	ProjectID string `json:"project_id"`
	SessionID string `json:"session_id"`
}

type ListTasksParams struct {
	Column       string
	AssignedRole string
	Tag          string
	Priority     string
	Search       string
	Limit        int
	Offset       int
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
	if c.err != nil {
		return nil, c.err
	}

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
	limited := io.LimitReader(resp.Body, maxResponseBytes)
	var zero T
	var result apiResponse[T]
	if err := json.NewDecoder(limited).Decode(&result); err != nil {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return zero, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
		}
		return zero, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if result.Error != nil {
			msg := sanitizeErrorMessage(fmt.Sprintf("%s: %s", result.Error.Code, result.Error.Message))
			return zero, fmt.Errorf("server error (HTTP %d): %s", resp.StatusCode, msg)
		}
		return zero, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}
	if result.Error != nil {
		msg := sanitizeErrorMessage(fmt.Sprintf("%s: %s", result.Error.Code, result.Error.Message))
		return zero, fmt.Errorf("%s", msg)
	}
	return result.Data, nil
}

func sanitizeErrorMessage(msg string) string {
	if reInternalError.MatchString(msg) {
		return "internal server error"
	}
	return msg
}

// Projects

func (c *Client) ListProjects() ([]pkgserver.ProjectResponse, error) {
	resp, err := c.do(http.MethodGet, "/api/projects", nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[[]pkgserver.ProjectResponse](resp)
}

func (c *Client) GetProject(id string) (*pkgserver.ProjectResponse, error) {
	resp, err := c.do(http.MethodGet, "/api/projects/"+escapePath(id), nil)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[pkgserver.ProjectResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateProject(req pkgserver.CreateProjectRequest) (*pkgserver.ProjectResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/projects", req)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[pkgserver.ProjectResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Per-project roles

func (c *Client) ListProjectRoles(projectID string) ([]pkgserver.AgentResponse, error) {
	resp, err := c.do(http.MethodGet, "/api/projects/"+url.PathEscape(projectID)+"/agents", nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[[]pkgserver.AgentResponse](resp)
}

func (c *Client) CreateProjectAgent(projectID string, req pkgserver.CreateAgentRequest) (*pkgserver.AgentResponse, error) {
	resp, err := c.do(http.MethodPost, "/api/projects/"+url.PathEscape(projectID)+"/agents", req)
	if err != nil {
		return nil, err
	}
	result, err := decodeResponse[pkgserver.AgentResponse](resp)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateProjectAgent(projectID, slug string, req pkgserver.UpdateAgentRequest) error {
	resp, err := c.do(http.MethodPatch, "/api/projects/"+url.PathEscape(projectID)+"/agents/"+url.PathEscape(slug), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

func (c *Client) DeleteProjectAgent(projectID, slug string) error {
	resp, err := c.do(http.MethodDelete, "/api/projects/"+url.PathEscape(projectID)+"/agents/"+url.PathEscape(slug), nil)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// Tasks

func (c *Client) GetNextTasks(projectID string, count int, role string, subProjectID *string, includeSubProjects bool) ([]NextTaskResult, error) {
	u := fmt.Sprintf("/api/projects/%s/next-tasks?count=%d", url.PathEscape(projectID), count)
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
	if c.err != nil {
		return c.err
	}
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
	if len(sessionID) > maxSessionIDLen {
		return fmt.Errorf("session_id exceeds maximum length of %d characters", maxSessionIDLen)
	}
	req := map[string]string{"session_id": sessionID}
	resp, err := c.do(http.MethodPatch, fmt.Sprintf("/api/projects/%s/tasks/%s/session", url.PathEscape(projectID), url.PathEscape(taskID)), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// UpdateTask updates a task via PATCH
func (c *Client) UpdateTask(projectID, taskID string, req pkgserver.UpdateTaskRequest) error {
	resp, err := c.do(http.MethodPatch, fmt.Sprintf("/api/projects/%s/tasks/%s", url.PathEscape(projectID), url.PathEscape(taskID)), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

func (c *Client) ListTasks(projectID string, params ListTasksParams) ([]pkgserver.TaskWithDetailsResponse, error) {
	u := fmt.Sprintf("/api/projects/%s/tasks", url.PathEscape(projectID))
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
	return decodeResponse[[]pkgserver.TaskWithDetailsResponse](resp)
}

func (c *Client) CreateTask(projectID string, req pkgserver.CreateTaskRequest) (string, error) {
	resp, err := c.do(http.MethodPost, "/api/projects/"+url.PathEscape(projectID)+"/tasks", req)
	if err != nil {
		return "", err
	}
	result, err := decodeResponse[pkgserver.TaskResponse](resp)
	if err != nil {
		return "", err
	}
	return result.ID, nil
}

func (c *Client) CompleteTask(projectID, taskID string, req pkgserver.CompleteTaskRequest) error {
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/complete", url.PathEscape(projectID), url.PathEscape(taskID)), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

func (c *Client) BlockTask(projectID, taskID string, req pkgserver.BlockTaskRequest) error {
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/block", url.PathEscape(projectID), url.PathEscape(taskID)), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

func (c *Client) MoveTask(projectID, taskID, targetColumn string) error {
	req := pkgserver.MoveTaskRequest{TargetColumn: targetColumn}
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/move", url.PathEscape(projectID), url.PathEscape(taskID)), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// Comments

func (c *Client) AddComment(projectID, taskID string, req pkgserver.CreateCommentRequest) (string, error) {
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/comments", url.PathEscape(projectID), url.PathEscape(taskID)), req)
	if err != nil {
		return "", err
	}
	result, err := decodeResponse[pkgserver.CommentResponse](resp)
	if err != nil {
		return "", err
	}
	return result.ID, nil
}

func (c *Client) ListComments(projectID, taskID string, limit, offset int) ([]pkgserver.CommentResponse, error) {
	u := fmt.Sprintf("/api/projects/%s/tasks/%s/comments?limit=%d&offset=%d", url.PathEscape(projectID), url.PathEscape(taskID), limit, offset)
	resp, err := c.do(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[[]pkgserver.CommentResponse](resp)
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
		tasks, err := c.ListTasks(projectID, ListTasksParams{Column: col, Limit: 100})
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

func (c *Client) GetColumns(projectID string) ([]pkgserver.ColumnResponse, error) {
	resp, err := c.do(http.MethodGet, "/api/projects/"+escapePath(projectID)+"/columns", nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[[]pkgserver.ColumnResponse](resp)
}

// Dependencies

func (c *Client) AddDependency(projectID, taskID, dependsOnTaskID string) error {
	req := pkgserver.AddDependencyRequest{DependsOnTaskID: dependsOnTaskID}
	resp, err := c.do(http.MethodPost, fmt.Sprintf("/api/projects/%s/tasks/%s/dependencies", url.PathEscape(projectID), url.PathEscape(taskID)), req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}
