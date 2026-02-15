package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
	"watchflare/backend/cache"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateAgent creates a new server with status "pending"
func CreateAgent(name, configuredIP string, allowAnyIP bool) (*models.Server, string, string, error) {
	// Generate agent ID
	agentID := uuid.New().String()

	// Generate registration token (wf_reg_{32_random_chars})
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", "", err
	}
	token := fmt.Sprintf("wf_reg_%s", hex.EncodeToString(tokenBytes))

	// Hash the token for storage
	hashedToken := hashToken(token)

	// Generate agent key (AES-256 = 32 bytes)
	agentKeyBytes := make([]byte, 32)
	if _, err := rand.Read(agentKeyBytes); err != nil {
		return nil, "", "", err
	}
	agentKey := hex.EncodeToString(agentKeyBytes)

	// Set expiration date
	expiresAt := time.Now().Add(time.Hour * 24) // 24 hours

	// Create server with status "pending"
	server := &models.Server{
		ID:                     uuid.New().String(),
		AgentID:                agentID,
		AgentKey:               agentKey,
		Name:                   name,
		ConfiguredIP:           &configuredIP,
		AllowAnyIPRegistration: allowAnyIP,
		RegistrationToken:      &hashedToken,
		ExpiresAt:              &expiresAt,
		Status:                 "pending",
	}

	if err := database.DB.Create(server).Error; err != nil {
		return nil, "", "", err
	}

	return server, token, agentKey, nil
}

// ListServers returns all servers with real-time status from cache
func ListServers(page, perPage int) ([]models.Server, int64, error) {
	var total int64
	if err := database.DB.Model(&models.Server{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var servers []models.Server
	query := database.DB.Order("created_at DESC")
	if perPage > 0 {
		query = query.Offset((page - 1) * perPage).Limit(perPage)
	}
	if err := query.Find(&servers).Error; err != nil {
		return nil, 0, err
	}

	// Merge with cache data for real-time status (cache may have more recent status than DB)
	heartbeatCache := cache.GetCache()
	for i := range servers {
		if cachedData, ok := heartbeatCache.Get(servers[i].AgentID); ok {
			// Update status and last_seen from cache if available
			servers[i].Status = cachedData.Status
			servers[i].LastSeen = &cachedData.LastSeen
			// Also update IPs if they changed
			if cachedData.IPv4Address != "" {
				servers[i].IPAddressV4 = &cachedData.IPv4Address
			}
			if cachedData.IPv6Address != "" {
				servers[i].IPAddressV6 = &cachedData.IPv6Address
			}
		}
	}

	return servers, total, nil
}

// ListAllServers returns all servers without pagination (for dashboard/SSE)
func ListAllServers() ([]models.Server, error) {
	var servers []models.Server
	if err := database.DB.Find(&servers).Error; err != nil {
		return nil, err
	}

	heartbeatCache := cache.GetCache()
	for i := range servers {
		if cachedData, ok := heartbeatCache.Get(servers[i].AgentID); ok {
			servers[i].Status = cachedData.Status
			servers[i].LastSeen = &cachedData.LastSeen
			if cachedData.IPv4Address != "" {
				servers[i].IPAddressV4 = &cachedData.IPv4Address
			}
			if cachedData.IPv6Address != "" {
				servers[i].IPAddressV6 = &cachedData.IPv6Address
			}
		}
	}

	return servers, nil
}

// GetServer returns a server by ID with real-time status from cache
func GetServer(serverID string) (*models.Server, error) {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("server not found")
		}
		return nil, err
	}

	// Merge with cache data for real-time status
	heartbeatCache := cache.GetCache()
	if cachedData, ok := heartbeatCache.Get(server.AgentID); ok {
		server.Status = cachedData.Status
		server.LastSeen = &cachedData.LastSeen
		if cachedData.IPv4Address != "" {
			server.IPAddressV4 = &cachedData.IPv4Address
		}
		if cachedData.IPv6Address != "" {
			server.IPAddressV6 = &cachedData.IPv6Address
		}
	}

	return &server, nil
}

// ValidateIP validates and updates the server IP
func ValidateIP(serverID string, selectedIP string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil{
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("server not found")
		}
		return err
	}

	// Update the IP and clear configured_ip
	server.IPAddressV4 = &selectedIP
	server.ConfiguredIP = nil

	if err := database.DB.Save(&server).Error; err != nil {
		return err
	}

	return nil
}

// UpdateConfiguredIP changes the configured IP for a server
func UpdateConfiguredIP(serverID string, newIP string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("server not found")
		}
		return err
	}

	// Save current configured IP to previous_configured_ip
	if server.ConfiguredIP != nil && *server.ConfiguredIP != "" {
		server.PreviousConfiguredIP = server.ConfiguredIP
	}

	// Update to new IP
	server.ConfiguredIP = &newIP

	// Reset ignore flag when IP is updated
	server.IgnoreIPMismatch = false

	if err := database.DB.Save(&server).Error; err != nil {
		return err
	}

	return nil
}

// IgnoreIPMismatch marks the IP mismatch warning as ignored by the user
func IgnoreIPMismatch(serverID string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("server not found")
		}
		return err
	}

	// Mark the mismatch as ignored
	server.IgnoreIPMismatch = true

	if err := database.DB.Save(&server).Error; err != nil {
		return err
	}

	return nil
}

// DismissReactivation clears the reactivation badge for an agent
func DismissReactivation(serverID string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("server not found")
		}
		return err
	}

	// Clear reactivated_at timestamp
	server.ReactivatedAt = nil

	if err := database.DB.Save(&server).Error; err != nil {
		return err
	}

	return nil
}

// RegenerateToken generates a new registration token for an expired server
func RegenerateToken(serverID string) (string, error) {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("server not found")
		}
		return "", err
	}

	// Only allow regeneration for pending or expired servers
	if server.Status != "pending" && server.Status != "expired" {
		return "", errors.New("can only regenerate token for pending or expired servers")
	}

	// Generate new registration token
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := fmt.Sprintf("wf_reg_%s", hex.EncodeToString(tokenBytes))

	// Hash the token for storage
	hashedToken := hashToken(token)

	// Update server with new token and expiration
	expiresAt := time.Now().Add(time.Hour * 24) // 24 hours
	server.RegistrationToken = &hashedToken
	server.ExpiresAt = &expiresAt
	server.Status = "pending"

	if err := database.DB.Save(&server).Error; err != nil {
		return "", err
	}

	return token, nil
}

// DeleteServer deletes a server
func DeleteServer(serverID string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("server not found")
		}
		return err
	}

	if err := database.DB.Delete(&server).Error; err != nil {
		return err
	}

	return nil
}

// hashToken hashes a registration token using SHA-256
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
