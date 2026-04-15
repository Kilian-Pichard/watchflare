package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"watchflare/backend/cache"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// GetHostPackages returns current packages for a host
// GET /api/v1/hosts/:id/packages
func GetHostPackages(c *gin.Context) {
	hostID := c.Param("id")

	// Query parameters
	limitStr := c.DefaultQuery("limit", "10000")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 10000 {
		limit = 10000
	}
	if offset < 0 {
		offset = 0
	}

	// Build query
	query := database.DB.Where("host_id = ?", hostID)

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

// GetHostPackageHistory returns package history for a host
// GET /api/v1/hosts/:id/packages/history
func GetHostPackageHistory(c *gin.Context) {
	hostID := c.Param("id")

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
	query := database.DB.Where("host_id = ?", hostID)

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

// GetHostPackageCollections returns package collection metadata
// GET /api/v1/hosts/:id/packages/collections
func GetHostPackageCollections(c *gin.Context) {
	hostID := c.Param("id")

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
		Where("host_id = ?", hostID).
		Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count collections"})
		return
	}

	// Get collections
	var collections []models.PackageCollection
	if err := database.DB.Where("host_id = ?", hostID).
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

// TriggerPackageCollect enqueues a "collect_packages" command for the agent.
// The command is delivered on the agent's next heartbeat (within ~5s).
// POST /api/v1/hosts/:id/packages/collect
func TriggerPackageCollect(c *gin.Context) {
	hostID := c.Param("id")

	var host models.Host
	if err := database.DB.Select("id, agent_id, status").Where("id = ?", hostID).First(&host).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch host"})
		}
		return
	}

	heartbeatCache := cache.GetCache()
	cacheEntry, inCache := heartbeatCache.Get(host.AgentID)
	isOnline := inCache && cacheEntry.Status == models.StatusOnline
	if !isOnline {
		c.JSON(http.StatusConflict, gin.H{"error": "host is not online"})
		return
	}

	cmdID := heartbeatCache.EnqueueCommand(host.AgentID, models.CommandCollectPackages)
	slog.Info("package collect command enqueued", "host_id", hostID, "command_id", cmdID)

	c.JSON(http.StatusAccepted, gin.H{"message": "collection requested", "command_id": cmdID})
}

// GetPackageStats returns aggregated package statistics
// GET /api/v1/hosts/:id/packages/stats
func GetPackageStats(c *gin.Context) {
	hostID := c.Param("id")

	// Package count by package manager
	var managerStats []struct {
		PackageManager string `json:"package_manager"`
		Count          int64  `json:"count"`
	}

	if err := database.DB.Model(&models.Package{}).
		Select("package_manager, COUNT(*) as count").
		Where("host_id = ?", hostID).
		Group("package_manager").
		Scan(&managerStats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}

	// Total packages
	var totalPackages int64
	if err := database.DB.Model(&models.Package{}).Where("host_id = ?", hostID).Count(&totalPackages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count packages"})
		return
	}

	// Recent changes (last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var recentChanges int64
	if err := database.DB.Model(&models.PackageHistory{}).
		Where("host_id = ? AND timestamp >= ? AND change_type != ?", hostID, thirtyDaysAgo, models.ChangeTypeInitial).
		Count(&recentChanges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count recent changes"})
		return
	}

	// Outdated packages count
	var outdatedCount int64
	if err := database.DB.Model(&models.Package{}).
		Where("host_id = ? AND available_version IS NOT NULL AND available_version != ''", hostID).
		Count(&outdatedCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count outdated packages"})
		return
	}

	// Security updates count
	var securityUpdatesCount int64
	if err := database.DB.Model(&models.Package{}).
		Where("host_id = ? AND has_security_update = true", hostID).
		Count(&securityUpdatesCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count security updates"})
		return
	}

	// Last collection (zero value is fine if none exists yet)
	var lastCollection models.PackageCollection
	if err := database.DB.Where("host_id = ?", hostID).Order("timestamp DESC").First(&lastCollection).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.Warn("failed to fetch last collection", "host_id", hostID, "error", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_packages":        totalPackages,
		"by_package_manager":    managerStats,
		"recent_changes":        recentChanges,
		"outdated_count":        outdatedCount,
		"security_updates_count": securityUpdatesCount,
		"last_collection":       lastCollection,
	})
}
