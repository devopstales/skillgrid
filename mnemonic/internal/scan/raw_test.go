package scan

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteReadRawRoundtrip covers raw scanner output persistence under
// <dataDir>/.skillgrid/cache/scans/<id>.json (0644) and byte-identical read
// back.
//
// SATISFIES: `happy path trivy scan ingests findings with stable hash` (raw leg)
func TestWriteReadRawRoundtrip(t *testing.T) {
	dataDir := t.TempDir()
	id := "test-scan-id-01"
	raw := []byte(`{"Results":[]}`)
	p, err := WriteRaw(dataDir, id, raw)
	if err != nil {
		t.Fatalf("WriteRaw: %v", err)
	}
	want := filepath.Join(dataDir, ".skillgrid", "cache", "scans", id+".json")
	if p != want {
		t.Fatalf("raw path = %q, want %q", p, want)
	}
	if !strings.Contains(p, ".skillgrid/cache/scans/"+id+".json") {
		t.Fatalf("raw path %q does not contain .skillgrid/cache/scans/<id>.json", p)
	}
	got, err := ReadRaw(dataDir, id)
	if err != nil {
		t.Fatalf("ReadRaw: %v", err)
	}
	if string(got) != string(raw) {
		t.Fatalf("roundtrip = %q, want %q", got, raw)
	}
}
