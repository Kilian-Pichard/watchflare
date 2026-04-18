package services

import (
	"errors"
	"testing"
)

func TestSMTPConfigError_Error(t *testing.T) {
	err := &SMTPConfigError{msg: "SMTP host is not configured"}
	if err.Error() != "SMTP host is not configured" {
		t.Errorf("got %q, want %q", err.Error(), "SMTP host is not configured")
	}
}

func TestNewConfigError_ReturnsSMTPConfigError(t *testing.T) {
	err := newConfigError("SMTP is disabled")
	var configErr *SMTPConfigError
	if !errors.As(err, &configErr) {
		t.Errorf("expected *SMTPConfigError, got %T", err)
	}
	if configErr.msg != "SMTP is disabled" {
		t.Errorf("got %q, want %q", configErr.msg, "SMTP is disabled")
	}
}

func TestNewConfigError_NotWrapped(t *testing.T) {
	err := newConfigError("test error")
	if errors.Unwrap(err) != nil {
		t.Error("expected no wrapped error")
	}
}

func TestSMTPConstants(t *testing.T) {
	if TLSModeNone != "none" {
		t.Errorf("TLSModeNone = %q, want %q", TLSModeNone, "none")
	}
	if TLSModeStartTLS != "starttls" {
		t.Errorf("TLSModeStartTLS = %q, want %q", TLSModeStartTLS, "starttls")
	}
	if TLSModeSSL != "tls" {
		t.Errorf("TLSModeSSL = %q, want %q", TLSModeSSL, "tls")
	}
	if SMTPAuthPlain != "plain" {
		t.Errorf("SMTPAuthPlain = %q, want %q", SMTPAuthPlain, "plain")
	}
	if SMTPAuthLogin != "login" {
		t.Errorf("SMTPAuthLogin = %q, want %q", SMTPAuthLogin, "login")
	}
}
