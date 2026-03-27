package e2eapi

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chatSession matches the ChatSessionResponse JSON shape.
type chatSession struct {
	ID               string  `json:"id"`
	FeatureID        string  `json:"feature_id"`
	ProjectID        string  `json:"project_id"`
	NodeID           string  `json:"node_id,omitempty"`
	State            string  `json:"state"`
	ClaudeSessionID  string  `json:"claude_session_id,omitempty"`
	JSONLPath        string  `json:"jsonl_path,omitempty"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	CacheReadTokens  int     `json:"cache_read_tokens"`
	CacheWriteTokens int     `json:"cache_write_tokens"`
	Model            string  `json:"model,omitempty"`
	CreatedAt        string  `json:"created_at"`
	EndedAt          *string `json:"ended_at,omitempty"`
	UpdatedAt        string  `json:"updated_at"`
}

// chatEndResponse matches the EndSession success payload.
type chatEndResponse struct {
	Message string `json:"message"`
}

// chatStatsResponse matches the UpdateStats success payload.
type chatStatsResponse struct {
	Message string `json:"message"`
}

// createChatFixtures creates a project and feature and returns their IDs.
func createChatFixtures(t *testing.T, token string) (projectID, featureID string) {
	t.Helper()

	type proj struct {
		ID string `json:"id"`
	}
	p := createAndDecode[proj](t, "/api/projects", token, map[string]any{
		"name": "Chat Test Project " + t.Name(),
	})

	type feat struct {
		ID string `json:"id"`
	}
	f := createAndDecode[feat](t,
		fmt.Sprintf("/api/projects/%s/features", p.ID), token,
		map[string]any{"name": "Chat Feature " + t.Name()})

	return p.ID, f.ID
}

// chatBasePath returns the base URL for chat endpoints under a feature.
func chatBasePath(projectID, featureID string) string {
	return fmt.Sprintf("/api/projects/%s/features/%s/chats", projectID, featureID)
}

// skipIfChatsDisabled sends a probe request and skips the test if chat
// endpoints are not registered (404).
func skipIfChatsDisabled(t *testing.T, projectID, featureID, token string) {
	t.Helper()
	resp := doAuth(t, "GET", chatBasePath(projectID, featureID), token, nil)
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		t.Skip("chat service not enabled")
	}
	resp.Body.Close()
}

// ---------- 1. Start and End -------------------------------------------------

func TestChats_StartAndEnd(t *testing.T) {
	token := adminToken(t)
	projectID, featureID := createChatFixtures(t, token)
	skipIfChatsDisabled(t, projectID, featureID, token)

	base := chatBasePath(projectID, featureID)

	// Start a session.
	session := createAndDecode[chatSession](t, base, token, map[string]any{})
	require.NotEmpty(t, session.ID, "session ID should be set")
	require.Equal(t, featureID, session.FeatureID)
	require.Equal(t, projectID, session.ProjectID)
	require.Equal(t, "active", session.State)

	// End the session.
	endPath := fmt.Sprintf("%s/%s/end", base, session.ID)
	endResp := createAndDecode[chatEndResponse](t, endPath, token, nil)
	require.Equal(t, "chat session ended", endResp.Message)

	// Verify state changed.
	got := getAndDecode[chatSession](t,
		fmt.Sprintf("%s/%s", base, session.ID), token)
	require.Equal(t, "ended", got.State)
	require.NotNil(t, got.EndedAt, "ended_at should be set after ending")
}

// ---------- 2. Update Stats --------------------------------------------------

func TestChats_UpdateStats(t *testing.T) {
	token := adminToken(t)
	projectID, featureID := createChatFixtures(t, token)
	skipIfChatsDisabled(t, projectID, featureID, token)

	base := chatBasePath(projectID, featureID)

	// Start a session.
	session := createAndDecode[chatSession](t, base, token, map[string]any{})
	require.NotEmpty(t, session.ID)

	// Update stats via PUT.
	statsPath := fmt.Sprintf("%s/%s/stats", base, session.ID)
	statsResp := doAuth(t, "PUT", statsPath, token, map[string]any{
		"input_tokens":       1500,
		"output_tokens":      3000,
		"cache_read_tokens":  200,
		"cache_write_tokens": 100,
		"model":              "claude-sonnet-4-20250514",
	})
	requireStatus(t, statsResp, http.StatusOK)
	sr := decode[chatStatsResponse](t, statsResp)
	require.Equal(t, "stats updated", sr.Message)

	// Get the session and verify stats.
	got := getAndDecode[chatSession](t,
		fmt.Sprintf("%s/%s", base, session.ID), token)
	require.Equal(t, 1500, got.InputTokens)
	require.Equal(t, 3000, got.OutputTokens)
	require.Equal(t, 200, got.CacheReadTokens)
	require.Equal(t, 100, got.CacheWriteTokens)
	require.Equal(t, "claude-sonnet-4-20250514", got.Model)
}

// ---------- 3. List and Get --------------------------------------------------

func TestChats_ListAndGet(t *testing.T) {
	token := adminToken(t)
	projectID, featureID := createChatFixtures(t, token)
	skipIfChatsDisabled(t, projectID, featureID, token)

	base := chatBasePath(projectID, featureID)

	// Start two sessions.
	s1 := createAndDecode[chatSession](t, base, token, map[string]any{})
	s2 := createAndDecode[chatSession](t, base, token, map[string]any{})
	require.NotEmpty(t, s1.ID)
	require.NotEmpty(t, s2.ID)
	require.NotEqual(t, s1.ID, s2.ID, "sessions should have distinct IDs")

	// List sessions for this feature.
	list := getAndDecode[[]chatSession](t, base, token)
	require.GreaterOrEqual(t, len(list), 2, "list should contain at least 2 sessions")

	ids := make(map[string]bool, len(list))
	for _, s := range list {
		ids[s.ID] = true
	}
	require.True(t, ids[s1.ID], "session 1 should appear in list")
	require.True(t, ids[s2.ID], "session 2 should appear in list")

	// Get single session by ID.
	got := getAndDecode[chatSession](t,
		fmt.Sprintf("%s/%s", base, s1.ID), token)
	require.Equal(t, s1.ID, got.ID)
	require.Equal(t, featureID, got.FeatureID)
	require.Equal(t, "active", got.State)
}

// ---------- 4. Upload and Download JSONL -------------------------------------

func TestChats_UploadAndDownload(t *testing.T) {
	token := adminToken(t)
	projectID, featureID := createChatFixtures(t, token)
	skipIfChatsDisabled(t, projectID, featureID, token)

	base := chatBasePath(projectID, featureID)

	// Start a session.
	session := createAndDecode[chatSession](t, base, token, map[string]any{})
	require.NotEmpty(t, session.ID)

	// Build a small JSONL payload.
	jsonlContent := `{"role":"user","content":"hello"}` + "\n" + `{"role":"assistant","content":"hi"}` + "\n"

	// Upload via multipart form.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("jsonl", "session.jsonl")
	require.NoError(t, err)
	_, err = io.Copy(part, strings.NewReader(jsonlContent))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	uploadURL := apiURL(fmt.Sprintf("%s/%s/upload", base, session.ID))
	req, err := http.NewRequest("POST", uploadURL, &buf)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	requireStatus(t, resp, http.StatusOK)

	type uploadResp struct {
		Message string `json:"message"`
		Path    string `json:"path"`
	}
	ur := decode[uploadResp](t, resp)
	assert.Contains(t, ur.Message, "uploaded")
	require.NotEmpty(t, ur.Path, "upload should return a relative path")

	// Download the JSONL.
	downloadURL := fmt.Sprintf("%s/%s/download", base, session.ID)
	downloadReq := authReq(t, "GET", apiURL(downloadURL), token, nil)
	resp, err = http.DefaultClient.Do(downloadReq)
	require.NoError(t, err)
	requireStatus(t, resp, http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, jsonlContent, string(body), "downloaded content should match uploaded content")
	assert.Contains(t, resp.Header.Get("Content-Type"), "ndjson",
		"Content-Type should be application/x-ndjson")
}

func TestChats_Download_NoUpload(t *testing.T) {
	token := adminToken(t)
	projectID, featureID := createChatFixtures(t, token)
	skipIfChatsDisabled(t, projectID, featureID, token)

	base := chatBasePath(projectID, featureID)

	// Start a session but do NOT upload.
	session := createAndDecode[chatSession](t, base, token, map[string]any{})
	require.NotEmpty(t, session.ID)

	// Download should fail — no JSONL associated.
	downloadURL := fmt.Sprintf("%s/%s/download", base, session.ID)
	resp := doAuth(t, "GET", downloadURL, token, nil)
	require.NotEqual(t, http.StatusOK, resp.StatusCode,
		"download without prior upload should fail")
	resp.Body.Close()
}

func TestChats_Upload_InvalidExtension(t *testing.T) {
	token := adminToken(t)
	projectID, featureID := createChatFixtures(t, token)
	skipIfChatsDisabled(t, projectID, featureID, token)

	base := chatBasePath(projectID, featureID)

	session := createAndDecode[chatSession](t, base, token, map[string]any{})
	require.NotEmpty(t, session.ID)

	// Upload a file with .txt extension — should be rejected.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("jsonl", "session.txt")
	require.NoError(t, err)
	_, _ = part.Write([]byte("not jsonl"))
	require.NoError(t, writer.Close())

	uploadURL := apiURL(fmt.Sprintf("%s/%s/upload", base, session.ID))
	req, err := http.NewRequest("POST", uploadURL, &buf)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NotEqual(t, http.StatusOK, resp.StatusCode,
		"upload with .txt extension should be rejected")
	resp.Body.Close()
}
