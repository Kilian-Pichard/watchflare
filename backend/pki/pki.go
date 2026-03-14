package pki

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
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
	log.Printf("Initializing PKI in auto mode (dir: %s)", p.config.PKIDir)

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
		log.Println("✓ PKI already exists, reusing existing certificates")
		return nil
	}

	// Generate new PKI
	log.Println("Generating new PKI (CA + server certificate)...")

	// Generate CA
	log.Println("  - Generating CA certificate (valid for 10 years)...")
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

	log.Printf("  ✓ CA certificate saved: %s", caPath)
	log.Printf("  ✓ CA private key saved: %s (permissions: 0600)", caKeyPath)

	// Generate server certificate
	log.Println("  - Generating server certificate (valid for 5 years)...")
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

	log.Printf("  ✓ Server certificate saved: %s", serverCertPath)
	log.Printf("  ✓ Server private key saved: %s (permissions: 0600)", serverKeyPath)

	log.Println("✓ PKI initialization complete")
	return nil
}

// initializeCustom validates custom certificates
func (p *PKI) initializeCustom() error {
	log.Println("Initializing PKI in custom mode")

	// Validate that all files exist (already done in config.Validate)
	log.Printf("  - Certificate: %s", p.config.CertFile)
	log.Printf("  - Private key: %s", p.config.KeyFile)
	log.Printf("  - CA certificate: %s", p.config.CAFile)

	// Load and validate certificate/key pair
	_, err := tls.LoadX509KeyPair(p.config.CertFile, p.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate/key pair: %w", err)
	}

	log.Println("✓ Custom certificates validated successfully")
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
