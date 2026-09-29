package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixture writes content to dir/name and returns the full path.
func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// ---------- ParseAssumptions ----------

const assumptionsFixture = `# ASSUMPTIONS

Intro paragraph.

### In-force set

| # | Title | Status | Supersedes | Amends | Date | In force |
|---|-------|--------|------------|--------|------|----------|
| 0001 | First ADR | accepted | — | — | 2026-09-01 | yes |
| 0002 | Second ADR | superseded | — | — | 2026-09-02 | no |
| 0003 | Third ADR | accepted | 0001 | — | 2026-09-03 | yes |
| 0004 | Fourth ADR | accepted | — | 0002 | 2026-09-04 | yes |

### ADR-0001 — First ADR

**Decision.** Some decision.
**Why.** Some reason.

### ADR-0002 — Second ADR

**Decision.** An old decision.

### ADR-0003 — Third ADR

**Decision.** A newer decision.

### ADR-0004 — Fourth ADR

**Decision.** A refining decision.

### Locked constraints

These are the locked constraints.

- Go 1.22+ minimum to build.
- No new dependencies without an ADR.
- Trivy is advisory-only.
`

func TestParseAssumptionsAdrs(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "ASSUMPTIONS.md", assumptionsFixture)

	concepts, _, err := ParseAssumptions(p)
	if err != nil {
		t.Fatalf("ParseAssumptions: %v", err)
	}

	byID := map[string]Concept{}
	for _, c := range concepts {
		byID[c.ID] = c
	}

	for id := range map[string]bool{"adr-0001": true, "adr-0002": true, "adr-0003": true, "adr-0004": true} {
		c, ok := byID[id]
		if !ok {
			t.Fatalf("missing concept %q; have %v", id, conceptIDs(concepts))
		}
		if c.Type != "ADR" {
			t.Errorf("%s: Type = %q, want ADR", id, c.Type)
		}
		if c.SourcePath != p {
			t.Errorf("%s: SourcePath = %q, want %q", id, c.SourcePath, p)
		}
	}

	// R4.1/4.2: superseded → deprecated, in-force → stable.
	if got := byID["adr-0002"].Status; got != "deprecated" {
		t.Errorf("adr-0002 Status = %q, want deprecated", got)
	}
	for _, id := range []string{"adr-0001", "adr-0003", "adr-0004"} {
		if got := byID[id].Status; got != "stable" {
			t.Errorf("%s Status = %q, want stable", id, got)
		}
	}

	// Titles come from the ADR-NNNN headings.
	if got := byID["adr-0003"].Title; got != "Third ADR" {
		t.Errorf("adr-0003 Title = %q, want %q", got, "Third ADR")
	}

	// ADR body includes the full text.
	if !strings.Contains(byID["adr-0001"].Body, "Some decision.") {
		t.Errorf("adr-0001 Body missing decision text: %q", byID["adr-0001"].Body)
	}
}

func TestParseAssumptionsEdges(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "ASSUMPTIONS.md", assumptionsFixture)

	_, edges, err := ParseAssumptions(p)
	if err != nil {
		t.Fatalf("ParseAssumptions: %v", err)
	}

	want := map[string]bool{
		"adr-0003→adr-0001:supersedes": true,
		"adr-0004→adr-0002:amends":     true,
	}
	for _, e := range edges {
		key := e.From + "→" + e.To + ":" + e.Kind
		if !want[key] {
			t.Errorf("unexpected edge %s", key)
		}
		if e.Confidence != ConfidenceExtracted {
			t.Errorf("edge %s: Confidence = %q, want EXTRACTED", key, e.Confidence)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Errorf("missing edges: %v", want)
	}
}

