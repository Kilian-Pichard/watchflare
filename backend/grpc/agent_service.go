package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"watchflare/backend/cache"
	"watchflare/backend/database"
	"watchflare/backend/models"
	"watchflare/backend/pki"
	"watchflare/backend/sse"
	pb "watchflare/shared/proto"

	"gorm.io/gorm"
)

// AgentServer implements the AgentService gRPC server
type AgentServer struct {
	pb.UnimplementedAgentServiceServer
}

// Global PKI instance (set during startup)
var pkiInstance *pki.PKI

// SetPKI stores the PKI instance for use in gRPC handlers
func SetPKI(p *pki.PKI) {
	pkiInstance = p
}

// NewAgentServer creates a new AgentServer instance
func NewAgentServer() *AgentServer {
	return &AgentServer{}
}

// RegisterServer handles initial agent registration
func (s *AgentServer) RegisterServer(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Step 1: Validate token and find the pending agent
	hashedToken := hashToken(req.RegistrationToken)
	var pendingAgent models.Server
	result := database.DB.Where("registration_token = ?", hashedToken).First(&pendingAgent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &pb.RegisterResponse{
				Success: false,
				Message: "Invalid registration token",
			}, nil
		}
		return nil, result.Error
	}

	// Step 2: Validate token hasn't expired
	if pendingAgent.ExpiresAt != nil && time.Now().After(*pendingAgent.ExpiresAt) {
		return &pb.RegisterResponse{
			Success: false,
			Message: "Registration token has expired",
		}, nil
	}

	// Step 3: Validate token is for a pending agent
	if pendingAgent.Status != "pending" && pendingAgent.Status != "expired" {
		return &pb.RegisterResponse{
			Success: false,
			Message: "Server is already registered",
		}, nil
	}

	// Step 4: Validate IP if required
	if !pendingAgent.AllowAnyIPRegistration {
		if pendingAgent.ConfiguredIP != nil && *pendingAgent.ConfiguredIP != "" {
			if req.IpAddressV4 != *pendingAgent.ConfiguredIP {
				return &pb.RegisterResponse{
					Success: false,
					Message: "IP address mismatch. Expected: " + *pendingAgent.ConfiguredIP + ", Got: " + req.IpAddressV4,
				}, nil
			}
		}
	}

	// Step 5: Check if this is a re-registration (existing UUID provided)
	var agentToUse *models.Server
	var deletePending bool

	if req.ExistingAgentUuid != "" {
		// Try to find existing agent by UUID
		var existingAgent models.Server
		result := database.DB.Where("agent_id = ?", req.ExistingAgentUuid).First(&existingAgent)
		if result.Error == nil {
			// Found existing agent - reactivate it instead of using pending
			log.Printf("Re-registration: Reactivating existing agent %s (hostname: %s)", existingAgent.AgentID, req.Hostname)
			agentToUse = &existingAgent
			deletePending = true // We'll delete the unused pending agent
		} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Database error (not "not found")
			return nil, result.Error
		}
		// If UUID not found, fall through to use pending agent
	}

	// Step 6: If no existing agent found, use the pending agent
	if agentToUse == nil {
		log.Printf("New registration: Using pending agent %s (hostname: %s)", pendingAgent.AgentID, req.Hostname)
		agentToUse = &pendingAgent
		deletePending = false
	}

	// Step 7: Update the agent with registration information
	now := time.Now()
	updates := map[string]interface{}{
		"hostname":           req.Hostname,
		"ip_address_v4":      req.IpAddressV4,
		"ip_address_v6":      req.IpAddressV6,
		"platform":           req.Platform,
		"platform_version":   req.PlatformVersion,
		"platform_family":    req.PlatformFamily,
		"architecture":       req.Architecture,
		"kernel":             req.Kernel,
		"environment_type":   req.EnvironmentType,
		"hypervisor":         req.Hypervisor,
		"container_runtime":  req.ContainerRuntime,
		"agent_version":      req.AgentVersion,
		"status":             "offline",
		"last_seen":          &now,
		"registration_token": nil, // Always clear token after successful registration
		"expires_at":         nil, // Clear expiration
	}

	// If this is a reactivation, set reactivated_at timestamp
	if deletePending {
		updates["reactivated_at"] = &now
	}

	if err := database.DB.Model(agentToUse).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Step 8: Delete the pending agent if we reactivated an existing one
	if deletePending {
		if err := database.DB.Delete(&pendingAgent).Error; err != nil {
			log.Printf("Warning: Failed to delete pending agent %s: %v", pendingAgent.AgentID, err)
			// Not fatal - continue with registration
		} else {
			log.Printf("Deleted unused pending agent %s", pendingAgent.AgentID)
		}
	}

	// Step 9: Broadcast SSE event for server update
	broker := sse.GetBroker()
	configuredIP := ""
	if agentToUse.ConfiguredIP != nil {
		configuredIP = *agentToUse.ConfiguredIP
	}
	broker.BroadcastServerUpdate(sse.ServerUpdate{
		ID:               agentToUse.ID,
		Status:           "offline",
		IPv4Address:      req.IpAddressV4,
		IPv6Address:      req.IpAddressV6,
		ConfiguredIP:     configuredIP,
		IgnoreIPMismatch: agentToUse.IgnoreIPMismatch,
		LastSeen:         now.Format(time.RFC3339),
		Reactivated:      deletePending, // True if existing agent was reactivated
		Hostname:         req.Hostname,  // For notification message
	})

	// Step 10: Get CA certificate for agent TLS verification
	caCertPEM, err := pkiInstance.GetCACertPEM()
	if err != nil {
		return nil, fmt.Errorf("failed to get CA certificate: %w", err)
	}

	return &pb.RegisterResponse{
		Success:     true,
		Message:     "Server registered successfully",
		AgentId:     agentToUse.AgentID,
		AgentKey:    agentToUse.AgentKey,
		CaCert:      string(caCertPEM),
		ServerName:  "watchflare",
		Reactivated: deletePending, // True if we reactivated existing agent
	}, nil
}

