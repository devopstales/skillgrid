package scan

import (
	"context"
	"database/sql"
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

// TestStoreFindingsMidUpsertFailureMarksPartial covers the red-team finding: a
// failure on a finding upsert after the raw parsed cleanly must mark the scan
// row partial (not leave it a phantom status='ok' with finding_count=0).
func TestStoreFindingsMidUpsertFailureMarksPartial(t *testing.T) {
	svc := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		return fixtureBytes(t, "trivy.json"), nil
	})
	scan, err := svc.Start(context.Background(), "trivy", ".")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Force the finding upsert to fail deterministically.
	svc.dbExec = func(_ context.Context, _ string, _ ...any) (sql.Result, error) {
		return nil, errors.New("disk full: cannot upsert finding")
	}
	if _, err := svc.StoreFindings(context.Background(), scan.ID); err == nil {
		t.Fatalf("StoreFindings = nil error, want the upsert error")
	}
	status, errText, count := scanRow(t, svc, scan.ID)
	if status != "partial" {
		t.Errorf("scan status = %q, want partial (a mid-upsert failure must not leave a phantom ok row)", status)
	}
	if !strings.Contains(errText, "cannot upsert finding") {
		t.Errorf("error = %q, want it to carry the failure reason", errText)
	}
	if count != 0 {
		t.Errorf("finding_count = %d, want 0 (no finding was stored)", count)
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

// TestUnchangedRescanDiffsToZeroAdded seeds two scans with identical trivy
// fixture output and asserts the diff is zero added / zero removed.
//
// SATISFIES: `happy path unchanged re-scan diffs to zero added`
func TestUnchangedRescanDiffsToZeroAdded(t *testing.T) {
	raw := fixtureBytes(t, "trivy.json")
	svc := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		return raw, nil
	})
	ctx := context.Background()

	scanA, err := svc.Start(ctx, "trivy", ".")
	if err != nil {
		t.Fatalf("Start A: %v", err)
	}
	if _, err := svc.StoreFindings(ctx, scanA.ID); err != nil {
		t.Fatalf("StoreFindings A: %v", err)
	}

	scanB, err := svc.Start(ctx, "trivy", ".")
	if err != nil {
		t.Fatalf("Start B: %v", err)
	}
	if _, err := svc.StoreFindings(ctx, scanB.ID); err != nil {
		t.Fatalf("StoreFindings B: %v", err)
	}

	diff, err := svc.Diff(ctx, scanA.ID, scanB.ID)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(diff.Added) != 0 {
		t.Errorf("added = %v, want 0", diff.Added)
	}
	if len(diff.Removed) != 0 {
		t.Errorf("removed = %v, want 0", diff.Removed)
	}
	if len(diff.Unchanged) != 1 {
		t.Errorf("unchanged = %d, want 1", len(diff.Unchanged))
	}
}

// TestFixedCveAppearsAsRemoved seeds scan A with the trivy fixture (1 CVE) and
// scan B with a different trivy report (the CVE is gone). The diff must report
// the CVE's dedup_hash in Removed, and scan A's findings row count must be
// unchanged (audit survival).
//
// SATISFIES: `happy path fixed cve appears as removed in diff`
func TestFixedCveAppearsAsRemoved(t *testing.T) {
	originalRaw := fixtureBytes(t, "trivy.json")
	// A trivy report with no vulnerabilities: the CVE is "fixed".
	fixedRaw := []byte(`{"Results":[{"Target":"requirements.txt","Vulnerabilities":[]}]}`)

	var calls int
	svc := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		calls++
		if calls == 1 {
			return originalRaw, nil
		}
		return fixedRaw, nil
	})
	ctx := context.Background()

	scanA, err := svc.Start(ctx, "trivy", ".")
	if err != nil {
		t.Fatalf("Start A: %v", err)
	}
	if _, err := svc.StoreFindings(ctx, scanA.ID); err != nil {
		t.Fatalf("StoreFindings A: %v", err)
	}
	_, _, countBefore := scanRow(t, svc, scanA.ID)

	scanB, err := svc.Start(ctx, "trivy", ".")
	if err != nil {
		t.Fatalf("Start B: %v", err)
	}
	if _, err := svc.StoreFindings(ctx, scanB.ID); err != nil {
		t.Fatalf("StoreFindings B: %v", err)
	}

	diff, err := svc.Diff(ctx, scanA.ID, scanB.ID)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	wantHash := DedupHash("trivy", "CVE-2024-1234", "", "flask", "2.0.1", "requirements.txt", 0)
	found := false
	for _, h := range diff.Removed {
		if h == wantHash {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("removed = %v, want it to contain %s", diff.Removed, wantHash)
	}

	// Audit survival: scan A's findings row count is unchanged.
	_, _, countAfter := scanRow(t, svc, scanA.ID)
	if countAfter != countBefore {
		t.Errorf("audit: scan A findings count = %d, want %d (unchanged)", countAfter, countBefore)
	}
}

