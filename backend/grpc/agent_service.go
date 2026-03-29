package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"watchflare/backend/cache"
	"watchflare/backend/database"
	"watchflare/backend/models"
	"watchflare/backend/pki"
	"watchflare/backend/sse"
	pb "watchflare/shared/proto/agent/v1"

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
func (s *AgentServer) RegisterServer(ctx context.Context, req *pb.RegisterServerRequest) (*pb.RegisterServerResponse, error) {
	// Step 1: Validate token and find the pending agent
	hashedToken := hashToken(req.RegistrationToken)
	var pendingAgent models.Server
	result := database.DB.Where("registration_token = ?", hashedToken).First(&pendingAgent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &pb.RegisterServerResponse{
				Success: false,
				Message: "Invalid registration token",
			}, nil
		}
		return nil, result.Error
	}

	// Step 2: Validate token hasn't expired
	if pendingAgent.ExpiresAt != nil && time.Now().After(*pendingAgent.ExpiresAt) {
		return &pb.RegisterServerResponse{
			Success: false,
			Message: "Registration token has expired",
		}, nil
	}

	// Step 3: Validate token is for a pending agent
	if pendingAgent.Status != models.StatusPending && pendingAgent.Status != models.StatusExpired {
		return &pb.RegisterServerResponse{
			Success: false,
			Message: "Server is already registered",
		}, nil
	}

	// Step 4: Validate IP if required
	if !pendingAgent.AllowAnyIPRegistration {
		if pendingAgent.ConfiguredIP != nil && *pendingAgent.ConfiguredIP != "" {
			if req.IpAddressV4 != *pendingAgent.ConfiguredIP {
				return &pb.RegisterServerResponse{
					Success: false,
					Message: "IP address mismatch. Expected: " + *pendingAgent.ConfiguredIP + ", Got: " + req.IpAddressV4,
				}, nil
			}
		}
	}

	// Step 5: Check if this is a re-registration (existing UUID provided).
	// The UUID must match the pending agent's own AgentID to prevent a token holder
	// from hijacking an arbitrary existing agent.
	var agentToUse *models.Server
	var deletePending bool

	if req.ExistingAgentUuid != "" {
		if req.ExistingAgentUuid != pendingAgent.AgentID {
			return &pb.RegisterServerResponse{
				Success: false,
				Message: "Invalid registration request",
			}, nil
		}

		// Try to find existing agent by UUID
		var existingAgent models.Server
		result := database.DB.Where("agent_id = ?", req.ExistingAgentUuid).First(&existingAgent)
		if result.Error == nil {
			// Found existing agent - reactivate it instead of using pending
			slog.Warn("re-registration: reactivating existing agent", "agent_id", existingAgent.AgentID, "hostname", req.Hostname)
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
		slog.Info("new registration", "agent_id", pendingAgent.AgentID, "hostname", req.Hostname)
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
		"status":             models.StatusOffline,
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

	// Step 8: Delete the pending agent if we reactivated an existing one.
	// Skip deletion if both point to the same record (token was regenerated on the existing agent).
	if deletePending && pendingAgent.ID != agentToUse.ID {
		if err := database.DB.Delete(&pendingAgent).Error; err != nil {
			slog.Warn("failed to delete pending agent", "agent_id", pendingAgent.AgentID, "error", err)
			// Not fatal - continue with registration
		} else {
			slog.Info("deleted unused pending agent", "agent_id", pendingAgent.AgentID)
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
		Status:           models.StatusOffline,
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

	return &pb.RegisterServerResponse{
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
	if server.Status == models.StatusPaused {
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
		Status:           models.StatusOnline,
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
func (s *AgentServer) SendMetrics(ctx context.Context, req *pb.SendMetricsRequest) (*pb.SendMetricsResponse, error) {
	// Find server by agent ID and verify agent key
	var server models.Server
	result := database.DB.Where("agent_id = ? AND agent_key = ?", req.AgentId, req.AgentKey).First(&server)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &pb.SendMetricsResponse{
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
				slog.Warn("failed to update agent version", "server_id", server.ID, "error", err)
			}
		}
	}

	if req.Metrics == nil {
		return &pb.SendMetricsResponse{
			Success: false,
			Message: "metrics payload is required",
		}, nil
	}

	// If server is paused, acknowledge but don't store metrics
	if server.Status == models.StatusPaused {
		slog.Info("metrics discarded for paused server", "name", server.Name, "server_id", server.ID)
		return &pb.SendMetricsResponse{
			Success: true,
			Message: "Server is paused, metrics discarded",
		}, nil
	}

	// Convert proto sensor readings to model type and SSE minified format in one pass
	var sensorReadings models.SensorReadings
	var sseSensorReadings []sse.SensorReadingMinified
	for _, sr := range req.Metrics.SensorReadings {
		sensorReadings = append(sensorReadings, models.SensorReading{
			Key:                sr.Key,
			TemperatureCelsius: sr.TemperatureCelsius,
		})
		sseSensorReadings = append(sseSensorReadings, sse.SensorReadingMinified{
			K: sr.Key,
			V: sr.TemperatureCelsius,
		})
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
		SensorReadings:        sensorReadings,
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
		SensorReadings:        sseSensorReadings,
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
			slog.Warn("failed to save container metrics", "server_id", server.ID, "error", err)
		}
	}

	// Persist system metric to DB (after SSE for lower latency)
	if err := database.DB.Create(metric).Error; err != nil {
		return nil, fmt.Errorf("failed to save metrics: %w", err)
	}

	// Persist normalized sensor readings for multi-range aggregation
	if len(sensorReadings) > 0 {
		sensorMetrics := make([]models.SensorMetric, len(sensorReadings))
		for i, sr := range sensorReadings {
			sensorMetrics[i] = models.SensorMetric{
				Time:        metric.Timestamp,
				ServerID:    server.ID,
				SensorKey:   sr.Key,
				Temperature: sr.TemperatureCelsius,
			}
		}
		if err := database.DB.Create(&sensorMetrics).Error; err != nil {
			slog.Warn("failed to save sensor metrics", "server_id", server.ID, "error", err)
		}
	}

	return &pb.SendMetricsResponse{
		Success: true,
		Message: "Metrics received successfully",
	}, nil
}

// ReportDroppedMetrics handles reports of metrics that were dropped by agents
func (s *AgentServer) ReportDroppedMetrics(ctx context.Context, req *pb.ReportDroppedMetricsRequest) (*pb.ReportDroppedMetricsResponse, error) {
	// Verify agent credentials
	var server models.Server
	result := database.DB.Where("agent_id = ? AND agent_key = ?", req.AgentId, req.AgentKey).First(&server)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &pb.ReportDroppedMetricsResponse{
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
		slog.Error("failed to insert dropped metrics report", "server_id", server.ID, "error", err)
		return nil, fmt.Errorf("failed to save dropped metrics report: %w", err)
	}

	// Calculate downtime duration for logging
	downtimeDuration := time.Unix(req.LastDroppedAt, 0).Sub(time.Unix(req.FirstDroppedAt, 0))

	slog.Warn("agent reported dropped metrics",
		"name", server.Name,
		"agent_id", req.AgentId,
		"count", req.Count,
		"downtime", downtimeDuration.Round(time.Second),
		"reason", req.Reason,
	)

	return &pb.ReportDroppedMetricsResponse{
		Success: true,
		Message: "Dropped metrics report received",
	}, nil
}

// SendPackageInventory handles package inventory updates from agents
func (s *AgentServer) SendPackageInventory(ctx context.Context, req *pb.SendPackageInventoryRequest) (*pb.SendPackageInventoryResponse, error) {
	// Verify agent credentials
	var server models.Server
	result := database.DB.Where("agent_id = ? AND agent_key = ?", req.AgentId, req.AgentKey).First(&server)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &pb.SendPackageInventoryResponse{
				Success: false,
				Message: "Invalid agent credentials",
			}, nil
		}
		return nil, result.Error
	}

	// Process package inventory
	packagesProcessed, changesDetected, err := processPackageInventory(server.ID, req)
	if err != nil {
		slog.Error("failed to process package inventory", "server_id", server.ID, "error", err)
		return &pb.SendPackageInventoryResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to process package inventory: %v", err),
		}, nil
	}

	slog.Info("package inventory processed",
		"name", server.Name,
		"server_id", server.ID,
		"packages", packagesProcessed,
		"changes", changesDetected,
		"type", req.InventoryType,
		"duration_ms", req.CollectionDurationMs,
	)

	return &pb.SendPackageInventoryResponse{
		Success:           true,
		Message:           "Package inventory received successfully",
		PackagesProcessed: int32(packagesProcessed),
		ChangesDetected:   int32(changesDetected),
	}, nil
}