func TestParseAssumptionsWikilinks(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "ASSUMPTIONS.md", `# ASSUMPTIONS

### In-force set

| # | Title | Status | Supersedes | Amends | Date | In force |
|---|-------|--------|------------|--------|------|----------|
| 0004 | Old ADR | superseded | — | — | 2026-09-01 | no |
| 0009 | New ADR | accepted | 0004 | — | 2026-09-02 | yes |

### ADR-0004 — Old ADR

**Decision.** An old decision.

### ADR-0009 — New ADR

**Decision.** A newer decision.
`)

	concepts, _, err := ParseAssumptions(p)
	if err != nil {
		t.Fatalf("ParseAssumptions: %v", err)
	}
	byID := map[string]Concept{}
	for _, c := range concepts {
		byID[c.ID] = c
	}

	// R4.1: the superseder's body links the superseded ADR.
	if c := byID["adr-0009"]; !strings.Contains(c.Body, "[[adr-0004]]") {
		t.Errorf("adr-0009 Body missing [[adr-0004]]: %q", c.Body)
	}
	// R4.2: the superseded ADR's body links the superseder.
	if c := byID["adr-0004"]; !strings.Contains(c.Body, "[[adr-0009]]") {
		t.Errorf("adr-0004 Body missing [[adr-0009]]: %q", c.Body)
	}
}

func TestParseAssumptionsWikilinkIdempotent(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "ASSUMPTIONS.md", `# ASSUMPTIONS

### In-force set

| # | Title | Status | Supersedes | Amends | Date | In force |
|---|-------|--------|------------|--------|------|----------|
| 0002 | Amended ADR | accepted | — | — | 2026-09-01 | yes |
| 0003 | Amending ADR | accepted | — | 0002 | 2026-09-02 | yes |

### ADR-0002 — Amended ADR

**Decision.** An old decision. See [[adr-0003]].

### ADR-0003 — Amending ADR

**Decision.** A refining decision.
`)

	concepts, _, err := ParseAssumptions(p)
	if err != nil {
		t.Fatalf("ParseAssumptions: %v", err)
	}
	byID := map[string]Concept{}
	for _, c := range concepts {
		byID[c.ID] = c
	}
	// The pre-existing [[adr-0003]] in adr-0002's body must not be duplicated.
	if n := strings.Count(byID["adr-0002"].Body, "[[adr-0003]]"); n != 1 {
		t.Errorf("adr-0002 Body has %d occurrences of [[adr-0003]], want 1: %q", n, byID["adr-0002"].Body)
	}
	// The amender's body gets the amended ADR's wikilink.
	if !strings.Contains(byID["adr-0003"].Body, "[[adr-0002]]") {
		t.Errorf("adr-0003 Body missing [[adr-0002]]: %q", byID["adr-0003"].Body)
	}
}

func TestParseAssumptionsConstraints(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "ASSUMPTIONS.md", assumptionsFixture)

	concepts, _, err := ParseAssumptions(p)
	if err != nil {
		t.Fatalf("ParseAssumptions: %v", err)
	}

	var constraints []Concept
	for _, c := range concepts {
		if c.Type == "Constraint" {
			constraints = append(constraints, c)
		}
	}
	// R4.3: 3 constraints.
	if len(constraints) != 3 {
		t.Fatalf("Constraint count = %d, want 3: %v", len(constraints), conceptIDs(concepts))
	}
	for i, c := range constraints {
		if c.Status != "stable" {
			t.Errorf("constraint[%d] Status = %q, want stable", i, c.Status)
		}
		if !c.StaleAfter.IsZero() {
			t.Errorf("constraint[%d] StaleAfter = %v, want zero", i, c.StaleAfter)
		}
		if TypeDir(c.Type) != "concepts" {
			t.Errorf("constraint[%d] TypeDir(%q) = %q, want concepts", i, c.Type, TypeDir(c.Type))
		}
	}
	if got := constraints[0].Title; got != "Go 1.22+ minimum to build." {
		t.Errorf("constraint[0] Title = %q", got)
	}
}

func TestParseAssumptionsMissingSection(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "ASSUMPTIONS.md", "# ASSUMPTIONS\n\nJust a title and an intro.\n")

	concepts, edges, err := ParseAssumptions(p)
	if err != nil {
		t.Fatalf("ParseAssumptions missing sections: %v", err)
	}
	if len(concepts) != 0 || len(edges) != 0 {
		t.Errorf("concepts=%d edges=%d, want 0/0 for file without sections", len(concepts), len(edges))
	}
}

func TestParseAssumptionsMissingFile(t *testing.T) {
	concepts, edges, err := ParseAssumptions(filepath.Join(t.TempDir(), "nope.md"))
	if err != nil {
		t.Fatalf("ParseAssumptions missing file: %v", err)
	}
	if len(concepts) != 0 || len(edges) != 0 {
		t.Errorf("concepts=%d edges=%d, want 0/0 for missing file", len(concepts), len(edges))
	}
}

