package report

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

// semgrepFixture / jscpdFixture are the SAST/duplication outputs the full-
// static-set test feeds through the real scan pipeline (same fixtures the
// scan package's parse tests use, inlined because that helper is unexported).
var semgrepFixture = []byte(`{"results":[{"check_id":"python.lang.security.eval-used",
  "extra":{"severity":"ERROR","message":"Use of eval() detected"},
  "path":"app/auth.py","start":{"line":12}}]}`)
var jscpdFixture = []byte(`{"duplication":{"clones":[
  {"offsets":[{"name":"src/a.ts","start":{"line":10}},{"name":"src/b.ts","start":{"line":5}}]}]}}`)

// coverProfileFixture is a minimal go cover profile the coverage source runs
// through in the static-set test (two statements, one covered → 50%).
var coverProfileFixture = []byte("mode: set\n./a.go:1.1,3.2 1 1\n./a.go:5.1,6.2 0 1\n")

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

// fullStaticRunner is the collect runner for the full-static-set test: it
// returns the fixture for each static bin and a stub error for anything else.
func fullStaticRunner() scan.RunnerFunc {
	runErr := errors.New("runner stub: not wired")
	return func(_ context.Context, bin string, _ ...string) ([]byte, error) {
		switch bin {
		case "trivy":
			return trivyFixture, nil
		case "semgrep":
			return semgrepFixture, nil
		case "jscpd":
			return jscpdFixture, nil
		case "go":
			return coverProfileFixture, nil
		}
		return nil, runErr
	}
}

