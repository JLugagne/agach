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
}

// FindConfigFile walks up from dir looking for filename, up to maxDepth levels.
// Returns the first matching path, or empty string if not found.
func FindConfigFile(dir string, filename string, maxDepth int) string {
	current := dir
	for depth := 0; depth < maxDepth; depth++ {
		candidate := filepath.Join(current, filename)
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

// LoadSecureYAML reads a YAML file at path into dest after verifying the file
// has permissions no broader than 0600.
func LoadSecureYAML(path string, dest any) error {
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
		return fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("parsing config file: %w", err)
	}

	return nil
}

// ValidateBaseURL returns an error if baseURL is empty or uses http:// for a
// non-localhost host.
func ValidateBaseURL(baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	if strings.HasPrefix(baseURL, "http://") {
		host := strings.TrimPrefix(baseURL, "http://")
		host = strings.SplitN(host, "/", 2)[0]
		host = strings.SplitN(host, ":", 2)[0]
		if host != "localhost" && host != "127.0.0.1" {
			return fmt.Errorf("base_url uses insecure http:// for remote host %q — use https:// to enable TLS", host)
		}
	}

	return nil
}

func Load(dir string) (*Config, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("dir must be an absolute path, got: %q", dir)
	}

	path := FindConfigFile(dir, ".agach.yml", maxWalkDepth)
	if path == "" {
		return nil, nil
	}

	var cfg Config
	if err := LoadSecureYAML(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) ResolvedBaseURL() string {
	return c.BaseURL
}

func (c *Config) Validate() error {
	return ValidateBaseURL(c.BaseURL)
}
