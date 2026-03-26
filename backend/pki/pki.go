package pki

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// PKI manages TLS certificates for the gRPC server.
// After Initialize() is called, tlsConfig and caCertPEM are loaded
// from disk and cached — no further disk reads occur.
type PKI struct {
	config    *Config
	tlsConfig *tls.Config
	caCertPEM []byte
}

// New creates a new PKI instance and validates the configuration.
func New(config *Config) (*PKI, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid PKI config: %w", err)
	}
	return &PKI{config: config}, nil
}

// Initialize generates or validates certificates, then loads them into memory.
// Must be called before GetTLSConfig or GetCACertPEM.
func (p *PKI) Initialize() error {
	switch p.config.Mode {
	case ModeAuto:
		if err := p.initializeAuto(); err != nil {
			return err
		}
	case ModeCustom:
		if err := p.initializeCustom(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid TLS mode: %s", p.config.Mode)
	}
	return p.load()
}

// initializeAuto generates CA + server certificate if they don't exist yet.
func (p *PKI) initializeAuto() error {
	slog.Info("initializing PKI", "mode", "auto", "dir", p.config.PKIDir)

	if err := os.MkdirAll(p.config.PKIDir, 0750); err != nil {
		return fmt.Errorf("failed to create PKI directory: %w", err)
	}

	caPath := filepath.Join(p.config.PKIDir, "ca.pem")
	caKeyPath := filepath.Join(p.config.PKIDir, "ca.key")
	serverCertPath := filepath.Join(p.config.PKIDir, "server.pem")
	serverKeyPath := filepath.Join(p.config.PKIDir, "server.key")

	if fileExists(caPath) && fileExists(serverCertPath) {
		slog.Info("PKI already exists, reusing existing certificates")
		return nil
	}

	slog.Info("generating new PKI (CA + server certificate)")

	slog.Info("generating CA certificate", "validity", "10y")
	caCert, caKey, err := generateCA()
	if err != nil {
		return fmt.Errorf("failed to generate CA: %w", err)
	}

	if err := saveCertificate(caCert, caPath); err != nil {
		return err
	}
	if err := savePrivateKey(caKey, caKeyPath); err != nil {
		return err
	}
	slog.Info("CA certificate saved", "path", caPath)

	slog.Info("generating server certificate", "validity", "5y")
	serverCert, serverKey, err := generateServerCert(caCert, caKey)
	if err != nil {
		return fmt.Errorf("failed to generate server certificate: %w", err)
	}

	if err := saveCertificate(serverCert, serverCertPath); err != nil {
		return err
	}
	if err := savePrivateKey(serverKey, serverKeyPath); err != nil {
		return err
	}
	slog.Info("server certificate saved", "path", serverCertPath)
	slog.Info("PKI initialization complete")

	return nil
}

// initializeCustom validates that the user-provided certificates are readable.
func (p *PKI) initializeCustom() error {
	slog.Info("initializing PKI", "mode", "custom",
		"cert", p.config.CertFile,
		"key", p.config.KeyFile,
		"ca", p.config.CAFile,
	)

	if _, err := tls.LoadX509KeyPair(p.config.CertFile, p.config.KeyFile); err != nil {
		return fmt.Errorf("failed to load certificate/key pair: %w", err)
	}

	slog.Info("custom certificates validated")
	return nil
}

// load reads the certificate files from disk and caches them in the struct.
// Called once at the end of Initialize().
func (p *PKI) load() error {
	certFile, keyFile, caFile := p.config.GetPaths()

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load server certificate: %w", err)
	}

	p.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
	}

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}
	p.caCertPEM = caPEM

	return nil
}

// GetTLSConfig returns the cached TLS configuration for the gRPC server.
func (p *PKI) GetTLSConfig() (*tls.Config, error) {
	if p.tlsConfig == nil {
		return nil, fmt.Errorf("PKI not initialized")
	}
	return p.tlsConfig, nil
}

// GetCACertPEM returns the cached CA certificate in PEM format.
// Used to send the CA cert to agents during registration.
func (p *PKI) GetCACertPEM() ([]byte, error) {
	if p.caCertPEM == nil {
		return nil, fmt.Errorf("PKI not initialized")
	}
	return p.caCertPEM, nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
