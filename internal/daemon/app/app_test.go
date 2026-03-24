package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JLugagne/agach-mcp/internal/daemon/app"
	"github.com/JLugagne/agach-mcp/internal/daemon/config"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func TestApp_Run_WithExistingTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			time.Sleep(200 * time.Millisecond)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	cfg := &config.Config{BaseURL: server.URL}

	tokenStore := app.NewTokenStore(dir)
	err := tokenStore.Save(&app.Tokens{
		AccessToken:  "test-token",
		RefreshToken: "refresh-token",
		NodeID:       "node-123",
	})
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	daemon := app.New(cfg, logger, dir)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = daemon.Run(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestApp_Run_Onboarding(t *testing.T) {
	onboardingCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/onboarding/complete" {
			onboardingCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data": map[string]any{
					"access_token":  "new-token",
					"refresh_token": "new-refresh",
					"node": map[string]any{
						"id":   "new-node-id",
						"name": "test-daemon",
						"mode": "default",
					},
				},
			})
			return
		}
		if r.URL.Path == "/ws" {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			time.Sleep(200 * time.Millisecond)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	cfg := &config.Config{
		BaseURL:        server.URL,
		OnboardingCode: "123456",
	}

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	daemon := app.New(cfg, logger, dir)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	daemon.Run(ctx)

	assert.True(t, onboardingCalled)

	tokenStore := app.NewTokenStore(dir)
	tokens, err := tokenStore.Load()
	require.NoError(t, err)
	assert.Equal(t, "new-token", tokens.AccessToken)
}

func TestTokenStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := app.NewTokenStore(dir)

	tokens := &app.Tokens{
		AccessToken:  "access",
		RefreshToken: "refresh",
		NodeID:       "node",
		NodeName:     "name",
	}

	err := store.Save(tokens)
	require.NoError(t, err)

	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, tokens.AccessToken, loaded.AccessToken)
	assert.Equal(t, tokens.RefreshToken, loaded.RefreshToken)
	assert.Equal(t, tokens.NodeID, loaded.NodeID)

	path := filepath.Join(dir, ".agach-daemon-tokens.json")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestTokenStore_Load_NotExists(t *testing.T) {
	dir := t.TempDir()
	store := app.NewTokenStore(dir)

	tokens, err := store.Load()
	require.NoError(t, err)
	assert.Nil(t, tokens)
}
