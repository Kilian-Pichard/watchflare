package security

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"

	pb "watchflare/shared/proto/agent/v1"
)

// knownMessage returns a simple deterministic proto message for tests.
func knownMessage() *pb.HeartbeatRequest {
	return &pb.HeartbeatRequest{
		AgentId:  "test-agent",
		AgentKey: "test-key",
	}
}

// --- ComputeHMAC ---

func TestComputeHMAC_Deterministic(t *testing.T) {
	msg := knownMessage()
	h1, err := ComputeHMAC("secret", 1700000000, "agent-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := ComputeHMAC("secret", 1700000000, "agent-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 != h2 {
		t.Errorf("ComputeHMAC is not deterministic: %q != %q", h1, h2)
	}
}

func TestComputeHMAC_IsHex(t *testing.T) {
	h, err := ComputeHMAC("secret", 1700000000, "agent-1", knownMessage())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(h) != 64 {
		t.Errorf("expected 64-char hex string (SHA-256), got len=%d: %q", len(h), h)
	}
	for _, c := range h {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("non-hex character %q in HMAC output %q", c, h)
		}
	}
}

func TestComputeHMAC_DifferentKey(t *testing.T) {
	msg := knownMessage()
	h1, err := ComputeHMAC("key-a", 1700000000, "agent-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := ComputeHMAC("key-b", 1700000000, "agent-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 == h2 {
		t.Error("different keys must produce different HMACs")
	}
}

func TestComputeHMAC_DifferentTimestamp(t *testing.T) {
	msg := knownMessage()
	h1, err := ComputeHMAC("secret", 1700000000, "agent-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := ComputeHMAC("secret", 1700000001, "agent-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 == h2 {
		t.Error("different timestamps must produce different HMACs")
	}
}

func TestComputeHMAC_DifferentAgentID(t *testing.T) {
	msg := knownMessage()
	h1, err := ComputeHMAC("secret", 1700000000, "agent-1", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := ComputeHMAC("secret", 1700000000, "agent-2", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 == h2 {
		t.Error("different agent IDs must produce different HMACs")
	}
}

func TestComputeHMAC_EmptyKey(t *testing.T) {
	_, err := ComputeHMAC("", 1700000000, "agent-1", knownMessage())
	if err == nil {
		t.Error("expected error for empty agent key")
	}
}

// --- AttachAuthMetadata ---

func TestAttachAuthMetadata_ContainsExpectedKeys(t *testing.T) {
	ctx := context.Background()
	newCtx, err := AttachAuthMetadata(ctx, "agent-1", "secret", 1700000000, knownMessage())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md, ok := metadata.FromOutgoingContext(newCtx)
	if !ok {
		t.Fatal("no outgoing metadata found in context")
	}

	if vals := md.Get("x-watchflare-hmac"); len(vals) == 0 || vals[0] == "" {
		t.Error("x-watchflare-hmac metadata key missing or empty")
	}
	if vals := md.Get("x-watchflare-timestamp"); len(vals) == 0 || vals[0] == "" {
		t.Error("x-watchflare-timestamp metadata key missing or empty")
	}
}

func TestAttachAuthMetadata_TimestampMatchesInput(t *testing.T) {
	ctx := context.Background()
	newCtx, err := AttachAuthMetadata(ctx, "agent-1", "secret", 1700000000, knownMessage())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md, _ := metadata.FromOutgoingContext(newCtx)
	vals := md.Get("x-watchflare-timestamp")
	if len(vals) == 0 || vals[0] != "1700000000" {
		t.Errorf("x-watchflare-timestamp: got %v, want %q", vals, "1700000000")
	}
}

func TestAttachAuthMetadata_EmptyKey(t *testing.T) {
	_, err := AttachAuthMetadata(context.Background(), "agent-1", "", 1700000000, knownMessage())
	if err == nil {
		t.Error("expected error for empty agent key")
	}
}
