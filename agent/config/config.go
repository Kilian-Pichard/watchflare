package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/BurntSushi/toml"
)

const (
	// Default system paths (FHS compliant)
	DefaultConfigDir = "/etc/watchflare"
	DefaultDataDir   = "/var/lib/watchflare"

	// File names
	ConfigFile     = "agent.conf"
	DefaultLogFile = "/var/log/watchflare-agent.log" // matches install.LogPath
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

// Config holds the agent configuration
type Config struct {
	ServerHost string `toml:"server_host"`
	ServerPort string `toml:"server_port"`
	AgentID    string `toml:"agent_id"`
	AgentKey   string `toml:"agent_key"`

	HeartbeatInterval int `toml:"heartbeat_interval"` // in seconds, default 5
	MetricsInterval   int `toml:"metrics_interval"`   // in seconds, default 30

	// TLS Configuration
	CACertFile string `toml:"ca_cert_file"` // Path to CA certificate for TLS
	ServerName string `toml:"server_name"`  // Server name for certificate validation

	// WAL Configuration (simplified V1)
	WALEnabled   *bool  `toml:"wal_enabled"`     // Enable WAL persistence (default: true)
	WALPath      string `toml:"wal_path"`        // WAL file path
	WALMaxSizeMB int    `toml:"wal_max_size_mb"` // Max WAL size before FIFO truncate

	// Log file path (optional — empty means stdout, captured by service manager)
	LogFile string `toml:"log_file"`

	// Docker metrics (opt-in: requires Docker socket access)
	DockerMetrics *bool `toml:"docker_metrics"` // Enable Docker container metrics (default: false)
}

// SetDefaults sets default values for optional configuration fields
func (c *Config) SetDefaults() {
	if c.HeartbeatInterval == 0 {
		c.HeartbeatInterval = 5
	}
	if c.MetricsInterval == 0 {
		c.MetricsInterval = 30
	}

	// WAL defaults
	if c.WALEnabled == nil {
		enabled := true
		c.WALEnabled = &enabled
	}
	if c.WALPath == "" {
		c.WALPath = filepath.Join(GetDataDir(), "metrics.wal")
	}
	if c.WALMaxSizeMB == 0 {
		c.WALMaxSizeMB = 10
	}

	// Docker metrics default: disabled
	if c.DockerMetrics == nil {
		disabled := false
		c.DockerMetrics = &disabled
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

	cfg.SetDefaults()
	return &cfg, nil
}

// Save writes the configuration to disk
func Save(cfg *Config) error {
	configDir := GetConfigDir()

	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, ConfigFile)

	file, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	// Set proper ownership when running as root (installation/registration)
	// Linux: root:watchflare, macOS: root:staff
	if os.Geteuid() == 0 {
		var groupName string
		switch runtime.GOOS {
		case "linux":
			groupName = "watchflare"
		case "darwin":
			groupName = "staff"
		}

		if groupName != "" {
			if group, err := user.LookupGroup(groupName); err == nil {
				if gid, err := strconv.Atoi(group.Gid); err == nil {
					// Change ownership to root:group (0 = root UID)
					if err := os.Chown(configPath, 0, gid); err != nil {
						// Don't fail on chown error, just warn
						fmt.Fprintf(os.Stderr, "Warning: failed to set ownership on %s: %v\n", configPath, err)
					}
				}
			}
		}
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
		GetConfigDir(): 0750,
		GetDataDir():   0750,
	}

	for dir, perm := range directories {
		if err := os.MkdirAll(dir, perm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
