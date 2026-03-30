package uuid

import (
	"os"
	"path/filepath"
	"testing"

	"watchflare-agent/config"
)

// setDataDir overrides the data directory for the duration of a test.
func setDataDir(t *testing.T, dir string) {
	t.Helper()
	orig := os.Getenv("WATCHFLARE_DATA_DIR")
	os.Setenv("WATCHFLARE_DATA_DIR", dir)
	t.Cleanup(func() { os.Setenv("WATCHFLARE_DATA_DIR", orig) })
}

// tempDataDir creates a temp directory and points the config data dir at it.
func tempDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	setDataDir(t, dir)
	_ = config.GetDataDir() // ensure the env var is picked up
	return dir
}

// --- Save ---

func TestSave_WritesFile(t *testing.T) {
	dir := tempDataDir(t)
	id := "550e8400-e29b-41d4-a716-446655440000"

	if err := Save(id); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, uuidFileName))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if got := string(data); got != id+"\n" {
		t.Errorf("file content = %q, want %q", got, id+"\n")
	}
}

func TestSave_EmptyUUID(t *testing.T) {
	tempDataDir(t)
	if err := Save(""); err == nil {
		t.Error("expected error for empty UUID")
	}
}

func TestSave_FilePermissions(t *testing.T) {
	dir := tempDataDir(t)
	if err := Save("550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, uuidFileName))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0640 {
		t.Errorf("file permissions = %04o, want 0640", perm)
	}
}

// --- Load ---

func TestLoad_ReturnsUUID(t *testing.T) {
	tempDataDir(t)
	id := "550e8400-e29b-41d4-a716-446655440000"

	if err := Save(id); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != id {
		t.Errorf("Load() = %q, want %q", got, id)
	}
}

func TestLoad_NotExist(t *testing.T) {
	tempDataDir(t)
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != "" {
		t.Errorf("Load() = %q, want empty string", got)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := tempDataDir(t)
	if err := os.WriteFile(filepath.Join(dir, uuidFileName), []byte("   \n"), 0640); err != nil {
		t.Fatalf("write empty file: %v", err)
	}
	_, err := Load()
	if err == nil {
		t.Error("expected error for empty UUID file")
	}
}

// --- Exists ---

func TestExists_True(t *testing.T) {
	tempDataDir(t)
	if err := Save("550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !Exists() {
		t.Error("Exists() = false, want true")
	}
}

func TestExists_False(t *testing.T) {
	tempDataDir(t)
	if Exists() {
		t.Error("Exists() = true, want false")
	}
}
