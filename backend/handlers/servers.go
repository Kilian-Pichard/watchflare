package handlers

import (
	"net/http"
	"strconv"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
)

// CreateAgentRequest represents the create agent request body
type CreateAgentRequest struct {
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required,oneof=physical vm docker lxc"`
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
		req.Type,
		req.ConfiguredIP,
		req.AllowAnyIP,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Server created successfully",
		"server":  server,
		"token":   token,    // Return plain token for installation
		"agent_key": agentKey, // Return agent key (will be given to agent during registration)
		"install_command": gin.H{
			"curl": "curl -sSL https://watchflare.io/install.sh | bash -s -- --token " + token + " --host backend.local --port 50051",
		},
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
	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
		return
	}

	server, err := services.GetServer(uint(serverID))
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
	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
		return
	}

	var req ValidateIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.ValidateIP(uint(serverID), req.SelectedIP); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "IP validated successfully",
	})
}

// UpdateConfiguredIP changes the configured IP for a server
func UpdateConfiguredIP(c *gin.Context) {
	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
		return
	}

	var req UpdateConfiguredIPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := services.UpdateConfiguredIP(uint(serverID), req.NewIP); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Configured IP updated successfully",
	})
}

// RegenerateToken regenerates a registration token for an expired/pending server
func RegenerateToken(c *gin.Context) {
	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
		return
	}

	token, err := services.RegenerateToken(uint(serverID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token regenerated successfully",
		"token":   token,
		"install_command": gin.H{
			"curl": "curl -sSL https://watchflare.io/install.sh | bash -s -- --token " + token + " --host backend.local --port 50051",
		},
	})
}

// DeleteServer deletes a server (only if pending or expired)
func DeleteServer(c *gin.Context) {
	serverID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
		return
	}

	if err := services.DeleteServer(uint(serverID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Server deleted successfully",
	})
}
