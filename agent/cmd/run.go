package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"watchflare-agent/client"
	"watchflare-agent/config"
	"watchflare-agent/errors"
	"watchflare-agent/metrics"
	"watchflare-agent/packages"
	pb "watchflare/shared/proto"
	"watchflare-agent/sysinfo"
	"watchflare-agent/wal"
)

// Run starts the agent in normal operation mode
func Run() {
	log.Println("Watchflare Agent V1 starting...")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Ensure directories exist
	if err := config.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	// Create gRPC client
	grpcClient, err := client.New(cfg.ServerHost, cfg.ServerPort, cfg.CACertFile, cfg.ServerName)
	if err != nil {
		log.Fatalf("Failed to create gRPC client: %v", err)
	}
	defer grpcClient.Close()

	log.Printf("Connected to backend: %s:%s", cfg.ServerHost, cfg.ServerPort)
	if cfg.CACertFile != "" {
		log.Printf("TLS enabled with CA cert: %s", cfg.CACertFile)
	}

	// Initialize WAL
	var walInstance *wal.WAL
	if cfg.WALEnabled != nil && *cfg.WALEnabled {
		walInstance, err = wal.New(cfg.WALPath, cfg.WALMaxSizeMB)
		if err != nil {
			log.Fatalf("Failed to initialize WAL: %v", err)
		}
		defer walInstance.Close()

		log.Printf("WAL enabled: %s (max: %d MB)", cfg.WALPath, cfg.WALMaxSizeMB)
	} else {
		log.Println("WAL disabled (metrics will be lost if send fails)")
	}

	// Detect environment and create metrics config
	env := sysinfo.DetectEnvironment()
	metricsConfig := sysinfo.GetMetricsConfig(env, *cfg.DockerMetrics)
	log.Printf("Environment: %s", env.String())
	if *cfg.DockerMetrics {
		log.Println("Docker container metrics: enabled")
	}

	// Initialize metrics collector (important for macOS CPU metrics)
	log.Println("Initializing metrics collector...")
	metrics.Initialize()

	// Create sender with metrics config
	sender := wal.NewSender(walInstance, grpcClient, cfg.AgentID, cfg.AgentKey, cfg.MetricsInterval, cfg.WALMaxSizeMB, metricsConfig)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start heartbeat in background
	go runHeartbeat(ctx, grpcClient, cfg)

	// Start sender in background
	go func() {
		if err := sender.Run(ctx); err != nil {
			log.Printf("Sender error: %v", err)
		}
	}()

	// Start package collector in background
	go runPackageCollector(ctx, grpcClient, cfg)

	// Wait for signal
	sig := <-sigCh
	log.Printf("Received signal %v, shutting down gracefully...", sig)

	// Cancel context (triggers shutdown in sender and heartbeat)
	cancel()

	// Give sender time to flush (handled internally with 5s timeout)
	time.Sleep(100 * time.Millisecond)

	log.Println("Shutdown complete")
}

