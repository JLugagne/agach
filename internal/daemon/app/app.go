package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JLugagne/agach-mcp/internal/daemon/client"
	"github.com/JLugagne/agach-mcp/internal/daemon/config"
	"github.com/JLugagne/agach-mcp/internal/daemon/outbound/sqlite"
	"github.com/JLugagne/agach-mcp/pkg/daemonws"
	dockerclient "github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type State int

const (
	StateInit State = iota
	StateOnboarding
	StateConnected
	StateReconnecting
	StateStopped
)

func (s State) String() string {
	switch s {
	case StateInit:
		return "init"
	case StateOnboarding:
		return "onboarding"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

type App struct {
	cfg           *config.Config
	logger        *logrus.Logger
	tokenStore    *TokenStore
	tokens        *Tokens
	state         State
	wsClient      *client.WSClient
	onboarding    *client.OnboardingClient
	authClient    *client.AuthClient
	dockerService *DockerService
	db            *sql.DB
	gitService    *GitService
	projectClient *client.ProjectClient
	chatManager   *ChatManager
}

func New(cfg *config.Config, logger *logrus.Logger) (*App, error) {
	tokenStore, err := NewTokenStore()
	if err != nil {
		return nil, fmt.Errorf("init token store: %w", err)
	}
	return &App{
		cfg:        cfg,
		logger:     logger,
		tokenStore: tokenStore,
		onboarding: client.NewOnboardingClient(cfg.BaseURL),
		authClient: client.NewAuthClient(cfg.BaseURL),
		state:      StateInit,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("Starting daemon")

	tokens, err := a.tokenStore.Load()
	if err != nil {
		return fmt.Errorf("load tokens: %w", err)
	}
	a.tokens = tokens

	if a.tokens == nil {
		if err := a.doOnboarding(ctx); err != nil {
			return fmt.Errorf("onboarding: %w", err)
		}
	}

	if err := a.refreshAccessToken(ctx); err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}

	// Initialize SQLite database for build history
	if err := a.initDockerService(ctx); err != nil {
		return fmt.Errorf("init docker service: %w", err)
	}

	gitService, err := NewGitService(a.logger)
	if err != nil {
		return fmt.Errorf("init git service: %w", err)
	}
	a.gitService = gitService
	a.projectClient = client.NewProjectClient(a.cfg.BaseURL)
	uploadClient := client.NewChatUploadClient(a.cfg.BaseURL)
	a.chatManager = NewChatManager(a.logger, a.gitService, a.projectClient, uploadClient, a.tokens.AccessToken, a.sendWSMessage)

	a.wsClient = client.NewWSClient(
		a.cfg.WebSocketURL(),
		a.tokens.AccessToken,
		a.logger,
		a.handleWSEvent,
	)

	a.state = StateConnected
	a.logger.WithField("node_id", a.tokens.NodeID).Info("Daemon connected")

	return a.wsClient.RunWithReconnect(ctx)
}

func (a *App) initDockerService(ctx context.Context) error {
	dbPath := a.cfg.SQLitePath()
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	db, err := sqlite.NewDB(dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	a.db = db

	if err := sqlite.RunMigrations(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	repo := sqlite.NewBuildRepository(db)

	docker, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		a.logger.WithError(err).Warn("Docker client unavailable, builds will fail")
		a.dockerService = NewDockerService(repo, nil, a.logger)
	} else {
		a.dockerService = NewDockerService(repo, docker, a.logger)
	}

	a.logger.Info("Docker service initialized")
	return nil
}

func (a *App) doOnboarding(ctx context.Context) error {
	if err := a.cfg.ValidateForOnboarding(); err != nil {
		return err
	}

	a.state = StateOnboarding
	a.logger.WithField("code", a.cfg.OnboardingCode).Info("Starting onboarding")

	result, err := a.onboarding.CompleteOnboarding(ctx, a.cfg.OnboardingCode, a.cfg.NodeName)
	if err != nil {
		return fmt.Errorf("complete onboarding: %w", err)
	}

	a.tokens = &Tokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		NodeID:       result.NodeID,
		NodeName:     result.NodeName,
	}

	if err := a.tokenStore.Save(a.tokens); err != nil {
		return fmt.Errorf("save tokens: %w", err)
	}

	a.logger.WithField("node_id", result.NodeID).Info("Onboarding complete")
	return nil
}

func (a *App) refreshAccessToken(ctx context.Context) error {
	a.logger.Debug("Refreshing access token")

	newToken, err := a.authClient.RefreshDaemonToken(ctx, a.tokens.NodeID, a.tokens.RefreshToken)
	if err != nil {
		return err
	}

	a.tokens.AccessToken = newToken
	if err := a.tokenStore.Save(a.tokens); err != nil {
		return fmt.Errorf("save refreshed tokens: %w", err)
	}

	a.logger.Debug("Access token refreshed")
	return nil
}

func (a *App) handleWSEvent(event client.WSEvent) {
	a.logger.WithFields(logrus.Fields{
		"type":       event.Type,
		"project_id": event.ProjectID,
	}).Debug("Received WebSocket event")

	// Handle daemon-targeted docker messages
	switch event.Type {
	case daemonws.TypeDockerList:
		a.handleDockerList(event)
	case daemonws.TypeDockerRebuild:
		a.handleDockerRebuild(event)
	case daemonws.TypeDockerLogs:
		a.handleDockerLogs(event)
	case daemonws.TypeDockerPrune:
		a.handleDockerPrune(event)
	case daemonws.TypeChatStart:
		a.handleChatStart(event)
	case daemonws.TypeChatUserMsg:
		a.handleChatUserMsg(event)
	case daemonws.TypeChatEnd:
		a.handleChatEndRequest(event)
	case daemonws.TypeChatPing:
		a.handleChatPing(event)
	}
}

func (a *App) handleDockerList(event client.WSEvent) {
	if a.dockerService == nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "docker service not initialized"})
		return
	}
	images, err := a.dockerService.ListImages(context.Background())
	if err != nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: err.Error()})
		return
	}
	type buildInfo struct {
		BuildID   string `json:"build_id"`
		Version   string `json:"version"`
		Status    string `json:"status"`
		ImageHash string `json:"image_hash,omitempty"`
		SizeBytes int64  `json:"size_bytes,omitempty"`
		CreatedAt string `json:"created_at"`
	}
	type dockerfileInfo struct {
		Slug          string      `json:"slug"`
		LatestVersion string      `json:"latest_version"`
		VersionCount  int         `json:"version_count"`
		IsHealthy     bool        `json:"is_healthy"`
		Builds        []buildInfo `json:"builds"`
	}
	resp := struct {
		Dockerfiles []dockerfileInfo `json:"dockerfiles"`
	}{Dockerfiles: make([]dockerfileInfo, 0, len(images))}
	for _, img := range images {
		info := dockerfileInfo{
			Slug:          img.Slug,
			LatestVersion: img.LatestTag,
			VersionCount:  len(img.Builds),
			IsHealthy:     img.IsHealthy,
			Builds:        make([]buildInfo, 0, len(img.Builds)),
		}
		for _, b := range img.Builds {
			info.Builds = append(info.Builds, buildInfo{
				BuildID:   string(b.ID),
				Version:   b.Version,
				Status:    string(b.Status),
				ImageHash: b.ImageHash,
				SizeBytes: b.ImageSize,
				CreatedAt: b.CreatedAt.Format("2006-01-02T15:04:05Z"),
			})
		}
		resp.Dockerfiles = append(resp.Dockerfiles, info)
	}
	payload, _ := json.Marshal(resp)
	a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeDockerList, Payload: payload})
}

