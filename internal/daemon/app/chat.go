package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/domain"
	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const sessionTTLWarning = 25 * time.Minute

type ChatSession struct {
	ID              string
	FeatureID       string
	ProjectID       string
	ClaudeSessionID string
	WorktreePath    string
	StartedAt       time.Time
	LastActivity    time.Time

	claudeCmd        *exec.Cmd
	claudeStdin      io.WriteCloser
	claudeStdout     *bufio.Reader
	claudeStderr     *bytes.Buffer
	jsonlFile        *os.File
	jsonlPath        string
	proxy            *SidecarProxy
	socketPath       string
	apiKey           string
	MessageCount     int
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
	TotalCost        float64
	Model            string
}

type ChatManager struct {
	sessions        map[string]*ChatSession
	mu              sync.RWMutex
	logger          *logrus.Logger
	gitService      *GitService
	projectClient   domain.ProjectFetcher
	uploadClient    domain.ChatUploader
	agentDownloader domain.AgentDownloader
	token           string
	sendMessage     func(daemonws.Message)
	stopCh          chan struct{}
	ttl             time.Duration
	serverURL       string
	getToken        func() string
	refreshToken    func() error
	resourceCache   *ResourceCache
}

func NewChatManager(
	logger *logrus.Logger,
	gitService *GitService,
	projectClient domain.ProjectFetcher,
	uploadClient domain.ChatUploader,
	agentDownloader domain.AgentDownloader,
	token string,
	sendMessage func(daemonws.Message),
	serverURL string,
	getToken func() string,
	refreshToken func() error,
	resourceCache *ResourceCache,
) *ChatManager {
	m := &ChatManager{
		sessions:        make(map[string]*ChatSession),
		logger:          logger,
		gitService:      gitService,
		projectClient:   projectClient,
		uploadClient:    uploadClient,
		agentDownloader: agentDownloader,
		token:           token,
		sendMessage:     sendMessage,
		stopCh:          make(chan struct{}),
		ttl:             30 * time.Minute,
		serverURL:       serverURL,
		getToken:        getToken,
		refreshToken:    refreshToken,
		resourceCache:   resourceCache,
	}
	go m.ttlChecker()
	go m.statsBroadcaster()
	return m
}

func (m *ChatManager) startClaudeProcess(session *ChatSession, resumeSessionID, agentName string) error {
	jsonlPath := fmt.Sprintf("/tmp/agach-chat-%s.jsonl", session.ID)
	f, err := os.Create(jsonlPath)
	if err != nil {
		return fmt.Errorf("create jsonl file: %w", err)
	}
	session.jsonlFile = f
	session.jsonlPath = jsonlPath

	args := []string{"--print", "--output-format", "stream-json", "--input-format", "stream-json", "--verbose"}
	if agentName != "" {
		args = append(args, "--agent", agentName)
	}
	if resumeSessionID != "" {
		args = append(args, "--resume", resumeSessionID)
	}

	cmd := exec.Command("claude", args...)
	cmd.Dir = session.WorktreePath
	cmd.Env = append(os.Environ(),
		"AGACH_PROXY="+session.socketPath,
		"AGACH_PROXY_KEY="+session.apiKey,
		"AGACH_FEATURE_ID="+session.FeatureID,
	)

	session.claudeStderr = &bytes.Buffer{}
	cmd.Stderr = session.claudeStderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		f.Close()
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		f.Close()
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		f.Close()
		return fmt.Errorf("start claude: %w", err)
	}

	session.claudeCmd = cmd
	session.claudeStdin = stdin
	session.claudeStdout = bufio.NewReader(stdout)

	go m.readClaudeOutput(session)
	return nil
}

func (m *ChatManager) readClaudeOutput(session *ChatSession) {
	buf := make([]byte, 1024*1024)
	scanner := bufio.NewScanner(session.claudeStdout)
	scanner.Buffer(buf, len(buf))

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if session.jsonlFile != nil {
			session.jsonlFile.Write(append(line, '\n'))
		}
		m.handleClaudeMessage(session, line)
	}

	if err := session.claudeCmd.Wait(); err != nil {
		stderr := ""
		if session.claudeStderr != nil {
			stderr = session.claudeStderr.String()
		}
		m.logger.WithError(err).WithFields(logrus.Fields{
			"session_id": session.ID,
			"stderr":     stderr,
		}).Warn("claude process exited")
	}
}

