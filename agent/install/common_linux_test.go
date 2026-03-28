//go:build linux

package install

import "testing"

func TestGetServiceManager_Linux_ReturnsManager(t *testing.T) {
	svc, err := GetServiceManager()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil ServiceManager on Linux")
	}
	if _, ok := svc.(*LinuxService); !ok {
		t.Errorf("expected *LinuxService, got %T", svc)
	}
}
