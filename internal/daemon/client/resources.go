package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ResourceClient downloads resources from the server.
type ResourceClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewResourceClient(baseURL string) *ResourceClient {
	return &ResourceClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

func (c *ResourceClient) DownloadResource(ctx context.Context, token, name string) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/api/resources/%s", c.baseURL, url.PathEscape(name))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("download resource %q: HTTP %d: %s", name, resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read resource body: %w", err)
	}
	return data, nil
}
