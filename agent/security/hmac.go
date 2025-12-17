package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// ComputeHMAC calculates HMAC-SHA256 for a message
// Format: HMAC-SHA256(agent_key, timestamp + agent_id + marshaled_message)
func ComputeHMAC(agentKey string, timestamp int64, agentID string, message proto.Message) (string, error) {
	// Marshal the protobuf message
	messageBytes, err := proto.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	// Build the data to sign: timestamp + agent_id + message
	data := fmt.Sprintf("%d%s", timestamp, agentID)
	data += string(messageBytes)

	// Compute HMAC-SHA256
	h := hmac.New(sha256.New, []byte(agentKey))
	h.Write([]byte(data))
	signature := h.Sum(nil)

	return hex.EncodeToString(signature), nil
}

// AttachAuthMetadata adds HMAC and timestamp to gRPC context metadata
// This should be called before each gRPC request
func AttachAuthMetadata(ctx context.Context, agentID, agentKey string, timestamp int64, message proto.Message) (context.Context, error) {
	// Compute HMAC
	hmacSignature, err := ComputeHMAC(agentKey, timestamp, agentID, message)
	if err != nil {
		return nil, fmt.Errorf("failed to compute HMAC: %w", err)
	}

	// Create metadata with HMAC and timestamp
	md := metadata.Pairs(
		"x-watchflare-hmac", hmacSignature,
		"x-watchflare-timestamp", fmt.Sprintf("%d", timestamp),
	)

	// Attach metadata to context
	return metadata.NewOutgoingContext(ctx, md), nil
}
