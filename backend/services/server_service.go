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

const registrationTokenTTL = 24 * time.Hour

// ErrServerNotFound is returned when a server lookup finds no matching record.
var ErrServerNotFound = errors.New("server not found")

// CreateAgent creates a new server with status "pending" and returns the server,
// plaintext registration token, and plaintext agent key.
func CreateAgent(name, configuredIP string, allowAnyIP bool) (*models.Server, string, string, error) {
	agentID := uuid.New().String()

	token, hashedToken, err := generateRegistrationToken()
	if err != nil {
		return nil, "", "", err
	}

	// 32-byte key for HMAC-SHA256
	agentKeyBytes := make([]byte, 32)
	if _, err := rand.Read(agentKeyBytes); err != nil {
		return nil, "", "", err
	}
	agentKey := hex.EncodeToString(agentKeyBytes)

	expiresAt := time.Now().Add(registrationTokenTTL)

	var configuredIPPtr *string
	if configuredIP != "" {
		configuredIPPtr = &configuredIP
	}

	server := &models.Server{
		ID:                     uuid.New().String(),
		AgentID:                agentID,
		AgentKey:               agentKey,
		Name:                   name,
		ConfiguredIP:           configuredIPPtr,
		AllowAnyIPRegistration: allowAnyIP,
		RegistrationToken:      &hashedToken,
		ExpiresAt:              &expiresAt,
		Status:                 models.StatusPending,
	}

	if err := database.DB.Create(server).Error; err != nil {
		return nil, "", "", err
	}

	return server, token, agentKey, nil
}

// ServerListParams holds parameters for listing servers with sort/filter.
type ServerListParams struct {
	Page        int
	PerPage     int
	Sort        string
	Order       string
	Status      string
	Search      string
	Environment string
}

// allowedSortColumns is a whitelist preventing SQL injection in ORDER BY.
var allowedSortColumns = map[string]string{
	"name":       "name",
	"status":     "status",
	"ip":         "ip_address_v4",
	"last_seen":  "last_seen",
	"created_at": "created_at",
}

// ListServers returns servers with sort, filter and pagination.
// Status filtering happens after cache merge because the cache holds real-time status.
func ListServers(params ServerListParams) ([]models.Server, int64, error) {
	query := database.DB.Model(&models.Server{})

	if params.Search != "" {
		search := "%" + params.Search + "%"
		query = query.Where("name ILIKE ? OR hostname ILIKE ?", search, search)
	}

	if params.Environment != "" {
		query = query.Where("environment_type = ?", params.Environment)
	}

	sortColumn := "created_at"
	if col, ok := allowedSortColumns[params.Sort]; ok {
		sortColumn = col
	}
	sortOrder := "DESC"
	if params.Order == "asc" {
		sortOrder = "ASC"
	}

	var allServers []models.Server
	if err := query.Order(sortColumn + " " + sortOrder).Find(&allServers).Error; err != nil {
		return nil, 0, err
	}

	mergeCache(allServers)

	if params.Status != "" {
		var filtered []models.Server
		for _, s := range allServers {
			if s.Status == params.Status {
				filtered = append(filtered, s)
			}
		}
		allServers = filtered
	}

	total := int64(len(allServers))

	// Pagination applied in memory after status filter.
	if params.PerPage > 0 {
		page := params.Page
		if page < 1 {
			page = 1
		}
		start := (page - 1) * params.PerPage
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

// ListAllServers returns all servers without pagination (used for dashboard/SSE).
func ListAllServers() ([]models.Server, error) {
	var servers []models.Server
	if err := database.DB.Find(&servers).Error; err != nil {
		return nil, err
	}
	mergeCache(servers)
	return servers, nil
}

// GetServer returns a single server by ID with real-time status from cache.
func GetServer(serverID string) (*models.Server, error) {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServerNotFound
		}
		return nil, err
	}

	if cachedData, ok := cache.GetCache().Get(server.AgentID); ok {
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

// ValidateIP confirms a selected IP for a server and clears the configured_ip.
func ValidateIP(serverID string, selectedIP string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}
		return err
	}

	server.IPAddressV4 = &selectedIP
	server.ConfiguredIP = nil

	return database.DB.Save(&server).Error
}

// RenameServer changes the display name of a server.
func RenameServer(serverID string, newName string) error {
	if len(newName) < 2 || len(newName) > 64 {
		return errors.New("name must be between 2 and 64 characters")
	}

	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}
		return err
	}

	server.Name = newName
	return database.DB.Save(&server).Error
}

// UpdateConfiguredIP changes the configured IP for a server and resets the ignore flag.
func UpdateConfiguredIP(serverID string, newIP string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}
		return err
	}

	if server.ConfiguredIP != nil && *server.ConfiguredIP != "" {
		server.PreviousConfiguredIP = server.ConfiguredIP
	}
	server.ConfiguredIP = &newIP
	server.IgnoreIPMismatch = false

	return database.DB.Save(&server).Error
}

// IgnoreIPMismatch marks the IP mismatch warning as dismissed by the user.
func IgnoreIPMismatch(serverID string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}
		return err
	}

	server.IgnoreIPMismatch = true
	return database.DB.Save(&server).Error
}

// DismissReactivation clears the reactivation badge for a server.
func DismissReactivation(serverID string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}
		return err
	}

	server.ReactivatedAt = nil
	return database.DB.Save(&server).Error
}

// RegenerateToken issues a new registration token and sets the server back to "pending".
func RegenerateToken(serverID string) (string, error) {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrServerNotFound
		}
		return "", err
	}

	token, hashedToken, err := generateRegistrationToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(registrationTokenTTL)
	server.RegistrationToken = &hashedToken
	server.ExpiresAt = &expiresAt
	server.Status = models.StatusPending

	if err := database.DB.Save(&server).Error; err != nil {
		return "", err
	}

	return token, nil
}

// PauseServer sets a server's status to "paused" and removes it from heartbeat cache.
func PauseServer(serverID string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}
		return err
	}

	if server.Status == models.StatusPending {
		return errors.New("cannot pause a pending server")
	}
	if server.Status == models.StatusPaused {
		return errors.New("server is already paused")
	}

	server.Status = models.StatusPaused
	if err := database.DB.Save(&server).Error; err != nil {
		return err
	}

	// Remove from heartbeat cache so the stale checker ignores it.
	cache.GetCache().Remove(server.AgentID)

	return nil
}

// ResumeServer sets a paused server back to "online".
func ResumeServer(serverID string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}
		return err
	}

	if server.Status != models.StatusPaused {
		return errors.New("server is not paused")
	}

	server.Status = models.StatusOnline
	return database.DB.Save(&server).Error
}

// DeleteServer permanently removes a server and its associated data.
func DeleteServer(serverID string) error {
	var server models.Server
	if err := database.DB.Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServerNotFound
		}
		return err
	}

	return database.DB.Delete(&server).Error
}

// generateRegistrationToken generates a new wf_reg_* token and returns the
// plaintext token and its SHA-256 hash for storage.
func generateRegistrationToken() (token, hashedToken string, err error) {
	tokenBytes := make([]byte, 16)
	if _, err = rand.Read(tokenBytes); err != nil {
		return "", "", err
	}
	token = fmt.Sprintf("wf_reg_%s", hex.EncodeToString(tokenBytes))
	hashedToken = hashToken(token)
	return token, hashedToken, nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// mergeCache overlays real-time heartbeat data onto a slice of servers.
func mergeCache(servers []models.Server) {
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
}
