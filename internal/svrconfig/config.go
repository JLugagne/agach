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
