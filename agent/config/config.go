package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	ConfigDir  = "/etc/watchflare"
	ConfigFile = "agent.conf"
)

// Config holds the agent configuration
type Config struct {
	ServerHost string `toml:"server_host"`
	ServerPort string `toml:"server_port"`
	AgentID    string `toml:"agent_id"`
	AgentKey   string `toml:"agent_key"`
}

// Load reads the configuration from disk
func Load() (*Config, error) {
	configPath := filepath.Join(ConfigDir, ConfigFile)

	var cfg Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to disk
func Save(cfg *Config) error {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(ConfigDir, ConfigFile)

	file, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// Exists checks if a configuration file already exists
func Exists() bool {
	configPath := filepath.Join(ConfigDir, ConfigFile)
	_, err := os.Stat(configPath)
	return err == nil
}