func (m *ChatManager) handleClaudeMessage(session *ChatSession, line []byte) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &envelope); err != nil {
		return
	}

	if envelope.Type == "result" {
		var result struct {
			Usage struct {
				InputTokens      int `json:"input_tokens"`
				OutputTokens     int `json:"output_tokens"`
				CacheReadTokens  int `json:"cache_read_input_tokens"`
				CacheWriteTokens int `json:"cache_creation_input_tokens"`
			} `json:"usage"`
			TotalCostUSD float64 `json:"total_cost_usd"`
			Model        string  `json:"model"`
			SessionID    string  `json:"session_id"`
		}
		if err := json.Unmarshal(line, &result); err == nil {
			m.mu.Lock()
			session.InputTokens += result.Usage.InputTokens
			session.OutputTokens += result.Usage.OutputTokens
			session.CacheReadTokens += result.Usage.CacheReadTokens
			session.CacheWriteTokens += result.Usage.CacheWriteTokens
			session.TotalCost += result.TotalCostUSD
			if result.Model != "" {
				session.Model = result.Model
			}
			if result.SessionID != "" {
				session.ClaudeSessionID = result.SessionID
			}
			session.MessageCount++
			session.LastActivity = time.Now()
			stats := domain.ChatStats{
				InputTokens:      session.InputTokens,
				OutputTokens:     session.OutputTokens,
				CacheReadTokens:  session.CacheReadTokens,
				CacheWriteTokens: session.CacheWriteTokens,
				Model:            session.Model,
			}
			projectID := session.ProjectID
			featureID := session.FeatureID
			sessionID := session.ID
			m.mu.Unlock()

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := m.uploadClient.UpdateStats(ctx, m.token, projectID, featureID, sessionID, stats); err != nil {
					m.logger.WithError(err).WithField("session_id", sessionID).Warn("failed to persist stats")
				}
			}()
		}
	} else if envelope.Type == "assistant" {
		m.mu.Lock()
		session.MessageCount++
		session.LastActivity = time.Now()
		m.mu.Unlock()
	}

	// Signal turn complete on result, forward text on assistant
	if envelope.Type == "result" {
		content, _ := json.Marshal(map[string]string{"text": ""})
		event := daemonws.ChatMessageEvent{
			SessionID:   session.ID,
			MessageType: "result",
			Content:     content,
			IsFinal:     true,
		}
		payload, _ := json.Marshal(event)
		m.sendMessage(daemonws.Message{
			Type:    daemonws.TypeChatMessage,
			Payload: payload,
		})
		return
	}
	if envelope.Type != "assistant" {
		return
	}

	var msg struct {
		Message struct {
			Model   string `json:"model"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
	}
	var textContent string
	if json.Unmarshal(line, &msg) == nil {
		for _, block := range msg.Message.Content {
			if block.Type == "text" {
				textContent += block.Text
			}
		}
		if msg.Message.Model != "" {
			m.mu.Lock()
			session.Model = msg.Message.Model
			m.mu.Unlock()
		}
	}
	if textContent == "" {
		return
	}

	content, _ := json.Marshal(map[string]string{"text": textContent})
	event := daemonws.ChatMessageEvent{
		SessionID:   session.ID,
		MessageType: envelope.Type,
		Content:     content,
	}
	payload, _ := json.Marshal(event)
	m.sendMessage(daemonws.Message{
		Type:    daemonws.TypeChatMessage,
		Payload: payload,
	})
}

func (m *ChatManager) stopClaudeProcess(session *ChatSession) {
	if session.claudeStdin != nil {
		session.claudeStdin.Close()
	}
	if session.claudeCmd != nil && session.claudeCmd.Process != nil {
		session.claudeCmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() {
			session.claudeCmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			session.claudeCmd.Process.Kill()
		}
	}
	if session.jsonlFile != nil {
		session.jsonlFile.Close()
		session.jsonlFile = nil
	}
}

func (m *ChatManager) StartSession(ctx context.Context, requestID string, req daemonws.ChatStartRequest) {
	project, err := m.projectClient.GetProject(ctx, m.token, req.ProjectID)
	if err != nil {
		m.logger.WithError(err).WithField("project_id", req.ProjectID).Error("fetch project")
		m.sendError(requestID, "", "failed to fetch project: "+err.Error())
		return
	}

	if project.DefaultRole == "" {
		m.sendError(requestID, "", "project must have a default agent set before starting a chat session")
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	worktreePath, err := m.gitService.CreateSessionWorktree(ctx, req.ProjectID, sessionID, project.GitURL, "")
	if err != nil {
		m.logger.WithError(err).WithField("project_id", req.ProjectID).Error("create session worktree")
		m.sendError(requestID, "", "failed to prepare worktree: "+err.Error())
		return
	}

	// Download agents and skills, write them into the worktree
	agentName, err := m.downloadAndWriteAgents(ctx, req.ProjectID, worktreePath, project.DefaultRole)
	if err != nil {
		m.logger.WithError(err).WithField("project_id", req.ProjectID).Error("download agents")
		m.sendError(requestID, "", "failed to download agents: "+err.Error())
		return
	}

	// Create sidecar proxy
	socketPath := fmt.Sprintf("/tmp/agach-sidecar-%s.sock", sessionID)
	apiKey, err := generateAPIKey()
	if err != nil {
		m.logger.WithError(err).Error("generate api key")
		m.sendError(requestID, "", "failed to generate api key: "+err.Error())
		return
	}

	proxy := NewSidecarProxy(
		socketPath, apiKey, req.ProjectID, req.FeatureID,
		m.serverURL, m.getToken, m.refreshToken, m.logger,
	)
	if err := proxy.Start(ctx); err != nil {
		m.logger.WithError(err).Error("start sidecar proxy")
		m.sendError(requestID, "", "failed to start sidecar proxy: "+err.Error())
		return
	}

	now := time.Now()

	session := &ChatSession{
		ID:           sessionID,
		FeatureID:    req.FeatureID,
		ProjectID:    req.ProjectID,
		WorktreePath: worktreePath,
		StartedAt:    now,
		LastActivity: now,
		proxy:        proxy,
		socketPath:   socketPath,
		apiKey:       apiKey,
	}

	if err := m.startClaudeProcess(session, req.ResumeSessionID, agentName); err != nil {
		m.logger.WithError(err).WithField("session_id", sessionID).Error("start claude process")
		m.sendError(requestID, sessionID, "failed to start claude: "+err.Error())
		return
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	// Send the initial message automatically for new sessions
	if req.ResumeSessionID == "" && req.InitialMessage != "" {
		if err := m.SendMessage(sessionID, req.InitialMessage); err != nil {
			m.logger.WithError(err).WithField("session_id", sessionID).Warn("failed to send initial message")
		}
	}

	m.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"project_id": req.ProjectID,
		"feature_id": req.FeatureID,
		"worktree":   worktreePath,
	}).Info("Chat session started")

	resp := daemonws.ChatStartResponse{
		SessionID:    sessionID,
		WorktreePath: worktreePath,
	}
	payload, _ := json.Marshal(resp)
	m.sendMessage(daemonws.Message{
		Type:      daemonws.TypeChatStart,
		RequestID: requestID,
		Payload:   payload,
	})
}

// downloadAndWriteAgents fetches the agent bundle from the server and writes
// each file into the worktree under .claude/agents/ and .claude/skills/.
// It returns the agent name (from frontmatter) to pass to `claude --agent <name>`.
func (m *ChatManager) downloadAndWriteAgents(ctx context.Context, projectID, worktreePath, defaultRole string) (string, error) {
	files, err := m.agentDownloader.DownloadAgents(ctx, m.token, projectID)
	if err != nil {
		return "", fmt.Errorf("download agent bundle: %w", err)
	}

	var agentName string
	defaultAgentPath := fmt.Sprintf("agents/%s.md", defaultRole)

	for _, f := range files {
		dest := filepath.Join(worktreePath, ".claude", f.Path)

		// Extract agent name from the default agent's frontmatter.
		if f.Path == defaultAgentPath {
			agentName = extractFrontmatterName(f.Content)
		}

		// Skip files that already exist in the repo — keep the user's version.
		if _, err := os.Stat(dest); err == nil {
			m.logger.WithField("path", f.Path).Debug("agent file already exists in repo, skipping")
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return "", fmt.Errorf("create directory for %s: %w", f.Path, err)
		}
		if err := os.WriteFile(dest, f.Content, 0644); err != nil {
			return "", fmt.Errorf("write %s: %w", f.Path, err)
		}
	}

	if agentName == "" {
		return "", fmt.Errorf("default agent %q not found in downloaded bundle", defaultRole)
	}

	return agentName, nil
}

// extractFrontmatterName parses the "name:" value from YAML frontmatter.
func extractFrontmatterName(content []byte) string {
	const prefix = "name: "
	for _, line := range strings.SplitN(string(content), "\n", 20) {
		if line == "---" {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(line[len(prefix):])
		}
	}
	return ""
}

func (m *ChatManager) GetSession(sessionID string) *ChatSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionID]
}

func (m *ChatManager) UpdateActivity(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[sessionID]; ok {
		s.LastActivity = time.Now()
	}
}

func (m *ChatManager) EndSession(sessionID, reason string) {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()

	if !ok {
		return
	}

	m.stopClaudeProcess(session)

	if session.proxy != nil {
		session.proxy.Stop()
	}

	if session.jsonlPath != "" {
		if _, err := os.Stat(session.jsonlPath); err == nil {
			uploadCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			if err := m.uploadClient.UploadJSONL(uploadCtx, m.token, session.ProjectID, session.FeatureID, sessionID, session.jsonlPath); err != nil {
				m.logger.WithError(err).WithField("session_id", sessionID).Error("failed to upload JSONL")
			} else {
				m.logger.WithField("session_id", sessionID).Info("JSONL uploaded successfully")
				os.Remove(session.jsonlPath)
			}
		}
	}

	// Remove the session worktree (discard all local changes, no commits)
	if m.gitService != nil {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := m.gitService.RemoveSessionWorktree(cleanupCtx, session.ProjectID, sessionID); err != nil {
			m.logger.WithError(err).WithField("session_id", sessionID).Warn("failed to remove session worktree")
		}
	}

	m.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"reason":     reason,
	}).Info("Chat session ended")

	payload, _ := json.Marshal(daemonws.ChatEndEvent{
		SessionID: sessionID,
		Reason:    reason,
		JSONLPath: session.jsonlPath,
	})
	m.sendMessage(daemonws.Message{
		Type:    daemonws.TypeChatEnd,
		Payload: payload,
	})
}

func (m *ChatManager) Stop() {
	close(m.stopCh)

	m.mu.Lock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		m.EndSession(id, "daemon_stopped")
	}
}

func (m *ChatManager) ttlChecker() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkTTL()
		}
	}
}

func (m *ChatManager) checkTTL() {
	now := time.Now()
	m.mu.RLock()
	var expired []string
	type warningEntry struct {
		id               string
		secondsRemaining int
	}
	var warnings []warningEntry
	for id, s := range m.sessions {
		idle := now.Sub(s.LastActivity)
		if idle > m.ttl {
			expired = append(expired, id)
		} else if idle > sessionTTLWarning {
			remaining := int((m.ttl - idle).Seconds())
			warnings = append(warnings, warningEntry{id: id, secondsRemaining: remaining})
		}
	}
	m.mu.RUnlock()

	for _, w := range warnings {
		payload, _ := json.Marshal(daemonws.ChatTTLWarningEvent{
			SessionID:        w.id,
			SecondsRemaining: w.secondsRemaining,
		})
		m.sendMessage(daemonws.Message{
			Type:    daemonws.TypeChatTTLWarning,
			Payload: payload,
		})
	}

	for _, id := range expired {
		m.logger.WithField("session_id", id).Info("Ending idle chat session (TTL exceeded)")
		m.EndSession(id, "ttl_expired")
	}
}

func (m *ChatManager) RefreshActivity(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	s.LastActivity = time.Now()
	return nil
}

func (m *ChatManager) SendMessage(sessionID string, content string) error {
	m.mu.Lock()
	session, ok := m.sessions[sessionID]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	if session.claudeStdin == nil {
		return fmt.Errorf("claude stdin not available for session: %s", sessionID)
	}

	m.mu.Lock()
	session.LastActivity = time.Now()
	m.mu.Unlock()

	msg, _ := json.Marshal(map[string]any{
		"type": "user",
		"message": map[string]string{
			"role":    "user",
			"content": content,
		},
	})
	if _, err := fmt.Fprintln(session.claudeStdin, string(msg)); err != nil {
		return fmt.Errorf("write to claude stdin: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"content":    content,
	}).Info("Sent message to Claude")

	return nil
}

func (m *ChatManager) broadcastStats() {
	m.mu.RLock()
	sessions := make([]*ChatSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.RUnlock()

	for _, s := range sessions {
		m.mu.RLock()
		event := daemonws.ChatStatsEvent{
			SessionID:        s.ID,
			MessageCount:     s.MessageCount,
			InputTokens:      s.InputTokens,
			OutputTokens:     s.OutputTokens,
			CacheReadTokens:  s.CacheReadTokens,
			CacheWriteTokens: s.CacheWriteTokens,
			TotalCost:        s.TotalCost,
			DurationSeconds:  int(time.Since(s.StartedAt).Seconds()),
			Model:            s.Model,
		}
		m.mu.RUnlock()

		payload, _ := json.Marshal(event)
		m.sendMessage(daemonws.Message{
			Type:    daemonws.TypeChatStats,
			Payload: payload,
		})
	}
}

func (m *ChatManager) statsBroadcaster() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.broadcastStats()
		}
	}
}

func (m *ChatManager) sendError(requestID, sessionID, errMsg string) {
	payload, _ := json.Marshal(daemonws.ChatErrorEvent{
		SessionID: sessionID,
		Error:     errMsg,
	})
	m.sendMessage(daemonws.Message{
		Type:      daemonws.TypeChatError,
		RequestID: requestID,
		Payload:   payload,
	})
}
