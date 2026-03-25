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
	"sync"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/client"
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
	jsonlFile        *os.File
	jsonlPath        string
	MessageCount     int
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
	Model            string
}

type ChatManager struct {
	sessions      map[string]*ChatSession
	mu            sync.RWMutex
	logger        *logrus.Logger
	gitService    *GitService
	projectClient *client.ProjectClient
	uploadClient  *client.ChatUploadClient
	token         string
	sendMessage   func(daemonws.Message)
	stopCh        chan struct{}
	ttl           time.Duration
}

func NewChatManager(
	logger *logrus.Logger,
	gitService *GitService,
	projectClient *client.ProjectClient,
	uploadClient *client.ChatUploadClient,
	token string,
	sendMessage func(daemonws.Message),
) *ChatManager {
	m := &ChatManager{
		sessions:      make(map[string]*ChatSession),
		logger:        logger,
		gitService:    gitService,
		projectClient: projectClient,
		uploadClient:  uploadClient,
		token:         token,
		sendMessage:   sendMessage,
		stopCh:        make(chan struct{}),
		ttl:           30 * time.Minute,
	}
	go m.ttlChecker()
	go m.statsBroadcaster()
	return m
}

func (m *ChatManager) startClaudeProcess(session *ChatSession, resumeSessionID string) error {
	jsonlPath := fmt.Sprintf("/tmp/agach-chat-%s.jsonl", session.ID)
	f, err := os.Create(jsonlPath)
	if err != nil {
		return fmt.Errorf("create jsonl file: %w", err)
	}
	session.jsonlFile = f
	session.jsonlPath = jsonlPath

	args := []string{"--output-format", "streaming-json", "--verbose"}
	if resumeSessionID != "" {
		args = append(args, "--resume", resumeSessionID)
	}

	cmd := exec.Command("claude", args...)
	cmd.Dir = session.WorktreePath

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
		m.logger.WithError(err).WithField("session_id", session.ID).Warn("claude process exited")
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
			Model     string `json:"model"`
			SessionID string `json:"session_id"`
		}
		if err := json.Unmarshal(line, &result); err == nil {
			m.mu.Lock()
			session.InputTokens += result.Usage.InputTokens
			session.OutputTokens += result.Usage.OutputTokens
			session.CacheReadTokens += result.Usage.CacheReadTokens
			session.CacheWriteTokens += result.Usage.CacheWriteTokens
			if result.Model != "" {
				session.Model = result.Model
			}
			if result.SessionID != "" {
				session.ClaudeSessionID = result.SessionID
			}
			session.MessageCount++
			session.LastActivity = time.Now()
			m.mu.Unlock()
		}
	} else if envelope.Type == "assistant" {
		m.mu.Lock()
		session.MessageCount++
		session.LastActivity = time.Now()
		m.mu.Unlock()
	}

	event := daemonws.ChatMessageEvent{
		SessionID:   session.ID,
		MessageType: envelope.Type,
		Content:     json.RawMessage(bytes.TrimSpace(line)),
		IsFinal:     envelope.Type == "result",
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

	worktreePath, err := m.gitService.EnsureWorktree(ctx, req.ProjectID, project.GitURL, "")
	if err != nil {
		m.logger.WithError(err).WithField("project_id", req.ProjectID).Error("ensure worktree")
		m.sendError(requestID, "", "failed to prepare worktree: "+err.Error())
		return
	}

	sessionID := uuid.New().String()
	now := time.Now()

	session := &ChatSession{
		ID:           sessionID,
		FeatureID:    req.FeatureID,
		ProjectID:    req.ProjectID,
		WorktreePath: worktreePath,
		StartedAt:    now,
		LastActivity: now,
	}

	if err := m.startClaudeProcess(session, req.ResumeSessionID); err != nil {
		m.logger.WithError(err).WithField("session_id", sessionID).Error("start claude process")
		m.sendError(requestID, sessionID, "failed to start claude: "+err.Error())
		return
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

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

	if _, err := fmt.Fprintln(session.claudeStdin, content); err != nil {
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
		cost := float64(s.InputTokens)/1_000_000*3.0 +
			float64(s.OutputTokens)/1_000_000*15.0 +
			float64(s.CacheReadTokens)/1_000_000*0.30 +
			float64(s.CacheWriteTokens)/1_000_000*3.75
		event := daemonws.ChatStatsEvent{
			SessionID:        s.ID,
			MessageCount:     s.MessageCount,
			InputTokens:      s.InputTokens,
			OutputTokens:     s.OutputTokens,
			CacheReadTokens:  s.CacheReadTokens,
			CacheWriteTokens: s.CacheWriteTokens,
			TotalCost:        cost,
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
