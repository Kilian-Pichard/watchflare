package main

import (
	"context"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	// Setup HTTP server
	router := setupRouter()
	httpServer := &http.Server{
		Addr:    ":" + config.AppConfig.Port,
		Handler: router.Handler(),
	}

	// Start HTTP server
	go func() {
		log.Printf("Starting HTTP server on port %s", config.AppConfig.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Start gRPC server
	grpcServer, err := createGRPCServer(config.AppConfig, pkiInstance)
	if err != nil {
		log.Fatalf("Failed to create gRPC server: %v", err)
	}

	grpcListener, err := net.Listen("tcp", ":"+config.AppConfig.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	go func() {
		log.Printf("Starting gRPC server on port %s", config.AppConfig.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	log.Println("Watchflare backend started successfully")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("Received signal %v, shutting down gracefully...", sig)

	// Graceful shutdown with 10s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop workers
	syncWorker.Stop()
	staleChecker.Stop()
	aggregatedMetricsScheduler.Stop()

	// Stop servers
	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Shutdown complete")
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

	// API routes under /api prefix
	api := router.Group("/api")

	// Health check endpoint
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Auth routes (public)
	authGroup := api.Group("/auth")
	{
		authGroup.GET("/setup-required", handlers.SetupRequired)
		authGroup.POST("/register", handlers.Register)
		authGroup.POST("/login", handlers.Login)
		authGroup.POST("/logout", handlers.Logout)
	}

	// Protected routes (require JWT)
	protectedGroup := api.Group("/auth")
	protectedGroup.Use(middleware.AuthMiddleware())
	{
		protectedGroup.GET("/user", handlers.GetCurrentUser)
		protectedGroup.PUT("/preferences", handlers.UpdatePreferences)
		protectedGroup.PUT("/change-password", handlers.ChangePassword)
		protectedGroup.PUT("/change-email", handlers.ChangeEmail)
		protectedGroup.PUT("/change-username", handlers.ChangeUsername)
	}

	// Server routes (protected)
	serverGroup := api.Group("/servers")
	serverGroup.Use(middleware.AuthMiddleware())
	{
		serverGroup.POST("", handlers.CreateAgent)
		serverGroup.GET("", handlers.ListServers)
		serverGroup.GET("/:id", handlers.GetServer)
		serverGroup.GET("/:id/metrics", handlers.GetMetrics)
		serverGroup.GET("/:id/container-metrics", handlers.GetContainerMetrics)
		serverGroup.GET("/metrics/aggregated", handlers.GetAggregatedMetrics)
		serverGroup.PUT("/:id/validate-ip", handlers.ValidateIP)
		serverGroup.PUT("/:id/rename", handlers.RenameServer)
		serverGroup.PUT("/:id/change-ip", handlers.UpdateConfiguredIP)
		serverGroup.PUT("/:id/ignore-ip-mismatch", handlers.IgnoreIPMismatch)
		serverGroup.PUT("/:id/dismiss-reactivation", handlers.DismissReactivation)
		serverGroup.PUT("/:id/pause", handlers.PauseServer)
		serverGroup.PUT("/:id/resume", handlers.ResumeServer)
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

	// Serve embedded frontend (SPA with fallback to index.html)
	frontendFiles, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Printf("Warning: Frontend files not found (dev mode?): %v", err)
	} else {
		fileServer := http.FileServer(http.FS(frontendFiles))
		router.NoRoute(func(c *gin.Context) {
			// Try to serve the exact file first
			path := c.Request.URL.Path
			f, err := frontendFiles.Open(path[1:]) // strip leading /
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
			// SPA fallback: serve index.html for all non-file routes
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
		})
		log.Println("Frontend embedded and served from /")
	}

	return router
}

// createGRPCServer initializes the gRPC server (does not start serving)
func createGRPCServer(cfg *config.Config, pkiInstance *pki.PKI) (*grpc.Server, error) {
	var opts []grpc.ServerOption

	// TLS configuration (mandatory)
	tlsConfig, err := pkiInstance.GetTLSConfig()
	if err != nil {
		return nil, err
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

	return grpcServer, nil
}
