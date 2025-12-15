package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID               string    `gorm:"type:char(36);primarykey" json:"id"`
	Email            string    `gorm:"unique;not null" json:"email"`
	Password         string    `gorm:"not null" json:"-"`
	DefaultTimeRange string    `gorm:"type:varchar(10);default:'24h'" json:"default_time_range"`
	Theme            string    `gorm:"type:varchar(10);default:'light'" json:"theme"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// BeforeCreate hook to generate UUID before creating a user
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	// Set default preferences if not specified
	if u.DefaultTimeRange == "" {
		u.DefaultTimeRange = "24h"
	}
	if u.Theme == "" {
		u.Theme = "light"
	}
	return nil
}

// HashPassword hashes the user's password using bcrypt
func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}