// TestCollect_FullStaticSet is the acceptance gate for the static set: all
// five sources (trivy, semgrep, jscpd, debt, coverage) are present, the
// scanner-backed ones carry their normalized findings, debt markers are
// recorded, and coverage rows are both returned and stored. No dast.url → no
// dynamic section.
//
// SATISFIES: happy path report-aggregates-static-sources
// SATISFIES: happy path sast-semgrep-report
// SATISFIES: happy path duplication-jscpd-report
func TestCollect_FullStaticSet(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n// TODO: fix later\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := New(st, root, t.TempDir(), Config{})
	svc.run = fullStaticRunner()

	rep, err := svc.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if rep.Dynamic != nil {
		t.Errorf("Dynamic = %v, want nil (no dast.url)", rep.Dynamic)
	}
	got := map[string]SourceResult{}
	for _, sr := range rep.Static {
		got[sr.Tool] = sr
	}
	want := []string{"trivy", "semgrep", "jscpd", "debt", "coverage"}
	for _, tool := range want {
		sr, ok := got[tool]
		if !ok {
			t.Errorf("static set missing %s; have %v", tool, got)
			continue
		}
		if sr.Status != "ok" {
			t.Errorf("%s status = %q, want ok", tool, sr.Status)
		}
	}
	if len(rep.Static) != len(want) {
		t.Errorf("Static = %d sources, want %d: %+v", len(rep.Static), len(want), rep.Static)
	}

	// semgrep carries the rule id + file + line.
	if sg := got["semgrep"]; sg.Status == "ok" {
		if len(sg.Findings) != 1 || sg.Findings[0].RuleID != "python.lang.security.eval-used" ||
			sg.Findings[0].File != "app/auth.py" || sg.Findings[0].Line != 12 {
			t.Errorf("semgrep findings = %+v, want the rule finding at app/auth.py:12", sg.Findings)
		}
	}
	// jscpd carries the clone's file + line with rule_id duplication.
	if js := got["jscpd"]; js.Status == "ok" {
		if len(js.Findings) != 1 || js.Findings[0].RuleID != "duplication" ||
			js.Findings[0].File != "src/a.ts" || js.Findings[0].Line != 10 {
			t.Errorf("jscpd findings = %+v, want the clone finding at src/a.ts:10", js.Findings)
		}
	}
	// debt records the marker with file + line.
	if db := got["debt"]; db.Status == "ok" {
		if len(db.Findings) != 1 || db.Findings[0].RuleID != "debt:TODO" ||
			db.Findings[0].File != "a.go" || db.Findings[0].Line != 2 {
			t.Errorf("debt findings = %+v, want debt:TODO at a.go:2", db.Findings)
		}
	}
	// coverage rows are returned and upserted into the coverage table.
	if len(rep.Coverage) != 1 || rep.Coverage[0].File != "./a.go" || rep.Coverage[0].PctLine != 50 {
		t.Errorf("Coverage = %+v, want ./a.go at 50", rep.Coverage)
	}
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM coverage WHERE file='./a.go'`).Scan(&n); err != nil {
		t.Fatalf("coverage count: %v", err)
	}
	if n != 1 {
		t.Errorf("coverage table rows for ./a.go = %d, want 1", n)
	}
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
	sr, ok := sourceByTool(rep, "trivy")
	if !ok {
		t.Fatalf("trivy source missing from %+v", rep.Static)
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
// propagated to the caller; the other sources still emit.
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
	got := map[string]SourceResult{}
	for _, sr := range rep.Static {
		got[sr.Tool] = sr
	}
	sr, ok := got["trivy"]
	if !ok {
		t.Fatalf("trivy source missing from %+v", rep.Static)
	}
	if sr.Status != "error" {
		t.Errorf("status = %q, want error", sr.Status)
	}
	if len(sr.Findings) != 0 {
		t.Errorf("findings = %d, want 0 for an error source", len(sr.Findings))
	}
	// The other sources still emit (debt always does; semgrep/jscpd fail
	// open to error with the stub runner).
	if _, ok := got["semgrep"]; !ok {
		t.Error("semgrep source missing (fail-open requires presence)")
	}
	if _, ok := got["jscpd"]; !ok {
		t.Error("jscpd source missing (fail-open requires presence)")
	}
	if db, ok := got["debt"]; !ok || db.Status != "ok" {
		t.Errorf("debt source = %+v, want ok", got["debt"])
	}
	if _, ok := got["coverage"]; !ok {
		t.Error("coverage source missing (fail-open requires presence)")
	}
}

// TestCollect_CoverageToolMissing_Unavailable: the detected kind's coverage
// tool is absent from PATH → the coverage source is present with Status
// "unavailable" and no rows; no error is propagated and the other static
// sources still emit.
//
// SATISFIES: edge coverage-tool-missing-unavailable
func TestCollect_CoverageToolMissing_Unavailable(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := New(st, root, t.TempDir(), Config{})
	svc.run = func(_ context.Context, bin string, _ ...string) ([]byte, error) {
		if bin == "trivy" {
			return trivyFixture, nil
		}
		return nil, errors.New("runner stub: not wired")
	}

	rep, err := svc.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect must be fail-open, got %v", err)
	}
	sr, ok := sourceByTool(rep, "coverage")
	if !ok {
		t.Fatalf("coverage source missing from %+v", rep.Static)
	}
	if sr.Status != "unavailable" {
		t.Errorf("coverage status = %q, want unavailable (cargo-llvm-cov is not on PATH in the test env)", sr.Status)
	}
	if len(rep.Coverage) != 0 {
		t.Errorf("Coverage rows = %d, want 0 for an unavailable tool", len(rep.Coverage))
	}
	// The other static sources are still present.
	got := map[string]SourceResult{}
	for _, s := range rep.Static {
		got[s.Tool] = s
	}
	for _, tool := range []string{"trivy", "semgrep", "jscpd", "debt"} {
		if _, ok := got[tool]; !ok {
			t.Errorf("static source %s missing although coverage was unavailable", tool)
		}
	}
}

// sourceByTool looks up one static source in a collected report.
func sourceByTool(rep *Report, tool string) (SourceResult, bool) {
	for _, sr := range rep.Static {
		if sr.Tool == tool {
			return sr, true
		}
	}
	return SourceResult{}, false
}
