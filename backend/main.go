package main

import (
	"log"
	"net"
	"sync"
	"time"
	"watchflare/backend/cache"
	"watchflare/backend/config"
	"watchflare/backend/database"
	grpcservice "watchflare/backend/grpc"
	"watchflare/backend/handlers"
	"watchflare/backend/middleware"
	"watchflare/backend/pki"
	"watchflare/backend/services"
	pb "watchflare/shared/proto"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// Load configuration
	config.Load()

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize PKI (auto-generate or validate custom certs)
	pkiConfig := &pki.Config{
		Mode:   pki.Mode(config.AppConfig.TLSMode),
		PKIDir: config.AppConfig.TLSPKIDir,

		// Custom mode fields
		CertFile: config.AppConfig.TLSCertFile,
		KeyFile:  config.AppConfig.TLSKeyFile,
		CAFile:   config.AppConfig.TLSCAFile,
	}

	pkiInstance, err := pki.New(pkiConfig)
	if err != nil {
		log.Fatalf("Failed to initialize PKI: %v", err)
	}

	if err := pkiInstance.Initialize(); err != nil {
		log.Fatalf("Failed to initialize PKI: %v", err)
	}

	// Store PKI instance in context for gRPC server and handlers
	grpcservice.SetPKI(pkiInstance)

	// Start heartbeat cache workers
	// Sync worker: writes cache to DB every 5 minutes
	syncWorker := cache.NewSyncWorker(5 * time.Minute)
	go syncWorker.Start()

	// Stale checker: marks agents offline if no heartbeat for 15s (3x 5s interval)
	staleChecker := cache.NewStaleChecker(10*time.Second, 15*time.Second)
	go staleChecker.Start()

	// Start aggregated metrics scheduler (broadcasts via SSE every 30s)
	aggregatedMetricsScheduler := services.NewAggregatedMetricsScheduler(30 * time.Second)
	go aggregatedMetricsScheduler.Start()

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
		if err := startGRPCServer(config.AppConfig.GRPCPort, config.AppConfig, pkiInstance); err != nil {
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
		protectedGroup.GET("/user", handlers.GetCurrentUser)
		protectedGroup.PUT("/preferences", handlers.UpdatePreferences)
		protectedGroup.PUT("/change-password", handlers.ChangePassword)
	}

	// Server routes (protected)
	serverGroup := router.Group("/servers")
	serverGroup.Use(middleware.AuthMiddleware())
	{
		serverGroup.POST("", handlers.CreateAgent)
		serverGroup.GET("", handlers.ListServers)
		serverGroup.GET("/:id", handlers.GetServer)
		serverGroup.GET("/:id/metrics", handlers.GetMetrics)
		serverGroup.GET("/metrics/aggregated", handlers.GetAggregatedMetrics)
		serverGroup.PUT("/:id/validate-ip", handlers.ValidateIP)
		serverGroup.PUT("/:id/change-ip", handlers.UpdateConfiguredIP)
		serverGroup.PUT("/:id/ignore-ip-mismatch", handlers.IgnoreIPMismatch)
		serverGroup.PUT("/:id/dismiss-reactivation", handlers.DismissReactivation)
		serverGroup.POST("/:id/regenerate-token", handlers.RegenerateToken)
		serverGroup.DELETE("/:id", handlers.DeleteServer)
		serverGroup.GET("/events", handlers.ServerEvents)
		serverGroup.GET("/dropped-metrics", handlers.GetDroppedMetrics)

		// Package inventory routes
		serverGroup.GET("/:id/packages", handlers.GetServerPackages)
		serverGroup.GET("/:id/packages/history", handlers.GetServerPackageHistory)
		serverGroup.GET("/:id/packages/collections", handlers.GetServerPackageCollections)
		serverGroup.GET("/:id/packages/stats", handlers.GetPackageStats)
	}

	return router
}

// startGRPCServer initializes and starts the gRPC server
func startGRPCServer(port string, cfg *config.Config, pkiInstance *pki.PKI) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	var opts []grpc.ServerOption

	// TLS configuration (mandatory)
	tlsConfig, err := pkiInstance.GetTLSConfig()
	if err != nil {
		return err
	}

	creds := credentials.NewTLS(tlsConfig)
	opts = append(opts, grpc.Creds(creds))
	log.Printf("gRPC TLS enabled (TLS 1.3, mode: %s)", cfg.TLSMode)

	// Authentication interceptor (HMAC mandatory)
	opts = append(opts, grpc.UnaryInterceptor(
		grpcservice.AuthInterceptor(cfg.GRPCTimestampWindow),
	))

	log.Printf("gRPC HMAC validation enabled (timestamp window: %ds)", cfg.GRPCTimestampWindow)

	grpcServer := grpc.NewServer(opts...)
	agentService := grpcservice.NewAgentServer()
	pb.RegisterAgentServiceServer(grpcServer, agentService)

	log.Printf("Starting gRPC server on port %s", port)
	return grpcServer.Serve(listener)
}
