package update

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const serviceCtlTimeout = 30 * time.Second

const (
	binaryInstallPath  = "/usr/local/bin/watchflare-agent"
	macOSServiceName   = "io.watchflare.agent"
	plistPath          = "/Library/LaunchDaemons/io.watchflare.agent.plist"
	systemdServiceName = "watchflare-agent"

	// Internal flags for the two-phase update (Phase 1 re-execs from /tmp for Phase 2)
	ApplyFlag   = "--_apply="
	UpdaterFlag = "--_updater="
	VersionFlag = "--_version="
)

// ApplyUpdate is Phase 1: download, verify, extract, then re-exec from /tmp.
//
// Re-execing from a temp path avoids the macOS security restriction that sends
// SIGKILL to any process replacing a binary currently memory-mapped by a
// running process (the agent service). By running Phase 2 from /tmp, the
// service binary can be freely replaced.
func logStep(msg string, args ...any) {
	slog.Info(msg, args...)
}

func ApplyUpdate(info *UpdateInfo) error {
	exe, _ := os.Executable()
	logStep("update phase 1 start", "pid", os.Getpid(), "exe", exe)

	logStep("downloading tarball", "url", info.TarballURL)
	tmpTarball, err := downloadToTemp(info.TarballURL, "watchflare-agent-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tmpTarball)
	logStep("download complete", "path", tmpTarball)

	logStep("verifying checksum")
	if err := verifyChecksum(tmpTarball, info.TarballName, info.ChecksumsURL); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}
	logStep("checksum ok")

	logStep("extracting binary")
	tmpBinary, err := extractBinary(tmpTarball)
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}
	logStep("binary extracted", "path", tmpBinary)

	// On Linux, the re-exec from /tmp is unnecessary: Linux allows replacing
	// an in-use binary via inode renaming — the running process keeps the old
	// inode until it exits. Apply directly without copying to /tmp.
	// (macOS routes through Homebrew and never reaches this point.)
	if runtime.GOOS == "linux" {
		return ApplyExtracted(tmpBinary, "")
	}

	// macOS (direct install): re-exec from /tmp to avoid the SIP restriction
	// that sends SIGKILL when a process replaces a binary it has memory-mapped.
	self, err := os.Executable()
	if err != nil {
		os.Remove(tmpBinary)
		return fmt.Errorf("failed to locate current binary: %w", err)
	}

	tmpUpdater, err := copyToTemp(self, "watchflare-updater-*")
	if err != nil {
		os.Remove(tmpBinary)
		return fmt.Errorf("failed to copy updater to temp: %w", err)
	}
	logStep("updater copied to temp", "path", tmpUpdater)

	args := []string{
		filepath.Base(self),
		"update",
		ApplyFlag + tmpBinary,
		UpdaterFlag + tmpUpdater,
		VersionFlag + info.LatestVersion,
	}
	logStep("re-executing from temp", "path", tmpUpdater)
	if err := syscall.Exec(tmpUpdater, args, os.Environ()); err != nil {
		os.Remove(tmpBinary)
		os.Remove(tmpUpdater)
		return fmt.Errorf("failed to re-exec updater: %w", err)
	}
	return nil // never reached
}