// Heartbeat handles periodic heartbeats from agents
func (s *AgentServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	// Verify agent credentials (read-only DB query)
	var server models.Server
	result := database.DB.Where("agent_id = ? AND agent_key = ?", req.AgentId, req.AgentKey).First(&server)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &pb.HeartbeatResponse{
				Success: false,
				Message: "Invalid agent credentials",
			}, nil
		}
		return nil, result.Error
	}

	// If server is paused, acknowledge but don't update cache or broadcast
	if server.Status == "paused" {
		return &pb.HeartbeatResponse{
			Success: true,
			Message: "Server is paused",
		}, nil
	}

	// Update heartbeat cache (in-memory, no DB write)
	heartbeatCache := cache.GetCache()
	heartbeatCache.Update(req.AgentId, req.IpAddressV4, req.IpAddressV6)

	// Broadcast SSE event for real-time dashboard
	broker := sse.GetBroker()
	configuredIP := ""
	if server.ConfiguredIP != nil {
		configuredIP = *server.ConfiguredIP
	}
	broker.BroadcastServerUpdate(sse.ServerUpdate{
		ID:               server.ID,
		Status:           "online",
		IPv4Address:      req.IpAddressV4,
		IPv6Address:      req.IpAddressV6,
		ConfiguredIP:     configuredIP,
		IgnoreIPMismatch: server.IgnoreIPMismatch,
		LastSeen:         time.Now().Format(time.RFC3339),
	})

	return &pb.HeartbeatResponse{
		Success: true,
		Message: "Heartbeat acknowledged",
	}, nil
}

