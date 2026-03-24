package svrconfig

import (
	"fmt"
	"os"
	"time"

	identitysvrconfig "github.com/JLugagne/agach-mcp/internal/identity/svrconfig"
	"gopkg.in/yaml.v3"
)

// Config is the root server configuration decoded from the YAML config file.
type Config struct {
	SSO          identitysvrconfig.SsoConfig `yaml:"sso"`
	DaemonJWTTTL time.Duration              `yaml:"daemon_jwt_ttl"`
}

// WriteDefault writes a default YAML config file at the given path.
// Returns an error if the file already exists.
func WriteDefault(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file %q already exists", path)
	}

	cfg := Config{
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

// Load reads and decodes a YAML config file at the given path.
// Returns an empty Config (no error) when the file does not exist.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading server config %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing server config %q: %w", path, err)
	}

	return &cfg, nil
}
