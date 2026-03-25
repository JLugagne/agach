package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JLugagne/agach-mcp/internal/agachconfig"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFileName = ".agach-daemon.yml"
	EnvOnboardingCode     = "AGACH_ONBOARDING_CODE"
	EnvServerURL          = "AGACH_SERVER_URL"
)

// Config holds daemon configuration.
type Config struct {
	// BaseURL is the Agach server URL (e.g., "https://agach.example.com")
	BaseURL string `yaml:"base_url"`

	// OnboardingCode is the 6-digit code for initial registration (from env only)
	OnboardingCode string `yaml:"-"`

	// NodeName is an optional name for this daemon instance
	NodeName string `yaml:"node_name"`
}

// DefaultConfigPath returns the default path for the daemon config file (~/.config/agach/daemon.yml).
func DefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}
	return filepath.Join(configDir, "agach", "daemon.yml"), nil
}

// WriteDefault writes a default daemon config file and creates the ~/.config/agach/ directory.
// Returns an error if the config file already exists.
func WriteDefault() error {
	path, err := DefaultConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file %q already exists", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	cfg := Config{
		BaseURL: "http://localhost:8322",
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshaling default config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config file %q: %w", path, err)
	}

	return nil
}

// Load reads daemon configuration from a YAML file and environment variables.
// It searches for .agach-daemon.yml starting from dir and walking up to parent directories.
func Load(dir string) (*Config, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("dir must be an absolute path, got: %q", dir)
	}

	cfg := &Config{}

	configPath := agachconfig.FindConfigFile(dir, DefaultConfigFileName, 5)
	if configPath == "" {
		if defaultPath, err := DefaultConfigPath(); err == nil {
			if _, err := os.Stat(defaultPath); err == nil {
				configPath = defaultPath
			}
		}
	}
	if configPath != "" {
		if err := agachconfig.LoadSecureYAML(configPath, cfg); err != nil {
			return nil, err
		}
	}

	if envURL := os.Getenv(EnvServerURL); envURL != "" {
		cfg.BaseURL = envURL
	}

	cfg.OnboardingCode = os.Getenv(EnvOnboardingCode)

	return cfg, nil
}

// Validate checks that all required configuration is present.
func (c *Config) Validate() error {
	if err := agachconfig.ValidateBaseURL(c.BaseURL); err != nil {
		if c.BaseURL == "" {
			return fmt.Errorf("base_url is required (set in %s or %s env var)", DefaultConfigFileName, EnvServerURL)
		}
		return err
	}
	return nil
}

// ValidateForOnboarding checks configuration is sufficient for initial onboarding.
func (c *Config) ValidateForOnboarding() error {
	if err := c.Validate(); err != nil {
		return err
	}

	if c.OnboardingCode == "" {
		return fmt.Errorf("onboarding code required: set %s environment variable", EnvOnboardingCode)
	}

	if len(c.OnboardingCode) != 6 {
		return fmt.Errorf("onboarding code must be exactly 6 digits")
	}

	for _, r := range c.OnboardingCode {
		if r < '0' || r > '9' {
			return fmt.Errorf("onboarding code must contain only digits")
		}
	}

	return nil
}

// SQLitePath returns the path for the daemon's SQLite database in the user config directory.
func (c *Config) SQLitePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	return filepath.Join(configDir, "agach", "daemon.db")
}

// WebSocketURL returns the WebSocket URL derived from BaseURL.
func (c *Config) WebSocketURL() string {
	url := c.BaseURL
	if strings.HasPrefix(url, "https://") {
		return strings.Replace(url, "https://", "wss://", 1) + "/ws"
	}
	if strings.HasPrefix(url, "http://") {
		return strings.Replace(url, "http://", "ws://", 1) + "/ws"
	}
	return url + "/ws"
}