func (a *App) handleDockerRebuild(event client.WSEvent) {
	if a.dockerService == nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "docker service not initialized"})
		return
	}
	var req struct {
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(event.Data, &req); err != nil || req.Slug == "" {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "slug is required"})
		return
	}

	// Acknowledge immediately
	ack, _ := json.Marshal(map[string]string{"status": "started", "slug": req.Slug})
	a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeDockerRebuild, Payload: ack})

	// Run build asynchronously and forward events
	go func() {
		eventCh := make(chan daemonws.BuildEvent, 10)
		go func() {
			for evt := range eventCh {
				payload, _ := json.Marshal(evt)
				a.sendWSMessage(daemonws.Message{Type: daemonws.TypeBuildEvent, Payload: payload})
			}
		}()
		if err := a.dockerService.Rebuild(context.Background(), req.Slug, eventCh); err != nil {
			a.logger.WithError(err).Error("rebuild failed")
		}
		close(eventCh)
	}()
}

func (a *App) handleDockerLogs(event client.WSEvent) {
	if a.dockerService == nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "docker service not initialized"})
		return
	}
	var req struct {
		BuildID string `json:"build_id"`
	}
	if err := json.Unmarshal(event.Data, &req); err != nil || req.BuildID == "" {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "build_id is required"})
		return
	}
	result, err := a.dockerService.GetBuildLogs(context.Background(), req.BuildID)
	if err != nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: err.Error()})
		return
	}
	if result == nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "build not found"})
		return
	}
	payload, _ := json.Marshal(result)
	a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeDockerLogs, Payload: payload})
}

