package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestEnv creates a temporary directory for tests
func setupTestEnv(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "watchflare-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set environment variables to use temp directory
	os.Setenv("WATCHFLARE_CONFIG_DIR", filepath.Join(tmpDir, "config"))
	os.Setenv("WATCHFLARE_DATA_DIR", filepath.Join(tmpDir, "data"))
	os.Setenv("WATCHFLARE_LOG_DIR", filepath.Join(tmpDir, "logs"))

	cleanup := func() {
		os.Unsetenv("WATCHFLARE_CONFIG_DIR")
		os.Unsetenv("WATCHFLARE_DATA_DIR")
		os.Unsetenv("WATCHFLARE_LOG_DIR")
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestGetConfigDir(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{
			name:     "default path when no env var",
			envValue: "",
			want:     DefaultConfigDir,
		},
		{
			name:     "custom path from env var",
			envValue: "/custom/config",
			want:     "/custom/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env var
			original := os.Getenv("WATCHFLARE_CONFIG_DIR")
			defer os.Setenv("WATCHFLARE_CONFIG_DIR", original)

			// Set test env var
			if tt.envValue != "" {
				os.Setenv("WATCHFLARE_CONFIG_DIR", tt.envValue)
			} else {
				os.Unsetenv("WATCHFLARE_CONFIG_DIR")
			}

			got := GetConfigDir()
			if got != tt.want {
				t.Errorf("GetConfigDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDataDir(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{
			name:     "default path when no env var",
			envValue: "",
			want:     DefaultDataDir,
		},
		{
			name:     "custom path from env var",
			envValue: "/custom/data",
			want:     "/custom/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := os.Getenv("WATCHFLARE_DATA_DIR")
			defer os.Setenv("WATCHFLARE_DATA_DIR", original)

			if tt.envValue != "" {
				os.Setenv("WATCHFLARE_DATA_DIR", tt.envValue)
			} else {
				os.Unsetenv("WATCHFLARE_DATA_DIR")
			}

			got := GetDataDir()
			if got != tt.want {
				t.Errorf("GetDataDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLogDir(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{
			name:     "default path when no env var",
			envValue: "",
			want:     DefaultLogDir,
		},
		{
			name:     "custom path from env var",
			envValue: "/custom/logs",
			want:     "/custom/logs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := os.Getenv("WATCHFLARE_LOG_DIR")
			defer os.Setenv("WATCHFLARE_LOG_DIR", original)

			if tt.envValue != "" {
				os.Setenv("WATCHFLARE_LOG_DIR", tt.envValue)
			} else {
				os.Unsetenv("WATCHFLARE_LOG_DIR")
			}

			got := GetLogDir()
			if got != tt.want {
				t.Errorf("GetLogDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create test config
	original := &Config{
		ServerHost: "test.example.com",
		ServerPort: "50051",
		AgentID:    "test-agent-123",
		AgentKey:   "test-key-456",
	}

	// Save config
	if err := Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(GetConfigDir(), ConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", configPath)
	}

	// Verify file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	// Check permissions (0640)
	expectedPerm := os.FileMode(0640)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("Config file permissions = %v, want %v", info.Mode().Perm(), expectedPerm)
	}

	// Load config back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Compare configs
	if loaded.ServerHost != original.ServerHost {
		t.Errorf("ServerHost = %v, want %v", loaded.ServerHost, original.ServerHost)
	}
	if loaded.ServerPort != original.ServerPort {
		t.Errorf("ServerPort = %v, want %v", loaded.ServerPort, original.ServerPort)
	}
	if loaded.AgentID != original.AgentID {
		t.Errorf("AgentID = %v, want %v", loaded.AgentID, original.AgentID)
	}
	if loaded.AgentKey != original.AgentKey {
		t.Errorf("AgentKey = %v, want %v", loaded.AgentKey, original.AgentKey)
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Try to load non-existent config
	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for non-existent config, got nil")
	}
}

func TestExists(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Check non-existent config
	if Exists() {
		t.Error("Exists() = true for non-existent config, want false")
	}

	// Create config
	cfg := &Config{
		ServerHost: "test.example.com",
		ServerPort: "50051",
		AgentID:    "test-agent",
		AgentKey:   "test-key",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Check existing config
	if !Exists() {
		t.Error("Exists() = false for existing config, want true")
	}
}

func TestEnsureDirectories(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create directories
	if err := EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories() error = %v", err)
	}

	// Verify all directories exist
	expectedDirs := []string{
		filepath.Join(tmpDir, "config"),
		filepath.Join(tmpDir, "data"),
		filepath.Join(tmpDir, "data", "logs"),
		filepath.Join(tmpDir, "data", "run"),
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Directory %s does not exist: %v", dir, err)
			continue
		}

		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}

		// Check permissions (0750)
		expectedPerm := os.FileMode(0750)
		if info.Mode().Perm() != expectedPerm {
			t.Errorf("Directory %s permissions = %v, want %v", dir, info.Mode().Perm(), expectedPerm)
		}
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Config directory doesn't exist yet
	configDir := GetConfigDir()
	if _, err := os.Stat(configDir); err == nil {
		t.Fatalf("Config directory already exists before Save()")
	}

	// Save should create the directory
	cfg := &Config{
		ServerHost: "test.example.com",
		ServerPort: "50051",
		AgentID:    "test-agent",
		AgentKey:   "test-key",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("Config directory was not created: %v", err)
	}

	if !info.IsDir() {
		t.Fatalf("Config path is not a directory")
	}

	// Check directory permissions (0750)
	expectedPerm := os.FileMode(0750)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("Config directory permissions = %v, want %v", info.Mode().Perm(), expectedPerm)
	}
}
