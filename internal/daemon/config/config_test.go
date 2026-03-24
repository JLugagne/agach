package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JLugagne/agach-mcp/internal/daemon/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".agach-daemon.yml")

	err := os.WriteFile(cfgPath, []byte(`base_url: "https://agach.example.com"`), 0600)
	require.NoError(t, err)

	cfg, err := config.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "https://agach.example.com", cfg.BaseURL)
}

func TestLoad_FromEnv(t *testing.T) {
	dir := t.TempDir()

	t.Setenv(config.EnvServerURL, "https://env.example.com")
	t.Setenv(config.EnvOnboardingCode, "123456")

	cfg, err := config.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "https://env.example.com", cfg.BaseURL)
	assert.Equal(t, "123456", cfg.OnboardingCode)
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".agach-daemon.yml")

	err := os.WriteFile(cfgPath, []byte(`base_url: "https://file.example.com"`), 0600)
	require.NoError(t, err)

	t.Setenv(config.EnvServerURL, "https://env.example.com")

	cfg, err := config.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "https://env.example.com", cfg.BaseURL)
}

func TestLoad_UnsafePermissions(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".agach-daemon.yml")

	err := os.WriteFile(cfgPath, []byte(`base_url: "https://example.com"`), 0644)
	require.NoError(t, err)

	_, err = config.Load(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsafe permissions")
}

func TestValidate_MissingBaseURL(t *testing.T) {
	cfg := &config.Config{}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url is required")
}

func TestValidate_InsecureHTTP(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://remote.example.com"}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insecure http://")
}

func TestValidate_LocalHTTPAllowed(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8322"}
	err := cfg.Validate()
	require.NoError(t, err)
}

func TestValidateForOnboarding_MissingCode(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://example.com"}
	err := cfg.ValidateForOnboarding()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "onboarding code required")
}

func TestValidateForOnboarding_InvalidCode(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://example.com", OnboardingCode: "12345"}
	err := cfg.ValidateForOnboarding()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "6 digits")
}

func TestWebSocketURL(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{"https://example.com", "wss://example.com/ws"},
		{"http://localhost:8322", "ws://localhost:8322/ws"},
	}
	for _, tt := range tests {
		cfg := &config.Config{BaseURL: tt.baseURL}
		assert.Equal(t, tt.want, cfg.WebSocketURL())
	}
}
