package services

import (
	"os"
	"strings"
	"testing"
	"time"
	"watchflare/backend/cache"
	"watchflare/backend/config"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"github.com/stretchr/testify/assert"
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

func setupTestDB(t *testing.T) {
	t.Helper()
	config.AppConfig = &config.Config{
		JWTSecret: "test-secret-key-must-be-32-chars!!",
	}
	if err := database.Connect(testDSN()); err != nil {
		t.Skipf("skipping test: database unavailable: %v", err)
	}
}

func teardownTestDB() {
	database.DB.Exec("DELETE FROM servers")
}

func TestCreateAgent(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, token, agentKey, err := CreateAgent("server01", "192.168.1.100", false)

	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, agentKey)

	// Verify server fields
	assert.Equal(t, "server01", server.Name)
	assert.Equal(t, "192.168.1.100", *server.ConfiguredIP)
	assert.False(t, server.AllowAnyIPRegistration)
	assert.Equal(t, models.StatusPending, server.Status)

	// Verify agent ID is UUID
	assert.Len(t, server.AgentID, 36) // UUID length

	// Verify token format (wf_reg_{32_chars})
	assert.True(t, strings.HasPrefix(token, "wf_reg_"))
	assert.Len(t, token, 39) // "wf_reg_" + 32 chars

	// Verify agent key is AES-256 (64 hex chars)
	assert.Len(t, agentKey, 64)

	// Verify registration token is hashed in DB
	assert.NotNil(t, server.RegistrationToken)
	assert.NotEqual(t, token, *server.RegistrationToken) // Should be hashed

	// Verify expiration is ~24 hours from now
	assert.NotNil(t, server.ExpiresAt)
	expectedExpiry := time.Now().Add(time.Hour * 24)
	assert.WithinDuration(t, expectedExpiry, *server.ExpiresAt, time.Minute)

	// Verify server is saved in DB
	var dbServer models.Server
	database.DB.Where("id = ?", server.ID).First(&dbServer)
	assert.Equal(t, server.ID, dbServer.ID)
	assert.Equal(t, models.StatusPending, dbServer.Status)
}

func TestListServers(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	// Create test servers
	CreateAgent("server01", "192.168.1.100", false)
	CreateAgent("server02", "192.168.1.101", true)
	CreateAgent("server03", "192.168.1.102", false)

	servers, _, err := ListServers(ServerListParams{})

	assert.NoError(t, err)
	assert.Len(t, servers, 3)

	// Verify servers are returned
	names := []string{servers[0].Name, servers[1].Name, servers[2].Name}
	assert.Contains(t, names, "server01")
	assert.Contains(t, names, "server02")
	assert.Contains(t, names, "server03")
}

func TestListServers_Empty(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	servers, _, err := ListServers(ServerListParams{})

	assert.NoError(t, err)
	assert.Empty(t, servers)
}

func TestGetServer(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	// Create test server
	createdServer, _, _, _ := CreateAgent("server01", "192.168.1.100", false)

	// Get server
	server, err := GetServer(createdServer.ID)

	assert.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, createdServer.ID, server.ID)
	assert.Equal(t, "server01", server.Name)
}

func TestGetServer_NotFound(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, err := GetServer("00000000-0000-0000-0000-000000000000")

	assert.Error(t, err)
	assert.Nil(t, server)
	assert.Contains(t, err.Error(), "not found")
}

func TestValidateIP(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	// Create test server
	server, _, _, _ := CreateAgent("server01", "192.168.1.100", false)

	// Validate IP
	err := ValidateIP(server.ID, "192.168.1.150")

	assert.NoError(t, err)

	// Verify IP was updated and configured_ip cleared
	updatedServer, _ := GetServer(server.ID)
	assert.Equal(t, "192.168.1.150", *updatedServer.IPAddressV4)
	assert.Nil(t, updatedServer.ConfiguredIP)
}

func TestUpdateConfiguredIP(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	// Create test server
	server, _, _, _ := CreateAgent("server01", "192.168.1.100", false)

	// Update configured IP
	err := UpdateConfiguredIP(server.ID, "192.168.1.200")

	assert.NoError(t, err)

	// Verify IP was updated
	updatedServer, _ := GetServer(server.ID)
	assert.Equal(t, "192.168.1.200", *updatedServer.ConfiguredIP)
}

