package handlers

import (
	"net/http"
	"watchflare/backend/config"

	"github.com/gin-gonic/gin"
)

// GetAppConfig returns frontend-relevant application configuration.
// This endpoint is public (no auth required).
func GetAppConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"cookie_secure": config.AppConfig.CookieSecure,
	})
}
