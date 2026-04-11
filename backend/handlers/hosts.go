package handlers

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"watchflare/backend/cache"
	"watchflare/backend/models"
	"watchflare/backend/services"
	"watchflare/backend/sse"

	"github.com/gin-gonic/gin"
)

// CreateAgentRequest represents the create agent request body
type CreateAgentRequest struct {
	Name         string `json:"name" binding:"required"`
	ConfiguredIP string `json:"configured_ip"`
	AllowAnyIP   bool   `json:"allow_any_ip"`
}

// ValidateIPRequest represents the validate IP request body
type ValidateIPRequest struct {
	SelectedIP string `json:"selected_ip" binding:"required"`
}

// RenameHostRequest represents the rename host request body
type RenameHostRequest struct {
	NewName string `json:"new_name" binding:"required"`
}

// UpdateConfiguredIPRequest represents the update configured IP request body
type UpdateConfiguredIPRequest struct {
	NewIP string `json:"new_ip" binding:"required"`
}

// CreateAgent creates a new host with status "pending" and returns installation command
func CreateAgent(c *gin.Context) {
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.AllowAnyIP && req.ConfiguredIP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "configured_ip is required when allow_any_ip is false"})
		return
	}

	host, token, agentKey, err := services.CreateAgent(
		req.Name,
		req.ConfiguredIP,
		req.AllowAnyIP,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Host created successfully",
		"host":      host,
		"token":     token,    // Return plain token for installation
		"agent_key": agentKey, // Return agent key for the agent
	})
}

// ListHosts returns hosts with optional pagination, sorting and filtering
func ListHosts(c *gin.Context) {
	params := services.HostListParams{
		Page:        1,
		PerPage:     0, // 0 = no pagination (backward compatible)
		Sort:        c.Query("sort"),
		Order:       c.Query("order"),
		Status:      c.Query("status"),
		Search:      c.Query("search"),
		Environment: c.Query("environment"),
	}

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			params.Page = v
		}
	}
	if pp := c.Query("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 {
			params.PerPage = v
		}
	}

	hosts, total, err := services.ListHosts(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hosts":    hosts,
		"total":    total,
		"page":     params.Page,
		"per_page": params.PerPage,
	})
}

// hostError writes a 404 for ErrHostNotFound, 400 for any other error.
func hostError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrHostNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// GetHost returns a specific host by ID
func GetHost(c *gin.Context) {
	hostID := c.Param("id")

	host, err := services.GetHost(hostID)
	if err != nil {
		hostError(c, err)
		return
	}

	// Check cache for clock desync flag
	clockDesync := false
	if cachedData, ok := cache.GetCache().Get(host.AgentID); ok {
		clockDesync = cachedData.ClockDesync
	}

	c.JSON(http.StatusOK, gin.H{
		"host":         host,
		"clock_desync": clockDesync,
	})
}

// ValidateIP validates and updates the host IP
func ValidateIP(c *gin.Context) {
	hostID := c.Param("id")

	var req ValidateIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.ValidateIP(hostID, req.SelectedIP); err != nil {
		hostError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "IP validated successfully",
	})
}

// RenameHost changes the display name of a host
func RenameHost(c *gin.Context) {
	hostID := c.Param("id")

	var req RenameHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.RenameHost(hostID, req.NewName); err != nil {
		hostError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Host renamed successfully",
	})
}

// UpdateConfiguredIP changes the configured IP for a host
func UpdateConfiguredIP(c *gin.Context) {
	hostID := c.Param("id")

	var req UpdateConfiguredIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if net.ParseIP(req.NewIP) == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid IP address format"})
		return
	}

	if err := services.UpdateConfiguredIP(hostID, req.NewIP); err != nil {
		hostError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Configured IP updated successfully",
	})
}

// RegenerateToken regenerates a registration token for an expired/pending host
func RegenerateToken(c *gin.Context) {
	hostID := c.Param("id")

	token, err := services.RegenerateToken(hostID)
	if err != nil {
		hostError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token regenerated successfully",
		"token":   token,
	})
}

// IgnoreIPMismatch marks the IP mismatch warning as ignored
func IgnoreIPMismatch(c *gin.Context) {
	hostID := c.Param("id")

	if err := services.IgnoreIPMismatch(hostID); err != nil {
		hostError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "IP mismatch warning ignored",
	})
}

// DismissReactivation clears the reactivation badge for an agent
func DismissReactivation(c *gin.Context) {
	hostID := c.Param("id")

	if err := services.DismissReactivation(hostID); err != nil {
		hostError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reactivation badge dismissed",
	})
}

// PauseHost pauses monitoring for a host
func PauseHost(c *gin.Context) {
	hostID := c.Param("id")

	if err := services.PauseHost(hostID); err != nil {
		hostError(c, err)
		return
	}

	// Broadcast SSE update
	broker := sse.GetBroker()
	broker.BroadcastHostUpdate(sse.HostUpdate{
		ID:     hostID,
		Status: models.StatusPaused,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Host paused successfully",
	})
}

// ResumeHost resumes monitoring for a paused host
func ResumeHost(c *gin.Context) {
	hostID := c.Param("id")

	if err := services.ResumeHost(hostID); err != nil {
		hostError(c, err)
		return
	}

	// Broadcast SSE update
	broker := sse.GetBroker()
	broker.BroadcastHostUpdate(sse.HostUpdate{
		ID:     hostID,
		Status: models.StatusOnline,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Host resumed successfully",
	})
}

// DeleteHost deletes a host
func DeleteHost(c *gin.Context) {
	hostID := c.Param("id")

	if err := services.DeleteHost(hostID); err != nil {
		hostError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Host deleted successfully",
	})
}

// GetDroppedMetrics returns summary of dropped metrics for the last 24 hours
func GetDroppedMetrics(c *gin.Context) {
	alerts, err := services.GetDroppedMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch dropped metrics"})
		return
	}
	c.JSON(http.StatusOK, alerts)
}

// GetAggregatedMetrics returns historical aggregated metrics with regular intervals
func GetAggregatedMetrics(c *gin.Context) {
	timeRange := c.Query("time_range")
	if timeRange == "" {
		timeRange = "1h"
	}

	points, err := services.GetAggregatedMetrics(timeRange)
	if err != nil {
		if errors.Is(err, services.ErrInvalidTimeRange) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time_range"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"metrics": points})
}
