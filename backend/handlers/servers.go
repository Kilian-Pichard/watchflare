package handlers

import (
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
