package install

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// --- CheckRoot ---

func TestCheckRoot_NotRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}

	err := CheckRoot()
	if err == nil {
		t.Fatal("expected error when not root")
	}
	if !strings.Contains(err.Error(), "sudo") {
		t.Errorf("expected hint about sudo in error, got: %v", err)
	}
}

// --- AskConfirmation ---

// askWithInput redirects os.Stdin so AskConfirmation reads from the provided string.
// Tests using this helper must NOT run in parallel (global os.Stdin mutation).
func askWithInput(t *testing.T, input string) bool {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	orig := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = orig
		r.Close()
	}()

	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}
	w.Close()

	return AskConfirmation("test?")
}

func TestAskConfirmation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"y", "y\n", true},
		{"Y", "Y\n", true},
		{"yes", "yes\n", true},
		{"YES", "YES\n", true},
		{"empty_defaults_no", "\n", false},
		{"n", "n\n", false},
		{"no", "no\n", false},
		{"garbage", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := askWithInput(t, tt.input)
			if got != tt.want {
				t.Errorf("input %q: got %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- GetBinaryPath ---

func TestGetBinaryPath_ReturnsNonEmpty(t *testing.T) {
	path, err := GetBinaryPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

// --- getUserID / getGroupID ---

func TestGetGroupID_Root(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("root group is named 'wheel' on macOS, not 'root'")
	}

	gid, err := getGroupID("root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gid != 0 {
		t.Errorf("expected GID 0 for root group, got %d", gid)
	}
}

func TestGetUserID_Root(t *testing.T) {
	uid, err := getUserID("root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != 0 {
		t.Errorf("expected UID 0 for root, got %d", uid)
	}
}
