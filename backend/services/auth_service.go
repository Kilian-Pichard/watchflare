package services

import (
	"errors"
	"time"
	"watchflare/backend/config"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Register creates a new user (first admin only) and returns a JWT token
func Register(email, password string) (*models.User, string, error) {
	// Check if any user already exists
	var count int64
	database.DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		return nil, "", errors.New("registration is closed - admin user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	// Create user
	user := &models.User{
		Email:    email,
		Password: string(hashedPassword),
	}

	if err := database.DB.Create(user).Error; err != nil {
		return nil, "", err
	}

	// Generate JWT token
	token, err := generateJWT(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login authenticates a user and returns a JWT token
func Login(email, password string) (string, error) {
	var user models.User

	// Find user by email
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("invalid credentials")
		}
		return "", err
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := generateJWT(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ChangePassword updates user password after verifying current password
func ChangePassword(userID string, currentPassword, newPassword string) error {
	var user models.User

	// Find user by ID
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	user.Password = string(hashedPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		return err
	}

	return nil
}

// generateJWT creates a new JWT token for a user
func generateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days expiration
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}
