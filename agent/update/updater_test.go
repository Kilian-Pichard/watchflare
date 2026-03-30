package update

import (
	"archive/tar"
	"compress/gzip"
	"os"
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
