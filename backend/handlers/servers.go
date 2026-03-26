package handlers

import (
	"errors"
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

// RenameServerRequest represents the rename server request body
type RenameServerRequest struct {
	NewName string `json:"new_name" binding:"required"`
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

	if !req.AllowAnyIP && req.ConfiguredIP == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "configured_ip is required when allow_any_ip is false"})
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

// ListServers returns servers with optional pagination, sorting and filtering
func ListServers(c *gin.Context) {
	params := services.ServerListParams{
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

	servers, total, err := services.ListServers(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"servers":  servers,
		"total":    total,
		"page":     params.Page,
		"per_page": params.PerPage,
	})
}

// serverError writes a 404 for ErrServerNotFound, 400 for any other error.
func serverError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrServerNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// GetServer returns a specific server by ID
func GetServer(c *gin.Context) {
	serverID := c.Param("id")

	server, err := services.GetServer(serverID)
	if err != nil {
		serverError(c, err)
		return
	}

	// Check cache for clock desync flag
	clockDesync := false
	if cachedData, ok := cache.GetCache().Get(server.AgentID); ok {
		clockDesync = cachedData.ClockDesync
	}

	c.JSON(http.StatusOK, gin.H{
		"server":       server,
		"clock_desync": clockDesync,
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
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "IP validated successfully",
	})
}

// RenameServer changes the display name of a server
func RenameServer(c *gin.Context) {
	serverID := c.Param("id")

	var req RenameServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.RenameServer(serverID, req.NewName); err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Server renamed successfully",
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
		serverError(c, err)
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
		serverError(c, err)
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
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "IP mismatch warning ignored",
	})
}

// DismissReactivation clears the reactivation badge for an agent
func DismissReactivation(c *gin.Context) {
	serverID := c.Param("id")

	if err := services.DismissReactivation(serverID); err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reactivation badge dismissed",
	})
}

// PauseServer pauses monitoring for a server
func PauseServer(c *gin.Context) {
	serverID := c.Param("id")

	if err := services.PauseServer(serverID); err != nil {
		serverError(c, err)
		return
	}

	// Broadcast SSE update
	broker := sse.GetBroker()
	broker.BroadcastServerUpdate(sse.ServerUpdate{
		ID:     serverID,
		Status: models.StatusPaused,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Server paused successfully",
	})
}

// ResumeServer resumes monitoring for a paused server
func ResumeServer(c *gin.Context) {
	serverID := c.Param("id")

	if err := services.ResumeServer(serverID); err != nil {
		serverError(c, err)
		return
	}

	// Broadcast SSE update
	broker := sse.GetBroker()
	broker.BroadcastServerUpdate(sse.ServerUpdate{
		ID:     serverID,
		Status: models.StatusOnline,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Server resumed successfully",
	})
}

// DeleteServer deletes a server
func DeleteServer(c *gin.Context) {
	serverID := c.Param("id")

	if err := services.DeleteServer(serverID); err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Server deleted successfully",
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
