package pki

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func TestCertProperties(t *testing.T) {
	dir := t.TempDir()
	p, _ := New(&Config{Mode: ModeAuto, PKIDir: dir})
	if err := p.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// Server cert: ECDSA key, correct SANs, ExtKeyUsageServerAuth.
	certPEM, err := os.ReadFile(dir + "/server.pem")
	if err != nil {
		t.Fatalf("read server cert: %v", err)
	}
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse server cert: %v", err)
	}
	if _, ok := cert.PublicKey.(*ecdsa.PublicKey); !ok {
		t.Error("expected ECDSA public key")
	}
	sans := map[string]bool{}
	for _, dns := range cert.DNSNames {
		sans[dns] = true
	}
	if !sans["watchflare"] {
		t.Error("expected 'watchflare' in DNSNames")
	}
	if !sans["localhost"] {
		t.Error("expected 'localhost' in DNSNames")
	}
	hasServerAuth := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
	}
	if !hasServerAuth {
		t.Error("expected ExtKeyUsageServerAuth")
	}

	// CA cert: ECDSA key, IsCA, signs server cert.
	caPEM, _ := os.ReadFile(dir + "/ca.pem")
	block, _ = pem.Decode(caPEM)
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}
	if _, ok := caCert.PublicKey.(*ecdsa.PublicKey); !ok {
		t.Error("expected ECDSA public key for CA")
	}
	if !caCert.IsCA {
		t.Error("expected CA cert to have IsCA=true")
	}

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	if _, err := cert.Verify(x509.VerifyOptions{Roots: pool, DNSName: "watchflare"}); err != nil {
		t.Errorf("server cert not verified by CA: %v", err)
	}
}

func TestMismatchedKeyPair(t *testing.T) {
	dir := t.TempDir()

	caCert1, caKey1, _ := generateCA()
	caCert2, caKey2, _ := generateCA()
	serverCert1, _, _ := generateServerCert(caCert1, caKey1)
	_, serverKey2, _ := generateServerCert(caCert2, caKey2)

	certPath := dir + "/server.pem"
	keyPath := dir + "/server.key"
	caPath := dir + "/ca.pem"
	saveCertificate(serverCert1, certPath)
	savePrivateKey(serverKey2, keyPath)
	saveCertificate(caCert1, caPath)

	p, err := New(&Config{Mode: ModeCustom, CertFile: certPath, KeyFile: keyPath, CAFile: caPath})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := p.Initialize(); err == nil {
		t.Error("expected error for mismatched cert/key pair")
	}
}
