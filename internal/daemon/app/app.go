package app

import (
	"context"
	"fmt"

	"github.com/JLugagne/agach-mcp/internal/daemon/client"
	"github.com/JLugagne/agach-mcp/internal/daemon/config"
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
	cfg        *config.Config
	logger     *logrus.Logger
	tokenStore *TokenStore
	tokens     *Tokens
	state      State
	wsClient   *client.WSClient
	onboarding *client.OnboardingClient
}

func New(cfg *config.Config, logger *logrus.Logger, tokenDir string) *App {
	return &App{
		cfg:        cfg,
		logger:     logger,
		tokenStore: NewTokenStore(tokenDir),
		onboarding: client.NewOnboardingClient(cfg.BaseURL),
		state:      StateInit,
	}
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

func (a *App) handleWSEvent(event client.WSEvent) {
	a.logger.WithFields(logrus.Fields{
		"type":       event.Type,
		"project_id": event.ProjectID,
	}).Debug("Received WebSocket event")
}

func (a *App) State() State {
	return a.state
}
