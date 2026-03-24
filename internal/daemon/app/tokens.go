package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	tokenFileName = "tokens.json"
	configDirName = "agach-daemon"
)

type TokenStore struct {
	path string
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	NodeID       string `json:"node_id"`
	NodeName     string `json:"node_name"`
}

func NewTokenStore() (*TokenStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("get user config dir: %w", err)
	}
	return &TokenStore{path: filepath.Join(configDir, configDirName, tokenFileName)}, nil
}

// NewTokenStoreWithDir creates a token store in a specific directory (for testing).
func NewTokenStoreWithDir(dir string) *TokenStore {
	return &TokenStore{path: filepath.Join(dir, tokenFileName)}
}

func (s *TokenStore) Load() (*Tokens, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tokens: %w", err)
	}

	var tokens Tokens
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("parse tokens: %w", err)
	}

	return &tokens, nil
}

func (s *TokenStore) Save(tokens *Tokens) error {
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tokens: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("create token dir: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("write tokens: %w", err)
	}

	return nil
}

func (s *TokenStore) Clear() error {
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove tokens: %w", err)
	}
	return nil
}
