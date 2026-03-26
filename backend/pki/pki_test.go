package pki

import (
	"crypto/tls"
	"os"
	"testing"
)

func TestInitialize_Auto(t *testing.T) {
	p, err := New(&Config{Mode: ModeAuto, PKIDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := p.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	tlsCfg, err := p.GetTLSConfig()
	if err != nil {
		t.Fatalf("GetTLSConfig: %v", err)
	}
	if tlsCfg.MinVersion != tls.VersionTLS13 {
		t.Error("expected MinVersion TLS 1.3")
	}
	if tlsCfg.MaxVersion != tls.VersionTLS13 {
		t.Error("expected MaxVersion TLS 1.3")
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(tlsCfg.Certificates))
	}

	caPEM, err := p.GetCACertPEM()
	if err != nil {
		t.Fatalf("GetCACertPEM: %v", err)
	}
	if len(caPEM) == 0 {
		t.Error("expected non-empty CA PEM")
	}
}

func TestInitialize_Auto_Idempotent(t *testing.T) {
	dir := t.TempDir()

	p, _ := New(&Config{Mode: ModeAuto, PKIDir: dir})
	if err := p.Initialize(); err != nil {
		t.Fatalf("first Initialize: %v", err)
	}

	cert1, err := os.ReadFile(dir + "/server.pem")
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}

	// Second instance on the same directory must reuse existing certs.
	p2, _ := New(&Config{Mode: ModeAuto, PKIDir: dir})
	if err := p2.Initialize(); err != nil {
		t.Fatalf("second Initialize: %v", err)
	}

	cert2, _ := os.ReadFile(dir + "/server.pem")
	if string(cert1) != string(cert2) {
		t.Error("certificates were regenerated instead of reused")
	}
}

func TestInitialize_Custom(t *testing.T) {
	dir := t.TempDir()

	caCert, caKey, err := generateCA()
	if err != nil {
		t.Fatalf("generateCA: %v", err)
	}
	serverCert, serverKey, err := generateServerCert(caCert, caKey)
	if err != nil {
		t.Fatalf("generateServerCert: %v", err)
	}

	caPath := dir + "/ca.pem"
	certPath := dir + "/server.pem"
	keyPath := dir + "/server.key"

	if err := saveCertificate(caCert, caPath); err != nil {
		t.Fatalf("save CA cert: %v", err)
	}
	if err := saveCertificate(serverCert, certPath); err != nil {
		t.Fatalf("save server cert: %v", err)
	}
	if err := savePrivateKey(serverKey, keyPath); err != nil {
		t.Fatalf("save server key: %v", err)
	}

	p, err := New(&Config{
		Mode:     ModeCustom,
		CertFile: certPath,
		KeyFile:  keyPath,
		CAFile:   caPath,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := p.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	tlsCfg, err := p.GetTLSConfig()
	if err != nil {
		t.Fatalf("GetTLSConfig: %v", err)
	}
	if tlsCfg.MinVersion != tls.VersionTLS13 {
		t.Error("expected MinVersion TLS 1.3")
	}
	if tlsCfg.MaxVersion != tls.VersionTLS13 {
		t.Error("expected MaxVersion TLS 1.3")
	}
}

func TestGetters_NotInitialized(t *testing.T) {
	p, _ := New(&Config{Mode: ModeAuto, PKIDir: t.TempDir()})

	if _, err := p.GetTLSConfig(); err == nil {
		t.Error("expected error from GetTLSConfig before Initialize")
	}
	if _, err := p.GetCACertPEM(); err == nil {
		t.Error("expected error from GetCACertPEM before Initialize")
	}
}
