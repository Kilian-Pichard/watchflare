package pki

import (
	"os"
	"testing"
)

func TestValidate_AutoMode_EmptyPKIDir(t *testing.T) {
	if _, err := New(&Config{Mode: ModeAuto, PKIDir: ""}); err == nil {
		t.Error("expected error for empty PKIDir")
	}
}

func TestValidate_InvalidMode(t *testing.T) {
	if _, err := New(&Config{Mode: Mode("invalid")}); err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestValidate_CustomMode_NonexistentFiles(t *testing.T) {
	if _, err := New(&Config{
		Mode:     ModeCustom,
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
		CAFile:   "/nonexistent/ca.pem",
	}); err == nil {
		t.Error("expected error for nonexistent cert files")
	}
}

func TestValidate_CustomMode_MissingFields(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/dummy"
	os.WriteFile(f, []byte("x"), 0644)

	cases := []struct {
		name string
		cfg  Config
	}{
		{"empty CertFile", Config{Mode: ModeCustom, CertFile: "", KeyFile: f, CAFile: f}},
		{"empty KeyFile", Config{Mode: ModeCustom, CertFile: f, KeyFile: "", CAFile: f}},
		{"empty CAFile", Config{Mode: ModeCustom, CertFile: f, KeyFile: f, CAFile: ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(&tc.cfg); err == nil {
				t.Error("expected error")
			}
		})
	}
}
