package main

import (
	"log"
	"watchflare/backend/config"
	"watchflare/backend/database"
	"watchflare/backend/handlers"
	"watchflare/backend/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config.Load()

	// Connect to database
	if err := database.Connect(config.AppConfig.DBPath); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Setup Gin router
	router := setupRouter()

	// Start HTTP server
	log.Printf("Starting HTTP server on port %s", config.AppConfig.Port)
	if err := router.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRouter() *gin.Engine {
	// Set Gin mode
	if config.AppConfig.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Configure CORS
	corsConfig := cors.Config{
		AllowOrigins:     config.AppConfig.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}
	router.Use(cors.New(corsConfig))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Auth routes (public)
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", handlers.Register)
		authGroup.POST("/login", handlers.Login)
		authGroup.POST("/logout", handlers.Logout)
	}

	// Protected routes (require JWT)
	protectedGroup := router.Group("/auth")
	protectedGroup.Use(middleware.AuthMiddleware())
	{
		protectedGroup.PUT("/change-password", handlers.ChangePassword)
	}

	return router
}