// loadConfig loads and validates configuration
func loadConfig() (*config.Config, error) {
	if !config.Exists() {
		return nil, fmt.Errorf("config file not found, run 'watchflare-agent register' first")
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	// Validate required fields
	if cfg.ServerHost == "" {
		return nil, fmt.Errorf("server_host is required")
	}
	if cfg.ServerPort == "" {
		return nil, fmt.Errorf("server_port is required")
	}
	if cfg.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if cfg.AgentKey == "" {
		return nil, fmt.Errorf("agent_key is required")
	}

	return cfg, nil
}

// runHeartbeat sends periodic heartbeats to the backend
func runHeartbeat(ctx context.Context, grpcClient *client.Client, cfg *config.Config) {
	ticker := time.NewTicker(time.Duration(cfg.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	log.Printf("Heartbeat started (interval: %ds)", cfg.HeartbeatInterval)

	for {
		select {
		case <-ticker.C:
			if err := grpcClient.Heartbeat(cfg.AgentID, cfg.AgentKey); err != nil {
				log.Println(errors.FormatError(err, "Heartbeat"))
			} else {
				log.Println("✓ Heartbeat sent")
			}

		case <-ctx.Done():
			log.Println("Heartbeat stopped")
			return
		}
	}
}

// runPackageCollector collects and sends package inventory
func runPackageCollector(ctx context.Context, grpcClient *client.Client, cfg *config.Config) {
	statePath := filepath.Join(config.GetDataDir(), "packages.state.json")

	log.Println("Package collector started")

	// Wait 60 seconds before initial collection (let system stabilize)
	log.Println("Waiting 60s before initial package collection...")
	select {
	case <-time.After(60 * time.Second):
		// Initial collection
		collectAndSendPackages(ctx, grpcClient, cfg, statePath)
	case <-ctx.Done():
		log.Println("Package collector stopped before initial collection")
		return
	}

	// Setup daily ticker for 3 AM
	// Calculate time until next 3 AM
	now := time.Now()
	next3AM := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
	if now.Hour() >= 3 {
		// If it's after 3 AM today, schedule for tomorrow
		next3AM = next3AM.Add(24 * time.Hour)
	}

	timeUntil3AM := time.Until(next3AM)
	log.Printf("Next package collection scheduled for: %s (in %v)", next3AM.Format("2006-01-02 15:04:05"), timeUntil3AM)

	// Create ticker for daily collection
	timer := time.NewTimer(timeUntil3AM)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// Daily collection at 3 AM
			collectAndSendPackages(ctx, grpcClient, cfg, statePath)

			// Reset timer for next day at 3 AM
			timer.Reset(24 * time.Hour)

		case <-ctx.Done():
			log.Println("Package collector stopped")
			return
		}
	}
}

// collectAndSendPackages performs package collection, delta calculation, and sending
func collectAndSendPackages(ctx context.Context, grpcClient *client.Client, cfg *config.Config, statePath string) {
	startTime := time.Now()
	log.Println("Starting package collection...")

	// Collect all packages
	allPackages, err := packages.CollectAll()
	if err != nil {
		log.Printf("Package collection failed: %v", err)
		return
	}

	collectionDuration := time.Since(startTime).Milliseconds()
	log.Printf("Collected %d packages in %dms", len(allPackages), collectionDuration)

	// Load previous state
	state, err := packages.LoadState(statePath)
	if err != nil {
		log.Printf("Warning: Failed to load package state: %v", err)
		state = &packages.PackageState{Packages: make([]*packages.Package, 0)}
	}

	// Compute delta
	added, removed, updated := state.ComputeDelta(allPackages)

	// Check if we should send
	isFirstRun := len(state.Packages) == 0
	hasChanges := packages.HasChanges(added, removed, updated)

	if !isFirstRun && !hasChanges {
		log.Println("No package changes detected, skipping send")
		return
	}

	// Determine inventory type
	var inventoryType string
	if isFirstRun {
		inventoryType = "full"
		log.Printf("First run detected, sending full inventory (%d packages)", len(allPackages))
	} else {
		inventoryType = "delta"
		log.Printf("Changes detected: +%d added, -%d removed, ~%d updated", len(added), len(removed), len(updated))
	}

	// Convert packages to protobuf format
	var addedProto, removedProto, updatedProto, allProto []*pb.Package

	if inventoryType == "full" {
		allProto = convertPackagesToProto(allPackages)
	} else {
		addedProto = convertPackagesToProto(added)
		removedProto = convertPackagesToProto(removed)
		updatedProto = convertPackagesToProto(updated)
	}

	// Send to backend
	inventoryData := &client.PackageInventoryData{
		InventoryType:        inventoryType,
		AddedPackages:        addedProto,
		RemovedPackages:      removedProto,
		UpdatedPackages:      updatedProto,
		AllPackages:          allProto,
		CollectionDurationMs: collectionDuration,
		TotalPackageCount:    int32(len(allPackages)),
	}

	if err := grpcClient.SendPackageInventory(cfg.AgentID, cfg.AgentKey, inventoryData); err != nil {
		log.Printf("Failed to send package inventory: %v", err)
		return
	}

	log.Printf("✓ Package inventory sent successfully (%s: +%d, -%d, ~%d)",
		inventoryType, len(added), len(removed), len(updated))

	// Update local state
	state.Packages = allPackages
	state.LastScan = time.Now()
	state.PackageCount = len(allPackages)

	if err := state.Save(statePath); err != nil {
		log.Printf("Warning: Failed to save package state: %v", err)
	} else {
		log.Printf("✓ Package state saved to %s", statePath)
	}
}

// convertPackagesToProto converts agent Package structs to protobuf Package structs
func convertPackagesToProto(packages []*packages.Package) []*pb.Package {
	protoPackages := make([]*pb.Package, len(packages))

	for i, pkg := range packages {
		var installedAt int64
		if !pkg.InstalledAt.IsZero() {
			installedAt = pkg.InstalledAt.Unix()
		}

		protoPackages[i] = &pb.Package{
			Name:           pkg.Name,
			Version:        pkg.Version,
			Architecture:   pkg.Architecture,
			PackageManager: pkg.PackageManager,
			Source:         pkg.Source,
			InstalledAt:    installedAt,
			PackageSize:    pkg.PackageSize,
			Description:    pkg.Description,
		}
	}

	return protoPackages
}
