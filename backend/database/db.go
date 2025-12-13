package database

import (
	"log"
	"watchflare/backend/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect establishes database connection and runs migrations
func Connect(dbPath string) error {
	var err error

	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	log.Println("Database connected successfully")

	// Auto-migrate models
	err = DB.AutoMigrate(
		&models.User{},
		&models.Server{},
	)
	if err != nil {
		return err
	}

	log.Println("Database migrations completed")
	return nil
}
