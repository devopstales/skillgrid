package report

import (
	"context"
	"errors"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/scan"
)

// dastDASTURL is the target the DAST tests point dast.url at.
const dastDASTURL = "http://dast.local:8080"

// dast fixtures are the real scanner output shapes (same as the scan package
// fixtures). Inline because the scan package's fixtureBytes is unexported.
var wapitiFixture = []byte(`{"report":[{"type":"SQL Injection","info":"sqli","url":"https://example.com/login"}]}`)
var nucleiFixture = []byte(`{"template-id":"cves/2023/cve-2023-12345.yaml","info":{"severity":"high"},"matcher-name":"cve-2023-12345","host":"https://example.com"}` + "\n")
var akcaFixture = []byte(`{"findings":[{"ruleId":"sqli","severity":"ERROR","url":"/search","file":"/search?x=%27%20OR%201=1","method":"GET"}]}`)

// dastRunner is the collect runner for the DAST tests: it returns the fixture
// for each DAST bin (wapiti3 is the installed wapiti binary), and the static
// fixtures for trivy/semgrep/jscpd so the static sources run ok.
func dastRunner(failBins map[string]bool) scan.RunnerFunc {
	return func(_ context.Context, bin string, _ ...string) ([]byte, error) {
		switch bin {
		case "trivy":
			return trivyFixture, nil
		case "semgrep":
			return semgrepFixture, nil
		case "jscpd":
			return jscpdFixture, nil
		case "wapiti3":
			if failBins["wapiti"] {
				return nil, errors.New("wapiti: exploded")
			}
			return wapitiFixture, nil
		case "nuclei":
			if failBins["nuclei"] {
				return nil, errors.New("nuclei: exploded")
			}
			return nucleiFixture, nil
		case "akca":
			if failBins["akca"] {
				return nil, errors.New("akca: exploded")
			}
			return akcaFixture, nil
		}
		return nil, errors.New("unexpected bin " + bin)
	}
}

// TestCollectDAST_Unset_Nil: an empty dast.url yields no DAST section at all.
//
// SATISFIES: dast-unset-is-absent
func TestCollectDAST_Unset_Nil(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	svc := New(st, t.TempDir(), t.TempDir(), Config{})
	if got := svc.collectDAST(ctx); got != nil {
		t.Fatalf("collectDAST = %v, want nil for an empty DastURL", got)
	}
}

// TestCollectDAST_Unreachable_FailOpen: dast.url set but the probe fails → all
// three DAST tools are present with Status "unavailable" and no scanner runs;
// the report's static sources are still present (fail-open, ADR-0016).
//
// SATISFIES: dast-unreachable-fail-open
func TestCollectDAST_Unreachable_FailOpen(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	svc := New(st, t.TempDir(), t.TempDir(), Config{DastURL: dastDASTURL})
	// A runner that records any call: an unreachable target must not exec any scanner.
	called := map[string]bool{}
	svc.run = func(_ context.Context, bin string, _ ...string) ([]byte, error) {
		called[bin] = true
		return trivyFixture, nil
	}
	// Probe fails: the target is unreachable.
	svc.probe = func(context.Context, string) error { return errors.New("dial tcp: connection refused") }

	rep, err := svc.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect must be fail-open, got %v", err)
	}
	if len(rep.Static) != 5 {
		t.Fatalf("Static = %d sources, want 5 (full static set): %v", len(rep.Static), rep.Static)
	}
	if len(rep.Dynamic) != 3 {
		t.Fatalf("Dynamic = %d sources, want 3: %v", len(rep.Dynamic), rep.Dynamic)
	}
	want := []string{"wapiti", "nuclei", "akca"}
	for i, tool := range want {
		sr := rep.Dynamic[i]
		if sr.Tool != tool {
			t.Errorf("Dynamic[%d].Tool = %q, want %q", i, sr.Tool, tool)
		}
		if sr.Status != "unavailable" {
			t.Errorf("%s status = %q, want unavailable", tool, sr.Status)
		}
		if len(sr.Findings) != 0 {
			t.Errorf("%s findings = %d, want 0 (unavailable)", tool, len(sr.Findings))
		}
	}
	// No DAST scanner may have run although the target was unreachable
	// (static sources — trivy/semgrep/jscpd — still run and are allowed).
	for _, tool := range dastTools {
		if called[tool] || called[scan.ScannerBin(tool)] {
			t.Errorf("DAST scanner %q ran although the target was unreachable", tool)
		}
	}
}

// TestCollectDAST_Reachable_RunsThree: dast.url set and reachable → wapiti,
// nuclei, akca all run through the real scan pipeline and come back as ok
// sources carrying their normalized findings.
//
// SATISFIES: dast-auto-from-config
func TestCollectDAST_Reachable_RunsThree(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	svc := New(st, t.TempDir(), t.TempDir(), Config{DastURL: dastDASTURL})
	svc.run = dastRunner(nil)
	svc.probe = func(context.Context, string) error { return nil }

	rep, err := svc.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(rep.Dynamic) != 3 {
		t.Fatalf("Dynamic = %d sources, want 3: %v", len(rep.Dynamic), rep.Dynamic)
	}
	want := []struct {
		tool    string
		minFind int
	}{
		{"wapiti", 1},
		{"nuclei", 1},
		{"akca", 1},
	}
	for i, w := range want {
		sr := rep.Dynamic[i]
		if sr.Tool != w.tool {
			t.Errorf("Dynamic[%d].Tool = %q, want %q", i, sr.Tool, w.tool)
			continue
		}
		if sr.Status != "ok" {
			t.Errorf("%s status = %q, want ok", w.tool, sr.Status)
			continue
		}
		if len(sr.Findings) < w.minFind {
			t.Errorf("%s findings = %d, want >= %d", w.tool, len(sr.Findings), w.minFind)
		}
		if w.tool == "akca" {
			// dast-auto-from-config: akca findings are parsed with rule id
			// and file.
			f := sr.Findings[0]
			if f.RuleID != "sqli" {
				t.Errorf("akca finding rule id = %q, want sqli", f.RuleID)
			}
			if f.File == "" {
				t.Errorf("akca finding file = %q, want a non-empty file", f.File)
			}
		}
	}
}

// TestCollectDAST_PerToolError_FailOpen: one tool's scan fails while the
// others succeed → that tool is present with Status "error", the others stay
// ok. Fail-open (ADR-0016): one tool's failure never drops the other tools.
//
// SATISFIES: dast-auto-from-config (per-tool failure clause)
func TestCollectDAST_PerToolError_FailOpen(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)

	svc := New(st, t.TempDir(), t.TempDir(), Config{DastURL: dastDASTURL})
	svc.run = dastRunner(map[string]bool{"nuclei": true})
	svc.probe = func(context.Context, string) error { return nil }

	rep, err := svc.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect must be fail-open, got %v", err)
	}
	if len(rep.Dynamic) != 3 {
		t.Fatalf("Dynamic = %d sources, want 3: %v", len(rep.Dynamic), rep.Dynamic)
	}
	statuses := map[string]string{}
	for _, sr := range rep.Dynamic {
		statuses[sr.Tool] = sr.Status
	}
	for tool, want := range map[string]string{"wapiti": "ok", "nuclei": "error", "akca": "ok"} {
		if got := statuses[tool]; got != want {
			t.Errorf("%s status = %q, want %q", tool, got, want)
		}
	}
}
