package pki

import (
	"fmt"
	"os"
	"path/filepath"
)

// Mode defines the TLS mode
type Mode string

const (
	ModeAuto   Mode = "auto"   // Auto-generated Watchflare PKI
	ModeCustom Mode = "custom" // User-provided certificates
)

// Config holds PKI configuration
type Config struct {
	Mode Mode

	// Auto mode - PKI directory
	PKIDir string

	// Custom mode - User-provided paths
	CertFile string
	KeyFile  string
	CAFile   string
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	switch c.Mode {
	case ModeAuto:
		if c.PKIDir == "" {
			return fmt.Errorf("pki_dir is required in auto mode")
		}

	case ModeCustom:
		if c.CertFile == "" {
			return fmt.Errorf("cert_file is required in custom mode")
		}
		if c.KeyFile == "" {
			return fmt.Errorf("key_file is required in custom mode")
		}
		if c.CAFile == "" {
			return fmt.Errorf("ca_file is required in custom mode")
		}

		// Check if files exist
		if _, err := os.Stat(c.CertFile); err != nil {
			return fmt.Errorf("cert_file not found: %w", err)
		}
		if _, err := os.Stat(c.KeyFile); err != nil {
			return fmt.Errorf("key_file not found: %w", err)
		}
		if _, err := os.Stat(c.CAFile); err != nil {
			return fmt.Errorf("ca_file not found: %w", err)
		}

	default:
		return fmt.Errorf("invalid TLS mode: %s (must be 'auto' or 'custom')", c.Mode)
	}

	return nil
}

// GetPaths returns the certificate paths based on mode
func (c *Config) GetPaths() (certFile, keyFile, caFile string) {
	if c.Mode == ModeCustom {
		return c.CertFile, c.KeyFile, c.CAFile
	}

	// Auto mode
	return filepath.Join(c.PKIDir, "server.pem"),
		filepath.Join(c.PKIDir, "server.key"),
		filepath.Join(c.PKIDir, "ca.pem")
}
