package handlers

import (
	"net/http"
	"watchflare/backend/database"
	"watchflare/backend/models"
	"watchflare/backend/services"

	"github.com/gin-gonic/gin"
)

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
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

// UpdatePreferencesRequest represents the update preferences request body
type UpdatePreferencesRequest struct {
	DefaultTimeRange string `json:"default_time_range"`
	Theme            string `json:"theme"`
}

// Register creates the first admin user and automatically logs them in
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := services.Register(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set JWT token in HttpOnly cookie (auto-login after registration)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"jwt_token",           // name
		token,                 // value
		60*60*24*7,           // maxAge (7 days in seconds)
		"/",                   // path
		"",                    // domain (empty = current domain)
		false,                 // secure (set to true in production with HTTPS)
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
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"jwt_token",           // name
		token,                 // value
		60*60*24*7,           // maxAge (7 days in seconds)
		"/",                   // path
		"",                    // domain (empty = current domain)
		false,                 // secure (set to true in production with HTTPS)
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

	// Get user ID from context (set by JWT middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	err := services.ChangePassword(userID.(string), req.CurrentPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// SetupRequired checks if initial setup is required (no users exist)
func SetupRequired(c *gin.Context) {
	var count int64
	database.DB.Model(&models.User{}).Count(&count)

	c.JSON(http.StatusOK, gin.H{
		"setup_required": count == 0,
	})
}

// GetCurrentUser returns the authenticated user's information including preferences
func GetCurrentUser(c *gin.Context) {
	// Get user ID from context (set by JWT middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := database.DB.Where("id = ?", userID.(string)).First(&user).Error; err != nil {
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

	// Get user ID from context (set by JWT middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Validate time range
	validTimeRanges := []string{"1h", "6h", "24h", "7d", "30d"}
	if req.DefaultTimeRange != "" {
		valid := false
		for _, tr := range validTimeRanges {
			if req.DefaultTimeRange == tr {
				valid = true
				break
			}
		}
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time range. Valid values: 1h, 6h, 24h, 7d, 30d"})
			return
		}
	}

	// Validate theme
	if req.Theme != "" && req.Theme != "light" && req.Theme != "dark" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme. Valid values: light, dark"})
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

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID.(string)).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}

	// Fetch updated user
	var user models.User
	database.DB.Where("id = ?", userID.(string)).First(&user)

	c.JSON(http.StatusOK, gin.H{
		"message": "Preferences updated successfully",
		"user":    user,
	})
}