// SendMetrics handles incoming system metrics from agents
func (s *AgentServer) SendMetrics(ctx context.Context, req *pb.MetricsRequest) (*pb.MetricsResponse, error) {
	// Find server by agent ID and verify agent key
	var server models.Server
	result := database.DB.Where("agent_id = ? AND agent_key = ?", req.AgentId, req.AgentKey).First(&server)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &pb.MetricsResponse{
				Success: false,
				Message: "Invalid agent credentials",
			}, nil
		}
		return nil, result.Error
	}

	// Update agent version if it changed (e.g. after an upgrade + restart)
	if req.AgentVersion != "" {
		currentVersion := ""
		if server.AgentVersion != nil {
			currentVersion = *server.AgentVersion
		}
		if req.AgentVersion != currentVersion {
			if err := database.DB.Model(&server).Update("agent_version", req.AgentVersion).Error; err != nil {
				log.Printf("Warning: failed to update agent version for %s: %v", server.ID, err)
			}
		}
	}

	// If server is paused, acknowledge but don't store metrics
	if server.Status == "paused" {
		log.Printf("⏸ Metrics discarded for paused server %s (%s)", server.Name, server.ID)
		return &pb.MetricsResponse{
			Success: true,
			Message: "Server is paused, metrics discarded",
		}, nil
	}

	// Create metric record
	metric := &models.Metric{
		ServerID:             server.ID,
		Timestamp:            time.Unix(req.Metrics.Timestamp, 0),
		CPUUsagePercent:      req.Metrics.CpuUsagePercent,
		MemoryTotalBytes:     req.Metrics.MemoryTotalBytes,
		MemoryUsedBytes:      req.Metrics.MemoryUsedBytes,
		MemoryAvailableBytes: req.Metrics.MemoryAvailableBytes,
		LoadAvg1Min:          req.Metrics.LoadAvg_1Min,
		LoadAvg5Min:          req.Metrics.LoadAvg_5Min,
		LoadAvg15Min:         req.Metrics.LoadAvg_15Min,
		DiskTotalBytes:        req.Metrics.DiskTotalBytes,
		DiskUsedBytes:         req.Metrics.DiskUsedBytes,
		DiskReadBytesPerSec:   req.Metrics.DiskReadBytesPerSec,
		DiskWriteBytesPerSec:  req.Metrics.DiskWriteBytesPerSec,
		NetworkRxBytesPerSec:  req.Metrics.NetworkRxBytesPerSec,
		NetworkTxBytesPerSec:  req.Metrics.NetworkTxBytesPerSec,
		CPUTemperatureCelsius: req.Metrics.CpuTemperatureCelsius,
		UptimeSeconds:         req.Metrics.UptimeSeconds,
	}

	// Broadcast SSE first for low-latency real-time display (Netdata/Prometheus pattern)
	broker := sse.GetBroker()
	broker.BroadcastMetricsUpdate(sse.MetricsUpdate{
		ServerID:             server.ID,
		Timestamp:            metric.Timestamp.Format(time.RFC3339),
		CPUUsagePercent:      metric.CPUUsagePercent,
		MemoryTotalBytes:     metric.MemoryTotalBytes,
		MemoryUsedBytes:      metric.MemoryUsedBytes,
		MemoryAvailableBytes: metric.MemoryAvailableBytes,
		LoadAvg1Min:          metric.LoadAvg1Min,
		LoadAvg5Min:          metric.LoadAvg5Min,
		LoadAvg15Min:         metric.LoadAvg15Min,
		DiskTotalBytes:        metric.DiskTotalBytes,
		DiskUsedBytes:         metric.DiskUsedBytes,
		DiskReadBytesPerSec:   metric.DiskReadBytesPerSec,
		DiskWriteBytesPerSec:  metric.DiskWriteBytesPerSec,
		NetworkRxBytesPerSec:  metric.NetworkRxBytesPerSec,
		NetworkTxBytesPerSec:  metric.NetworkTxBytesPerSec,
		CPUTemperatureCelsius: metric.CPUTemperatureCelsius,
		UptimeSeconds:         metric.UptimeSeconds,
	})

	if len(req.ContainerMetrics) > 0 {
		var containerModels []models.ContainerMetric
		var sseContainerMetrics []sse.ContainerMetricMinified

		for _, cm := range req.ContainerMetrics {
			containerModels = append(containerModels, models.ContainerMetric{
				ServerID:             server.ID,
				Timestamp:            metric.Timestamp,
				ContainerID:          cm.ContainerId,
				ContainerName:        cm.ContainerName,
				Image:                cm.Image,
				CPUPercent:           cm.CpuPercent,
				MemoryUsedBytes:      cm.MemoryUsedBytes,
				MemoryLimitBytes:     cm.MemoryLimitBytes,
				NetworkRxBytesPerSec: cm.NetworkRxBytesPerSec,
				NetworkTxBytesPerSec: cm.NetworkTxBytesPerSec,
			})

			sseContainerMetrics = append(sseContainerMetrics, sse.ContainerMetricMinified{
				ID:   cm.ContainerId,
				Name: cm.ContainerName,
				CPU:  cm.CpuPercent,
				MU:   cm.MemoryUsedBytes,
				ML:   cm.MemoryLimitBytes,
				NR:   cm.NetworkRxBytesPerSec,
				NT:   cm.NetworkTxBytesPerSec,
			})
		}

		broker.BroadcastContainerMetricsUpdate(sse.ContainerMetricsUpdate{
			ServerID:  server.ID,
			Timestamp: metric.Timestamp.Unix(),
			Metrics:   sseContainerMetrics,
		})

		// Persist container metrics to DB (after SSE for lower latency)
		if err := database.DB.Create(&containerModels).Error; err != nil {
			log.Printf("Warning: Failed to save container metrics: %v", err)
		}
	}

	// Persist system metric to DB (after SSE for lower latency)
	if err := database.DB.Create(metric).Error; err != nil {
		return nil, fmt.Errorf("failed to save metrics: %w", err)
	}

	return &pb.MetricsResponse{
		Success: true,
		Message: "Metrics received successfully",
	}, nil
}

