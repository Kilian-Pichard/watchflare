package client

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"watchflare-agent/metrics"
	pb "watchflare/shared/proto/agent/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// mockAgentServer is a configurable in-process gRPC server for testing
type mockAgentServer struct {
	pb.UnimplementedAgentServiceServer

	registerFn         func(*pb.RegisterHostRequest) (*pb.RegisterHostResponse, error)
	heartbeatFn        func(*pb.HeartbeatRequest) (*pb.HeartbeatResponse, error)
	sendMetricsFn      func(*pb.SendMetricsRequest) (*pb.SendMetricsResponse, error)
	sendPackageInvFn   func(*pb.SendPackageInventoryRequest) (*pb.SendPackageInventoryResponse, error)
}

func (m *mockAgentServer) RegisterHost(_ context.Context, req *pb.RegisterHostRequest) (*pb.RegisterHostResponse, error) {
	if m.registerFn != nil {
		return m.registerFn(req)
	}
	return nil, status.Error(codes.Unimplemented, "not configured")
}

func (m *mockAgentServer) Heartbeat(_ context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	if m.heartbeatFn != nil {
		return m.heartbeatFn(req)
	}
	return nil, status.Error(codes.Unimplemented, "not configured")
}

func (m *mockAgentServer) SendMetrics(_ context.Context, req *pb.SendMetricsRequest) (*pb.SendMetricsResponse, error) {
	if m.sendMetricsFn != nil {
		return m.sendMetricsFn(req)
	}
	return nil, status.Error(codes.Unimplemented, "not configured")
}

func (m *mockAgentServer) SendPackageInventory(_ context.Context, req *pb.SendPackageInventoryRequest) (*pb.SendPackageInventoryResponse, error) {
	if m.sendPackageInvFn != nil {
		return m.sendPackageInvFn(req)
	}
	return nil, status.Error(codes.Unimplemented, "not configured")
}

// startMockServer starts a local gRPC server and returns a Client connected to it.
// The server is stopped automatically when the test ends.
func startMockServer(t *testing.T, srv *mockAgentServer) *Client {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterAgentServiceServer(grpcSrv, srv)

	go func() { _ = grpcSrv.Serve(lis) }()
	t.Cleanup(grpcSrv.Stop)

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create test gRPC client: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return &Client{conn: conn, client: pb.NewAgentServiceClient(conn)}
}

// --- SaveCACertificate ---

func TestSaveCACertificate_CreatesFileAndDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pki", "sub")
	path := filepath.Join(dir, "ca.pem")
	pem := "-----BEGIN CERTIFICATE-----\nfakecert\n-----END CERTIFICATE-----\n"

	if err := SaveCACertificate(pem, path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(data) != pem {
		t.Fatalf("content mismatch: got %q", string(data))
	}
}

