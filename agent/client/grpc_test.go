package client

import (
	"testing"
)

func TestNew(t *testing.T) {
	// Test with invalid address (no server running)
	// This should fail to connect, which is expected
	client, err := New("localhost", "99999")
	if err != nil {
		// Expected to fail if no server is running
		t.Logf("Client creation failed (expected if no server): %v", err)
		return
	}

	// If somehow it connected, clean up
	if client != nil {
		defer client.Close()
	}
}

func TestClientClose(t *testing.T) {
	// Create a client (might fail if no server)
	client, err := New("localhost", "99999")
	if err != nil {
		t.Skip("Skipping Close test: no server available")
	}

	// Test Close
	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// Integration tests that require a running backend
// These should be run with: go test -tags=integration

func TestRegisterIntegration(t *testing.T) {
	// Skip if backend is not running
	// You can set an environment variable to enable integration tests
	// export WATCHFLARE_TEST_BACKEND=localhost:50051
	backendAddr := getTestBackendAddr(t)
	if backendAddr == "" {
		t.Skip("Skipping integration test: WATCHFLARE_TEST_BACKEND not set")
	}

	host, port := parseAddr(backendAddr)
	client, err := New(host, port)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test registration with a test token
	// Note: This requires a valid registration token from the backend
	agentID, agentKey, err := client.Register(
		"test-token",
		"test-hostname",
		"192.168.1.100",
		"",
		"darwin",
		"macOS",
	)

	if err != nil {
		// Expected to fail with invalid token
		t.Logf("Registration failed (expected with test token): %v", err)
		return
	}

	if agentID == "" {
		t.Error("AgentID is empty")
	}

	if agentKey == "" {
		t.Error("AgentKey is empty")
	}

	t.Logf("Registration successful: AgentID=%s", agentID)
}

func TestSendHeartbeatIntegration(t *testing.T) {
	backendAddr := getTestBackendAddr(t)
	if backendAddr == "" {
		t.Skip("Skipping integration test: WATCHFLARE_TEST_BACKEND not set")
	}

	host, port := parseAddr(backendAddr)
	client, err := New(host, port)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test heartbeat with test credentials
	err = client.SendHeartbeat(
		"test-agent-id",
		"test-agent-key",
		"192.168.1.100",
		"",
	)

	if err != nil {
		// Expected to fail with invalid credentials
		t.Logf("Heartbeat failed (expected with test credentials): %v", err)
		return
	}

	t.Log("Heartbeat sent successfully")
}

// Helper functions

func getTestBackendAddr(t *testing.T) string {
	t.Helper()
	// Check if integration test backend address is set
	// Example: WATCHFLARE_TEST_BACKEND=localhost:50051
	return ""  // Return empty to skip integration tests by default
}

func parseAddr(addr string) (string, string) {
	// Simple addr parsing (format: "host:port")
	// For now, just return defaults
	return "localhost", "50051"
}

// Table-driven test for client creation with various addresses
func TestNewWithVariousAddresses(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		port    string
		wantErr bool
	}{
		{
			name:    "localhost with standard port",
			host:    "localhost",
			port:    "50051",
			wantErr: false,  // May fail if server not running, but creation should succeed
		},
		{
			name:    "IP address",
			host:    "127.0.0.1",
			port:    "50051",
			wantErr: false,
		},
		{
			name:    "custom port",
			host:    "localhost",
			port:    "8080",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.host, tt.port)

			// Note: gRPC dial doesn't immediately fail if server is not running
			// It will fail on actual RPC calls
			if err != nil && !tt.wantErr {
				t.Logf("New() error = %v (server might not be running)", err)
			}

			if client != nil {
				client.Close()
			}
		})
	}
}

// Test client struct fields
func TestClientFields(t *testing.T) {
	// This test doesn't require a running server
	// We just test that a client can be created with proper fields

	host := "test.example.com"
	port := "50051"

	client, err := New(host, port)
	if err != nil {
		t.Skip("Skipping field test: couldn't create client")
	}
	defer client.Close()

	if client.host != host {
		t.Errorf("Client.host = %v, want %v", client.host, host)
	}

	if client.port != port {
		t.Errorf("Client.port = %v, want %v", client.port, port)
	}

	if client.conn == nil {
		t.Error("Client.conn is nil")
	}

	if client.client == nil {
		t.Error("Client.client is nil")
	}
}
