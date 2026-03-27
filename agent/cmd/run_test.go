package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"watchflare-agent/config"
	"watchflare-agent/packages"
)

// writeTestConfig writes a TOML config file into dir.
func writeTestConfig(t *testing.T, dir, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, config.ConfigFile), []byte(content), 0640)
	if err != nil {
		t.Fatalf("writeTestConfig: %v", err)
	}
}

// --- loadConfig ---

func TestLoadConfig_FileNotFound(t *testing.T) {
	t.Setenv("WATCHFLARE_CONFIG_DIR", t.TempDir())

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "register") {
		t.Errorf("expected hint to register, got: %v", err)
	}
}

func TestLoadConfig_MissingServerHost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WATCHFLARE_CONFIG_DIR", dir)
	writeTestConfig(t, dir, `
server_port = "50051"
agent_id    = "test-id"
agent_key   = "test-key"
`)

	_, err := loadConfig()
	if err == nil || !strings.Contains(err.Error(), "server_host") {
		t.Errorf("expected server_host error, got: %v", err)
	}
}

func TestLoadConfig_MissingServerPort(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WATCHFLARE_CONFIG_DIR", dir)
	writeTestConfig(t, dir, `
server_host = "localhost"
agent_id    = "test-id"
agent_key   = "test-key"
`)

	_, err := loadConfig()
	if err == nil || !strings.Contains(err.Error(), "server_port") {
		t.Errorf("expected server_port error, got: %v", err)
	}
}

func TestLoadConfig_MissingAgentID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WATCHFLARE_CONFIG_DIR", dir)
	writeTestConfig(t, dir, `
server_host = "localhost"
server_port = "50051"
agent_key   = "test-key"
`)

	_, err := loadConfig()
	if err == nil || !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("expected agent_id error, got: %v", err)
	}
}

func TestLoadConfig_MissingAgentKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WATCHFLARE_CONFIG_DIR", dir)
	writeTestConfig(t, dir, `
server_host = "localhost"
server_port = "50051"
agent_id    = "test-id"
`)

	_, err := loadConfig()
	if err == nil || !strings.Contains(err.Error(), "agent_key") {
		t.Errorf("expected agent_key error, got: %v", err)
	}
}

func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WATCHFLARE_CONFIG_DIR", dir)
	writeTestConfig(t, dir, `
server_host = "backend.example.com"
server_port = "50051"
agent_id    = "test-id"
agent_key   = "test-key"
`)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ServerHost != "backend.example.com" {
		t.Errorf("ServerHost: got %q, want %q", cfg.ServerHost, "backend.example.com")
	}
	if cfg.ServerPort != "50051" {
		t.Errorf("ServerPort: got %q, want %q", cfg.ServerPort, "50051")
	}
	if cfg.AgentID != "test-id" {
		t.Errorf("AgentID: got %q, want %q", cfg.AgentID, "test-id")
	}
	if cfg.AgentKey != "test-key" {
		t.Errorf("AgentKey: got %q, want %q", cfg.AgentKey, "test-key")
	}
	// Verify SetDefaults() was applied
	if cfg.HeartbeatInterval != config.DefaultHeartbeatInterval {
		t.Errorf("HeartbeatInterval: got %d, want %d (default)", cfg.HeartbeatInterval, config.DefaultHeartbeatInterval)
	}
	if cfg.MetricsInterval != config.DefaultMetricsInterval {
		t.Errorf("MetricsInterval: got %d, want %d (default)", cfg.MetricsInterval, config.DefaultMetricsInterval)
	}
}

// --- convertPackagesToProto ---

func TestConvertPackagesToProto_Empty(t *testing.T) {
	result := convertPackagesToProto([]*packages.Package{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(result))
	}
}

func TestConvertPackagesToProto_ZeroInstalledAt(t *testing.T) {
	pkgs := []*packages.Package{
		{Name: "curl", Version: "7.88", InstalledAt: time.Time{}},
	}
	result := convertPackagesToProto(pkgs)
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	if result[0].InstalledAt != 0 {
		t.Errorf("InstalledAt: got %d, want 0 for zero time", result[0].InstalledAt)
	}
}

func TestConvertPackagesToProto_InstalledAt(t *testing.T) {
	ts := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	pkgs := []*packages.Package{
		{Name: "curl", Version: "7.88", InstalledAt: ts},
	}
	result := convertPackagesToProto(pkgs)
	if result[0].InstalledAt != ts.Unix() {
		t.Errorf("InstalledAt: got %d, want %d", result[0].InstalledAt, ts.Unix())
	}
}

func TestConvertPackagesToProto_AllFields(t *testing.T) {
	ts := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	pkgs := []*packages.Package{{
		Name:           "nginx",
		Version:        "1.24.0",
		Architecture:   "amd64",
		PackageManager: "apt",
		Source:         "ubuntu",
		InstalledAt:    ts,
		PackageSize:    1024,
		Description:    "HTTP server",
	}}
	result := convertPackagesToProto(pkgs)
	if len(result) != 1 {
		t.Fatalf("expected 1 element, got %d", len(result))
	}
	p := result[0]
	if p.Name != "nginx" {
		t.Errorf("Name: got %q, want %q", p.Name, "nginx")
	}
	if p.Version != "1.24.0" {
		t.Errorf("Version: got %q, want %q", p.Version, "1.24.0")
	}
	if p.Architecture != "amd64" {
		t.Errorf("Architecture: got %q, want %q", p.Architecture, "amd64")
	}
	if p.PackageManager != "apt" {
		t.Errorf("PackageManager: got %q, want %q", p.PackageManager, "apt")
	}
	if p.Source != "ubuntu" {
		t.Errorf("Source: got %q, want %q", p.Source, "ubuntu")
	}
	if p.InstalledAt != ts.Unix() {
		t.Errorf("InstalledAt: got %d, want %d", p.InstalledAt, ts.Unix())
	}
	if p.PackageSize != 1024 {
		t.Errorf("PackageSize: got %d, want %d", p.PackageSize, 1024)
	}
	if p.Description != "HTTP server" {
		t.Errorf("Description: got %q, want %q", p.Description, "HTTP server")
	}
}

func TestConvertPackagesToProto_PreservesOrder(t *testing.T) {
	pkgs := []*packages.Package{
		{Name: "alpha"},
		{Name: "beta"},
		{Name: "gamma"},
	}
	result := convertPackagesToProto(pkgs)
	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result))
	}
	for i, want := range []string{"alpha", "beta", "gamma"} {
		if result[i].Name != want {
			t.Errorf("[%d] Name: got %q, want %q", i, result[i].Name, want)
		}
	}
}