func (a *App) handleDockerPrune(event client.WSEvent) {
	if a.dockerService == nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "docker service not initialized"})
		return
	}

	ack, _ := json.Marshal(map[string]string{"status": "started"})
	a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeDockerPrune, Payload: ack})

	go func() {
		eventCh := make(chan daemonws.PruneEvent, 10)
		go func() {
			for evt := range eventCh {
				payload, _ := json.Marshal(evt)
				a.sendWSMessage(daemonws.Message{Type: daemonws.TypePruneEvent, Payload: payload})
			}
		}()
		result, err := a.dockerService.PruneNonLatest(context.Background(), eventCh)
		close(eventCh)
		if err != nil {
			a.logger.WithError(err).Error("prune failed")
		} else {
			a.logger.WithField("removed", result.Removed).Info("prune completed")
		}
	}()
}

func (a *App) handleChatStart(event client.WSEvent) {
	if a.chatManager == nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "chat manager not initialized"})
		return
	}
	var req daemonws.ChatStartRequest
	if err := json.Unmarshal(event.Data, &req); err != nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "invalid chat.start payload"})
		return
	}
	if req.ProjectID == "" || req.FeatureID == "" {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "project_id and feature_id are required"})
		return
	}
	go a.chatManager.StartSession(context.Background(), event.RequestID, req)
}

func (a *App) handleChatUserMsg(event client.WSEvent) {
	if a.chatManager == nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "chat manager not initialized"})
		return
	}
	var req daemonws.ChatUserMessage
	if err := json.Unmarshal(event.Data, &req); err != nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "invalid chat.user_message payload"})
		return
	}
	if req.SessionID == "" {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "session_id is required"})
		return
	}
	if err := a.chatManager.SendMessage(req.SessionID, req.Content); err != nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: err.Error()})
	}
}

func (a *App) handleChatEndRequest(event client.WSEvent) {
	if a.chatManager == nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "chat manager not initialized"})
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(event.Data, &req); err != nil || req.SessionID == "" {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "session_id is required"})
		return
	}
	a.chatManager.EndSession(req.SessionID, "user_ended")
}

func (a *App) handleChatPing(event client.WSEvent) {
	if a.chatManager == nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "chat manager not initialized"})
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(event.Data, &req); err != nil || req.SessionID == "" {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: "session_id is required"})
		return
	}
	if err := a.chatManager.RefreshActivity(req.SessionID); err != nil {
		a.sendWSResponse(event, daemonws.Message{Type: daemonws.TypeError, Error: err.Error()})
	}
}

func (a *App) sendWSResponse(event client.WSEvent, msg daemonws.Message) {
	if msg.RequestID == "" {
		msg.RequestID = event.RequestID
	}
	a.sendWSMessage(msg)
}

// wsResponse is the message format sent back through the server WS relay.
// It uses "data" (not "payload") to match the browser's WSEvent parsing.
type wsResponse struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Data      any    `json:"data,omitempty"`
}

func (a *App) sendWSMessage(msg daemonws.Message) {
	if a.wsClient == nil {
		return
	}
	// Convert daemonws.Message to wsResponse format for browser compatibility
	var data any
	if msg.Payload != nil {
		_ = json.Unmarshal(msg.Payload, &data)
	}
	if msg.Error != "" {
		data = map[string]string{"error": msg.Error}
	}
	a.wsClient.Send(wsResponse{
		Type:      msg.Type,
		RequestID: msg.RequestID,
		Data:      data,
	})
}

func (a *App) State() State {
	return a.state
}
