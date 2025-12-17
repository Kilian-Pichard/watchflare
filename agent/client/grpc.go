package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"watchflare/agent/metrics"
	pb "watchflare/agent/proto"
	"watchflare/agent/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client handles gRPC communication with the backend
type Client struct {
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
	host   string
	port   string
}

// New creates a new gRPC client with optional TLS support
func New(host, port, caCertFile, serverName string) (*Client, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	var opts []grpc.DialOption

	// TLS configuration if CA cert file is provided
	if caCertFile != "" {
		// Load CA certificate
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		// Create cert pool and add CA cert
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}

		// Create TLS config
		tlsConfig := &tls.Config{
			RootCAs:    certPool,
			ServerName: serverName, // For certificate verification
		}

		// Create TLS credentials
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		// Fallback to insecure connection (backward compatibility)
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(addr, opts...)
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

// Register attempts to register the agent with the backend
func (c *Client) Register(token, hostname, ipv4, ipv6, platform, platformVersion, platformFamily, architecture, kernel string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &pb.RegisterRequest{
		RegistrationToken: token,
		Hostname:          hostname,
		IpAddressV4:       ipv4,
		IpAddressV6:       ipv6,
		Platform:          platform,
		PlatformVersion:   platformVersion,
		PlatformFamily:    platformFamily,
		Architecture:      architecture,
		Kernel:            kernel,
		Timestamp:         time.Now().Unix(), // Add timestamp for anti-replay
	}

	// Note: Registration uses token-based auth, not HMAC
	// HMAC is only used after successful registration

	resp, err := c.client.RegisterServer(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("registration failed: %w", err)
	}

	if !resp.Success {
		return "", "", fmt.Errorf("registration rejected: %s", resp.Message)
	}

	return resp.AgentId, resp.AgentKey, nil
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
func (c *Client) SendMetrics(agentID, agentKey string, m *metrics.SystemMetrics) error {
	timestamp := time.Now().Unix()

	req := &pb.MetricsRequest{
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
		},
		Timestamp: timestamp, // Request-level timestamp for anti-replay
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
