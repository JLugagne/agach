package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/sidecar/domain"
)

const maxResponseBytes = 10 * 1024 * 1024 // 10 MB

// Client implements domain.ServerAPI by making HTTP requests through a Unix socket.
type Client struct {
	httpClient *http.Client
	apiKey     string
}

// New creates a Client that connects to the daemon proxy via Unix socket.
func New(socketPath, apiKey string) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &Client{
		httpClient: &http.Client{Transport: transport},
		apiKey:     apiKey,
	}
}

type apiResponse[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	// Use http://localhost as base — the transport dials the Unix socket
	req, err := http.NewRequestWithContext(ctx, method, "http://localhost"+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	return c.httpClient.Do(req)
}

func decodeResponse[T any](resp *http.Response) (T, error) {
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, maxResponseBytes)
	var result apiResponse[T]
	if err := json.NewDecoder(limited).Decode(&result); err != nil {
		var zero T
		return zero, fmt.Errorf("decode response: %w", err)
	}
	if result.Error != nil {
		var zero T
		return zero, fmt.Errorf("%s: %s", result.Error.Code, result.Error.Message)
	}
	return result.Data, nil
}

// CreateTask creates a task via POST /api/projects/_/tasks.
// The "_" placeholder is replaced by the proxy with the real project ID.
func (c *Client) CreateTask(ctx context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/projects/_/tasks", req)
	if err != nil {
		return domain.CreateTaskResponse{}, err
	}
	type taskResp struct {
		ID string `json:"id"`
	}
	result, err := decodeResponse[taskResp](resp)
	if err != nil {
		return domain.CreateTaskResponse{}, err
	}
	return domain.CreateTaskResponse{ID: result.ID}, nil
}

// AddDependency adds a dependency via POST /api/projects/_/tasks/{taskID}/dependencies.
func (c *Client) AddDependency(ctx context.Context, taskID, dependsOnTaskID string) error {
	body := map[string]string{"depends_on_task_id": dependsOnTaskID}
	resp, err := c.do(ctx, http.MethodPost, "/api/projects/_/tasks/"+taskID+"/dependencies", body)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// CompleteTask completes a task via POST /api/projects/_/tasks/{taskID}/complete.
func (c *Client) CompleteTask(ctx context.Context, taskID string, req domain.CompleteTaskRequest) error {
	resp, err := c.do(ctx, http.MethodPost, "/api/projects/_/tasks/"+taskID+"/complete", req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// MoveTask moves a task via POST /api/projects/_/tasks/{taskID}/move.
func (c *Client) MoveTask(ctx context.Context, taskID, targetColumn string) error {
	body := map[string]string{"target_column": targetColumn}
	resp, err := c.do(ctx, http.MethodPost, "/api/projects/_/tasks/"+taskID+"/move", body)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// BlockTask blocks a task via POST /api/projects/_/tasks/{taskID}/block.
func (c *Client) BlockTask(ctx context.Context, taskID string, req domain.BlockTaskRequest) error {
	resp, err := c.do(ctx, http.MethodPost, "/api/projects/_/tasks/"+taskID+"/block", req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// RequestWontDo requests won't-do via POST /api/projects/_/tasks/{taskID}/wont-do.
func (c *Client) RequestWontDo(ctx context.Context, taskID string, req domain.WontDoRequest) error {
	resp, err := c.do(ctx, http.MethodPost, "/api/projects/_/tasks/"+taskID+"/wont-do", req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}

// UpdateFeatureChangelogs updates changelogs via PATCH /api/projects/_/features/_/changelogs.
// The proxy replaces both "_" placeholders with real project and feature IDs.
func (c *Client) UpdateFeatureChangelogs(ctx context.Context, req domain.FeatureChangelogsRequest) error {
	resp, err := c.do(ctx, http.MethodPatch, "/api/projects/_/features/_/changelogs", req)
	if err != nil {
		return err
	}
	_, err = decodeResponse[map[string]string](resp)
	return err
}
