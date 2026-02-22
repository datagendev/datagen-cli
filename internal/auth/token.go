package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TokenStore holds persisted OAuth tokens.
type TokenStore struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// CredentialsPath returns the path to the credentials file.
// Typically: ~/.config/datagen/credentials.json on Linux/macOS.
func CredentialsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate config directory: %w", err)
	}
	return filepath.Join(dir, "datagen", "credentials.json"), nil
}

// SaveTokens writes tokens to the credentials file with mode 0600.
func SaveTokens(tokens TokenStore) error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}
	data, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("failed to encode tokens: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// LoadTokens reads tokens from the credentials file.
// Returns nil, nil if the file does not exist.
func LoadTokens() (*TokenStore, error) {
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}
	var t TokenStore
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}
	return &t, nil
}
