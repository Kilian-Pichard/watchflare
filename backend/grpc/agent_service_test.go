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
	pb "watchflare/shared/proto/agent/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDSN() string {
	get := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}
	return "host=" + get("POSTGRES_HOST", "localhost") +
		" port=" + get("POSTGRES_PORT", "5432") +
		" user=" + get("POSTGRES_USER", "watchflare") +
		" password=" + get("POSTGRES_PASSWORD", "watchflare_dev") +
		" dbname=" + get("POSTGRES_TEST_DB", "watchflare_test") +
		" sslmode=" + get("POSTGRES_SSLMODE", "disable")
}

// setupGRPCTestDB connects to the local PostgreSQL database for testing.
func setupGRPCTestDB(t *testing.T) {
	t.Helper()
	if err := database.Connect(testDSN()); err != nil {
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

// createPendingHost inserts a pending host with a registration token into the DB.
func createPendingHost(t *testing.T, token string) *models.Host {
	t.Helper()
	expiry := time.Now().Add(24 * time.Hour)
	host := &models.Host{
		ID:                     uuid.New().String(),
		AgentID:                uuid.New().String(),
		DisplayName: "test-host-" + token[:8],
		Status:                 models.StatusPending,
		RegistrationToken:      strPtr(hashTestToken(token)),
		ExpiresAt:              &expiry,
		AllowAnyIPRegistration: true,
		AgentKey:               "test-agent-key-" + token[:8],
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() {
		database.DB.Unscoped().Delete(host)
	})
	return host
}

func strPtr(s string) *string { return &s }

// --- Tests ---

func TestRegisterHost_InvalidToken(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	req := &pb.RegisterHostRequest{
		RegistrationToken: "wf_reg_doesnotexist",
		Hostname:          "test-host",
	}
	resp, err := s.RegisterHost(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "Invalid registration token", resp.Message)
}

func TestRegisterHost_AlreadyRegistered(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	const token = "wf_reg_alreadyregistered01"

	expiry := time.Now().Add(24 * time.Hour)
	host := &models.Host{
		ID:                     uuid.New().String(),
		AgentID:                uuid.New().String(),
		DisplayName: "already-registered",
		Status:                 models.StatusOnline,
		RegistrationToken:      strPtr(hashTestToken(token)),
		ExpiresAt:              &expiry,
		AllowAnyIPRegistration: true,
		AgentKey:               "some-key",
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(host) })

	req := &pb.RegisterHostRequest{
		RegistrationToken: token,
		Hostname:          "test-host",
	}
	resp, err := s.RegisterHost(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "Host is already registered", resp.Message)
}

func TestRegisterHost_ExpiredToken(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	const token = "wf_reg_expiredtoken00001"
	expiry := time.Now().Add(-1 * time.Hour) // already expired
	host := &models.Host{
		ID:                     uuid.New().String(),
		AgentID:                uuid.New().String(),
		DisplayName: "expired-host",
		Status:                 models.StatusPending,
		RegistrationToken:      strPtr(hashTestToken(token)),
		ExpiresAt:              &expiry,
		AllowAnyIPRegistration: true,
		AgentKey:               "some-key",
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(host) })

	req := &pb.RegisterHostRequest{
		RegistrationToken: token,
		Hostname:          "test-host",
	}
	resp, err := s.RegisterHost(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "Registration token has expired", resp.Message)
}

func TestRegisterHost_Success(t *testing.T) {
	setupGRPCTestDB(t)
	setupTestPKI(t)
	s := NewAgentServer()

	const token = "wf_reg_successtoken0001"
	host := createPendingHost(t, token)

	req := &pb.RegisterHostRequest{
		RegistrationToken: token,
		Hostname:          "my-host",
		IpAddressV4:       "1.2.3.4",
		Platform:          "Linux",
		AgentVersion:      "0.28.0",
	}
	resp, err := s.RegisterHost(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.AgentId)
	assert.Equal(t, host.AgentKey, resp.AgentKey)

	// Token must be cleared in DB after successful registration
	var updated models.Host
	require.NoError(t, database.DB.First(&updated, "id = ?", host.ID).Error)
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

func TestSendMetrics_PausedHost(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	host := &models.Host{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		DisplayName: "paused-host",
		Status:   models.StatusPaused,
		AgentKey: "paused-agent-key-abc123",
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(host) })

	req := &pb.SendMetricsRequest{
		AgentId:  host.AgentID,
		AgentKey: host.AgentKey,
		Metrics:  &pb.Metrics{Timestamp: time.Now().Unix()},
	}
	resp, err := s.SendMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Message, "paused")
}

func TestSendMetrics_NilMetrics(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	host := &models.Host{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		DisplayName: "online-host",
		Status:   models.StatusOnline,
		AgentKey: "online-agent-key-abc123",
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(host) })

	req := &pb.SendMetricsRequest{
		AgentId:  host.AgentID,
		AgentKey: host.AgentKey,
		Metrics:  nil,
	}
	resp, err := s.SendMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "required")
}

