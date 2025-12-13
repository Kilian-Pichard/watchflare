package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateAgent creates a new server with status "pending"
func CreateAgent(name, serverType, configuredIP string, allowAnyIP bool) (*models.Server, string, string, error) {
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
		Type:                   serverType,
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

// ListServers returns all servers
func ListServers() ([]models.Server, error) {
	var servers []models.Server
	if err := database.DB.Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

// GetServer returns a server by ID
func GetServer(serverID string) (*models.Server, error) {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("server not found")
		}
		return nil, err
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

	server.ConfiguredIP = &newIP

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
