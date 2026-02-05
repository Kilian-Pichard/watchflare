package handlers

import (
	"log"
	"net/http"
	"time"
	"watchflare/backend/database"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
)

// CreateAgentRequest represents the create agent request body
type CreateAgentRequest struct {
	Name         string `json:"name" binding:"required"`
	ConfiguredIP string `json:"configured_ip" binding:"required"`
	AllowAnyIP   bool   `json:"allow_any_ip"`
}

// ValidateIPRequest represents the validate IP request body
type ValidateIPRequest struct {
	SelectedIP string `json:"selected_ip" binding:"required"`
}

// UpdateConfiguredIPRequest represents the update configured IP request body
type UpdateConfiguredIPRequest struct {
	NewIP string `json:"new_ip" binding:"required"`
}

// CreateAgent creates a new server with status "pending" and returns installation command
func CreateAgent(c *gin.Context) {
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, token, agentKey, err := services.CreateAgent(
		req.Name,
		req.ConfiguredIP,
		req.AllowAnyIP,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Server created successfully",
		"server":    server,
		"token":     token,    // Return plain token for installation
		"agent_key": agentKey, // Return agent key for the agent
	})
}

// ListServers returns all servers
func ListServers(c *gin.Context) {
	servers, err := services.ListServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"servers": servers,
	})
}

// GetServer returns a specific server by ID
func GetServer(c *gin.Context) {
	serverID := c.Param("id")

	server, err := services.GetServer(serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"server": server,
	})
}

// ValidateIP validates and updates the server IP
func ValidateIP(c *gin.Context) {
	serverID := c.Param("id")

	var req ValidateIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.ValidateIP(serverID, req.SelectedIP); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "IP validated successfully",
	})
}

// UpdateConfiguredIP changes the configured IP for a server
func UpdateConfiguredIP(c *gin.Context) {
	serverID := c.Param("id")

	var req UpdateConfiguredIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.UpdateConfiguredIP(serverID, req.NewIP); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Configured IP updated successfully",
	})
}

// RegenerateToken regenerates a registration token for an expired/pending server
func RegenerateToken(c *gin.Context) {
	serverID := c.Param("id")

	token, err := services.RegenerateToken(serverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token regenerated successfully",
		"token":   token,
	})
}

// IgnoreIPMismatch marks the IP mismatch warning as ignored
func IgnoreIPMismatch(c *gin.Context) {
	serverID := c.Param("id")

	if err := services.IgnoreIPMismatch(serverID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "IP mismatch warning ignored",
	})
}

// DeleteServer deletes a server
func DeleteServer(c *gin.Context) {
	serverID := c.Param("id")

	if err := services.DeleteServer(serverID); err != nil{
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Server deleted successfully",
	})
}

// DroppedMetricsAlert represents a dropped metrics alert for the dashboard
type DroppedMetricsAlert struct {
	AgentID          string        `json:"agent_id"`
	Hostname         string        `json:"hostname"`
	TotalDropped     int           `json:"total_dropped"`
	FirstDroppedAt   time.Time     `json:"first_dropped_at"`
	LastDroppedAt    time.Time     `json:"last_dropped_at"`
	LastReportedAt   time.Time     `json:"last_reported_at"`
	DowntimeDuration time.Duration `json:"downtime_duration"`
}

// GetDroppedMetrics returns summary of dropped metrics for the last 24 hours
func GetDroppedMetrics(c *gin.Context) {
	// Query the dropped metrics summary view
	rows, err := database.DB.Raw(`
		SELECT agent_id, hostname, total_dropped,
		       first_dropped_at, last_dropped_at, last_reported_at
		FROM agent_dropped_metrics_summary
		ORDER BY total_dropped DESC
	`).Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dropped metrics"})
		return
	}
	defer rows.Close()

	var alerts []DroppedMetricsAlert
	for rows.Next() {
		var alert DroppedMetricsAlert
		var agentIDStr string

		if err := rows.Scan(
			&agentIDStr,
			&alert.Hostname,
			&alert.TotalDropped,
			&alert.FirstDroppedAt,
			&alert.LastDroppedAt,
			&alert.LastReportedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan dropped metrics"})
			return
		}

		alert.AgentID = agentIDStr
		alert.DowntimeDuration = alert.LastDroppedAt.Sub(alert.FirstDroppedAt)
		alerts = append(alerts, alert)
	}

	// Return empty array if no alerts
	if alerts == nil {
		alerts = []DroppedMetricsAlert{}
	}

	c.JSON(http.StatusOK, alerts)
}