// pkgFields returns the mutable fields map used for package upsert/update.
func pkgFields(pkg *pb.Package, installedAt *time.Time, now time.Time) map[string]interface{} {
	return map[string]interface{}{
		"version":      pkg.Version,
		"architecture": pkg.Architecture,
		"source":       pkg.Source,
		"installed_at": installedAt,
		"package_size": pkg.PackageSize,
		"description":  pkg.Description,
		"last_seen":    now,
	}
}

// writeHistory inserts a package history record within a transaction.
func writeHistory(tx *gorm.DB, serverID string, pkg *pb.Package, changeType string, now time.Time) error {
	return tx.Create(&models.PackageHistory{
		Timestamp:      now,
		ServerID:       serverID,
		Name:           pkg.Name,
		Version:        pkg.Version,
		Architecture:   pkg.Architecture,
		PackageManager: pkg.PackageManager,
		Source:         pkg.Source,
		PackageSize:    pkg.PackageSize,
		Description:    pkg.Description,
		ChangeType:     changeType,
	}).Error
}

// processPackageInventory handles the business logic for package inventory updates
func processPackageInventory(serverID string, req *pb.SendPackageInventoryRequest) (int, int, error) {
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

	if req.InventoryType != models.CollectionTypeFull && req.InventoryType != models.CollectionTypeDelta {
		tx.Rollback()
		return 0, 0, fmt.Errorf("unknown inventory_type: %q", req.InventoryType)
	}

	if req.InventoryType == models.CollectionTypeFull {
		for _, pkg := range req.AllPackages {
			installedAt := convertTimestamp(pkg.InstalledAt)
			model := models.Package{ServerID: serverID, Name: pkg.Name, Version: pkg.Version, Architecture: pkg.Architecture, PackageManager: pkg.PackageManager, Source: pkg.Source, InstalledAt: installedAt, PackageSize: pkg.PackageSize, Description: pkg.Description, FirstSeen: now, LastSeen: now}
			if err := tx.Where("server_id = ? AND name = ? AND package_manager = ?", serverID, pkg.Name, pkg.PackageManager).Assign(pkgFields(pkg, installedAt, now)).FirstOrCreate(&model).Error; err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to upsert package %s: %w", pkg.Name, err)
			}
			if err := writeHistory(tx, serverID, pkg, models.ChangeTypeInitial, now); err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to create history record for %s: %w", pkg.Name, err)
			}
			packagesProcessed++
		}
		changesDetected = packagesProcessed

	} else if req.InventoryType == models.CollectionTypeDelta {
		for _, pkg := range req.AddedPackages {
			installedAt := convertTimestamp(pkg.InstalledAt)
			model := models.Package{ServerID: serverID, Name: pkg.Name, Version: pkg.Version, Architecture: pkg.Architecture, PackageManager: pkg.PackageManager, Source: pkg.Source, InstalledAt: installedAt, PackageSize: pkg.PackageSize, Description: pkg.Description, FirstSeen: now, LastSeen: now}
			if err := tx.Where("server_id = ? AND name = ? AND package_manager = ?", serverID, pkg.Name, pkg.PackageManager).Assign(pkgFields(pkg, installedAt, now)).FirstOrCreate(&model).Error; err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to upsert added package %s: %w", pkg.Name, err)
			}
			if err := writeHistory(tx, serverID, pkg, models.ChangeTypeAdded, now); err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to create history for added package %s: %w", pkg.Name, err)
			}
			changesDetected++
		}

		for _, pkg := range req.RemovedPackages {
			if err := tx.Where("server_id = ? AND name = ? AND package_manager = ?", serverID, pkg.Name, pkg.PackageManager).Delete(&models.Package{}).Error; err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to delete removed package %s: %w", pkg.Name, err)
			}
			if err := writeHistory(tx, serverID, pkg, models.ChangeTypeRemoved, now); err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to create history for removed package %s: %w", pkg.Name, err)
			}
			changesDetected++
		}

		for _, pkg := range req.UpdatedPackages {
			installedAt := convertTimestamp(pkg.InstalledAt)
			if err := tx.Model(&models.Package{}).Where("server_id = ? AND name = ? AND package_manager = ?", serverID, pkg.Name, pkg.PackageManager).Updates(pkgFields(pkg, installedAt, now)).Error; err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to update package %s: %w", pkg.Name, err)
			}
			if err := writeHistory(tx, serverID, pkg, models.ChangeTypeUpdated, now); err != nil {
				tx.Rollback()
				return 0, 0, fmt.Errorf("failed to create history for updated package %s: %w", pkg.Name, err)
			}
			changesDetected++
		}

		packagesProcessed = int(req.TotalPackageCount)
	}

	if err := tx.Create(&models.PackageCollection{
		ServerID:       serverID,
		Timestamp:      now,
		CollectionType: req.InventoryType,
		PackageCount:   int(req.TotalPackageCount),
		ChangesCount:   changesDetected,
		DurationMs:     int(req.CollectionDurationMs),
		Status:         models.PackageCollectionStatusSuccess,
	}).Error; err != nil {
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
