package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type ChatUploadClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewChatUploadClient(baseURL string) *ChatUploadClient {
	return &ChatUploadClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

func (c *ChatUploadClient) UploadJSONL(ctx context.Context, token, projectID, featureID, sessionID, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open jsonl file: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	fw, err := w.CreateFormFile("jsonl", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(fw, f); err != nil {
		return fmt.Errorf("copy file contents: %w", err)
	}
	w.Close()

	url := fmt.Sprintf("%s/api/projects/%s/features/%s/chats/%s/upload", c.baseURL, projectID, featureID, sessionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}

	return nil
}