func TestParseAssumptionsRealRepo(t *testing.T) {
	p := filepath.Join("..", "..", "..", "..", ".skillgrid", "ASSUMPTIONS.md")
	concepts, edges, err := ParseAssumptions(p)
	if err != nil {
		t.Fatalf("ParseAssumptions real repo: %v", err)
	}

	var adrs, constraints []Concept
	for _, c := range concepts {
		switch c.Type {
		case "ADR":
			adrs = append(adrs, c)
		case "Constraint":
			constraints = append(constraints, c)
		}
	}
	// The real repo holds 15 ADR entries and 11 locked constraints.
	if len(adrs) != 15 {
		t.Errorf("ADR count = %d, want 15", len(adrs))
	}
	if len(constraints) != 11 {
		t.Errorf("Constraint count = %d, want 11", len(constraints))
	}

	byID := map[string]Concept{}
	for _, c := range concepts {
		byID[c.ID] = c
	}
	// ADR-0009 amends ADR-0006.
	found := false
	for _, e := range edges {
		if e.From == "adr-0009" && e.To == "adr-0006" && e.Kind == "amends" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing edge adr-0009 → adr-0006 (amends)")
	}
	if got := byID["adr-0014"].Status; got != "stable" {
		t.Errorf("adr-0014 Status = %q, want stable", got)
	}
}

// ---------- ParseState ----------

func TestParseStateHappyPath(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "state.yaml", `schema: skillgrid/state/v1
pipeline:
  current_phase: "slicing"
  current_change: "2026-09-29-llmwiki-okf-wiki"
  status: in_progress
progress:
  completed_changes: 17
  blocked_changes: 0
constraints_ref: .skillgrid/ASSUMPTIONS.md
`)

	c, warnings, err := ParseState(p)
	if err != nil {
		t.Fatalf("ParseState: %v", err)
	}
	if c == nil {
		t.Fatal("ParseState returned nil concept")
	}
	if c.Type != "State" {
		t.Errorf("Type = %q, want State", c.Type)
	}
	if c.Status != "draft" {
		t.Errorf("Status = %q, want draft", c.Status)
	}
	if c.SourcePath != p {
		t.Errorf("SourcePath = %q, want %q", c.SourcePath, p)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
	for _, want := range []string{"slicing", "2026-09-29-llmwiki-okf-wiki", "in_progress"} {
		if !strings.Contains(c.Body, want) {
			t.Errorf("Body missing %q: %q", want, c.Body)
		}
	}
	if !strings.Contains(c.Body, "completed_changes: 17") {
		t.Errorf("Body missing progress: %q", c.Body)
	}
}

func TestParseStateUnknownKeyWarning(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "state.yaml", `pipeline:
  current_phase: "executing"
  current_change: "x"
  status: in_progress
bogus_key: something
`)

	_, warnings, err := ParseState(p)
	if err != nil {
		t.Fatalf("ParseState: %v", err)
	}
	found := false
	for _, w := range warnings {
		if w.Key == "bogus_key" {
			found = true
		}
	}
	if !found {
		t.Errorf("warnings = %v, want one for bogus_key", warnings)
	}
}

func TestParseStateMissingFile(t *testing.T) {
	c, _, err := ParseState(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("ParseState missing file: %v", err)
	}
	if c != nil {
		t.Errorf("concept = %+v, want nil", c)
	}
}

// ---------- ParseTerms ----------

