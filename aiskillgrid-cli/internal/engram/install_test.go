package engram

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	goos, arch, err := DetectPlatform()
	if err != nil {
		t.Fatalf("DetectPlatform failed: %v", err)
	}
	if goos != "darwin" && goos != "linux" {
		t.Fatalf("unexpected goos: %s", goos)
	}
	if arch != "amd64" && arch != "arm64" {
		t.Fatalf("unexpected arch: %s", arch)
	}
}

func TestInstallBinarySkipsWhenAlreadyPresent(t *testing.T) {
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	os.MkdirAll(binDir, 0755)
	os.WriteFile(filepath.Join(binDir, "engram"), []byte("fake"), 0755)

	err := InstallBinary(tmp)
	if err != nil {
		t.Fatalf("InstallBinary failed: %v", err)
	}
}

func TestInstallBinaryDownloadsAndExtracts(t *testing.T) {
	tmp := t.TempDir()
	if err := InstallBinary(tmp); err != nil {
		t.Fatalf("InstallBinary failed: %v", err)
	}
	binaryPath := filepath.Join(tmp, "bin", "engram")
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("engram binary not installed: %v", err)
	}
}
