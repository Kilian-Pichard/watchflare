package update

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"strings"
	"testing"
)

// makeTarball creates a .tar.gz in a temp file containing the given entries.
// Each entry is (name, content).
func makeTarball(t *testing.T, entries []struct{ name, content string }) string {
	t.Helper()
	tmp, err := os.CreateTemp(t.TempDir(), "test-*.tar.gz")
	if err != nil {
		t.Fatalf("create temp tarball: %v", err)
	}
	defer tmp.Close()

	gw := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: tar.TypeReg,
			Size:     int64(len(e.content)),
			Mode:     0755,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(e.content)); err != nil {
			t.Fatalf("write tar entry: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	return tmp.Name()
}

// --- extractBinary ---

func TestExtractBinary_Found(t *testing.T) {
	tarball := makeTarball(t, []struct{ name, content string }{
		{"watchflare-agent", "binary-content"},
	})

	out, err := extractBinary(tarball)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(out)

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read extracted binary: %v", err)
	}
	if string(data) != "binary-content" {
		t.Errorf("content = %q, want %q", string(data), "binary-content")
	}
}

func TestExtractBinary_FoundInSubdir(t *testing.T) {
	// filepath.Base strips the directory prefix — binary in a subdir must still be found.
	tarball := makeTarball(t, []struct{ name, content string }{
		{"v1.2.3/watchflare-agent", "nested-binary"},
	})

	out, err := extractBinary(tarball)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(out)

	data, _ := os.ReadFile(out)
	if string(data) != "nested-binary" {
		t.Errorf("content = %q, want %q", string(data), "nested-binary")
	}
}

func TestExtractBinary_NotFound(t *testing.T) {
	tarball := makeTarball(t, []struct{ name, content string }{
		{"other-binary", "irrelevant"},
	})

	_, err := extractBinary(tarball)
	if err == nil {
		t.Error("expected error when binary not in tarball")
	}
}

func TestExtractBinary_SkipsNonRegular(t *testing.T) {
	// A directory entry named "watchflare-agent" must not be extracted.
	tmp, err := os.CreateTemp(t.TempDir(), "test-*.tar.gz")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer tmp.Close()

	gw := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gw)

	// Directory entry with the target name
	if err := tw.WriteHeader(&tar.Header{
		Name:     "watchflare-agent",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}); err != nil {
		t.Fatalf("write dir header: %v", err)
	}
	tw.Close()
	gw.Close()

	_, err = extractBinary(tmp.Name())
	if err == nil {
		t.Error("expected error: directory entry must not be extracted as binary")
	}
}

// --- writeTriggerFile ---

func TestWriteTriggerFile_WritesContent(t *testing.T) {
	dir := t.TempDir()
	triggerPath := dir + "/update-pending"
	binaryPath := "/tmp/watchflare-agent-new-12345"

	if err := writeTriggerFile(triggerPath, binaryPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(triggerPath)
	if err != nil {
		t.Fatalf("read trigger file: %v", err)
	}
	if strings.TrimSpace(string(data)) != binaryPath {
		t.Errorf("content = %q, want %q", strings.TrimSpace(string(data)), binaryPath)
	}
}

func TestWriteTriggerFile_NoDir(t *testing.T) {
	err := writeTriggerFile("/nonexistent/dir/update-pending", "/tmp/something")
	if err == nil {
		t.Error("expected error when directory does not exist")
	}
}

// --- applyFromTrigger ---

func TestApplyFromTrigger_NoTriggerFile(t *testing.T) {
	dir := t.TempDir()
	err := applyFromTrigger(dir+"/update-pending", dir+"/watchflare-agent")
	if err == nil {
		t.Error("expected error when trigger file does not exist")
	}
}

func TestApplyFromTrigger_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	triggerPath := dir + "/update-pending"
	os.WriteFile(triggerPath, []byte("\n"), 0640) //nolint:errcheck

	err := applyFromTrigger(triggerPath, dir+"/watchflare-agent")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected 'empty' error, got %v", err)
	}
}

func TestApplyFromTrigger_PathNotUnderTmp(t *testing.T) {
	dir := t.TempDir()
	triggerPath := dir + "/update-pending"
	os.WriteFile(triggerPath, []byte("/var/lib/something\n"), 0640) //nolint:errcheck

	err := applyFromTrigger(triggerPath, dir+"/watchflare-agent")
	if err == nil || !strings.Contains(err.Error(), "must be under /tmp") {
		t.Errorf("expected path validation error, got %v", err)
	}
}

func TestApplyFromTrigger_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	triggerPath := dir + "/update-pending"
	// filepath.Clean resolves /tmp/../etc/passwd → /etc/passwd, which then fails the /tmp/ check
	os.WriteFile(triggerPath, []byte("/tmp/../etc/passwd\n"), 0640) //nolint:errcheck

	err := applyFromTrigger(triggerPath, dir+"/watchflare-agent")
	if err == nil || !strings.Contains(err.Error(), "must be under /tmp") {
		t.Errorf("expected path traversal rejection, got %v", err)
	}
}

func TestApplyFromTrigger_Symlink(t *testing.T) {
	dir := t.TempDir()
	triggerPath := dir + "/update-pending"

	// Create a symlink inside /tmp pointing to a file outside /tmp
	realTarget := dir + "/real-file"
	os.WriteFile(realTarget, []byte("data"), 0644) //nolint:errcheck

	tmpDir, err := os.MkdirTemp("/tmp", "wf-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	linkPath := tmpDir + "/watchflare-link"
	if err := os.Symlink(realTarget, linkPath); err != nil {
		t.Fatal(err)
	}

	os.WriteFile(triggerPath, []byte(linkPath+"\n"), 0640) //nolint:errcheck

	err = applyFromTrigger(triggerPath, dir+"/watchflare-agent")
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("expected symlink rejection, got %v", err)
	}
}

// --- copyFile ---

func TestCopyFile_ContentAndPermissions(t *testing.T) {
	dir := t.TempDir()
	src := dir + "/src"
	dst := dir + "/dst"

	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := copyFile(src, dst, 0755); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", string(data), "hello")
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat dst: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("permissions = %v, want 0755", info.Mode().Perm())
	}
}
