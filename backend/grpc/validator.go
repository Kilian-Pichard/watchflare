package grpc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

// ValidateTimestamp verifies that the timestamp is within the acceptable window
func ValidateTimestamp(timestamp int64, windowSeconds int) error {
	now := time.Now().Unix()
	minTime := now - int64(windowSeconds)
	maxTime := now + int64(windowSeconds)

	if timestamp < minTime {
		return fmt.Errorf("timestamp too old: %d is before %d (window: %ds)", timestamp, minTime, windowSeconds)
	}

	if timestamp > maxTime {
		return fmt.Errorf("timestamp too far in future: %d is after %d (window: %ds)", timestamp, maxTime, windowSeconds)
	}

	return nil
}

// ValidateHMAC verifies the HMAC signature from gRPC metadata
func ValidateHMAC(ctx context.Context, agentID, agentKey string, message proto.Message) error {
	// Extract HMAC from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errors.New("no metadata in context")
	}

	hmacValues := md.Get("x-watchflare-hmac")
	if len(hmacValues) == 0 {
		return errors.New("no HMAC in metadata")
	}
	receivedHMAC := hmacValues[0]

	timestampValues := md.Get("x-watchflare-timestamp")
	if len(timestampValues) == 0 {
		return errors.New("no timestamp in metadata")
	}
	timestampStr := timestampValues[0]

	// Extract timestamp from message using reflection
	messageTimestamp, err := extractTimestamp(message)
	if err != nil {
		return fmt.Errorf("failed to extract timestamp from message: %w", err)
	}

	// Verify metadata timestamp matches message timestamp
	if timestampStr != fmt.Sprintf("%d", messageTimestamp) {
		return errors.New("timestamp mismatch between metadata and message")
	}

	// Compute expected HMAC
	expectedHMAC, err := computeHMAC(agentKey, messageTimestamp, agentID, message)
	if err != nil {
		return fmt.Errorf("failed to compute HMAC: %w", err)
	}

	// Compare HMACs using constant-time comparison
	if !hmac.Equal([]byte(receivedHMAC), []byte(expectedHMAC)) {
		return errors.New("HMAC validation failed")
	}

	return nil
}

// computeHMAC calculates HMAC-SHA256 for a message
// Format: HMAC-SHA256(agent_key, timestamp (binary 8 bytes) + "|" + agentID + "|" + message)
func computeHMAC(agentKey string, timestamp int64, agentID string, message proto.Message) (string, error) {
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

// extractAgentID extracts the agent_id field from a message using reflection
func extractAgentID(message interface{}) (string, error) {
	v := reflect.ValueOf(message)

	// If it's a pointer, get the underlying value
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Check if it's a struct
	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("message is not a struct: %v", v.Kind())
	}

	// Look for AgentId field
	agentIDField := v.FieldByName("AgentId")
	if !agentIDField.IsValid() {
		return "", errors.New("message does not have AgentId field")
	}

	// Check if it's a string
	if agentIDField.Kind() != reflect.String {
		return "", fmt.Errorf("AgentId field is not a string: %v", agentIDField.Kind())
	}

	return agentIDField.String(), nil
}

// extractTimestamp extracts the timestamp field from a message using reflection
func extractTimestamp(message interface{}) (int64, error) {
	v := reflect.ValueOf(message)

	// If it's a pointer, get the underlying value
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Check if it's a struct
	if v.Kind() != reflect.Struct {
		return 0, fmt.Errorf("message is not a struct: %v", v.Kind())
	}

	// Look for Timestamp field
	timestampField := v.FieldByName("Timestamp")
	if !timestampField.IsValid() {
		return 0, errors.New("message does not have Timestamp field")
	}

	// Check if it's an int64
	if timestampField.Kind() != reflect.Int64 {
		return 0, fmt.Errorf("Timestamp field is not int64: %v", timestampField.Kind())
	}

	return timestampField.Int(), nil
}
