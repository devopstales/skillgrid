package scan

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func newTestService(t *testing.T, run RunnerFunc) *Service {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "scan-test")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	svc := New(st, dataDir)
	svc.Run = run
	return svc
}

func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("fixtures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func scanRow(t *testing.T, svc *Service, id string) (status, errText string, count int) {
	t.Helper()
	row := svc.store.DB.QueryRow(
		`SELECT status, COALESCE(error,''), finding_count FROM scans WHERE id = ?`, id)
	if err := row.Scan(&status, &errText, &count); err != nil {
		t.Fatalf("scan row %s: %v", id, err)
	}
	return status, errText, count
}

// TestStartTrivyStoresFindings runs the full happy path: Start with a stubbed
// runner emitting the trivy fixture, then StoreFindings; the scans row ends
// ok with finding_count=1 and the finding carries the stable dedup hash.
//
// SATISFIES: `happy path trivy scan ingests findings with stable hash`
func TestStartTrivyStoresFindings(t *testing.T) {
	raw := fixtureBytes(t, "trivy.json")
	svc := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		return raw, nil
	})
	scan, err := svc.Start(context.Background(), "trivy", ".")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := svc.StoreFindings(context.Background(), scan.ID); err != nil {
		t.Fatalf("StoreFindings: %v", err)
	}
	status, _, count := scanRow(t, svc, scan.ID)
	if status != "ok" {
		t.Errorf("scan status = %q, want ok", status)
	}
	if count != 1 {
		t.Fatalf("finding_count = %d, want 1", count)
	}
	wantHash := DedupHash("trivy", "CVE-2024-1234", "", "flask", "2.0.1", "requirements.txt", 0)
	var gotHash string
	err = svc.store.DB.QueryRow(
		`SELECT dedup_hash FROM findings WHERE scan_id = ?`, scan.ID,
	).Scan(&gotHash)
	if err != nil {
		t.Fatalf("findings row: %v", err)
	}
	if gotHash != wantHash {
		t.Fatalf("hash = %s, want %s", gotHash, wantHash)
	}
	rawPath := filepath.Join(svc.dataDir, ".skillgrid", "cache", "scans", scan.ID+".json")
	if _, err := os.Stat(rawPath); err != nil {
		t.Fatalf("raw file missing at %s: %v", rawPath, err)
	}
}

// TestStartFailingStubIsFailOpen covers ADR-0016: a failing scanner records
// status='error' + non-empty error on the scans row and returns no error.
//
// SATISFIES: `error path scanner failure records status=error and is fail-open`
func TestStartFailingStubIsFailOpen(t *testing.T) {
	svc := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		return nil, errors.New("trivy blew up: connection refused")
	})
	scan, err := svc.Start(context.Background(), "trivy", ".")
	if err != nil {
		t.Fatalf("Start must be fail-open, got %v", err)
	}
	status, errText, count := scanRow(t, svc, scan.ID)
	if status != "error" {
		t.Errorf("scan status = %q, want error", status)
	}
	if !strings.Contains(errText, "trivy blew up") {
		t.Errorf("error = %q, want it to contain the runner failure", errText)
	}
	if count != 0 {
		t.Errorf("finding_count = %d, want 0", count)
	}
}

// TestStoreFindingsUnknownScan is a second StoreFindings input (triangulation):
// an unknown scan id is reported, not swallowed.
func TestStoreFindingsUnknownScan(t *testing.T) {
	svc := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		return fixtureBytes(t, "trivy.json"), nil
	})
	if _, err := svc.StoreFindings(context.Background(), "no-such-scan"); err == nil {
		t.Fatal("StoreFindings(unknown) = nil error, want an error")
	}
}
