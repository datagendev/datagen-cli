package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// LoadConfig reads and parses a datagen.toml file
func LoadConfig(path string) (*DatagenConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config DatagenConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// SaveConfig writes a DatagenConfig to a TOML file
func SaveConfig(config *DatagenConfig, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode TOML: %w", err)
	}

	return nil
}
