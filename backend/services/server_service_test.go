package services

import (
	"strings"
	"testing"
	"time"
	"watchflare/backend/config"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) {
	config.AppConfig = &config.Config{
		DBPath:    ":memory:",
		JWTSecret: "test-secret-key",
	}

	if err := database.Connect(config.AppConfig.DBPath); err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
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
	assert.Equal(t, "pending", server.Status)

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
	assert.Equal(t, "pending", dbServer.Status)
}

func TestListServers(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	// Create test servers
	CreateAgent("server01", "192.168.1.100", false)
	CreateAgent("server02", "192.168.1.101", true)
	CreateAgent("server03", "192.168.1.102", false)

	servers, err := ListServers()

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

	servers, err := ListServers()

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
	assert.Equal(t, "pending", updatedServer.Status)
}

func TestRegenerateToken_OnlineServer(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	// Create server and set to online
	server, _, _, _ := CreateAgent("server01", "192.168.1.100", false)
	database.DB.Model(&server).Update("status", "online")

	// Try to regenerate token
	_, err := RegenerateToken(server.ID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can only regenerate token")
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
	database.DB.Model(&server).Update("status", "online")

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