func TestHeartbeat_InvalidCredentials(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	req := &pb.HeartbeatRequest{
		AgentId:  "00000000-0000-0000-0000-000000000000",
		AgentKey: "invalid-key",
	}
	resp, err := s.Heartbeat(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "Invalid agent credentials", resp.Message)
}

func TestHeartbeat_PausedHost(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	host := &models.Host{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		DisplayName: "paused-hb-host",
		Status:   models.StatusPaused,
		AgentKey: "paused-hb-key-abc123",
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(host) })

	req := &pb.HeartbeatRequest{
		AgentId:  host.AgentID,
		AgentKey: host.AgentKey,
	}
	resp, err := s.Heartbeat(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Message, "paused")
}

func TestHeartbeat_Online(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	host := &models.Host{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		DisplayName: "online-hb-host",
		Status:   models.StatusOnline,
		AgentKey: "online-hb-key-abc123",
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(host) })

	req := &pb.HeartbeatRequest{
		AgentId:     host.AgentID,
		AgentKey:    host.AgentKey,
		IpAddressV4: "10.0.0.1",
	}
	resp, err := s.Heartbeat(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestReportDroppedMetrics_InvalidCredentials(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	req := &pb.ReportDroppedMetricsRequest{
		AgentId:  "00000000-0000-0000-0000-000000000000",
		AgentKey: "invalid-key",
	}
	resp, err := s.ReportDroppedMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestReportDroppedMetrics_Success(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	host := &models.Host{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		DisplayName: "drop-host",
		Status:   models.StatusOnline,
		AgentKey: "drop-agent-key-abc123",
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(host) })

	now := time.Now().Unix()
	req := &pb.ReportDroppedMetricsRequest{
		AgentId:        host.AgentID,
		AgentKey:       host.AgentKey,
		Count:          5,
		FirstDroppedAt: now - 60,
		LastDroppedAt:  now,
		Reason:         "max_retries_exceeded",
	}
	resp, err := s.ReportDroppedMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestSendPackageInventory_InvalidCredentials(t *testing.T) {
	setupGRPCTestDB(t)
	s := NewAgentServer()

	req := &pb.SendPackageInventoryRequest{
		AgentId:  "00000000-0000-0000-0000-000000000000",
		AgentKey: "invalid-key",
	}
	resp, err := s.SendPackageInventory(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestProcessPackageInventory_UnknownType(t *testing.T) {
	setupGRPCTestDB(t)

	host := &models.Host{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		DisplayName: "pkg-inv-host",
		Status:   models.StatusOnline,
		AgentKey: "pkg-inv-key-abc123",
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(host) })

	req := &pb.SendPackageInventoryRequest{
		AgentId:       host.AgentID,
		AgentKey:      host.AgentKey,
		InventoryType: "unknown_type",
	}
	_, _, err := processPackageInventory(host.ID, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown inventory_type")
}

func TestProcessPackageInventory_FullInventory(t *testing.T) {
	setupGRPCTestDB(t)

	host := &models.Host{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		DisplayName: "pkg-full-host",
		Status:   models.StatusOnline,
		AgentKey: "pkg-full-key-abc123",
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(host) })

	req := &pb.SendPackageInventoryRequest{
		InventoryType: models.CollectionTypeFull,
		AllPackages: []*pb.Package{
			{Name: "curl", Version: "7.88.0", PackageManager: "apt"},
			{Name: "git", Version: "2.39.0", PackageManager: "apt"},
		},
		TotalPackageCount: 2,
	}
	processed, changes, err := processPackageInventory(host.ID, req)
	require.NoError(t, err)
	assert.Equal(t, 2, processed)
	assert.Equal(t, 2, changes)
}

func TestProcessPackageInventory_DeltaInventory(t *testing.T) {
	setupGRPCTestDB(t)

	host := &models.Host{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		DisplayName: "pkg-delta-host",
		Status:   models.StatusOnline,
		AgentKey: "pkg-delta-key-abc123",
	}
	require.NoError(t, database.DB.Create(host).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(host) })

	// Seed an existing package to remove/update
	fullReq := &pb.SendPackageInventoryRequest{
		InventoryType: models.CollectionTypeFull,
		AllPackages: []*pb.Package{
			{Name: "curl", Version: "7.88.0", PackageManager: "apt"},
			{Name: "vim", Version: "8.2", PackageManager: "apt"},
		},
		TotalPackageCount: 2,
	}
	_, _, err := processPackageInventory(host.ID, fullReq)
	require.NoError(t, err)

	deltaReq := &pb.SendPackageInventoryRequest{
		InventoryType:     models.CollectionTypeDelta,
		AddedPackages:     []*pb.Package{{Name: "htop", Version: "3.2.0", PackageManager: "apt"}},
		RemovedPackages:   []*pb.Package{{Name: "vim", Version: "8.2", PackageManager: "apt"}},
		UpdatedPackages:   []*pb.Package{{Name: "curl", Version: "8.0.0", PackageManager: "apt"}},
		TotalPackageCount: 2,
	}
	processed, changes, err := processPackageInventory(host.ID, deltaReq)
	require.NoError(t, err)
	assert.Equal(t, 2, processed) // TotalPackageCount
	assert.Equal(t, 3, changes)   // 1 added + 1 removed + 1 updated
}
