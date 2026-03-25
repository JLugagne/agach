package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
)

const (
	onboardingEndpoint = "/api/onboarding/complete"
	defaultTimeout     = 30 * time.Second
)

// OnboardingClient handles the daemon onboarding HTTP flow.
type OnboardingClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewOnboardingClient creates a new onboarding client.
func NewOnboardingClient(baseURL string) *OnboardingClient {
	return &OnboardingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// CompleteOnboarding sends the onboarding code to the server and receives tokens.
func (c *OnboardingClient) CompleteOnboarding(ctx context.Context, code, nodeName string) (*domain.OnboardingResult, error) {
	reqBody := map[string]string{
		"code":      code,
		"node_name": nodeName,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + onboardingEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
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
		return nil, c.parseError(resp.StatusCode, body)
	}

	return c.parseSuccess(body)
}

func (c *OnboardingClient) parseSuccess(body []byte) (*domain.OnboardingResult, error) {
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			Node         struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Mode string `json:"mode"`
			} `json:"node"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return &domain.OnboardingResult{
		AccessToken:  resp.Data.AccessToken,
		RefreshToken: resp.Data.RefreshToken,
		NodeID:       resp.Data.Node.ID,
		NodeName:     resp.Data.Node.Name,
		Mode:         resp.Data.Node.Mode,
	}, nil
}

func (c *OnboardingClient) parseError(statusCode int, body []byte) error {
	var resp struct {
		Status string `json:"status"`
		Error  struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("server returned status %d: %s", statusCode, string(body))
	}

	return &OnboardingError{
		StatusCode: statusCode,
		Code:       resp.Error.Code,
		Message:    resp.Error.Message,
	}
}

// OnboardingError represents an error response from the server.
type OnboardingError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *OnboardingError) Error() string {
	return fmt.Sprintf("onboarding failed: %s - %s (HTTP %d)", e.Code, e.Message, e.StatusCode)
}

func (e *OnboardingError) IsCodeNotFound() bool { return e.Code == "CODE_NOT_FOUND" }
func (e *OnboardingError) IsCodeExpired() bool  { return e.Code == "CODE_EXPIRED" }
func (e *OnboardingError) IsCodeUsed() bool     { return e.Code == "CODE_ALREADY_USED" }
