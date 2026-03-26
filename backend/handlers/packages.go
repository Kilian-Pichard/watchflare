package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

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
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	// Build query
	query := database.DB.Where("server_id = ?", serverID)

	if searchQuery != "" {
		query = query.Where("name ILIKE ?", "%"+searchQuery+"%")
	}

	if packageManager != "" {
		managers := strings.Split(packageManager, ",")
		query = query.Where("package_manager IN ?", managers)
	}

	// Get total count
	var totalCount int64
	if err := query.Model(&models.Package{}).Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count packages"})
		return
	}

	// Get packages
	var packages []models.Package
	if err := query.Order("name ASC").Limit(limit).Offset(offset).Find(&packages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch packages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"packages":    packages,
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
	excludeInitial := c.Query("exclude_initial") == "true"
	packageName := c.Query("package")
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	// Validate change_type if provided
	if changeType != "" &&
		changeType != models.ChangeTypeAdded &&
		changeType != models.ChangeTypeRemoved &&
		changeType != models.ChangeTypeUpdated &&
		changeType != models.ChangeTypeInitial {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid change_type, valid values: added, removed, updated, initial"})
		return
	}

	// Build query
	query := database.DB.Where("server_id = ?", serverID)

	if changeType != "" {
		query = query.Where("change_type = ?", changeType)
	} else if excludeInitial {
		query = query.Where("change_type != ?", models.ChangeTypeInitial)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count history records"})
		return
	}

	// Get history
	var history []models.PackageHistory
	if err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history":     history,
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
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	offset, _ := strconv.Atoi(offsetStr)
	if offset < 0 {
		offset = 0
	}

	// Get total count
	var totalCount int64
	if err := database.DB.Model(&models.PackageCollection{}).
		Where("server_id = ?", serverID).
		Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count collections"})
		return
	}

	// Get collections
	var collections []models.PackageCollection
	if err := database.DB.Where("server_id = ?", serverID).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&collections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch collections"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"collections": collections,
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}

	// Total packages
	var totalPackages int64
	if err := database.DB.Model(&models.Package{}).Where("server_id = ?", serverID).Count(&totalPackages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count packages"})
		return
	}

	// Recent changes (last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var recentChanges int64
	if err := database.DB.Model(&models.PackageHistory{}).
		Where("server_id = ? AND timestamp >= ? AND change_type != ?", serverID, thirtyDaysAgo, models.ChangeTypeInitial).
		Count(&recentChanges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count recent changes"})
		return
	}

	// Last collection (zero value is fine if none exists yet)
	var lastCollection models.PackageCollection
	if err := database.DB.Where("server_id = ?", serverID).Order("timestamp DESC").First(&lastCollection).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Warn("failed to fetch last collection", "server_id", serverID, "error", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_packages":     totalPackages,
		"by_package_manager": managerStats,
		"recent_changes":     recentChanges,
		"last_collection":    lastCollection,
	})
}
