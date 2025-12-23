package handlers

import (
	"net/http"
	"strconv"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"github.com/gin-gonic/gin"
)

// PackageResponse represents a package with additional metadata
type PackageResponse struct {
	ID             int64     `json:"id"`
	ServerID       string    `json:"server_id"`
	Name           string    `json:"name"`
	Version        string    `json:"version"`
	Architecture   string    `json:"architecture"`
	PackageManager string    `json:"package_manager"`
	Source         string    `json:"source"`
	InstalledAt    *time.Time `json:"installed_at"`
	PackageSize    int64     `json:"package_size"`
	Description    string    `json:"description"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
}

// PackageHistoryResponse represents a package history record
type PackageHistoryResponse struct {
	ID             int64     `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	ServerID       string    `json:"server_id"`
	Name           string    `json:"name"`
	Version        string    `json:"version"`
	Architecture   string    `json:"architecture"`
	PackageManager string    `json:"package_manager"`
	Source         string    `json:"source"`
	PackageSize    int64     `json:"package_size"`
	Description    string    `json:"description"`
	ChangeType     string    `json:"change_type"`
}

// PackageCollectionResponse represents collection metadata
type PackageCollectionResponse struct {
	ID             int64     `json:"id"`
	ServerID       string    `json:"server_id"`
	Timestamp      time.Time `json:"timestamp"`
	CollectionType string    `json:"collection_type"`
	PackageCount   int       `json:"package_count"`
	ChangesCount   int       `json:"changes_count"`
	DurationMs     int       `json:"duration_ms"`
	Status         string    `json:"status"`
	ErrorMessage   string    `json:"error_message"`
}

// GetServerPackages returns current packages for a server
// GET /api/servers/:id/packages
func GetServerPackages(c *gin.Context) {
	serverID := c.Param("id")

	// Query parameters
	searchQuery := c.Query("search")
	packageManager := c.Query("package_manager")
	limitStr := c.DefaultQuery("limit", "1000")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Build query
	query := database.DB.Where("server_id = ?", serverID)

	if searchQuery != "" {
		query = query.Where("name ILIKE ?", "%"+searchQuery+"%")
	}

	if packageManager != "" {
		query = query.Where("package_manager = ?", packageManager)
	}

	// Get total count
	var totalCount int64
	if err := query.Model(&models.Package{}).Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count packages"})
		return
	}

	// Get packages
	var packages []models.Package
	if err := query.Order("name ASC").Limit(limit).Offset(offset).Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch packages"})
		return
	}

	// Convert to response format
	response := make([]PackageResponse, len(packages))
	for i, pkg := range packages {
		response[i] = PackageResponse{
			ID:             pkg.ID,
			ServerID:       pkg.ServerID,
			Name:           pkg.Name,
			Version:        pkg.Version,
			Architecture:   pkg.Architecture,
			PackageManager: pkg.PackageManager,
			Source:         pkg.Source,
			InstalledAt:    pkg.InstalledAt,
			PackageSize:    pkg.PackageSize,
			Description:    pkg.Description,
			FirstSeen:      pkg.FirstSeen,
			LastSeen:       pkg.LastSeen,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"packages":    response,
		"total_count": totalCount,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetServerPackageHistory returns package history for a server
// GET /api/servers/:id/packages/history
func GetServerPackageHistory(c *gin.Context) {
	serverID := c.Param("id")

	// Query parameters
	changeType := c.Query("change_type") // 'added', 'removed', 'updated', 'initial'
	packageName := c.Query("package")
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Build query
	query := database.DB.Where("server_id = ?", serverID)

	if changeType != "" {
		query = query.Where("change_type = ?", changeType)
	}

	if packageName != "" {
		query = query.Where("name ILIKE ?", "%"+packageName+"%")
	}

	if startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query = query.Where("timestamp >= ?", startTime)
		}
	}

	if endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query = query.Where("timestamp <= ?", endTime)
		}
	}

	// Get total count
	var totalCount int64
	if err := query.Model(&models.PackageHistory{}).Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count history records"})
		return
	}

	// Get history
	var history []models.PackageHistory
	if err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	// Convert to response format
	response := make([]PackageHistoryResponse, len(history))
	for i, h := range history {
		response[i] = PackageHistoryResponse{
			ID:             h.ID,
			Timestamp:      h.Timestamp,
			ServerID:       h.ServerID,
			Name:           h.Name,
			Version:        h.Version,
			Architecture:   h.Architecture,
			PackageManager: h.PackageManager,
			Source:         h.Source,
			PackageSize:    h.PackageSize,
			Description:    h.Description,
			ChangeType:     h.ChangeType,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"history":     response,
		"total_count": totalCount,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetServerPackageCollections returns package collection metadata
// GET /api/servers/:id/packages/collections
func GetServerPackageCollections(c *gin.Context) {
	serverID := c.Param("id")

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Get total count
	var totalCount int64
	if err := database.DB.Model(&models.PackageCollection{}).
		Where("server_id = ?", serverID).
		Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count collections"})
		return
	}

	// Get collections
	var collections []models.PackageCollection
	if err := database.DB.Where("server_id = ?", serverID).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&collections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collections"})
		return
	}

	// Convert to response format
	response := make([]PackageCollectionResponse, len(collections))
	for i, col := range collections {
		response[i] = PackageCollectionResponse{
			ID:             col.ID,
			ServerID:       col.ServerID,
			Timestamp:      col.Timestamp,
			CollectionType: col.CollectionType,
			PackageCount:   col.PackageCount,
			ChangesCount:   col.ChangesCount,
			DurationMs:     col.DurationMs,
			Status:         col.Status,
			ErrorMessage:   col.ErrorMessage,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"collections": response,
		"total_count": totalCount,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetPackageStats returns aggregated package statistics
// GET /api/servers/:id/packages/stats
func GetPackageStats(c *gin.Context) {
	serverID := c.Param("id")

	// Package count by package manager
	var managerStats []struct {
		PackageManager string `json:"package_manager"`
		Count          int64  `json:"count"`
	}

	if err := database.DB.Model(&models.Package{}).
		Select("package_manager, COUNT(*) as count").
		Where("server_id = ?", serverID).
		Group("package_manager").
		Scan(&managerStats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	// Total packages
	var totalPackages int64
	database.DB.Model(&models.Package{}).Where("server_id = ?", serverID).Count(&totalPackages)

	// Recent changes (last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var recentChanges int64
	database.DB.Model(&models.PackageHistory{}).
		Where("server_id = ? AND timestamp >= ? AND change_type != 'initial'", serverID, thirtyDaysAgo).
		Count(&recentChanges)

	// Last collection
	var lastCollection models.PackageCollection
	database.DB.Where("server_id = ?", serverID).
		Order("timestamp DESC").
		First(&lastCollection)

	c.JSON(http.StatusOK, gin.H{
		"total_packages":      totalPackages,
		"by_package_manager":  managerStats,
		"recent_changes":      recentChanges,
		"last_collection":     lastCollection,
	})
}