func TestSaveCACertificate_InvalidPath(t *testing.T) {
	// Use a file as a directory component — cannot be created
	tmp := t.TempDir()
	blockingFile := filepath.Join(tmp, "notadir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := SaveCACertificate("pem", filepath.Join(blockingFile, "ca.pem"))
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

// --- New() ---

func TestNew_MissingCACertFile(t *testing.T) {
	_, err := New("localhost", "50051", "/nonexistent/ca.pem", "server")
	if err == nil {
		t.Fatal("expected error for missing CA cert file, got nil")
	}
}

func TestNew_InvalidPEM(t *testing.T) {
	tmp := t.TempDir()
	badPEM := filepath.Join(tmp, "bad.pem")
	if err := os.WriteFile(badPEM, []byte("this is not valid PEM"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := New("localhost", "50051", badPEM, "server")
	if err == nil {
		t.Fatal("expected error for invalid PEM, got nil")
	}
}

// --- NewForRegistration() ---

func TestNewForRegistration_Succeeds(t *testing.T) {
	// gRPC connection is lazy — no actual network dial happens here
	c, err := NewForRegistration("localhost", "50051")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c.Close()
}

// --- Register() ---

func TestRegister_Success(t *testing.T) {
	mock := &mockAgentServer{
		registerFn: func(req *pb.RegisterHostRequest) (*pb.RegisterHostResponse, error) {
			if req.RegistrationToken != "tok" {
				return nil, status.Error(codes.InvalidArgument, "bad token")
			}
			return &pb.RegisterHostResponse{
				Success:    true,
				AgentId:    "agent-1",
				AgentKey:   "key-1",
				CaCert:     "pem-data",
				ServerName: "backend.local",
				Reactivated: false,
			}, nil
		},
	}

	c := startMockServer(t, mock)

	resp, err := c.Register(RegisterRequest{
		Token:    "tok",
		Hostname: "host1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.AgentID != "agent-1" {
		t.Errorf("AgentID: got %q, want %q", resp.AgentID, "agent-1")
	}
	if resp.AgentKey != "key-1" {
		t.Errorf("AgentKey: got %q, want %q", resp.AgentKey, "key-1")
	}
	if resp.CACert != "pem-data" {
		t.Errorf("CACert: got %q, want %q", resp.CACert, "pem-data")
	}
	if resp.ServerName != "backend.local" {
		t.Errorf("ServerName: got %q, want %q", resp.ServerName, "backend.local")
	}
}

func TestRegister_Rejected(t *testing.T) {
	mock := &mockAgentServer{
		registerFn: func(_ *pb.RegisterHostRequest) (*pb.RegisterHostResponse, error) {
			return &pb.RegisterHostResponse{
				Success: false,
				Message: "token expired",
			}, nil
		},
	}

	c := startMockServer(t, mock)

	_, err := c.Register(RegisterRequest{Token: "bad"})
	if err == nil {
		t.Fatal("expected error for rejected registration, got nil")
	}
}

func TestRegister_ServerError(t *testing.T) {
	mock := &mockAgentServer{
		registerFn: func(_ *pb.RegisterHostRequest) (*pb.RegisterHostResponse, error) {
			return nil, status.Error(codes.Internal, "internal error")
		},
	}

	c := startMockServer(t, mock)

	_, err := c.Register(RegisterRequest{Token: "tok"})
	if err == nil {
		t.Fatal("expected error for server failure, got nil")
	}
}

func TestRegister_FieldsMapping(t *testing.T) {
	var received *pb.RegisterHostRequest

	mock := &mockAgentServer{
		registerFn: func(req *pb.RegisterHostRequest) (*pb.RegisterHostResponse, error) {
			received = req
			return &pb.RegisterHostResponse{Success: true, AgentId: "x", AgentKey: "y"}, nil
		},
	}

	c := startMockServer(t, mock)

	_, err := c.Register(RegisterRequest{
		Token:            "token123",
		Hostname:         "myhost",
		IPv4:             "1.2.3.4",
		IPv6:             "::1",
		Platform:         "linux",
		PlatformVersion:  "6.1",
		PlatformFamily:   "debian",
		Architecture:     "amd64",
		Kernel:           "6.1.0",
		EnvironmentType:  "vm",
		Hypervisor:       "kvm",
		ContainerRuntime: "docker",
		ExistingUUID:     "old-uuid",
		AgentVersion:     "1.2.3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []struct {
		name string
		got  string
		want string
	}{
		{"RegistrationToken", received.RegistrationToken, "token123"},
		{"Hostname", received.Hostname, "myhost"},
		{"IpAddressV4", received.IpAddressV4, "1.2.3.4"},
		{"IpAddressV6", received.IpAddressV6, "::1"},
		{"Platform", received.Platform, "linux"},
		{"PlatformVersion", received.PlatformVersion, "6.1"},
		{"PlatformFamily", received.PlatformFamily, "debian"},
		{"Architecture", received.Architecture, "amd64"},
		{"Kernel", received.Kernel, "6.1.0"},
		{"EnvironmentType", received.EnvironmentType, "vm"},
		{"Hypervisor", received.Hypervisor, "kvm"},
		{"ContainerRuntime", received.ContainerRuntime, "docker"},
		{"ExistingAgentUuid", received.ExistingAgentUuid, "old-uuid"},
		{"AgentVersion", received.AgentVersion, "1.2.3"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, c.got, c.want)
		}
	}
}

// --- Heartbeat / SendHeartbeat ---

func TestHeartbeat_DelegatesToSendHeartbeat(t *testing.T) {
	var receivedIPv4, receivedIPv6 string

	mock := &mockAgentServer{
		heartbeatFn: func(req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
			receivedIPv4 = req.IpAddressV4
			receivedIPv6 = req.IpAddressV6
			return &pb.HeartbeatResponse{Success: true}, nil
		},
	}

	c := startMockServer(t, mock)

	if err := c.Heartbeat("agent-1", "key-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedIPv4 != "" {
		t.Errorf("Heartbeat should send empty IPv4, got %q", receivedIPv4)
	}
	if receivedIPv6 != "" {
		t.Errorf("Heartbeat should send empty IPv6, got %q", receivedIPv6)
	}
}

func TestSendHeartbeat_Success(t *testing.T) {
	mock := &mockAgentServer{
		heartbeatFn: func(req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
			if req.AgentId != "agent-1" || req.AgentKey != "key-1" {
				return nil, status.Error(codes.InvalidArgument, "bad credentials")
			}
			return &pb.HeartbeatResponse{Success: true}, nil
		},
	}

	c := startMockServer(t, mock)

	if err := c.SendHeartbeat("agent-1", "key-1", "1.2.3.4", "::1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendHeartbeat_Rejected(t *testing.T) {
	mock := &mockAgentServer{
		heartbeatFn: func(_ *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
			return &pb.HeartbeatResponse{Success: false, Message: "agent not found"}, nil
		},
	}

	c := startMockServer(t, mock)

	err := c.SendHeartbeat("agent-1", "key-1", "", "")
	if err == nil {
		t.Fatal("expected error for rejected heartbeat, got nil")
	}
}

// --- SendMetrics ---

func TestSendMetrics_Success(t *testing.T) {
	var received *pb.SendMetricsRequest

	mock := &mockAgentServer{
		sendMetricsFn: func(req *pb.SendMetricsRequest) (*pb.SendMetricsResponse, error) {
			received = req
			return &pb.SendMetricsResponse{Success: true}, nil
		},
	}

	c := startMockServer(t, mock)

	m := &metrics.SystemMetrics{
		CPUUsagePercent:  45.5,
		MemoryTotalBytes: 8 * 1024 * 1024 * 1024,
		MemoryUsedBytes:  4 * 1024 * 1024 * 1024,
		SensorReadings: []metrics.SensorReading{
			{Key: "cpu0", TemperatureCelsius: 55.0},
		},
		ContainerMetrics: []metrics.ContainerMetric{
			{ContainerID: "c1", ContainerName: "app", CPUPercent: 10.0},
		},
	}

	if err := c.SendMetrics("agent-1", "key-1", "1.0.0", m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Metrics.CpuUsagePercent != 45.5 {
		t.Errorf("CpuUsagePercent: got %v, want 45.5", received.Metrics.CpuUsagePercent)
	}
	if len(received.Metrics.SensorReadings) != 1 {
		t.Errorf("SensorReadings: got %d, want 1", len(received.Metrics.SensorReadings))
	} else if received.Metrics.SensorReadings[0].Key != "cpu0" {
		t.Errorf("SensorReadings[0].Key: got %q, want %q", received.Metrics.SensorReadings[0].Key, "cpu0")
	}
	if len(received.ContainerMetrics) != 1 {
		t.Errorf("ContainerMetrics: got %d, want 1", len(received.ContainerMetrics))
	}
	if received.AgentVersion != "1.0.0" {
		t.Errorf("AgentVersion: got %q, want %q", received.AgentVersion, "1.0.0")
	}
}

func TestSendMetrics_Rejected(t *testing.T) {
	mock := &mockAgentServer{
		sendMetricsFn: func(_ *pb.SendMetricsRequest) (*pb.SendMetricsResponse, error) {
			return &pb.SendMetricsResponse{Success: false, Message: "invalid agent"}, nil
		},
	}

	c := startMockServer(t, mock)

	err := c.SendMetrics("agent-1", "key-1", "1.0.0", &metrics.SystemMetrics{})
	if err == nil {
		t.Fatal("expected error for rejected metrics, got nil")
	}
}

func TestSendMetrics_NoContainerMetrics(t *testing.T) {
	var received *pb.SendMetricsRequest

	mock := &mockAgentServer{
		sendMetricsFn: func(req *pb.SendMetricsRequest) (*pb.SendMetricsResponse, error) {
			received = req
			return &pb.SendMetricsResponse{Success: true}, nil
		},
	}

	c := startMockServer(t, mock)

	if err := c.SendMetrics("agent-1", "key-1", "1.0.0", &metrics.SystemMetrics{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(received.ContainerMetrics) != 0 {
		t.Errorf("expected no container metrics, got %d", len(received.ContainerMetrics))
	}
}

// --- SendPackageInventory ---

func TestSendPackageInventory_Success(t *testing.T) {
	var received *pb.SendPackageInventoryRequest

	mock := &mockAgentServer{
		sendPackageInvFn: func(req *pb.SendPackageInventoryRequest) (*pb.SendPackageInventoryResponse, error) {
			received = req
			return &pb.SendPackageInventoryResponse{Success: true}, nil
		},
	}

	c := startMockServer(t, mock)

	data := &PackageInventoryData{
		InventoryType:        "full",
		TotalPackageCount:    3,
		CollectionDurationMs: 120,
		AllPackages: []*pb.Package{
			{Name: "curl", Version: "7.0", PackageManager: "apt"},
		},
	}

	if err := c.SendPackageInventory("agent-1", "key-1", data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.InventoryType != "full" {
		t.Errorf("InventoryType: got %q, want %q", received.InventoryType, "full")
	}
	if received.TotalPackageCount != 3 {
		t.Errorf("TotalPackageCount: got %d, want 3", received.TotalPackageCount)
	}
	if len(received.AllPackages) != 1 || received.AllPackages[0].Name != "curl" {
		t.Errorf("AllPackages: unexpected value")
	}
}

func TestSendPackageInventory_Rejected(t *testing.T) {
	mock := &mockAgentServer{
		sendPackageInvFn: func(_ *pb.SendPackageInventoryRequest) (*pb.SendPackageInventoryResponse, error) {
			return &pb.SendPackageInventoryResponse{Success: false, Message: "quota exceeded"}, nil
		},
	}

	c := startMockServer(t, mock)

	err := c.SendPackageInventory("agent-1", "key-1", &PackageInventoryData{})
	if err == nil {
		t.Fatal("expected error for rejected package inventory, got nil")
	}
}