// GetAggregatedMetrics returns historical aggregated metrics with regular intervals
func GetAggregatedMetrics(c *gin.Context) {
	timeRange := c.Query("time_range")
	if timeRange == "" {
		timeRange = "1h" // Default to 1 hour
	}

	// Determine time range and interval
	var duration time.Duration
	var interval string
	switch timeRange {
	case "1h":
		duration = 1 * time.Hour
		interval = "30 seconds"
	case "12h":
		duration = 12 * time.Hour
		interval = "10 minutes"
	case "24h":
		duration = 24 * time.Hour
		interval = "20 minutes"
	case "7d":
		duration = 7 * 24 * time.Hour
		interval = "2 hours"
	case "30d":
		duration = 30 * 24 * time.Hour
		interval = "10 hours"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time_range"})
		return
	}

	endTime := time.Now()
	startTime := endTime.Add(-duration)

	// Query using time_bucket to create regular intervals
	// For each bucket, we aggregate metrics with timestamp > bucket_start AND <= bucket_end
	// Two-level aggregation:
	// 1. First, average metrics per server per bucket (to avoid summing multiple metrics from same server)
	// 2. Then, sum across servers (and average CPU)
	query := `
		WITH time_buckets AS (
			SELECT time_bucket($1::interval, m.timestamp) as bucket,
				   m.server_id,
				   m.cpu_usage_percent,
				   m.memory_total_bytes,
				   m.memory_used_bytes,
				   m.disk_total_bytes,
				   m.disk_used_bytes,
				   s.environment_type
			FROM metrics m
			JOIN servers s ON m.server_id = s.id
			WHERE s.status = 'online'
			  AND m.timestamp > $2
			  AND m.timestamp <= $3
		),
		server_aggregates AS (
			-- Average metrics per server per bucket (one row per server per bucket)
			SELECT
				bucket,
				server_id,
				COALESCE(AVG(cpu_usage_percent), 0) as cpu_usage_percent,
				COALESCE(AVG(memory_total_bytes), 0) as memory_total_bytes,
				COALESCE(AVG(memory_used_bytes), 0) as memory_used_bytes,
				COALESCE(AVG(disk_total_bytes), 0) as disk_total_bytes,
				COALESCE(AVG(disk_used_bytes), 0) as disk_used_bytes,
				MAX(environment_type) as environment_type
			FROM time_buckets
			GROUP BY bucket, server_id
		)
		SELECT
			bucket as timestamp,
			COALESCE(AVG(cpu_usage_percent), 0) as cpu_usage_percent,
			COALESCE(SUM(memory_total_bytes), 0)::BIGINT as memory_total_bytes,
			COALESCE(SUM(memory_used_bytes), 0)::BIGINT as memory_used_bytes,
			COALESCE(SUM(CASE WHEN environment_type != 'container' THEN disk_total_bytes ELSE 0 END), 0)::BIGINT as disk_total_bytes,
			COALESCE(SUM(CASE WHEN environment_type != 'container' THEN disk_used_bytes ELSE 0 END), 0)::BIGINT as disk_used_bytes
		FROM server_aggregates
		GROUP BY bucket
		ORDER BY bucket ASC
	`

	rows, err := database.DB.Raw(query, interval, startTime, endTime).Rows()
	if err != nil {
		log.Printf("Error querying aggregated metrics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query metrics"})
		return
	}
	defer rows.Close()

	type AggregatedPoint struct {
		Timestamp         time.Time `json:"timestamp"`
		CPUUsagePercent   float64   `json:"cpu_usage_percent"`
		MemoryTotalBytes  uint64    `json:"memory_total_bytes"`
		MemoryUsedBytes   uint64    `json:"memory_used_bytes"`
		MemoryAvailableBytes uint64 `json:"memory_available_bytes"`
		DiskTotalBytes    uint64    `json:"disk_total_bytes"`
		DiskUsedBytes     uint64    `json:"disk_used_bytes"`
	}

	var points []AggregatedPoint
	for rows.Next() {
		var point AggregatedPoint
		if err := rows.Scan(
			&point.Timestamp,
			&point.CPUUsagePercent,
			&point.MemoryTotalBytes,
			&point.MemoryUsedBytes,
			&point.DiskTotalBytes,
			&point.DiskUsedBytes,
		); err != nil {
			log.Printf("Error scanning aggregated metrics: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan metrics"})
			return
		}
		// Calculate memory available
		point.MemoryAvailableBytes = point.MemoryTotalBytes - point.MemoryUsedBytes
		points = append(points, point)
	}

	if points == nil {
		points = []AggregatedPoint{}
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": points,
	})
}
