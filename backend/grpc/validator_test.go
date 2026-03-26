package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	pb "watchflare/shared/proto/agent/v1"

	"google.golang.org/grpc/metadata"
)

// --- ValidateTimestamp ---

func TestValidateTimestamp_Valid(t *testing.T) {
	if err := ValidateTimestamp(time.Now().Unix(), 300); err != nil {
		t.Errorf("expected no error for current timestamp, got %v", err)
	}
}

func TestValidateTimestamp_TooOld(t *testing.T) {
	old := time.Now().Add(-10 * time.Minute).Unix()
	if err := ValidateTimestamp(old, 300); err == nil {
		t.Error("expected error for too-old timestamp")
	}
}

func TestValidateTimestamp_TooFarFuture(t *testing.T) {
	future := time.Now().Add(10 * time.Minute).Unix()
	if err := ValidateTimestamp(future, 300); err == nil {
		t.Error("expected error for far-future timestamp")
	}
}

func TestValidateTimestamp_EdgeWithinWindow(t *testing.T) {
	// Exactly at the boundary (within 1 second margin)
	ts := time.Now().Add(-4 * time.Minute).Unix()
	if err := ValidateTimestamp(ts, 300); err != nil {
		t.Errorf("expected no error at boundary, got %v", err)
	}
}

// --- extractAgentID ---

func TestExtractAgentID_Valid(t *testing.T) {
	msg := &pb.HeartbeatRequest{AgentId: "test-agent-id"}
	id, err := extractAgentID(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "test-agent-id" {
		t.Errorf("expected 'test-agent-id', got %q", id)
	}
}

func TestExtractAgentID_NotAStruct(t *testing.T) {
	_, err := extractAgentID("not a struct")
	if err == nil {
		t.Error("expected error for non-struct input")
	}
}

// --- extractTimestamp ---

func TestExtractTimestamp_Valid(t *testing.T) {
	now := time.Now().Unix()
	msg := &pb.HeartbeatRequest{Timestamp: now}
	ts, err := extractTimestamp(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts != now {
		t.Errorf("expected %d, got %d", now, ts)
	}
}

func TestExtractTimestamp_NotAStruct(t *testing.T) {
	_, err := extractTimestamp(42)
	if err == nil {
		t.Error("expected error for non-struct input")
	}
}

// --- ValidateHMAC ---

func TestValidateHMAC_ValidSignature(t *testing.T) {
	agentID := "agent-123"
	agentKey := "supersecretkey"
	now := time.Now().Unix()

	msg := &pb.HeartbeatRequest{
		AgentId:   agentID,
		Timestamp: now,
	}

	// Compute expected HMAC
	expectedHMAC, err := computeHMAC(agentKey, now, agentID, msg)
	if err != nil {
		t.Fatalf("computeHMAC failed: %v", err)
	}

	// Build context with metadata
	md := metadata.Pairs(
		"x-watchflare-hmac", expectedHMAC,
		"x-watchflare-timestamp", formatTimestamp(now),
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	if err := ValidateHMAC(ctx, agentID, agentKey, msg); err != nil {
		t.Errorf("expected valid HMAC, got error: %v", err)
	}
}

func TestValidateHMAC_WrongKey(t *testing.T) {
	agentID := "agent-123"
	now := time.Now().Unix()

	msg := &pb.HeartbeatRequest{AgentId: agentID, Timestamp: now}

	hmacVal, _ := computeHMAC("correct-key", now, agentID, msg)
	md := metadata.Pairs(
		"x-watchflare-hmac", hmacVal,
		"x-watchflare-timestamp", formatTimestamp(now),
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// Validate with wrong key
	if err := ValidateHMAC(ctx, agentID, "wrong-key", msg); err == nil {
		t.Error("expected HMAC validation to fail with wrong key")
	}
}

func TestValidateHMAC_MissingMetadata(t *testing.T) {
	msg := &pb.HeartbeatRequest{AgentId: "a", Timestamp: time.Now().Unix()}
	if err := ValidateHMAC(context.Background(), "a", "key", msg); err == nil {
		t.Error("expected error with no metadata in context")
	}
}

func TestValidateHMAC_TimestampMismatch(t *testing.T) {
	agentID := "agent-123"
	agentKey := "key"
	now := time.Now().Unix()

	msg := &pb.HeartbeatRequest{AgentId: agentID, Timestamp: now}
	hmacVal, _ := computeHMAC(agentKey, now, agentID, msg)

	// Use a different timestamp in metadata
	md := metadata.Pairs(
		"x-watchflare-hmac", hmacVal,
		"x-watchflare-timestamp", formatTimestamp(now+1),
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	if err := ValidateHMAC(ctx, agentID, agentKey, msg); err == nil {
		t.Error("expected error for timestamp mismatch")
	}
}

// formatTimestamp mirrors the format used by the agent.
func formatTimestamp(ts int64) string {
	return fmt.Sprintf("%d", ts)
}