// ReportDroppedMetrics handles reports of metrics that were dropped by agents
func (s *AgentServer) ReportDroppedMetrics(ctx context.Context, req *pb.DroppedMetricsReport) (*pb.DroppedMetricsResponse, error) {
	// Verify agent credentials
	var server models.Server
	result := database.DB.Where("agent_id = ? AND agent_key = ?", req.AgentId, req.AgentKey).First(&server)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &pb.DroppedMetricsResponse{
				Success: false,
				Message: "Invalid agent credentials",
			}, nil
		}
		return nil, result.Error
	}

	// Insert dropped metrics report into database
	err := database.DB.Exec(`
		INSERT INTO dropped_metrics
		(agent_id, count, first_dropped_at, last_dropped_at, reason)
		VALUES ($1, $2, $3, $4, $5)
	`,
		server.ID,
		req.Count,
		time.Unix(req.FirstDroppedAt, 0),
		time.Unix(req.LastDroppedAt, 0),
		req.Reason,
	).Error

	if err != nil {
		log.Printf("Error: Failed to insert dropped metrics report: %v", err)
		return nil, fmt.Errorf("failed to save dropped metrics report: %w", err)
	}

	// Calculate downtime duration for logging
	downtimeDuration := time.Unix(req.LastDroppedAt, 0).Sub(time.Unix(req.FirstDroppedAt, 0))

	log.Printf("⚠️  Agent %s (%s) reported %d dropped metrics (downtime: %v, reason: %s)",
		server.Name,
		req.AgentId,
		req.Count,
		downtimeDuration.Round(time.Second),
		req.Reason,
	)

	return &pb.DroppedMetricsResponse{
		Success: true,
		Message: "Dropped metrics report received",
	}, nil
}

