package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"watchflare/backend/database"
	"watchflare/backend/models"
	pb "watchflare/shared/proto/agent/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// noopHandler is a gRPC handler that does nothing and returns success.
var noopHandler grpc.UnaryHandler = func(ctx context.Context, req interface{}) (interface{}, error) {
	return "ok", nil
}

func unaryInfo(method string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: method}
}

func grpcCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	return status.Code(err)
}

// TestInterceptor_SkipsRegisterServer verifies that RegisterServer bypasses HMAC auth.
func TestInterceptor_SkipsRegisterServer(t *testing.T) {
	interceptor := AuthInterceptor(300)

	req := &pb.RegisterServerRequest{RegistrationToken: "wf_reg_test"}
	info := unaryInfo("/agent.v1.AgentService/RegisterServer")

	_, err := interceptor(context.Background(), req, info, noopHandler)
	assert.NoError(t, err)
}

// TestInterceptor_MissingHMAC verifies that requests without HMAC metadata are rejected.
func TestInterceptor_MissingHMAC(t *testing.T) {
	interceptor := AuthInterceptor(300)

	req := &pb.HeartbeatRequest{AgentId: "some-id", Timestamp: time.Now().Unix()}
	info := unaryInfo("/agent.v1.AgentService/Heartbeat")

	// No metadata in context.
	_, err := interceptor(context.Background(), req, info, noopHandler)
	assert.Equal(t, codes.Unauthenticated, grpcCode(err))
}

// TestInterceptor_MissingHMAC_WithEmptyMetadata verifies rejection when metadata exists but HMAC header is absent.
func TestInterceptor_MissingHMAC_WithEmptyMetadata(t *testing.T) {
	interceptor := AuthInterceptor(300)

	req := &pb.HeartbeatRequest{AgentId: "some-id", Timestamp: time.Now().Unix()}
	info := unaryInfo("/agent.v1.AgentService/Heartbeat")

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-other-header", "value"))
	_, err := interceptor(ctx, req, info, noopHandler)
	assert.Equal(t, codes.Unauthenticated, grpcCode(err))
}

// TestInterceptor_AgentNotFound verifies rejection when the agent_id is not in the database.
func TestInterceptor_AgentNotFound(t *testing.T) {
	setupGRPCTestDB(t)

	interceptor := AuthInterceptor(300)

	req := &pb.HeartbeatRequest{
		AgentId:   "00000000-0000-0000-0000-000000000000",
		AgentKey:  "irrelevant",
		Timestamp: time.Now().Unix(),
	}
	info := unaryInfo("/agent.v1.AgentService/Heartbeat")

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-watchflare-hmac", "dummy"))
	_, err := interceptor(ctx, req, info, noopHandler)
	assert.Equal(t, codes.Unauthenticated, grpcCode(err))
}

// TestInterceptor_TimestampTooOld verifies rejection when the timestamp is outside the window.
func TestInterceptor_TimestampTooOld(t *testing.T) {
	setupGRPCTestDB(t)

	server := &models.Server{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		Name:     "intercept-ts-server",
		Status:   models.StatusOnline,
		AgentKey: "intercept-ts-key-abc",
	}
	require.NoError(t, database.DB.Create(server).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(server) })

	interceptor := AuthInterceptor(300)

	staleTS := time.Now().Add(-10 * time.Minute).Unix()
	req := &pb.HeartbeatRequest{
		AgentId:   server.AgentID,
		Timestamp: staleTS,
	}
	info := unaryInfo("/agent.v1.AgentService/Heartbeat")

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-watchflare-hmac", "dummy"))
	_, err := interceptor(ctx, req, info, noopHandler)
	assert.Equal(t, codes.InvalidArgument, grpcCode(err))
}

// TestInterceptor_InvalidHMAC verifies rejection when HMAC signature is wrong.
func TestInterceptor_InvalidHMAC(t *testing.T) {
	setupGRPCTestDB(t)

	server := &models.Server{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		Name:     "intercept-hmac-server",
		Status:   models.StatusOnline,
		AgentKey: "intercept-hmac-key-abc",
	}
	require.NoError(t, database.DB.Create(server).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(server) })

	interceptor := AuthInterceptor(300)

	ts := time.Now().Unix()
	req := &pb.HeartbeatRequest{
		AgentId:   server.AgentID,
		Timestamp: ts,
	}
	info := unaryInfo("/agent.v1.AgentService/Heartbeat")

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-watchflare-hmac", "badhmacsignature",
		"x-watchflare-timestamp", fmt.Sprintf("%d", ts),
	))
	_, err := interceptor(ctx, req, info, noopHandler)
	assert.Equal(t, codes.Unauthenticated, grpcCode(err))
}

// TestInterceptor_ValidHMAC verifies that a correctly signed request reaches the handler.
func TestInterceptor_ValidHMAC(t *testing.T) {
	setupGRPCTestDB(t)

	server := &models.Server{
		ID:       uuid.New().String(),
		AgentID:  uuid.New().String(),
		Name:     "intercept-valid-server",
		Status:   models.StatusOnline,
		AgentKey: "intercept-valid-key-abc",
	}
	require.NoError(t, database.DB.Create(server).Error)
	t.Cleanup(func() { database.DB.Unscoped().Delete(server) })

	interceptor := AuthInterceptor(300)

	ts := time.Now().Unix()
	req := &pb.HeartbeatRequest{
		AgentId:   server.AgentID,
		Timestamp: ts,
	}

	sig, err := computeHMAC(server.AgentKey, ts, server.AgentID, req)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-watchflare-hmac", sig,
		"x-watchflare-timestamp", fmt.Sprintf("%d", ts),
	))
	info := unaryInfo("/agent.v1.AgentService/Heartbeat")

	result, err := interceptor(ctx, req, info, noopHandler)
	assert.NoError(t, err)
	assert.Equal(t, "ok", result)
}
