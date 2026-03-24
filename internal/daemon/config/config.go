package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFileName = ".agach-daemon.yml"
	EnvOnboardingCode     = "AGACH_ONBOARDING_CODE"
	EnvServerURL          = "AGACH_SERVER_URL"
	maxWalkDepth          = 5
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

// Load reads daemon configuration from a YAML file and environment variables.
// It searches for .agach-daemon.yml starting from dir and walking up to parent directories.
func Load(dir string) (*Config, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("dir must be an absolute path, got: %q", dir)
	}

	cfg := &Config{}

	configPath := findConfigFile(dir)
	if configPath != "" {
		if err := loadFromFile(configPath, cfg); err != nil {
			return nil, err
		}
	}

	if envURL := os.Getenv(EnvServerURL); envURL != "" {
		cfg.BaseURL = envURL
	}

	cfg.OnboardingCode = os.Getenv(EnvOnboardingCode)

	return cfg, nil
}

func findConfigFile(dir string) string {
	current := dir
	for depth := 0; depth < maxWalkDepth; depth++ {
		candidate := filepath.Join(current, DefaultConfigFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

func loadFromFile(path string, cfg *Config) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat config file: %w", err)
	}
	mode := info.Mode().Perm()
	if mode&0o177 != 0 {
		return fmt.Errorf("config file %q has unsafe permissions (mode %04o): must be 0600 or stricter", path, mode)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	return nil
}

// Validate checks that all required configuration is present.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required (set in %s or %s env var)", DefaultConfigFileName, EnvServerURL)
	}

	if strings.HasPrefix(c.BaseURL, "http://") {
		host := strings.TrimPrefix(c.BaseURL, "http://")
		host = strings.SplitN(host, "/", 2)[0]
		host = strings.SplitN(host, ":", 2)[0]
		if host != "localhost" && host != "127.0.0.1" {
			return fmt.Errorf("base_url uses insecure http:// for remote host %q — use https:// for production", host)
		}
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