// ApplyExtracted is Phase 2: called when running from /tmp.
// Stops the service, atomically replaces the binary, starts the service,
// then cleans up temp files.
func ApplyExtracted(extractedBinaryPath, updaterPath string) error {
	exe, _ := os.Executable()
	logStep("update phase 2 start", "pid", os.Getpid(), "exe", exe)
	defer os.Remove(extractedBinaryPath)
	if updaterPath != "" {
		defer os.Remove(updaterPath)
	}

	logStep("stopping service")
	if err := stopService(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	logStep("service stopped")

	stagingPath := binaryInstallPath + ".new"
	logStep("staging binary", "src", extractedBinaryPath, "dst", stagingPath)
	if err := copyFile(extractedBinaryPath, stagingPath, 0755); err != nil {
		os.Remove(stagingPath)
		startService()
		return fmt.Errorf("failed to stage binary (are you root?): %w", err)
	}

	logStep("replacing binary", "src", stagingPath, "dst", binaryInstallPath)
	if err := os.Rename(stagingPath, binaryInstallPath); err != nil {
		os.Remove(stagingPath)
		startService()
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	logStep("starting service")
	err := startService()
	if err != nil {
		logStep("failed to start service", "error", err)
	} else {
		logStep("service started")
	}
	return err
}

// copyToTemp copies src to a new randomly-named temp file and returns its path
func copyToTemp(src, pattern string) (string, error) {
	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	tmp.Close()

	if err := copyFile(src, tmp.Name(), 0755); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// downloadToTemp downloads a URL to a temporary file and returns its path
func downloadToTemp(url, pattern string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	const maxTarballBytes = 250 * 1024 * 1024 // 250 MB
	if _, err := io.Copy(tmp, io.LimitReader(resp.Body, maxTarballBytes)); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tmp.Close()

	return tmp.Name(), nil
}

// verifyChecksum downloads the checksums file and verifies the tarball SHA256
func verifyChecksum(tarballPath, tarballName, checksumsURL string) error {
	client := &http.Client{Timeout: httpTimeout}

	resp, err := client.Get(checksumsURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	const maxChecksumsBytes = 1 * 1024 * 1024 // 1 MB
	var expectedHash string
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, maxChecksumsBytes))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == tarballName {
			expectedHash = fields[0]
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read checksums: %w", err)
	}
	if expectedHash == "" {
		return fmt.Errorf("checksum not found for %s", tarballName)
	}

	f, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("failed to open tarball: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to hash tarball: %w", err)
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// extractBinary extracts the watchflare-agent binary from a .tar.gz archive
func extractBinary(tarballPath string) (string, error) {
	f, err := os.Open(tarballPath)
	if err != nil {
		return "", fmt.Errorf("failed to open tarball: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("failed to decompress tarball: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tarball: %w", err)
		}

		name := filepath.Base(hdr.Name)
		if hdr.Typeflag == tar.TypeReg && name == "watchflare-agent" {
			tmp, err := os.CreateTemp("", "watchflare-agent-new-*")
			if err != nil {
				return "", fmt.Errorf("failed to create temp binary: %w", err)
			}

			if _, err := io.Copy(tmp, tr); err != nil {
				tmp.Close()
				os.Remove(tmp.Name())
				return "", fmt.Errorf("failed to extract binary: %w", err)
			}
			tmp.Close()

			return tmp.Name(), nil
		}
	}

	return "", fmt.Errorf("watchflare-agent binary not found in tarball")
}

// copyFile copies src to dst with the given permissions
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func stopService() error {
	ctx, cancel := context.WithTimeout(context.Background(), serviceCtlTimeout)
	defer cancel()
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("systemctl"); err != nil {
			return nil
		}
		cmd := exec.CommandContext(ctx, "systemctl", "stop", systemdServiceName)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to stop service: %w\n%s", err, string(out))
		}
		return nil
	case "darwin":
		exec.CommandContext(ctx, "launchctl", "bootout", "system/"+macOSServiceName).Run() //nolint:errcheck
		return nil
	default:
		return nil
	}
}

func startService() error {
	ctx, cancel := context.WithTimeout(context.Background(), serviceCtlTimeout)
	defer cancel()
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("systemctl"); err != nil {
			return nil
		}
		cmd := exec.CommandContext(ctx, "systemctl", "start", systemdServiceName)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to start service: %w\n%s", err, string(out))
		}
		return nil
	case "darwin":
		cmd := exec.CommandContext(ctx, "launchctl", "bootstrap", "system", plistPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to start service: %w\n%s", err, string(out))
		}
		return nil
	default:
		return fmt.Errorf("unsupported OS: %s — start the service manually", runtime.GOOS)
	}
}
