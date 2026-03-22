package handlers

import (
	"net/http"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
)

// GetLatestAgentVersion returns the latest agent version cached from GitHub.
func GetLatestAgentVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"latest_version": services.GetCachedLatestAgentVersion(),
	})
}
