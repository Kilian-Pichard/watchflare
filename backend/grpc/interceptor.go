package grpc

import (
	"context"
	"log"

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
// HMAC authentication is mandatory for all requests (except RegisterServer)
func AuthInterceptor(timestampWindow int) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip authentication for RegisterServer (uses registration token instead)
		if info.FullMethod == "/agent.v1.AgentService/RegisterServer" {
			return handler(ctx, req)
		}

		// Check if HMAC is present in metadata
		md, ok := metadata.FromIncomingContext(ctx)
		hasHMAC := ok && len(md.Get("x-watchflare-hmac")) > 0

		// HMAC is mandatory
		if !hasHMAC {
			log.Printf("HMAC authentication required but not present in request to %s", info.FullMethod)
			return nil, status.Error(codes.Unauthenticated, "HMAC authentication required")
		}

		// HMAC is present, validate it
		// 1. Extract agent_id from message
		agentID, err := extractAgentID(req)
		if err != nil {
			log.Printf("Failed to extract agent_id from request: %v", err)
			return nil, status.Error(codes.InvalidArgument, "Invalid request format")
		}

		// 2. Lookup agent_key from database
		var server models.Server
		if err := database.DB.Where("agent_id = ?", agentID).First(&server).Error; err != nil {
			log.Printf("Agent not found: %s", agentID)
			return nil, status.Error(codes.Unauthenticated, "Invalid agent credentials")
		}

		// 3. Extract and validate timestamp
		message, ok := req.(proto.Message)
		if !ok {
			log.Printf("Request is not a proto.Message")
			return nil, status.Error(codes.Internal, "Internal error")
		}

		timestamp, err := extractTimestamp(message)
		if err != nil {
			log.Printf("Failed to extract timestamp: %v", err)
			return nil, status.Error(codes.InvalidArgument, "Invalid request format: missing timestamp")
		}

		if err := ValidateTimestamp(timestamp, timestampWindow); err != nil {
			log.Printf("Timestamp validation failed for agent %s: %v", agentID, err)
			// Track clock desync for heartbeat RPCs so frontend can show a banner
			if info.FullMethod == "/agent.v1.AgentService/Heartbeat" || info.FullMethod == "/agent.v1.AgentService/SendMetrics" {
				cache.GetCache().SetClockDesync(agentID)
			}
			return nil, status.Error(codes.InvalidArgument, "Timestamp outside acceptable window")
		}

		// 4. Validate HMAC
		if err := ValidateHMAC(ctx, agentID, server.AgentKey, message); err != nil {
			log.Printf("HMAC validation failed for agent %s: %v", agentID, err)
			return nil, status.Error(codes.Unauthenticated, "HMAC validation failed")
		}

		// Authentication successful, call the handler
		return handler(ctx, req)
	}
}
