package report

import (
	"context"
	"errors"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/scan"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// trivyFixture is the trivy report the report package tests run through the
// real scan pipeline (Start + StoreFindings). Inline rather than a cross-
// package fixture read: the scan package's fixtureBytes is unexported.
var trivyFixture = []byte(`{"Results":[{"Target":"go.mod","Vulnerabilities":[
  {"VulnerabilityID":"CVE-2024-0001","PkgName":"example.com/lib","InstalledVersion":"1.0.0",
   "FixedVersion":"1.0.1","Severity":"HIGH","Title":"test vuln","PrimaryURL":"https://x"}]}]}`)

// newTestStore opens a throwaway store for report tests. There is no
// store.OpenForTest; this mirrors scan/service_test.go's newTestService.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), "report-test")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// TestCollect_TrvySource_FailOpen is the trivy tracer: a seeded trivy scan +
// findings must come back through Collect as an ok trivy source carrying the
// normalized finding (cve_id, package, fixed_version).
//
// SATISFIES: happy path cve-trivy-report
// SATISFIES: happy path report-aggregates-static-sources
func TestCollect_TrvySource_FailOpen(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	root := t.TempDir()

	// Seed a trivy scan row + findings via the real scan service with a
	// stubbed runner.
	seed := scan.New(st, t.TempDir())
	seed.Run = func(_ context.Context, bin string, _ ...string) ([]byte, error) {
		if bin != "trivy" {
			t.Fatalf("unexpected bin %s", bin)
		}
		return trivyFixture, nil
	}
	sc, err := seed.Start(ctx, "trivy", root)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := seed.StoreFindings(ctx, sc.ID); err != nil {
		t.Fatalf("StoreFindings: %v", err)
	}

	svc := New(st, root, t.TempDir(), Config{})
	// The stub is the collect runner: it returns the fixture for trivy and an
	// error for any other bin (later sources fail-open to error).
	runErr := errors.New("runner stub: not wired yet")
	svc.run = func(_ context.Context, bin string, _ ...string) ([]byte, error) {
		if bin == "trivy" {
			return trivyFixture, nil
		}
		return nil, runErr
	}
	rep, err := svc.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(rep.Static) != 1 {
		t.Fatalf("Static = %d sources, want 1: %v", len(rep.Static), rep.Static)
	}
	sr := rep.Static[0]
	if sr.Tool != "trivy" {
		t.Errorf("tool = %q, want trivy", sr.Tool)
	}
	if sr.Status != "ok" {
		t.Errorf("status = %q, want ok", sr.Status)
	}
	if len(sr.Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(sr.Findings))
	}
	f := sr.Findings[0]
	if f.CVEID != "CVE-2024-0001" {
		t.Errorf("cve_id = %q, want CVE-2024-0001", f.CVEID)
	}
	if f.Package != "example.com/lib" {
		t.Errorf("package = %q, want example.com/lib", f.Package)
	}
	if f.FixedVersion != "1.0.1" {
		t.Errorf("fixed_version = %q, want 1.0.1", f.FixedVersion)
	}
}

// TestCollect_SourceError_FailOpen is the fail-open shell: a source whose
// scan errors is present with status="error" (ADR-0016) and no error is
// propagated to the caller.
//
// SATISFIES: edge report-source-error-is-fail-open
func TestCollect_SourceError_FailOpen(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	svc := New(st, t.TempDir(), t.TempDir(), Config{})
	svc.run = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, errors.New("trivy: connection refused")
	}
	rep, err := svc.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect must be fail-open, got %v", err)
	}
	if len(rep.Static) != 1 {
		t.Fatalf("Static = %d sources, want 1: %v", len(rep.Static), rep.Static)
	}
	sr := rep.Static[0]
	if sr.Tool != "trivy" {
		t.Errorf("tool = %q, want trivy", sr.Tool)
	}
	if sr.Status != "error" {
		t.Errorf("status = %q, want error", sr.Status)
	}
	if len(sr.Findings) != 0 {
		t.Errorf("findings = %d, want 0 for an error source", len(sr.Findings))
	}
}
