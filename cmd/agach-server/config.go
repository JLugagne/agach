package main

import (
	"fmt"
	"os"
	"time"

	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
	"gopkg.in/yaml.v3"
)

type serverConfig struct {
	SSO                    identitydomain.SsoConfig `yaml:"sso"`
	DaemonJWTTTL           time.Duration            `yaml:"daemon_jwt_ttl"`
	AuthRateLimitPerSecond float64                  `yaml:"auth_rate_limit_per_second"`
	AuthRateLimitBurst     int                      `yaml:"auth_rate_limit_burst"`
}

func writeDefaultConfig(path string) error {
	cfg := serverConfig{
		DaemonJWTTTL: 24 * time.Hour,
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshaling default config: %w", err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("config file %q already exists", path)
		}
		return fmt.Errorf("creating config file %q: %w", path, err)
	}
	_ = f.Close()

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config file %q: %w", path, err)
	}

	return nil
}

func loadConfig(configPath string) (*serverConfig, error) {
	info, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &serverConfig{}, nil
		}
		return nil, fmt.Errorf("stating server config %q: %w", configPath, err)
	}

	if info.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "WARNING: config file %q has permissions %04o; recommended 0600 (owner-only)\n", configPath, info.Mode().Perm())
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading server config %q: %w", configPath, err)
	}

	var cfg serverConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing server config %q: %w", configPath, err)
	}

	if cfg.AuthRateLimitPerSecond <= 0 && cfg.AuthRateLimitPerSecond != 0 {
		return nil, fmt.Errorf("invalid auth rate limit: AuthRateLimitPerSecond must be positive, got %v", cfg.AuthRateLimitPerSecond)
	}

	return &cfg, nil
}
