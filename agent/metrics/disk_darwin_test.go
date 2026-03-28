//go:build darwin

package metrics

import "testing"

func TestGetDiskUsage_ReturnsNonZero(t *testing.T) {
	total, used, err := getDiskUsage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total == 0 {
		t.Error("expected non-zero total disk size")
	}
	if used > total {
		t.Errorf("used (%d) > total (%d): underflow", used, total)
	}
}