// TestLatestDiffFewerThanTwoScans returns a clear error when fewer than two
// scans exist.
func TestLatestDiffFewerThanTwoScans(t *testing.T) {
	raw := fixtureBytes(t, "trivy.json")
	svc := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		return raw, nil
	})
	ctx := context.Background()

	if _, err := svc.Start(ctx, "trivy", "."); err != nil {
		t.Fatalf("Start: %v", err)
	}

	_, _, _, err := svc.LatestDiff(ctx)
	if err == nil {
		t.Fatal("LatestDiff with one scan = nil error, want an error")
	}
}

// TestStatusAggregatesByTool seeds a trivy scan with one HIGH finding and a
// wapiti scan with one finding, then asserts per-tool severity counts and
// latest started_at.
func TestStatusAggregatesByTool(t *testing.T) {
	trivyRaw := fixtureBytes(t, "trivy.json")
	wapitiRaw := []byte(`{"report":[{"type":"XSS","info":"reflected","url":"https://example.com"}]}`)

	var calls int
	svc := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		calls++
		switch calls {
		case 1:
			return trivyRaw, nil
		case 2:
			return wapitiRaw, nil
		}
		return nil, nil
	})
	ctx := context.Background()

	scanTrivy, err := svc.Start(ctx, "trivy", ".")
	if err != nil {
		t.Fatalf("Start trivy: %v", err)
	}
	if _, err := svc.StoreFindings(ctx, scanTrivy.ID); err != nil {
		t.Fatalf("StoreFindings trivy: %v", err)
	}

	scanWapiti, err := svc.Start(ctx, "wapiti", ".")
	if err != nil {
		t.Fatalf("Start wapiti: %v", err)
	}
	if _, err := svc.StoreFindings(ctx, scanWapiti.ID); err != nil {
		t.Fatalf("StoreFindings wapiti: %v", err)
	}

	status, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	trivyRow, ok := status["trivy"]
	if !ok {
		t.Fatal("status missing trivy")
	}
	if trivyRow.SeverityCounts["HIGH"] != 1 {
		t.Errorf("trivy HIGH = %d, want 1", trivyRow.SeverityCounts["HIGH"])
	}
	if trivyRow.Latest.IsZero() {
		t.Error("trivy latest is zero")
	}

	wapitiRow, ok := status["wapiti"]
	if !ok {
		t.Fatal("status missing wapiti")
	}
	if wapitiRow.Latest.IsZero() {
		t.Error("wapiti latest is zero")
	}
}

// TestListFiltersByToolAndStatus seeds trivy (ok) and wapiti (error) scans,
// then asserts List filters by tool, by status, and respects limit.
func TestListFiltersByToolAndStatus(t *testing.T) {
	raw := fixtureBytes(t, "trivy.json")
	var calls int
	svc := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		calls++
		return raw, nil
	})
	ctx := context.Background()

	scanTrivy, err := svc.Start(ctx, "trivy", ".")
	if err != nil {
		t.Fatalf("Start trivy: %v", err)
	}
	if _, err := svc.StoreFindings(ctx, scanTrivy.ID); err != nil {
		t.Fatalf("StoreFindings trivy: %v", err)
	}

	// A failing wapiti scan.
	svc2 := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		return nil, errors.New("wapiti failed")
	})
	// Use the same store for both scans.
	svc2.store = svc.store
	scanWapiti, err := svc2.Start(ctx, "wapiti", ".")
	if err != nil {
		t.Fatalf("Start wapiti: %v", err)
	}

	all, err := svc.List(ctx, ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("List all = %d, want 2", len(all))
	}

	byTool, err := svc.List(ctx, ListFilter{Tool: "trivy", Limit: 10})
	if err != nil {
		t.Fatalf("List trivy: %v", err)
	}
	if len(byTool) != 1 || byTool[0].ID != scanTrivy.ID {
		t.Errorf("List trivy = %d scans, want 1 matching %s", len(byTool), scanTrivy.ID)
	}

	byStatus, err := svc.List(ctx, ListFilter{Status: "error", Limit: 10})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(byStatus) != 1 || byStatus[0].ID != scanWapiti.ID {
		t.Errorf("List error = %d scans, want 1 matching %s", len(byStatus), scanWapiti.ID)
	}

	limited, err := svc.List(ctx, ListFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List limit 1: %v", err)
	}
	if len(limited) != 1 {
		t.Errorf("List limit 1 = %d, want 1", len(limited))
	}
}

// TestGetReturnsScan retrieves a scan by id and asserts the row; an unknown
// id returns an error.
func TestGetReturnsScan(t *testing.T) {
	raw := fixtureBytes(t, "trivy.json")
	svc := newTestService(t, func(ctx context.Context, bin string, args ...string) ([]byte, error) {
		return raw, nil
	})
	ctx := context.Background()

	scan, err := svc.Start(ctx, "trivy", ".")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	got, err := svc.Get(ctx, scan.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != scan.ID {
		t.Errorf("Get id = %s, want %s", got.ID, scan.ID)
	}
	if got.Tool != "trivy" {
		t.Errorf("Get tool = %s, want trivy", got.Tool)
	}

	if _, err := svc.Get(ctx, "no-such-scan"); err == nil {
		t.Error("Get(unknown) = nil error, want an error")
	}
}
