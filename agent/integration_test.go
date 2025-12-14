// +build integration

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"watchflare/agent/client"
	"watchflare/agent/config"
	"watchflare/agent/sysinfo"
)

// TestEndToEndRegistration tests the complete registration flow
// Run with: go test -tags=integration -v
// Requires: Backend server running on localhost:50051
func TestEndToEndRegistration(t *testing.T) {
	// Setup test environment
	tmpDir := setupIntegrationTest(t)
	defer os.RemoveAll(tmpDir)

	// Get backend address from environment
	backendHost := os.Getenv("WATCHFLARE_TEST_BACKEND_HOST")
	if backendHost == "" {
		backendHost = "localhost"
	}

	backendPort := os.Getenv("WATCHFLARE_TEST_BACKEND_PORT")
	if backendPort == "" {
		backendPort = "50051"
	}

	// Get test registration token from environment
	// You need to create a server in the dashboard and get a token
	token := os.Getenv("WATCHFLARE_TEST_TOKEN")
	if token == "" {
		t.Skip("Skipping integration test: WATCHFLARE_TEST_TOKEN not set")
	}

	// Step 1: Collect system information
	t.Log("Step 1: Collecting system information...")
	info, err := sysinfo.Collect()
	if err != nil {
		t.Fatalf("Failed to collect system info: %v", err)
	}

	t.Logf("  Hostname: %s", info.Hostname)
	t.Logf("  OS: %s %s", info.OS, info.OSVersion)
	t.Logf("  IPv4: %s", info.IPv4Address)
	t.Logf("  IPv6: %s", info.IPv6Address)

	// Step 2: Connect to backend
	t.Log("Step 2: Connecting to backend...")
	grpcClient, err := client.New(backendHost, backendPort)
	if err != nil {
		t.Fatalf("Failed to connect to backend: %v", err)
	}
	defer grpcClient.Close()

	t.Logf("  Connected to %s:%s", backendHost, backendPort)

	// Step 3: Register with backend
	t.Log("Step 3: Registering agent...")
	agentID, agentKey, err := grpcClient.Register(
		token,
		info.Hostname,
		info.IPv4Address,
		info.IPv6Address,
		info.OS,
		info.OSVersion,
	)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	t.Logf("  ✅ Registration successful!")
	t.Logf("  Agent ID: %s", agentID)
	t.Logf("  Agent Key: %s", agentKey[:20]+"...")

	// Verify we got valid credentials
	if agentID == "" {
		t.Error("Agent ID is empty")
	}

	if agentKey == "" {
		t.Error("Agent Key is empty")
	}

	// Step 4: Save configuration
	t.Log("Step 4: Saving configuration...")
	cfg := &config.Config{
		ServerHost: backendHost,
		ServerPort: backendPort,
		AgentID:    agentID,
		AgentKey:   agentKey,
	}

	if err := config.Save(cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	t.Logf("  Config saved to: %s", filepath.Join(config.GetConfigDir(), config.ConfigFile))

	// Step 5: Verify configuration can be loaded
	t.Log("Step 5: Verifying configuration...")
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.AgentID != agentID {
		t.Errorf("Loaded AgentID = %v, want %v", loadedCfg.AgentID, agentID)
	}

	if loadedCfg.AgentKey != agentKey {
		t.Errorf("Loaded AgentKey doesn't match")
	}

	t.Log("  ✅ Configuration verified!")

	// Step 6: Send heartbeat
	t.Log("Step 6: Sending heartbeat...")
	err = grpcClient.SendHeartbeat(
		agentID,
		agentKey,
		info.IPv4Address,
		info.IPv6Address,
	)

	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	t.Log("  ✅ Heartbeat sent successfully!")

	// Step 7: Send multiple heartbeats to simulate ongoing monitoring
	t.Log("Step 7: Sending 3 heartbeats (5s interval)...")
	for i := 1; i <= 3; i++ {
		time.Sleep(5 * time.Second)

		// Get fresh IP addresses
		freshInfo, err := sysinfo.Collect()
		if err != nil {
			t.Fatalf("Failed to collect system info: %v", err)
		}

		err = grpcClient.SendHeartbeat(
			agentID,
			agentKey,
			freshInfo.IPv4Address,
			freshInfo.IPv6Address,
		)

		if err != nil {
			t.Fatalf("Heartbeat %d failed: %v", i, err)
		}

		t.Logf("  ✅ Heartbeat %d/3 sent", i)
	}

	t.Log("✅ End-to-end integration test completed successfully!")
}

// TestEndToEndWithExistingConfig tests loading existing config and sending heartbeats
func TestEndToEndWithExistingConfig(t *testing.T) {
	// Check if config exists
	if !config.Exists() {
		t.Skip("Skipping test: no existing config found")
	}

	// Load existing config
	t.Log("Loading existing configuration...")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	t.Logf("Loaded config: AgentID=%s", cfg.AgentID)

	// Connect to backend
	t.Log("Connecting to backend...")
	grpcClient, err := client.New(cfg.ServerHost, cfg.ServerPort)
	if err != nil {
		t.Fatalf("Failed to connect to backend: %v", err)
	}
	defer grpcClient.Close()

	// Get system info
	info, err := sysinfo.Collect()
	if err != nil {
		t.Fatalf("Failed to collect system info: %v", err)
	}

	// Send heartbeat
	t.Log("Sending heartbeat...")
	err = grpcClient.SendHeartbeat(
		cfg.AgentID,
		cfg.AgentKey,
		info.IPv4Address,
		info.IPv6Address,
	)

	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	t.Log("✅ Heartbeat sent successfully!")
}

// setupIntegrationTest prepares the test environment
func setupIntegrationTest(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "watchflare-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set environment variables to use temp directory
	os.Setenv("WATCHFLARE_CONFIG_DIR", filepath.Join(tmpDir, "config"))
	os.Setenv("WATCHFLARE_DATA_DIR", filepath.Join(tmpDir, "data"))
	os.Setenv("WATCHFLARE_LOG_DIR", filepath.Join(tmpDir, "logs"))

	// Create directories
	if err := config.EnsureDirectories(); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	return tmpDir
}