func TestRegenerateToken(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	// Create test server
	server, originalToken, _, _ := CreateAgent("server01", "192.168.1.100", false)

	// Regenerate token
	newToken, err := RegenerateToken(server.ID)

	assert.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, originalToken, newToken)

	// Verify token format
	assert.True(t, strings.HasPrefix(newToken, "wf_reg_"))

	// Verify expiration was updated
	updatedServer, _ := GetServer(server.ID)
	expectedExpiry := time.Now().Add(time.Hour * 24)
	assert.WithinDuration(t, expectedExpiry, *updatedServer.ExpiresAt, time.Minute)
	assert.Equal(t, models.StatusPending, updatedServer.Status)
}

func TestRegenerateToken_OnlineServer(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("server01", "192.168.1.100", false)
	database.DB.Model(&server).Update("status", models.StatusOnline)

	// Regenerating token on an online server is allowed — agent will re-register.
	newToken, err := RegenerateToken(server.ID)

	assert.NoError(t, err)
	assert.NotEmpty(t, newToken)

	updatedServer, _ := GetServer(server.ID)
	assert.Equal(t, models.StatusPending, updatedServer.Status)
}

func TestDeleteServer(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	// Create test server
	server, _, _, _ := CreateAgent("server01", "192.168.1.100", false)

	// Delete server
	err := DeleteServer(server.ID)

	assert.NoError(t, err)

	// Verify server was deleted
	_, err = GetServer(server.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeleteServer_OnlineServer(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	// Create server and set to online
	server, _, _, _ := CreateAgent("server01", "192.168.1.100", false)
	database.DB.Model(&server).Update("status", models.StatusOnline)

	// Delete server (should succeed now - restriction removed)
	err := DeleteServer(server.ID)

	assert.NoError(t, err)

	// Verify server was deleted
	var deletedServer models.Server
	err = database.DB.Where("id = ?", server.ID).First(&deletedServer).Error
	assert.Error(t, err) // Should not find the server
}

func TestDeleteServer_NotFound(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	err := DeleteServer("00000000-0000-0000-0000-000000000000")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListAllServers(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	CreateAgent("server01", "192.168.1.1", false)
	CreateAgent("server02", "192.168.1.2", false)

	servers, err := ListAllServers()

	assert.NoError(t, err)
	assert.Len(t, servers, 2)
}

func TestListServers_Pagination(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	CreateAgent("server01", "192.168.1.1", false)
	CreateAgent("server02", "192.168.1.2", false)
	CreateAgent("server03", "192.168.1.3", false)

	servers, total, err := ListServers(ServerListParams{Page: 1, PerPage: 2})

	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, servers, 2)
}

func TestListServers_PaginationBeyondEnd(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	CreateAgent("server01", "192.168.1.1", false)

	servers, total, err := ListServers(ServerListParams{Page: 99, PerPage: 10})

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Empty(t, servers)
}

func TestListServers_SearchFilter(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	CreateAgent("web-server", "192.168.1.1", false)
	CreateAgent("db-server", "192.168.1.2", false)

	servers, total, err := ListServers(ServerListParams{Search: "web"})

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "web-server", servers[0].Name)
}

func TestRenameServer(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("old-name", "192.168.1.1", false)

	err := RenameServer(server.ID, "new-name")

	assert.NoError(t, err)
	updated, _ := GetServer(server.ID)
	assert.Equal(t, "new-name", updated.Name)
}

func TestRenameServer_TooShort(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("old-name", "192.168.1.1", false)

	err := RenameServer(server.ID, "x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "between 2 and 64")
}

func TestRenameServer_TooLong(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("old-name", "192.168.1.1", false)

	err := RenameServer(server.ID, strings.Repeat("a", 65))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "between 2 and 64")
}

func TestIgnoreIPMismatch(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("server01", "192.168.1.1", false)

	err := IgnoreIPMismatch(server.ID)

	assert.NoError(t, err)
	updated, _ := GetServer(server.ID)
	assert.True(t, updated.IgnoreIPMismatch)
}

func TestDismissReactivation(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("server01", "192.168.1.1", false)
	now := time.Now()
	database.DB.Model(&server).Update("reactivated_at", now)

	err := DismissReactivation(server.ID)

	assert.NoError(t, err)
	updated, _ := GetServer(server.ID)
	assert.Nil(t, updated.ReactivatedAt)
}

func TestPauseServer(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("server01", "192.168.1.1", false)
	database.DB.Model(&server).Update("status", models.StatusOnline)

	err := PauseServer(server.ID)

	assert.NoError(t, err)
	updated, _ := GetServer(server.ID)
	assert.Equal(t, models.StatusPaused, updated.Status)
}

func TestPauseServer_AlreadyPaused(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("server01", "192.168.1.1", false)
	database.DB.Model(&server).Update("status", models.StatusPaused)

	err := PauseServer(server.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already paused")
}

func TestPauseServer_Pending(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("server01", "192.168.1.1", false)

	err := PauseServer(server.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot pause a pending")
}

func TestResumeServer(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("server01", "192.168.1.1", false)
	database.DB.Model(&server).Update("status", models.StatusPaused)

	err := ResumeServer(server.ID)

	assert.NoError(t, err)
	updated, _ := GetServer(server.ID)
	assert.Equal(t, models.StatusOnline, updated.Status)
}

func TestResumeServer_NotPaused(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, _ := CreateAgent("server01", "192.168.1.1", false)
	database.DB.Model(&server).Update("status", models.StatusOnline)

	err := ResumeServer(server.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not paused")
}

func TestGenerateRegistrationToken(t *testing.T) {
	token1, hash1, err1 := generateRegistrationToken()
	token2, hash2, err2 := generateRegistrationToken()

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	assert.True(t, strings.HasPrefix(token1, "wf_reg_"))
	assert.Len(t, token1, 39) // "wf_reg_" + 32 hex chars

	assert.NotEqual(t, token1, token2)
	assert.NotEqual(t, hash1, hash2)

	// Hash must differ from plaintext token.
	assert.NotEqual(t, token1, hash1)
	assert.Len(t, hash1, 64) // SHA-256 hex
}

func TestMergeCache(t *testing.T) {
	c := cache.GetCache()
	c.Remove("agent-abc")
	c.Remove("agent-xyz")
	defer c.Remove("agent-abc")

	c.Update("agent-abc", "10.0.0.1", "::1")

	servers := []models.Server{
		{AgentID: "agent-abc", Status: models.StatusPending},
		{AgentID: "agent-xyz", Status: models.StatusPending}, // not in cache
	}
	mergeCache(servers)

	// Agent in cache: status and IPs must be overridden.
	if servers[0].Status != models.StatusOnline {
		t.Errorf("status: got %s, want online", servers[0].Status)
	}
	if servers[0].IPAddressV4 == nil || *servers[0].IPAddressV4 != "10.0.0.1" {
		t.Errorf("ipv4: got %v, want 10.0.0.1", servers[0].IPAddressV4)
	}
	if servers[0].IPAddressV6 == nil || *servers[0].IPAddressV6 != "::1" {
		t.Errorf("ipv6: got %v, want ::1", servers[0].IPAddressV6)
	}

	// Agent not in cache: status must be unchanged.
	if servers[1].Status != models.StatusPending {
		t.Errorf("status: got %s, want pending", servers[1].Status)
	}
}

func TestListServers_PageZeroTreatedAsOne(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	CreateAgent("server01", "192.168.1.1", false)
	CreateAgent("server02", "192.168.1.2", false)

	// Page=0 must not panic and must behave like Page=1.
	servers, total, err := ListServers(ServerListParams{Page: 0, PerPage: 1})

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, servers, 1)
}

func TestCreateAgent_EmptyConfiguredIP(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	server, _, _, err := CreateAgent("server01", "", false)

	assert.NoError(t, err)
	assert.Nil(t, server.ConfiguredIP) // empty string must not be stored as non-nil pointer
}

func TestHashToken(t *testing.T) {
	token1 := "test_token_123"
	token2 := "test_token_456"

	hash1 := hashToken(token1)
	hash2 := hashToken(token2)

	// Verify hashes are different
	assert.NotEqual(t, hash1, hash2)

	// Verify same token produces same hash
	hash1Again := hashToken(token1)
	assert.Equal(t, hash1, hash1Again)

	// Verify hash is SHA-256 (64 hex chars)
	assert.Len(t, hash1, 64)
}
