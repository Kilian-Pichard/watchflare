package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"watchflare/backend/database"
	"watchflare/backend/models"
	pb "watchflare/backend/proto"
	"watchflare/backend/sse"

	"gorm.io/gorm"
)

// AgentServer implements the AgentService gRPC server
type AgentServer struct {
	pb.UnimplementedAgentServiceServer
}

// NewAgentServer creates a new AgentServer instance
func NewAgentServer() *AgentServer {
	return &AgentServer{}
}

// RegisterServer handles initial agent registration
func (s *AgentServer) RegisterServer(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Hash the provided token to compare with stored hash
	hashedToken := hashToken(req.RegistrationToken)

	// Find server with matching registration token
	var server models.Server
	result := database.DB.Where("registration_token = ?", hashedToken).First(&server)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &pb.RegisterResponse{
				Success: false,
				Message: "Invalid registration token",
			}, nil
		}
		return nil, result.Error
	}

	// Check if token has expired
	if server.ExpiresAt != nil && time.Now().After(*server.ExpiresAt) {
		return &pb.RegisterResponse{
			Success: false,
			Message: "Registration token has expired",
		}, nil
	}

	// Check if server is in pending status
	if server.Status != "pending" && server.Status != "expired" {
		return &pb.RegisterResponse{
			Success: false,
			Message: "Server is already registered",
		}, nil
	}

	// Validate IP address if allow_any_ip_registration is false
	if !server.AllowAnyIPRegistration {
		if server.ConfiguredIP != nil && *server.ConfiguredIP != "" {
			// Check if the actual IP matches the configured IP
			if req.IpAddressV4 != *server.ConfiguredIP {
				return &pb.RegisterResponse{
					Success: false,
					Message: "IP address mismatch. Expected: " + *server.ConfiguredIP + ", Got: " + req.IpAddressV4,
				}, nil
			}
		}
	}

	// Update server with agent information
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
		"status":             "online",
		"last_seen":          &now,
		"registration_token": nil, // Clear the token after successful registration
		"expires_at":         nil, // Clear expiration
	}

	if err := database.DB.Model(&server).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Broadcast SSE event for server registration
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
		LastSeen:         now.Format(time.RFC3339),
	})

	return &pb.RegisterResponse{
		Success:  true,
		Message:  "Server registered successfully",
		AgentId:  server.AgentID,
		AgentKey: server.AgentKey,
	}, nil
}

// Heartbeat handles periodic heartbeats from agents
func (s *AgentServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	// Find server by agent ID and verify agent key
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

	// Update last seen and status
	now := time.Now()
	updates := map[string]interface{}{
		"last_seen":     &now,
		"status":        "online",
		"ip_address_v4": req.IpAddressV4,
		"ip_address_v6": req.IpAddressV6,
	}

	if err := database.DB.Model(&server).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Broadcast SSE event for heartbeat
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
		LastSeen:         now.Format(time.RFC3339),
	})

	return &pb.HeartbeatResponse{
		Success: true,
		Message: "Heartbeat acknowledged",
	}, nil
}

// hashToken creates a SHA-256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
