package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"watchflare-agent/metrics"
	"watchflare-agent/security"
	pb "watchflare/shared/proto/agent/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client handles gRPC communication with the backend
type Client struct {
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
	host   string
	port   string
}

// New creates a new gRPC client with strict TLS verification
// Requires a valid CA certificate file for TLS verification
func New(host, port, caCertFile, serverName string) (*Client, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	var opts []grpc.DialOption

	// Load CA certificate (mandatory for TLS)
	caCert, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	// Create cert pool and add CA cert
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	// Create TLS config with strict verification
	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		ServerName: serverName, // For SNI and certificate verification
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}

	// Create TLS credentials
	creds := credentials.NewTLS(tlsConfig)
	opts = append(opts, grpc.WithTransportCredentials(creds))

	// Connect with timeout to avoid blocking indefinitely
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, addr, append(opts, grpc.WithBlock())...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewAgentServiceClient(conn),
		host:   host,
		port:   port,
	}, nil
}

// NewForRegistration creates a gRPC client for initial registration with permissive TLS
// This allows the agent to connect without prior knowledge of the CA certificate
// The CA cert will be received during registration and used for strict verification afterward
func NewForRegistration(host, port string) (*Client, error) {
	addr := fmt.Sprintf("%s:%s", host, port)

	// Use permissive TLS for bootstrap (accepts any certificate)
	// This is safe because registration requires a secret token as root of trust
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Only for initial registration
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
	}

	creds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	// Connect with timeout to avoid blocking indefinitely
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dialCancel()

	conn, err := grpc.DialContext(dialCtx, addr, append(opts, grpc.WithBlock())...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewAgentServiceClient(conn),
		host:   host,
		port:   port,
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// RegistrationResponse contains the result of a successful registration
type RegistrationResponse struct {
	AgentID     string
	AgentKey    string
	CACert      string // CA certificate in PEM format
	ServerName  string // Server name for TLS verification
	Reactivated bool   // True if existing agent was reactivated (UUID reused)
}

// Register attempts to register the agent with the backend
// Returns registration credentials and TLS information
func (c *Client) Register(token, hostname, ipv4, ipv6, platform, platformVersion, platformFamily, architecture, kernel, environmentType, hypervisor, containerRuntime, existingUUID, agentVersion string) (*RegistrationResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.RegisterServerRequest{
		RegistrationToken:   token,
		Hostname:            hostname,
		IpAddressV4:         ipv4,
		IpAddressV6:         ipv6,
		Platform:            platform,
		PlatformVersion:     platformVersion,
		PlatformFamily:      platformFamily,
		Architecture:        architecture,
		Kernel:              kernel,
		Timestamp:           time.Now().Unix(), // Add timestamp for anti-replay
		EnvironmentType:     environmentType,
		Hypervisor:          hypervisor,
		ContainerRuntime:    containerRuntime,
		ExistingAgentUuid:   existingUUID, // For re-registration
		AgentVersion:        agentVersion,
	}

	// Note: Registration uses token-based auth, not HMAC
	// HMAC is only used after successful registration

	resp, err := c.client.RegisterServer(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("registration failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("registration rejected: %s", resp.Message)
	}

	return &RegistrationResponse{
		AgentID:     resp.AgentId,
		AgentKey:    resp.AgentKey,
		CACert:      resp.CaCert,
		ServerName:  resp.ServerName,
		Reactivated: resp.Reactivated,
	}, nil
}

// SaveCACertificate saves the CA certificate to disk
// The directory will be created if it doesn't exist
func SaveCACertificate(caCertPEM, certPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(certPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create PKI directory: %w", err)
	}

	// Write CA certificate with restricted permissions
	if err := os.WriteFile(certPath, []byte(caCertPEM), 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	return nil
}

// Heartbeat sends a simple heartbeat (wrapper for SendHeartbeat with empty IPs)
func (c *Client) Heartbeat(agentID, agentKey string) error {
	return c.SendHeartbeat(agentID, agentKey, "", "")
}

// SendHeartbeat sends a heartbeat to the backend
func (c *Client) SendHeartbeat(agentID, agentKey, ipv4, ipv6 string) error {
	timestamp := time.Now().Unix()

	req := &pb.HeartbeatRequest{
		AgentId:     agentID,
		AgentKey:    agentKey,
		IpAddressV4: ipv4,
		IpAddressV6: ipv6,
		Timestamp:   timestamp,
	}

	// Attach HMAC authentication metadata
	ctx := context.Background()
	ctx, err := security.AttachAuthMetadata(ctx, agentID, agentKey, timestamp, req)
	if err != nil {
		return fmt.Errorf("failed to attach auth metadata: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.Heartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("heartbeat rejected: %s", resp.Message)
	}

	return nil
}

// SendMetrics sends system metrics to the backend
func (c *Client) SendMetrics(agentID, agentKey, agentVersion string, m *metrics.SystemMetrics) error {
	timestamp := time.Now().Unix()

	var pbSensorReadings []*pb.SensorReading
	for _, sr := range m.SensorReadings {
		pbSensorReadings = append(pbSensorReadings, &pb.SensorReading{
			Key:                sr.Key,
			TemperatureCelsius: sr.TemperatureCelsius,
		})
	}

	req := &pb.SendMetricsRequest{
		AgentId:  agentID,
		AgentKey: agentKey,
		Metrics: &pb.Metrics{
			CpuUsagePercent:      m.CPUUsagePercent,
			MemoryTotalBytes:     m.MemoryTotalBytes,
			MemoryUsedBytes:      m.MemoryUsedBytes,
			MemoryAvailableBytes: m.MemoryAvailableBytes,
			LoadAvg_1Min:         m.LoadAvg1Min,
			LoadAvg_5Min:         m.LoadAvg5Min,
			LoadAvg_15Min:        m.LoadAvg15Min,
			DiskTotalBytes:       m.DiskTotalBytes,
			DiskUsedBytes:        m.DiskUsedBytes,
			UptimeSeconds:        m.UptimeSeconds,
			Timestamp:            m.Timestamp,

			DiskReadBytesPerSec:   m.DiskReadBytesPerSec,
			DiskWriteBytesPerSec:  m.DiskWriteBytesPerSec,
			NetworkRxBytesPerSec:  m.NetworkRxBytesPerSec,
			NetworkTxBytesPerSec:  m.NetworkTxBytesPerSec,
			CpuTemperatureCelsius: m.CPUTemperatureCelsius,
			SensorReadings:        pbSensorReadings,
		},
		Timestamp:    timestamp, // Request-level timestamp for anti-replay
		AgentVersion: agentVersion,
	}

	// Map container metrics if present
	for _, cm := range m.ContainerMetrics {
		req.ContainerMetrics = append(req.ContainerMetrics, &pb.ContainerMetric{
			ContainerId:          cm.ContainerID,
			ContainerName:        cm.ContainerName,
			Image:                cm.Image,
			CpuPercent:           cm.CPUPercent,
			MemoryUsedBytes:      cm.MemoryUsedBytes,
			MemoryLimitBytes:     cm.MemoryLimitBytes,
			NetworkRxBytesPerSec: cm.NetworkRxBytesPerSec,
			NetworkTxBytesPerSec: cm.NetworkTxBytesPerSec,
		})
	}

	// Attach HMAC authentication metadata
	ctx := context.Background()
	ctx, err := security.AttachAuthMetadata(ctx, agentID, agentKey, timestamp, req)
	if err != nil {
		return fmt.Errorf("failed to attach auth metadata: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.SendMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("send metrics failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("metrics rejected: %s", resp.Message)
	}

	return nil
}

// ReportDroppedMetrics reports metrics that were dropped after max retries
func (c *Client) ReportDroppedMetrics(agentID, agentKey string, count int32, firstDroppedAt, lastDroppedAt int64, reason string) error {
	timestamp := time.Now().Unix()

	req := &pb.ReportDroppedMetricsRequest{
		AgentId:        agentID,
		AgentKey:       agentKey,
		Timestamp:      timestamp,
		Count:          count,
		FirstDroppedAt: firstDroppedAt,
		LastDroppedAt:  lastDroppedAt,
		Reason:         reason,
	}

	// Attach HMAC authentication metadata
	ctx := context.Background()
	ctx, err := security.AttachAuthMetadata(ctx, agentID, agentKey, timestamp, req)
	if err != nil {
		return fmt.Errorf("failed to attach auth metadata: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.ReportDroppedMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("report dropped metrics failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("dropped metrics report rejected: %s", resp.Message)
	}

	return nil
}

// PackageInventoryData contains the package inventory to send
type PackageInventoryData struct {
	InventoryType        string
	AddedPackages        []*pb.Package
	RemovedPackages      []*pb.Package
	UpdatedPackages      []*pb.Package
	AllPackages          []*pb.Package
	CollectionDurationMs int64
	TotalPackageCount    int32
}

// SendPackageInventory sends package inventory to the backend
func (c *Client) SendPackageInventory(agentID, agentKey string, data *PackageInventoryData) error {
	timestamp := time.Now().Unix()

	req := &pb.SendPackageInventoryRequest{
		AgentId:              agentID,
		AgentKey:             agentKey,
		Timestamp:            timestamp,
		InventoryType:        data.InventoryType,
		AddedPackages:        data.AddedPackages,
		RemovedPackages:      data.RemovedPackages,
		UpdatedPackages:      data.UpdatedPackages,
		AllPackages:          data.AllPackages,
		CollectionDurationMs: data.CollectionDurationMs,
		TotalPackageCount:    data.TotalPackageCount,
	}

	// Attach HMAC authentication metadata
	ctx := context.Background()
	ctx, err := security.AttachAuthMetadata(ctx, agentID, agentKey, timestamp, req)
	if err != nil {
		return fmt.Errorf("failed to attach auth metadata: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second) // Longer timeout for package inventory
	defer cancel()

	resp, err := c.client.SendPackageInventory(ctx, req)
	if err != nil {
		return fmt.Errorf("send package inventory failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("package inventory rejected: %s", resp.Message)
	}

	return nil
}