// SendPackageInventory handles package inventory updates from agents
func (s *AgentServer) SendPackageInventory(ctx context.Context, req *pb.PackageInventoryRequest) (*pb.PackageInventoryResponse, error) {
	// Verify agent credentials
	var server models.Server
	result := database.DB.Where("agent_id = ? AND agent_key = ?", req.AgentId, req.AgentKey).First(&server)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &pb.PackageInventoryResponse{
				Success: false,
				Message: "Invalid agent credentials",
			}, nil
		}
		return nil, result.Error
	}

	// Process package inventory
	packagesProcessed, changesDetected, err := processPackageInventory(server.ID, req)
	if err != nil {
		log.Printf("Error: Failed to process package inventory for server %s: %v", server.ID, err)
		return &pb.PackageInventoryResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to process package inventory: %v", err),
		}, nil
	}

	// Log successful processing
	log.Printf("✓ Package inventory processed for %s (%s): %d packages, %d changes (%s, %dms)",
		server.Name,
		server.ID,
		packagesProcessed,
		changesDetected,
		req.InventoryType,
		req.CollectionDurationMs,
	)

	return &pb.PackageInventoryResponse{
		Success:           true,
		Message:           "Package inventory received successfully",
		PackagesProcessed: int32(packagesProcessed),
		ChangesDetected:   int32(changesDetected),
	}, nil
}

