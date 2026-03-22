package pki

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// PKI manages the TLS certificates
type PKI struct {
	config *Config
}

// New creates a new PKI instance
func New(config *Config) (*PKI, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid PKI config: %w", err)
	}

	return &PKI{
		config: config,
	}, nil
}

// Initialize sets up the PKI (generates certificates if needed in auto mode)
func (p *PKI) Initialize() error {
	switch p.config.Mode {
	case ModeAuto:
		return p.initializeAuto()
	case ModeCustom:
		return p.initializeCustom()
	default:
		return fmt.Errorf("invalid TLS mode: %s", p.config.Mode)
	}
}

// initializeAuto handles auto-generated PKI
func (p *PKI) initializeAuto() error {
	slog.Info("initializing PKI", "mode", "auto", "dir", p.config.PKIDir)

	// Create PKI directory if it doesn't exist
	if err := os.MkdirAll(p.config.PKIDir, 0755); err != nil {
		return fmt.Errorf("failed to create PKI directory: %w", err)
	}

	caPath := filepath.Join(p.config.PKIDir, "ca.pem")
	caKeyPath := filepath.Join(p.config.PKIDir, "ca.key")
	serverCertPath := filepath.Join(p.config.PKIDir, "server.pem")
	serverKeyPath := filepath.Join(p.config.PKIDir, "server.key")

	// Check if PKI already exists
	if fileExists(caPath) && fileExists(serverCertPath) {
		slog.Info("PKI already exists, reusing existing certificates")
		return nil
	}

	// Generate new PKI
	slog.Info("generating new PKI (CA + server certificate)")

	// Generate CA
	slog.Info("generating CA certificate", "validity", "10y")
	caCert, caKey, err := generateCA()
	if err != nil {
		return fmt.Errorf("failed to generate CA: %w", err)
	}

	// Save CA certificate and key
	if err := saveCertificate(caCert, caPath); err != nil {
		return err
	}
	if err := savePrivateKey(caKey, caKeyPath); err != nil {
		return err
	}

	slog.Info("CA certificate saved", "path", caPath)
	slog.Info("CA private key saved", "path", caKeyPath, "permissions", "0600")

	// Generate server certificate
	slog.Info("generating server certificate", "validity", "5y")
	serverCert, serverKey, err := generateServerCert(caCert, caKey)
	if err != nil {
		return fmt.Errorf("failed to generate server certificate: %w", err)
	}

	// Save server certificate and key
	if err := saveCertificate(serverCert, serverCertPath); err != nil {
		return err
	}
	if err := savePrivateKey(serverKey, serverKeyPath); err != nil {
		return err
	}

	slog.Info("server certificate saved", "path", serverCertPath)
	slog.Info("server private key saved", "path", serverKeyPath, "permissions", "0600")
	slog.Info("PKI initialization complete")
	return nil
}

// initializeCustom validates custom certificates
func (p *PKI) initializeCustom() error {
	slog.Info("initializing PKI", "mode", "custom")
	slog.Info("using custom certificates",
		"cert", p.config.CertFile,
		"key", p.config.KeyFile,
		"ca", p.config.CAFile,
	)

	// Load and validate certificate/key pair
	_, err := tls.LoadX509KeyPair(p.config.CertFile, p.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate/key pair: %w", err)
	}

	slog.Info("custom certificates validated")
	return nil
}

// GetTLSConfig returns a TLS configuration for the gRPC server
func (p *PKI) GetTLSConfig() (*tls.Config, error) {
	certFile, keyFile, _ := p.config.GetPaths()

	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13, // TLS 1.3 only
		MaxVersion:   tls.VersionTLS13,
	}

	return tlsConfig, nil
}

// GetCACertPEM returns the CA certificate in PEM format
// This is used to send the CA cert to agents during registration
func (p *PKI) GetCACertPEM() ([]byte, error) {
	_, _, caFile := p.config.GetPaths()

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	return caPEM, nil
}

// GetCACertPool returns a certificate pool with the CA certificate
// This can be used for mTLS if needed in the future
func (p *PKI) GetCACertPool() (*x509.CertPool, error) {
	caPEM, err := p.GetCACertPEM()
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to append CA certificate to pool")
	}

	return pool, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
