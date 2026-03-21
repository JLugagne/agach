package agachconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxWalkDepth = 5

type Config struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

func Load(dir string) (*Config, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("dir must be an absolute path, got: %q", dir)
	}

	current := dir
	for depth := 0; depth < maxWalkDepth; depth++ {
		candidate := filepath.Join(current, ".agach.yml")
		info, err := os.Stat(candidate)
		if err == nil {
			mode := info.Mode().Perm()
			if mode&0o177 != 0 {
				return nil, fmt.Errorf("config file %q has unsafe permissions (mode %04o): must be 0600 or stricter", candidate, mode)
			}
			data, err := os.ReadFile(candidate)
			if err != nil {
				return nil, fmt.Errorf("reading config file: %w", err)
			}
			var cfg Config
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("parsing config file: %w", err)
			}
			return &cfg, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return nil, nil
}

func (c *Config) ResolvedBaseURL() string {
	return c.BaseURL
}

func (c *Config) ResolvedAPIKey() string {
	return c.APIKey
}

func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	if c.APIKey == "" {
		return fmt.Errorf("api_key is required and must not be empty")
	}

	if strings.HasPrefix(c.APIKey, "$") {
		varName := strings.TrimPrefix(c.APIKey, "$")
		if _, set := os.LookupEnv(varName); !set {
			return fmt.Errorf("api_key references env var $%s which is not set", varName)
		}
	}

	if strings.HasPrefix(c.BaseURL, "http://") {
		host := strings.TrimPrefix(c.BaseURL, "http://")
		host = strings.SplitN(host, "/", 2)[0]
		host = strings.SplitN(host, ":", 2)[0]
		if host != "localhost" && host != "127.0.0.1" {
			return fmt.Errorf("base_url uses insecure http:// for remote host %q — use https:// to enable TLS", host)
		}
	}

	return nil
}
