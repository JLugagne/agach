package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
)

type ProjectClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewProjectClient(baseURL string) *ProjectClient {
	return &ProjectClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *ProjectClient) GetProject(ctx context.Context, token, projectID string) (*domain.ProjectInfo, error) {
	url := c.baseURL + "/api/projects/" + projectID
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get project failed: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string             `json:"status"`
		Data   domain.ProjectInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("unexpected status: %s", result.Status)
	}

	return &result.Data, nil
}
