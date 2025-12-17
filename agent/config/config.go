package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	// Default system paths (FHS compliant)
	DefaultConfigDir = "/etc/watchflare"
	DefaultDataDir   = "/var/lib/watchflare"
	DefaultLogDir    = "/var/log/watchflare"

	// File names
	ConfigFile = "agent.conf"
	PidFile    = "watchflare-agent.pid"
	LogFile    = "watchflare-agent.log"
)

// GetConfigDir returns the configuration directory
// Priority: WATCHFLARE_CONFIG_DIR env var > default system path
func GetConfigDir() string {
	if dir := os.Getenv("WATCHFLARE_CONFIG_DIR"); dir != "" {
		return dir
	}
	return DefaultConfigDir
}

// GetDataDir returns the data directory
// Priority: WATCHFLARE_DATA_DIR env var > default system path
func GetDataDir() string {
	if dir := os.Getenv("WATCHFLARE_DATA_DIR"); dir != "" {
		return dir
	}
	return DefaultDataDir
}

// GetLogDir returns the log directory
// Priority: WATCHFLARE_LOG_DIR env var > default system path
func GetLogDir() string {
	if dir := os.Getenv("WATCHFLARE_LOG_DIR"); dir != "" {
		return dir
	}
	return DefaultLogDir
}

// Config holds the agent configuration
type Config struct {
	ServerHost       string `toml:"server_host"`
	ServerPort       string `toml:"server_port"`
	AgentID          string `toml:"agent_id"`
	AgentKey         string `toml:"agent_key"`
	HeartbeatInterval int   `toml:"heartbeat_interval"` // in seconds, default 30
	MetricsInterval   int   `toml:"metrics_interval"`   // in seconds, default 30

	// TLS Configuration
	CACertFile string `toml:"ca_cert_file"` // Path to CA certificate for TLS
	ServerName string `toml:"server_name"`   // Server name for certificate validation
}

// SetDefaults sets default values for optional configuration fields
func (c *Config) SetDefaults() {
	if c.HeartbeatInterval == 0 {
		c.HeartbeatInterval = 30
	}
	if c.MetricsInterval == 0 {
		c.MetricsInterval = 30
	}
}

// Load reads the configuration from disk
func Load() (*Config, error) {
	configPath := filepath.Join(GetConfigDir(), ConfigFile)

	var cfg Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set default values for optional fields
	cfg.SetDefaults()

	return &cfg, nil
}

// Save writes the configuration to disk
// Uses permissions: dir 750 (root:watchflare), file 640 (root:watchflare)
func Save(cfg *Config) error {
	configDir := GetConfigDir()

	// Create config directory if it doesn't exist
	// 0750 = rwxr-x--- (owner can rwx, group can rx, others nothing)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, ConfigFile)

	// Create config file with restricted permissions
	// 0640 = rw-r----- (owner can rw, group can r, others nothing)
	file, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
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
	configPath := filepath.Join(GetConfigDir(), ConfigFile)
	_, err := os.Stat(configPath)
	return err == nil
}

// EnsureDirectories creates all required directories with proper permissions
func EnsureDirectories() error {
	directories := map[string]os.FileMode{
		GetConfigDir():                   0750, // rwxr-x---
		GetDataDir():                     0750,
		filepath.Join(GetDataDir(), "logs"): 0750,
		filepath.Join(GetDataDir(), "run"):  0750,
	}

	for dir, perm := range directories {
		if err := os.MkdirAll(dir, perm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