// processPackageInventory handles the business logic for package inventory updates
func processPackageInventory(serverID string, req *pb.PackageInventoryRequest) (int, int, error) {
	tx := database.DB.Begin()
	if tx.Error != nil {
		return 0, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()
	packagesProcessed := 0
	changesDetected := 0

	// Handle full inventory
	if req.InventoryType == "full" {
		// Process all packages (first run)
		for _, pkg := range req.AllPackages {
			// Upsert package (insert or update if exists)
			installedAt := convertTimestamp(pkg.InstalledAt)

			packageModel := models.Package{
				ServerID:       serverID,
				Name:           pkg.Name,
				Version:        pkg.Version,
				Architecture:   pkg.Architecture,
				PackageManager: pkg.PackageManager,
				Source:         pkg.Source,
				InstalledAt:    installedAt,
				PackageSize:    pkg.PackageSize,
				Description:    pkg.Description,
				FirstSeen:      now,
				LastSeen:       now,
			}

			// Try to update existing package, or insert if not exists
			result := tx.Where("server_id = ? AND name = ? AND package_manager = ?",
				serverID, pkg.Name, pkg.PackageManager).
				Assign(map[string]interface{}{
					"version":      pkg.Version,
					"architecture": pkg.Architecture,
					"source":       pkg.Source,
					"installed_at": installedAt,
					"package_size": pkg.PackageSize,
					"description":  pkg.Description,
					"last_seen":    now,
				}).
				FirstOrCreate(&packageModel)

			if result.Error != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to upsert package %s: %w", pkg.Name, result.Error)
			}

			// Create history record for initial state
			historyRecord := models.PackageHistory{
				Timestamp:      now,
				ServerID:       serverID,
				Name:           pkg.Name,
				Version:        pkg.Version,
				Architecture:   pkg.Architecture,
				PackageManager: pkg.PackageManager,
				Source:         pkg.Source,
				PackageSize:    pkg.PackageSize,
				Description:    pkg.Description,
				ChangeType:     "initial",
			}

			if err := tx.Create(&historyRecord).Error; err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to create history record for %s: %w", pkg.Name, err)
			}

			packagesProcessed++
		}

		changesDetected = packagesProcessed

	} else if req.InventoryType == "delta" {
		// Process added packages
		for _, pkg := range req.AddedPackages {
			installedAt := convertTimestamp(pkg.InstalledAt)

			packageModel := models.Package{
				ServerID:       serverID,
				Name:           pkg.Name,
				Version:        pkg.Version,
				Architecture:   pkg.Architecture,
				PackageManager: pkg.PackageManager,
				Source:         pkg.Source,
				InstalledAt:    installedAt,
				PackageSize:    pkg.PackageSize,
				Description:    pkg.Description,
				FirstSeen:      now,
				LastSeen:       now,
			}

			// Use FirstOrCreate to handle cases where package already exists (desync between agent and backend)
			result := tx.Where("server_id = ? AND name = ? AND package_manager = ?",
				serverID, pkg.Name, pkg.PackageManager).
				Assign(map[string]interface{}{
					"version":      pkg.Version,
					"architecture": pkg.Architecture,
					"source":       pkg.Source,
					"installed_at": installedAt,
					"package_size": pkg.PackageSize,
					"description":  pkg.Description,
					"last_seen":    now,
				}).
				FirstOrCreate(&packageModel)

			if result.Error != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to upsert added package %s: %w", pkg.Name, result.Error)
			}

			// History record
			historyRecord := models.PackageHistory{
				Timestamp:      now,
				ServerID:       serverID,
				Name:           pkg.Name,
				Version:        pkg.Version,
				Architecture:   pkg.Architecture,
				PackageManager: pkg.PackageManager,
				Source:         pkg.Source,
				PackageSize:    pkg.PackageSize,
				Description:    pkg.Description,
				ChangeType:     "added",
			}

			if err := tx.Create(&historyRecord).Error; err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to create history for added package %s: %w", pkg.Name, err)
			}

			changesDetected++
		}

		// Process removed packages
		for _, pkg := range req.RemovedPackages {
			// Delete from packages table
			result := tx.Where("server_id = ? AND name = ? AND package_manager = ?",
				serverID, pkg.Name, pkg.PackageManager).
				Delete(&models.Package{})

			if result.Error != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to delete removed package %s: %w", pkg.Name, result.Error)
			}

			// History record
			historyRecord := models.PackageHistory{
				Timestamp:      now,
				ServerID:       serverID,
				Name:           pkg.Name,
				Version:        pkg.Version,
				Architecture:   pkg.Architecture,
				PackageManager: pkg.PackageManager,
				Source:         pkg.Source,
				PackageSize:    pkg.PackageSize,
				Description:    pkg.Description,
				ChangeType:     "removed",
			}

			if err := tx.Create(&historyRecord).Error; err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to create history for removed package %s: %w", pkg.Name, err)
			}

			changesDetected++
		}

		// Process updated packages
		for _, pkg := range req.UpdatedPackages {
			installedAt := convertTimestamp(pkg.InstalledAt)

			// Update existing package
			result := tx.Model(&models.Package{}).
				Where("server_id = ? AND name = ? AND package_manager = ?",
					serverID, pkg.Name, pkg.PackageManager).
				Updates(map[string]interface{}{
					"version":      pkg.Version,
					"architecture": pkg.Architecture,
					"source":       pkg.Source,
					"installed_at": installedAt,
					"package_size": pkg.PackageSize,
					"description":  pkg.Description,
					"last_seen":    now,
				})

			if result.Error != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to update package %s: %w", pkg.Name, result.Error)
			}

			// History record
			historyRecord := models.PackageHistory{
				Timestamp:      now,
				ServerID:       serverID,
				Name:           pkg.Name,
				Version:        pkg.Version,
				Architecture:   pkg.Architecture,
				PackageManager: pkg.PackageManager,
				Source:         pkg.Source,
				PackageSize:    pkg.PackageSize,
				Description:    pkg.Description,
				ChangeType:     "updated",
			}

			if err := tx.Create(&historyRecord).Error; err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to create history for updated package %s: %w", pkg.Name, err)
			}

			changesDetected++
		}

		packagesProcessed = int(req.TotalPackageCount)
	}

	// Create package collection metadata record
	collectionRecord := models.PackageCollection{
		ServerID:       serverID,
		Timestamp:      now,
		CollectionType: req.InventoryType,
		PackageCount:   int(req.TotalPackageCount),
		ChangesCount:   changesDetected,
		DurationMs:     int(req.CollectionDurationMs),
		Status:         "success",
	}

	if err := tx.Create(&collectionRecord).Error; err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to create collection record: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return packagesProcessed, changesDetected, nil
}

// convertTimestamp converts Unix timestamp to *time.Time (nil if 0)
func convertTimestamp(ts int64) *time.Time {
	if ts == 0 {
		return nil
	}
	t := time.Unix(ts, 0)
	return &t
}

// hashToken creates a SHA-256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
