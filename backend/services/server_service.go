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

// ServerListParams holds parameters for listing servers with sort/filter
type ServerListParams struct {
	Page        int
	PerPage     int
	Sort        string
	Order       string
	Status      string
	Search      string
	Environment string
}

// allowedSortColumns is a whitelist of columns that can be used for sorting
var allowedSortColumns = map[string]string{
	"name":       "name",
	"status":     "status",
	"ip":         "ip_address_v4",
	"last_seen":  "last_seen",
	"created_at": "created_at",
}

// ListServers returns servers with sort, filter and pagination
func ListServers(params ServerListParams) ([]models.Server, int64, error) {
	query := database.DB.Model(&models.Server{})

	// Apply search filter (name or hostname)
	if params.Search != "" {
		search := "%" + params.Search + "%"
		query = query.Where("name ILIKE ? OR hostname ILIKE ?", search, search)
	}

	// Apply environment filter
	if params.Environment != "" {
		query = query.Where("environment_type = ?", params.Environment)
	}

	// Apply sorting with whitelist validation
	sortColumn := "created_at"
	if col, ok := allowedSortColumns[params.Sort]; ok {
		sortColumn = col
	}
	sortOrder := "DESC"
	if params.Order == "asc" {
		sortOrder = "ASC"
	}

	// Status filter is applied after cache merge, so fetch all matching servers first
	// (cache has the real-time status, DB status may be stale)
	var allServers []models.Server
	if err := query.Order(sortColumn + " " + sortOrder).Find(&allServers).Error; err != nil {
		return nil, 0, err
	}

	// Merge with cache data for real-time status
	heartbeatCache := cache.GetCache()
	for i := range allServers {
		if cachedData, ok := heartbeatCache.Get(allServers[i].AgentID); ok {
			allServers[i].Status = cachedData.Status
			allServers[i].LastSeen = &cachedData.LastSeen
			if cachedData.IPv4Address != "" {
				allServers[i].IPAddressV4 = &cachedData.IPv4Address
			}
			if cachedData.IPv6Address != "" {
				allServers[i].IPAddressV6 = &cachedData.IPv6Address
			}
		}
	}

	// Apply status filter after cache merge (real-time status)
	if params.Status != "" {
		filtered := make([]models.Server, 0)
		for _, s := range allServers {
			if s.Status == params.Status {
				filtered = append(filtered, s)
			}
		}
		allServers = filtered
	}

	total := int64(len(allServers))

	// Apply pagination in memory (after status filter)
	if params.PerPage > 0 {
		start := (params.Page - 1) * params.PerPage
		if start >= int(total) {
			return []models.Server{}, total, nil
		}
		end := start + params.PerPage
		if end > int(total) {
			end = int(total)
		}
		allServers = allServers[start:end]
	}

	return allServers, total, nil
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

// RenameServer changes the display name of a server
func RenameServer(serverID string, newName string) error {
	if len(newName) < 2 || len(newName) > 64 {
		return errors.New("name must be between 2 and 64 characters")
	}

	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("server not found")
		}
		return err
	}

	server.Name = newName

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

	// Only allow regeneration for pending servers
	if server.Status != "pending" {
		return "", errors.New("can only regenerate token for pending servers")
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
