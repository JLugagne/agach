package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const daemonRefreshEndpoint = "/api/daemon/refresh"

// AuthClient handles daemon authentication HTTP flows.
type AuthClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAuthClient creates a new auth client.
func NewAuthClient(baseURL string) *AuthClient {
	return &AuthClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// RefreshDaemonToken exchanges a refresh token for a new access token.
func (c *AuthClient) RefreshDaemonToken(ctx context.Context, nodeID, refreshToken string) (string, error) {
	reqBody := map[string]string{
		"node_id":       nodeID,
		"refresh_token": refreshToken,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + daemonRefreshEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("refresh failed: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if result.Status != "success" {
		return "", fmt.Errorf("unexpected status: %s", result.Status)
	}

	return result.Data.AccessToken, nil
}
