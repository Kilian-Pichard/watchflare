package handlers

import (
	"net/http"
	"strings"
	"watchflare/backend/config"
	"watchflare/backend/database"
	"watchflare/backend/models"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
)

// getUserID extracts the authenticated user ID from the Gin context
func getUserID(c *gin.Context) (string, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return "", false
	}
	id, ok := val.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return "", false
	}
	return id, true
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Username string `json:"username" binding:"max=50"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// ChangePasswordRequest represents the change password request body
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ChangeEmailRequest represents the change email request body
type ChangeEmailRequest struct {
	NewEmail string `json:"new_email" binding:"required,email"`
}

// ChangeUsernameRequest represents the change username request body
type ChangeUsernameRequest struct {
	Username string `json:"username" binding:"min=1,max=50"`
}

// UpdatePreferencesRequest represents the update preferences request body
type UpdatePreferencesRequest struct {
	DefaultTimeRange       string `json:"default_time_range"`
	Theme                  string `json:"theme"`
	TimeFormat             string `json:"time_format"`
	TemperatureUnit        string `json:"temperature_unit"`
	NetworkUnit            string `json:"network_unit"`
	DiskUnit               string `json:"disk_unit"`
	GaugeWarningThreshold  *int   `json:"gauge_warning_threshold"`
	GaugeCriticalThreshold *int   `json:"gauge_critical_threshold"`
}

// Register creates the first admin user and automatically logs them in
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := services.Register(req.Email, req.Password, req.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set JWT token in HttpOnly cookie (auto-login after registration)
	isProd := config.AppConfig.Environment == "production"
	domain := config.AppConfig.CookieDomain
	secure := isProd && domain != "" // Only secure if HTTPS is likely (custom domain set)

	c.SetCookie(
		"jwt_token",           // name
		token,                 // value
		60*60*24*7,           // maxAge (7 days in seconds)
		"/",                   // path
		domain,                // domain (empty = current host)
		secure,                // secure (only with custom domain/HTTPS)
		true,                  // httpOnly
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user":    user,
	})
}

// Login authenticates a user and sets JWT token in HttpOnly cookie
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := services.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Set JWT token in HttpOnly cookie
	isProd := config.AppConfig.Environment == "production"
	domain := config.AppConfig.CookieDomain
	secure := isProd && domain != "" // Only secure if HTTPS is likely (custom domain set)

	c.SetCookie(
		"jwt_token",           // name
		token,                 // value
		60*60*24*7,           // maxAge (7 days in seconds)
		"/",                   // path
		domain,                // domain (empty = current host)
		secure,                // secure (only with custom domain/HTTPS)
		true,                  // httpOnly
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
	})
}

// Logout clears the JWT cookie
func Logout(c *gin.Context) {
	c.SetCookie(
		"jwt_token",
		"",
		-1,    // maxAge -1 deletes the cookie
		"/",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

// ChangePassword updates the authenticated user's password
func ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	err := services.ChangePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// ChangeEmail updates the authenticated user's email
func ChangeEmail(c *gin.Context) {
	var req ChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).Update("email", req.NewEmail).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already in use"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update email"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email updated successfully",
	})
}

// ChangeUsername updates the authenticated user's username
func ChangeUsername(c *gin.Context) {
	var req ChangeUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).Update("username", req.Username).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update username"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch updated user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Username updated successfully",
		"user":    user,
	})
}

// SetupRequired checks if initial setup is required (no users exist)
func SetupRequired(c *gin.Context) {
	var count int64
	if err := database.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check setup status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"setup_required": count == 0,
	})
}

// GetCurrentUser returns the authenticated user's information including preferences
func GetCurrentUser(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// UpdatePreferences updates the authenticated user's preferences
func UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserID(c)
	if !ok {
		return
	}

	// Validate time range
	validTimeRanges := []string{"1h", "12h", "24h", "7d", "30d"}
	if req.DefaultTimeRange != "" {
		valid := false
		for _, tr := range validTimeRanges {
			if req.DefaultTimeRange == tr {
				valid = true
				break
			}
		}
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time range. Valid values: 1h, 12h, 24h, 7d, 30d"})
			return
		}
	}

	// Validate theme
	if req.Theme != "" && req.Theme != "light" && req.Theme != "dark" && req.Theme != "system" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme. Valid values: light, dark, system"})
		return
	}

	// Validate time_format
	if req.TimeFormat != "" && req.TimeFormat != "24h" && req.TimeFormat != "12h" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time_format. valid values: 24h, 12h"})
		return
	}

	// Validate temperature_unit
	if req.TemperatureUnit != "" && req.TemperatureUnit != "celsius" && req.TemperatureUnit != "fahrenheit" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid temperature_unit. valid values: celsius, fahrenheit"})
		return
	}

	// Validate network_unit
	if req.NetworkUnit != "" && req.NetworkUnit != "bytes" && req.NetworkUnit != "bits" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid network_unit. valid values: bytes, bits"})
		return
	}

	// Validate disk_unit
	if req.DiskUnit != "" && req.DiskUnit != "bytes" && req.DiskUnit != "bits" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid disk_unit. valid values: bytes, bits"})
		return
	}

	// Validate gauge thresholds
	if req.GaugeWarningThreshold != nil && (*req.GaugeWarningThreshold < 1 || *req.GaugeWarningThreshold > 99) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid gauge_warning_threshold. must be between 1 and 99"})
		return
	}
	if req.GaugeCriticalThreshold != nil && (*req.GaugeCriticalThreshold < 1 || *req.GaugeCriticalThreshold > 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid gauge_critical_threshold. must be between 1 and 100"})
		return
	}

	// Update user preferences
	updates := make(map[string]interface{})
	if req.DefaultTimeRange != "" {
		updates["default_time_range"] = req.DefaultTimeRange
	}
	if req.Theme != "" {
		updates["theme"] = req.Theme
	}
	if req.TimeFormat != "" {
		updates["time_format"] = req.TimeFormat
	}
	if req.TemperatureUnit != "" {
		updates["temperature_unit"] = req.TemperatureUnit
	}
	if req.NetworkUnit != "" {
		updates["network_unit"] = req.NetworkUnit
	}
	if req.DiskUnit != "" {
		updates["disk_unit"] = req.DiskUnit
	}
	if req.GaugeWarningThreshold != nil {
		updates["gauge_warning_threshold"] = *req.GaugeWarningThreshold
	}
	if req.GaugeCriticalThreshold != nil {
		updates["gauge_critical_threshold"] = *req.GaugeCriticalThreshold
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}

	// Fetch updated user
	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Preferences updated successfully",
		"user":    user,
	})
}
