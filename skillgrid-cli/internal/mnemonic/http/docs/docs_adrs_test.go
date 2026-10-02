package docs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// seedADRs writes three ADR files (two in force, one removed) plus an
// ASSUMPTIONS.md index so the empty-index case stays distinct.
func seedADRs(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".skillgrid", "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"04-adr-0002-engine-first.md": `# Engine-first framing

**Decision.** The product is framed engine-first.
**Why.** The code has drifted engine-first.
**Consequences.** Good: consistent vision. Bad: README is stale.
`,
		"04-adr-0001-prd-scope.md": `# PRD scope is the whole Hub Product

**Decision.** One product document covering CLI + engine + surface + content.
**Why.** The binary ships 20+ subcommands.
**Consequences.** Good: matches what the user receives.
`,
		"04-adr-0003-removed.md": `# (removed 2026-09-30)

This ADR was removed on 2026-09-30 by user request. The sequence number is retired.
`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	idx := "# ASSUMPTIONS\n\n## LOCKED\n\n| 0001 | scope | `.skillgrid/artifacts/04-adr-0001-prd-scope.md` |\n"
	if err := os.WriteFile(filepath.Join(root, ".skillgrid", "ASSUMPTIONS.md"), []byte(idx), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// 05.1 [RED] GET /docs/adrs parses ADR blocks out of ASSUMPTIONS.md: id,
// number, title, removed flag, and the Decision/Why/Consequences fields,
// sorted by number, with a correct summary.
func TestStep05_ListADRs(t *testing.T) {
	cwd := seedADRs(t)
	h := NewADRs(cwd)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs/adrs", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Source  string     `json:"source"`
		Summary adrSummary `json:"summary"`
		ADRs    []ADR      `json:"adrs"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Source != ".skillgrid/artifacts" {
		t.Errorf("source = %q", resp.Source)
	}
	// sorted by number ascending: 1, 2, 3.
	if len(resp.ADRs) != 3 {
		t.Fatalf("expected 3 adrs, got %d", len(resp.ADRs))
	}
	gotNums := []int{}
	for _, a := range resp.ADRs {
		gotNums = append(gotNums, a.Number)
	}
	sort.Ints(gotNums)
	for i, want := range []int{1, 2, 3} {
		if resp.ADRs[i].Number != want {
			t.Errorf("adr[%d].Number = %d, want %d (order %v)", i, resp.ADRs[i].Number, want, gotNums)
		}
	}
	// ADR-0001 in force, carries fields, not removed.
	a1 := resp.ADRs[0]
	if a1.ID != "ADR-0001" || a1.Removed {
		t.Errorf("ADR-0001 = %+v", a1)
	}
	if a1.Title != "PRD scope is the whole Hub Product" {
		t.Errorf("ADR-0001 title = %q", a1.Title)
	}
	if a1.Decision == "" || a1.Why == "" || a1.Conseq == "" {
		t.Errorf("ADR-0001 missing fields: %+v", a1)
	}
	if a1.Decision != "One product document covering CLI + engine + surface + content." {
		t.Errorf("ADR-0001 decision = %q", a1.Decision)
	}
	// ADR-0003 removed, no decision field.
	a3 := resp.ADRs[2]
	if !a3.Removed {
		t.Errorf("ADR-0003 should be removed: %+v", a3)
	}
	if a3.Decision != "" {
		t.Errorf("ADR-0003 should have no decision, got %q", a3.Decision)
	}
	// summary: total 3, in_force 2, removed 1, next 4.
	if resp.Summary.Total != 3 || resp.Summary.InForce != 2 || resp.Summary.Removed != 1 || resp.Summary.Next != 4 {
		t.Errorf("summary = %+v", resp.Summary)
	}
}

// 05.2 [RED] missing ASSUMPTIONS.md → 404, empty file → 200 with [].
func TestStep05_ADREdge(t *testing.T) {
	// missing file.
	empty := t.TempDir()
	h := NewADRs(empty)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs/adrs", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("missing file: expected 404, got %d (%s)", rr.Code, rr.Body.String())
	}
	// empty file (no ADR headings).
	if err := os.MkdirAll(filepath.Join(empty, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(empty, ".skillgrid", "ASSUMPTIONS.md"), []byte("# ASSUMPTIONS\n\n## VERIFIED\n\n- fact\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs/adrs", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("no-adrs file: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		ADRs []ADR `json:"adrs"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ADRs == nil || len(resp.ADRs) != 0 {
		t.Errorf("expected empty adrs, got %+v", resp.ADRs)
	}
}
