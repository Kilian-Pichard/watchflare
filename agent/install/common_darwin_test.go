//go:build darwin

package install

import (
	"strings"
	"testing"
)

func TestGetServiceManager_Darwin_ReturnsError(t *testing.T) {
	svc, err := GetServiceManager()
	if err == nil {
		t.Fatal("expected error on macOS, got nil")
	}
	if svc != nil {
		t.Error("expected nil ServiceManager on macOS")
	}
	if !strings.Contains(err.Error(), "Homebrew") {
		t.Errorf("expected Homebrew hint in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "brew services") {
		t.Errorf("expected brew services command in error, got: %v", err)
	}
}
