package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
)

// AgentDownloadClient downloads the multipart agent+skill bundle from the server.
type AgentDownloadClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAgentDownloadClient(baseURL string) *AgentDownloadClient {
	return &AgentDownloadClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

func (c *AgentDownloadClient) DownloadAgents(ctx context.Context, token, projectID string) ([]domain.AgentFile, error) {
	url := fmt.Sprintf("%s/api/projects/%s/agents/download", c.baseURL, projectID)
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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("download agents failed: HTTP %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return nil, fmt.Errorf("unexpected content-type: %s", contentType)
	}

	reader := multipart.NewReader(resp.Body, params["boundary"])
	var files []domain.AgentFile

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read multipart part: %w", err)
		}

		content, err := io.ReadAll(part)
		if err != nil {
			return nil, fmt.Errorf("read part content: %w", err)
		}

		_, partParams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		filename := partParams["filename"]

		// The last part is the manifest — parse it to get checksums
		if filename == "manifest.json" {
			var manifest []struct {
				Path   string `json:"path"`
				SHA256 string `json:"sha256"`
			}
			if err := json.Unmarshal(content, &manifest); err != nil {
				return nil, fmt.Errorf("parse manifest: %w", err)
			}
			// Attach checksums to already-collected files
			checksumMap := make(map[string]string, len(manifest))
			for _, m := range manifest {
				checksumMap[m.Path] = m.SHA256
			}
			for i := range files {
				files[i].SHA256 = checksumMap[files[i].Path]
			}
			continue
		}

		files = append(files, domain.AgentFile{
			Path:    filename,
			Content: content,
		})
	}

	return files, nil
}