func TestParseTerms(t *testing.T) {
	dir := t.TempDir()
	biz := writeFixture(t, dir, "01-business-terms.md", `# Glossary — Business

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| Step | A sequenced unit. See [[Change]]. | Writing change.md. | "phase". |
| Change | A self-contained SDD unit. | Referencing SDD changes. | "ticket". |
`)
	tech := writeFixture(t, dir, "02-technical-terms.md", `# Glossary — Technical

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| Module | Anything with an interface. | Architecture Decisions. | "unit". |
`)

	concepts := ParseTerms(biz, tech)
	if len(concepts) != 3 {
		t.Fatalf("concept count = %d, want 3", len(concepts))
	}
	byID := map[string]Concept{}
	for _, c := range concepts {
		byID[c.ID] = c
	}
	c, ok := byID["step"]
	if !ok {
		t.Fatalf("missing term 'step'; have %v", conceptIDs(concepts))
	}
	if c.Type != "Term" {
		t.Errorf("step Type = %q, want Term", c.Type)
	}
	if c.SourcePath != biz {
		t.Errorf("step SourcePath = %q, want %q", c.SourcePath, biz)
	}
	if !strings.Contains(c.Body, "A sequenced unit") {
		t.Errorf("step Body missing definition: %q", c.Body)
	}
	if !strings.Contains(c.Body, "Change") {
		t.Errorf("step Body should contain the [[Change]] cross-ref: %q", c.Body)
	}
}

func TestParseTermsCrossRefEdges(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "01-business-terms.md", `| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| Step | A sequenced unit. See [[Change]]. | x | y |
| Change | A SDD unit. | x | y |
`)

	concepts := ParseTerms(p)
	if len(concepts) != 2 {
		t.Fatalf("concept count = %d, want 2", len(concepts))
	}
	// The [[Change]] cross-ref stays in the body as an Obsidian link.
	var step Concept
	for _, c := range concepts {
		if c.ID == "step" {
			step = c
		}
	}
	if !strings.Contains(step.Body, "[[Change]]") {
		t.Errorf("step Body missing [[Change]] wikilink: %q", step.Body)
	}
}

func TestParseTermsMissingFile(t *testing.T) {
	concepts := ParseTerms(filepath.Join(t.TempDir(), "nope.md"))
	if len(concepts) != 0 {
		t.Errorf("concepts = %d, want 0 for missing file", len(concepts))
	}
}

// ---------- ParseSpecs ----------

func TestParseSpecs(t *testing.T) {
	dir := t.TempDir()
	wikiBriefing := "# Change: llmwiki-okf-wiki — Dual-Compat OKF Compiler\n\n> **STATUS:** " + "`draft`" + " (2026-09-29)\n\n**Goal:** Add a wiki capability.\n"
	writeFixture(t, dir, "2026-09-29-llmwiki-okf-wiki/briefing.md", wikiBriefing)
	writeFixture(t, dir, "2026-09-21-session-events-layer/briefing.md", `# Session/Checkpoint Layer Consolidation

**Goal:** Consolidate session events.
`)
	// A dir without briefing.md.
	writeFixture(t, dir, "2026-09-22-skill-workflow-review/tasks.md", "tasks\n")
	// A non-dir file is ignored.
	writeFixture(t, dir, "stray.txt", "x\n")

	concepts := ParseSpecs(dir)
	byDir := map[string]Concept{}
	for _, c := range concepts {
		byDir[filepath.Base(c.SourcePath)] = c
	}
	if len(concepts) != 3 {
		t.Fatalf("concept count = %d, want 3 (3 dirs): %v", len(concepts), conceptIDs(concepts))
	}
	c, ok := byDir["2026-09-29-llmwiki-okf-wiki"]
	if !ok {
		t.Fatalf("missing spec 2026-09-29-llmwiki-okf-wiki; have %v", byDir)
	}
	if c.Type != "Spec" {
		t.Errorf("Type = %q, want Spec", c.Type)
	}
	if c.Status != "draft" {
		t.Errorf("Status = %q, want draft", c.Status)
	}
	if !strings.Contains(c.Title, "llmwiki-okf-wiki") {
		t.Errorf("Title = %q, want to contain llmwiki-okf-wiki", c.Title)
	}
	if !strings.Contains(c.Body, "Add a wiki capability") {
		t.Errorf("Body missing goal: %q", c.Body)
	}
	if !strings.Contains(c.SourcePath, "2026-09-29-llmwiki-okf-wiki") {
		t.Errorf("SourcePath = %q, want the change dir", c.SourcePath)
	}
	// The dir without a briefing still yields a spec concept (title from dir name).
	if _, ok := byDir["2026-09-22-skill-workflow-review"]; !ok {
		t.Errorf("missing spec for dir without briefing.md; have %v", byDir)
	}
}

