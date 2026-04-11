package grpc

import (
	"context"
	"log/slog"

	"watchflare/backend/cache"
	"watchflare/backend/database"
	"watchflare/backend/models"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// AuthInterceptor creates a gRPC unary interceptor for authentication and validation
// HMAC authentication is mandatory for all requests (except RegisterHost)
func AuthInterceptor(timestampWindow int) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip authentication for RegisterHost (uses registration token instead)
		if info.FullMethod == "/agent.v1.AgentService/RegisterHost" {
			return handler(ctx, req)
		}

		// Check if HMAC is present in metadata
		md, ok := metadata.FromIncomingContext(ctx)
		hasHMAC := ok && len(md.Get("x-watchflare-hmac")) > 0

		// HMAC is mandatory
		if !hasHMAC {
			slog.Warn("HMAC authentication missing", "method", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, "HMAC authentication required")
		}

		// HMAC is present, validate it
		// 1. Extract agent_id from message
		agentID, err := extractAgentID(req)
		if err != nil {
			slog.Warn("failed to extract agent_id from request", "error", err)
			return nil, status.Error(codes.InvalidArgument, "Invalid request format")
		}

		// 2. Lookup agent_key from database
		var host models.Host
		if err := database.DB.Where("agent_id = ?", agentID).First(&host).Error; err != nil {
			slog.Warn("agent not found", "agent_id", agentID)
			return nil, status.Error(codes.Unauthenticated, "Invalid agent credentials")
		}

		// 3. Extract and validate timestamp
		message, ok := req.(proto.Message)
		if !ok {
			slog.Error("request is not a proto.Message", "method", info.FullMethod)
			return nil, status.Error(codes.Internal, "Internal error")
		}

		timestamp, err := extractTimestamp(message)
		if err != nil {
			slog.Warn("failed to extract timestamp", "error", err)
			return nil, status.Error(codes.InvalidArgument, "Invalid request format: missing timestamp")
		}

		if err := ValidateTimestamp(timestamp, timestampWindow); err != nil {
			slog.Warn("timestamp validation failed", "agent_id", agentID, "error", err)
			// Track clock desync for heartbeat RPCs so frontend can show a banner
			if info.FullMethod == "/agent.v1.AgentService/Heartbeat" || info.FullMethod == "/agent.v1.AgentService/SendMetrics" {
				cache.GetCache().SetClockDesync(agentID)
			}
			return nil, status.Error(codes.InvalidArgument, "Timestamp outside acceptable window")
		}

		// 4. Validate HMAC
		if err := ValidateHMAC(ctx, agentID, host.AgentKey, message); err != nil {
			slog.Warn("HMAC validation failed", "agent_id", agentID, "error", err)
			return nil, status.Error(codes.Unauthenticated, "HMAC validation failed")
		}

		// Authentication successful, call the handler
		return handler(ctx, req)
	}
}
