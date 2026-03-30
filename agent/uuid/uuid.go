package uuid

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"watchflare-agent/config"
)

const uuidFileName = "agent.uuid"

// GetUUIDPath returns the path to the UUID file
func GetUUIDPath() string {
	return filepath.Join(config.GetDataDir(), uuidFileName)
}

// Load loads the agent UUID from disk
// Returns empty string if file doesn't exist
func Load() (string, error) {
	path := GetUUIDPath()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil // No UUID yet (first installation)
	}

	// Read UUID from file
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to read UUID file: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 256))
	if err != nil {
		return "", fmt.Errorf("failed to read UUID file: %w", err)
	}

	uuid := strings.TrimSpace(string(data))
	if uuid == "" {
		return "", fmt.Errorf("uuid file is empty")
	}

	return uuid, nil
}

// Save saves the agent UUID to disk
func Save(uuid string) error {
	if uuid == "" {
		return fmt.Errorf("cannot save empty UUID")
	}

	path := GetUUIDPath()

	// Ensure data directory exists
	dataDir := config.GetDataDir()
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Write UUID to file
	if err := os.WriteFile(path, []byte(uuid+"\n"), 0640); err != nil {
		return fmt.Errorf("failed to write UUID file: %w", err)
	}

	return nil
}

// Exists checks if the UUID file exists
func Exists() bool {
	path := GetUUIDPath()
	_, err := os.Stat(path)
	return err == nil
}