func TestParseSpecsEmptyDir(t *testing.T) {
	concepts := ParseSpecs(t.TempDir())
	if len(concepts) != 0 {
		t.Errorf("concepts = %d, want 0 for empty dir", len(concepts))
	}
}

func TestParseSpecsRealRepo(t *testing.T) {
	concepts := ParseSpecs(filepath.Join("..", "..", "..", "..", ".skillgrid", "specs"))
	if len(concepts) != 11 {
		t.Errorf("spec count = %d, want 11", len(concepts))
	}
	var statuses []string
	for _, c := range concepts {
		statuses = append(statuses, c.Status)
	}
	for _, s := range []string{"draft", "revised", ""} {
		_ = s
	}
	// At least one draft and one revised briefing must be parsed.
	joined := strings.Join(statuses, ",")
	if !strings.Contains(joined, "draft") || !strings.Contains(joined, "revised") {
		t.Errorf("statuses = %v, want draft and revised present", statuses)
	}
}

// ---------- ParseSpikes ----------

func TestParseSpikes(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "001-cgo-free-vector-db/spike.md", `# Spike: 001-cgo-free-vector-db

## Verdict

**PARTIAL** ⚠ — with constraints.
`)
	// A dir without a verdict.
	writeFixture(t, dir, "002-other/findings.md", "no verdict here\n")
	// A non-dir file is ignored.
	writeFixture(t, dir, "stray.md", "x\n")

	concepts := ParseSpikes(dir)
	byDir := map[string]Concept{}
	for _, c := range concepts {
		byDir[filepath.Base(c.SourcePath)] = c
	}
	if len(concepts) != 2 {
		t.Fatalf("concept count = %d, want 2: %v", len(concepts), conceptIDs(concepts))
	}
	if got := byDir["001-cgo-free-vector-db"].Status; got != "PARTIAL" {
		t.Errorf("spike 001 Status = %q, want PARTIAL", got)
	}
	if got := byDir["001-cgo-free-vector-db"].Type; got != "Spike" {
		t.Errorf("Type = %q, want Spike", got)
	}
	if got := byDir["002-other"].Status; got != "" {
		t.Errorf("spike 002 Status = %q, want empty (no verdict)", got)
	}
}

func TestParseSpikesValidated(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "003-x/report.md", "The approach is VALIDATED.\n")
	concepts := ParseSpikes(dir)
	if len(concepts) != 1 {
		t.Fatalf("concept count = %d, want 1", len(concepts))
	}
	if got := concepts[0].Status; got != "VALIDATED" {
		t.Errorf("Status = %q, want VALIDATED", got)
	}
}

func TestParseSpikesEmptyDir(t *testing.T) {
	concepts := ParseSpikes(t.TempDir())
	if len(concepts) != 0 {
		t.Errorf("concepts = %d, want 0 for empty dir", len(concepts))
	}
}

func TestParseSpikesRealRepo(t *testing.T) {
	concepts := ParseSpikes(filepath.Join("..", "..", "..", "..", ".skillgrid", "spikes"))
	if len(concepts) != 1 {
		t.Fatalf("spike count = %d, want 1", len(concepts))
	}
	if got := concepts[0].Status; got != "PARTIAL" {
		t.Errorf("spike 001 Status = %q, want PARTIAL", got)
	}
}

// ---------- ParseArchitecture ----------

func TestParseArchitecture(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "ARCHITECTURE.md", "# ARCHITECTURE\n\nLive structure.\n")

	c := ParseArchitecture(p)
	if c == nil {
		t.Fatal("ParseArchitecture returned nil for existing file")
	}
	if c.Type != "Architecture" {
		t.Errorf("Type = %q, want Architecture", c.Type)
	}
	if c.SourcePath != p {
		t.Errorf("SourcePath = %q, want %q", c.SourcePath, p)
	}
	if !strings.Contains(c.Body, "Live structure") {
		t.Errorf("Body missing content: %q", c.Body)
	}
}

func TestParseArchitectureMissing(t *testing.T) {
	// R4.4: file absent → nil, no error.
	c := ParseArchitecture(filepath.Join(t.TempDir(), "ARCHITECTURE.md"))
	if c != nil {
		t.Errorf("concept = %+v, want nil for missing file", c)
	}
}

// ---------- helpers ----------

func conceptIDs(cs []Concept) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}
