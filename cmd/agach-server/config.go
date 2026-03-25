package main

import (
	"fmt"
	"os"
	"time"

	identitydomain "github.com/JLugagne/agach-mcp/internal/identity/domain"
	"gopkg.in/yaml.v3"
)

type serverConfig struct {
	SSO          identitydomain.SsoConfig `yaml:"sso"`
	DaemonJWTTTL time.Duration            `yaml:"daemon_jwt_ttl"`
}

func writeDefaultConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file %q already exists", path)
	}

	cfg := serverConfig{
		DaemonJWTTTL: 24 * time.Hour,
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshaling default config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file %q: %w", path, err)
	}

	return nil
}

func loadConfig(path string) (*serverConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &serverConfig{}, nil
		}
		return nil, fmt.Errorf("reading server config %q: %w", path, err)
	}

	var cfg serverConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing server config %q: %w", path, err)
	}

	return &cfg, nil
}
