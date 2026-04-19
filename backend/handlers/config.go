package handlers

import (
	"net/http"
	"watchflare/backend/config"

	"github.com/gin-gonic/gin"
)

// GetAppConfig returns frontend-relevant application configuration.
// This endpoint is public (no auth required).
func GetAppConfig(c *gin.Context) {
	secure := config.CookieSecure(c.Request.TLS != nil, c.Request.RemoteAddr, c.GetHeader("X-Forwarded-Proto"))
	c.JSON(http.StatusOK, gin.H{
		"cookie_secure": secure,
	})
}
