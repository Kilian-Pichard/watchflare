package main

import (
	"log"
	"net"
	"sync"
	"watchflare/backend/config"
	"watchflare/backend/database"
	grpcservice "watchflare/backend/grpc"
	"watchflare/backend/handlers"
	"watchflare/backend/middleware"
	pb "watchflare/backend/proto"
	"watchflare/backend/scheduler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	// Load configuration
	config.Load()

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Start offline checker
	scheduler.StartOfflineChecker()

	// Use WaitGroup to run both servers concurrently
	var wg sync.WaitGroup
	wg.Add(2)

	// Start HTTP server
	go func() {
		defer wg.Done()
		router := setupRouter()
		log.Printf("Starting HTTP server on port %s", config.AppConfig.Port)
		if err := router.Run(":" + config.AppConfig.Port); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Start gRPC server
	go func() {
		defer wg.Done()
		if err := startGRPCServer(config.AppConfig.GRPCPort); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	log.Println("Watchflare backend started successfully")
	wg.Wait()
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
		authGroup.GET("/setup-required", handlers.SetupRequired)
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

	// Server routes (protected)
	serverGroup := router.Group("/servers")
	serverGroup.Use(middleware.AuthMiddleware())
	{
		serverGroup.POST("", handlers.CreateAgent)
		serverGroup.GET("", handlers.ListServers)
		serverGroup.GET("/:id", handlers.GetServer)
		serverGroup.PUT("/:id/validate-ip", handlers.ValidateIP)
		serverGroup.PUT("/:id/change-ip", handlers.UpdateConfiguredIP)
		serverGroup.PUT("/:id/ignore-ip-mismatch", handlers.IgnoreIPMismatch)
		serverGroup.POST("/:id/regenerate-token", handlers.RegenerateToken)
		serverGroup.DELETE("/:id", handlers.DeleteServer)
		serverGroup.GET("/events", handlers.ServerEvents)
	}

	return router
}

// startGRPCServer initializes and starts the gRPC server
func startGRPCServer(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	agentService := grpcservice.NewAgentServer()
	pb.RegisterAgentServiceServer(grpcServer, agentService)

	log.Printf("Starting gRPC server on port %s", port)
	return grpcServer.Serve(listener)
}
