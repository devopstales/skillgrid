package scan

import (
	"fmt"
	"os"
	"path/filepath"
)

// RawPath returns the on-disk location of a scan's raw output:
// <dataDir>/.skillgrid/cache/scans/<id>.json.
func RawPath(dataDir, scanID string) string {
	return filepath.Join(dataDir, ".skillgrid", "cache", "scans", scanID+".json")
}

// WriteRaw persists raw scanner output at 0644 and returns the path.
func WriteRaw(dataDir, scanID string, raw []byte) (string, error) {
	p := RawPath(dataDir, scanID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", fmt.Errorf("create scan cache dir: %w", err)
	}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		return "", fmt.Errorf("write scan raw: %w", err)
	}
	return p, nil
}

// ReadRaw loads the raw scanner output previously written by WriteRaw.
func ReadRaw(dataDir, scanID string) ([]byte, error) {
	b, err := os.ReadFile(RawPath(dataDir, scanID))
	if err != nil {
		return nil, fmt.Errorf("read scan raw: %w", err)
	}
	return b, nil
}
