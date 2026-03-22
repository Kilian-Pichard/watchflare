package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"watchflare/backend/database"
	"watchflare/backend/models"
	"watchflare/backend/pki"
	pb "watchflare/shared/proto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupGRPCTestDB connects to the local PostgreSQL database for testing.
func setupGRPCTestDB(t *testing.T) {
	t.Helper()
	if err := database.Connect(); err != nil {
		t.Skipf("skipping grpc tests: database unavailable: %v", err)
	}
}

// setupTestPKI creates a temporary PKI directory with a dummy CA cert.
func setupTestPKI(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	// Write a dummy CA cert (not a real cert — just needs to be readable)
	caPath := dir + "/ca.pem"
	err := os.WriteFile(caPath, []byte("-----BEGIN CERTIFICATE-----\ndummy\n-----END CERTIFICATE-----\n"), 0644)
	require.NoError(t, err)

	p, err := pki.New(&pki.Config{
		Mode:   pki.ModeAuto,
		PKIDir: dir,
	})
	require.NoError(t, err)
	SetPKI(p)
}

// hashTestToken computes SHA-256 of a registration token (matches hashToken in agent_service.go).
func hashTestToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// createPendingServer inserts a pending server with a registration token into the DB.
func createPendingServer(t *testing.T, token string) *models.Server {
	t.Helper()
	expiry := time.Now().Add(24 * time.Hour)
	server := &models.Server{
		ID:                     uuid.New().String(),
		AgentID:                uuid.New().String(),
		Name:                   "test-server-" + token[:8],
		Status:                 "pending",
		RegistrationToken:      strPtr(hashTestToken(token)),
		ExpiresAt:              &expiry,
		AllowAnyIPRegistration: true,
		AgentKey:               "test-agent-key-" + token[:8],
	}
	require.NoError(t, database.DB.Create(server).Error)
	t.Cleanup(func() {
		database.DB.Unscoped().Delete(server)
	})
	return server
}

func strPtr(s string) *string { return &s }

// --- Tests ---

func TestRegisterServer_InvalidToken(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	req := &pb.RegisterServerRequest{
		RegistrationToken: "wf_reg_doesnotexist",
		Hostname:          "test-host",
	}
	resp, err := s.RegisterServer(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "Invalid registration token", resp.Message)
}

func TestRegisterServer_AlreadyRegistered(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	const token = "wf_reg_alreadyregistered01"

	expiry := time.Now().Add(24 * time.Hour)
	server := &models.Server{
		ID:                     uuid.New().String(),
		AgentID:                uuid.New().String(),
		Name:                   "already-registered",
		Status:                 "online",
		RegistrationToken:      strPtr(hashTestToken(token)),
		ExpiresAt:              &expiry,
		AllowAnyIPRegistration: true,
		AgentKey:               "some-key",
	}
	require.NoError(t, database.DB.Create(server).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(server) })

	req := &pb.RegisterServerRequest{
		RegistrationToken: token,
		Hostname:          "test-host",
	}
	resp, err := s.RegisterServer(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "Server is already registered", resp.Message)
}

func TestRegisterServer_ExpiredToken(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	const token = "wf_reg_expiredtoken00001"
	expiry := time.Now().Add(-1 * time.Hour) // already expired
	server := &models.Server{
		ID:                     uuid.New().String(),
		AgentID:                uuid.New().String(),
		Name:                   "expired-server",
		Status:                 "pending",
		RegistrationToken:      strPtr(hashTestToken(token)),
		ExpiresAt:              &expiry,
		AllowAnyIPRegistration: true,
		AgentKey:               "some-key",
	}
	require.NoError(t, database.DB.Create(server).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(server) })

	req := &pb.RegisterServerRequest{
		RegistrationToken: token,
		Hostname:          "test-host",
	}
	resp, err := s.RegisterServer(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "Registration token has expired", resp.Message)
}

func TestRegisterServer_Success(t *testing.T) {
	setupGRPCTestDB(t)
	setupTestPKI(t)
	s := NewAgentServer()

	const token = "wf_reg_successtoken0001"
	server := createPendingServer(t, token)

	req := &pb.RegisterServerRequest{
		RegistrationToken: token,
		Hostname:          "my-host",
		IpAddressV4:       "1.2.3.4",
		Platform:          "Linux",
		AgentVersion:      "0.28.0",
	}
	resp, err := s.RegisterServer(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.AgentId)
	assert.Equal(t, server.AgentKey, resp.AgentKey)

	// Token must be cleared in DB after successful registration
	var updated models.Server
	require.NoError(t, database.DB.First(&updated, "id = ?", server.ID).Error)
	assert.Nil(t, updated.RegistrationToken)
	assert.Equal(t, "offline", updated.Status)
}

func TestSendMetrics_InvalidCredentials(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	req := &pb.SendMetricsRequest{
		AgentId:  "00000000-0000-0000-0000-000000000000",
		AgentKey: "invalid-key",
		Metrics:  &pb.Metrics{Timestamp: time.Now().Unix()},
	}
	resp, err := s.SendMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "Invalid agent credentials", resp.Message)
}

func TestSendMetrics_PausedServer(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	server := &models.Server{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		Name:     "paused-server",
		Status:   "paused",
		AgentKey: "paused-agent-key-abc123",
	}
	require.NoError(t, database.DB.Create(server).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(server) })

	req := &pb.SendMetricsRequest{
		AgentId:  server.AgentID,
		AgentKey: server.AgentKey,
		Metrics:  &pb.Metrics{Timestamp: time.Now().Unix()},
	}
	resp, err := s.SendMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Message, "paused")
}
