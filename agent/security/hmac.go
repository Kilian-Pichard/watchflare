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

	// Build the data to sign with proper delimiters to prevent collision attacks
	// Format: timestamp (8 bytes binary) + "|" + agentID + "|" + message
	h := hmac.New(sha256.New, []byte(agentKey))

	// Write timestamp as binary (8 bytes, big-endian)
	timestampBytes := make([]byte, 8)
	timestampBytes[0] = byte(timestamp >> 56)
	timestampBytes[1] = byte(timestamp >> 48)
	timestampBytes[2] = byte(timestamp >> 40)
	timestampBytes[3] = byte(timestamp >> 32)
	timestampBytes[4] = byte(timestamp >> 24)
	timestampBytes[5] = byte(timestamp >> 16)
	timestampBytes[6] = byte(timestamp >> 8)
	timestampBytes[7] = byte(timestamp)
	h.Write(timestampBytes)

	// Write delimiter
	h.Write([]byte("|"))

	// Write agentID
	h.Write([]byte(agentID))

	// Write delimiter
	h.Write([]byte("|"))

	// Write message
	h.Write(messageBytes)

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
