package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// ComputeHMAC calculates HMAC-SHA256 for a message.
// Payload format: [8-byte timestamp big-endian] | [agentID] | [marshaled proto]
// Delimiters prevent length-extension collisions. agentID must be a UUID
// (no "|" characters) for the delimiter scheme to hold.
func ComputeHMAC(agentKey string, timestamp int64, agentID string, message proto.Message) (string, error) {
	if agentKey == "" {
		return "", fmt.Errorf("agent key must not be empty")
	}

	messageBytes, err := proto.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	h := hmac.New(sha256.New, []byte(agentKey))

	var timestampBytes [8]byte
	binary.BigEndian.PutUint64(timestampBytes[:], uint64(timestamp))
	h.Write(timestampBytes[:])
	h.Write([]byte("|"))
	h.Write([]byte(agentID))
	h.Write([]byte("|"))
	h.Write(messageBytes)

	return hex.EncodeToString(h.Sum(nil)), nil
}

// AttachAuthMetadata adds HMAC and timestamp to gRPC context metadata.
// Must be called before each authenticated gRPC request.
func AttachAuthMetadata(ctx context.Context, agentID, agentKey string, timestamp int64, message proto.Message) (context.Context, error) {
	hmacSignature, err := ComputeHMAC(agentKey, timestamp, agentID, message)
	if err != nil {
		return nil, fmt.Errorf("failed to compute HMAC: %w", err)
	}

	md := metadata.Pairs(
		"x-watchflare-hmac", hmacSignature,
		"x-watchflare-timestamp", fmt.Sprintf("%d", timestamp),
	)

	return metadata.NewOutgoingContext(ctx, md), nil
}
