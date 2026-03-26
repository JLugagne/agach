package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
)

type DockerfileClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewDockerfileClient(baseURL string) *DockerfileClient {
	return &DockerfileClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *DockerfileClient) GetDockerfileBySlug(ctx context.Context, token, slug string) (*domain.DockerfileContent, error) {
	url := fmt.Sprintf("%s/api/dockerfiles/by-slug/%s", c.baseURL, slug)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch dockerfile failed with status %d", resp.StatusCode)
	}

	var envelope struct {
		Data struct {
			Slug    string `json:"slug"`
			Version string `json:"version"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &domain.DockerfileContent{
		Slug:    envelope.Data.Slug,
		Version: envelope.Data.Version,
		Content: envelope.Data.Content,
	}, nil
}
