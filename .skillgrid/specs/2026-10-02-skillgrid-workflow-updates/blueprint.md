---
type: Blueprint
title: "2026-10-02-skillgrid-workflow-updates — blueprint"
status: draft

---

# Skillgrid Release/2 — Tracer-Bullet Slice Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread — the thin path "artifact → validation → verdict" through the Go CLI, the JS helper, and the skill wiring works end-to-end in the first wave; later Release/2 phases thicken the runtime.

**Goal:** Prove the Release/2 tracer bullet: a deterministic `skillgrid validate` command (Go) that verifies a spec folder's phase order and a tested JS validation-matrix script that blocks blueprints whose requirements lack a `## Validation` row.

**Architecture:** Two deterministic validators over existing committed artifacts. (1) Go: new package `skillgrid-cli/internal/validate` — typed artifact schemas for `Briefing` and `ValidationRow` (markdown extraction; `gopkg.in/yaml.v3` remains available for later schema files but is not needed by the tracer bullet), a directional phase-order checker over `.skillgrid/specs/<topic>/`, and a `skillgrid validate <topic>` CLI subcommand (exit 0 pass / 1 findings / 2 usage). (2) JS: `scripts/validation-matrix-check.mjs` — coverage-only checker (same extraction semantics as the Go side) that exits 1 when a locked requirement has no `## Validation` row; wired into the `writing-blueprints` skill as a pre-completion gate. ADR-0046 records the Go runtime home (the existing `skillgrid-cli` module, not a new module, not `mnemonic/`).

**Tech Stack:** Go 1.22+ (module declares 1.26.0), stdlib `regexp`/`bufio` for markdown extraction (`gopkg.in/yaml.v3` already in `go.mod` for later schema files — no new dep either way, per AGENTS.md § Rules), Node.js `.mjs` helpers with fixture tests (existing `scripts/test-*.mjs` pattern, ADR-0008).

**Spec:** `.skillgrid/specs/2026-10-02-skillgrid-workflow-updates/briefing.md` — Release/2 tracer-bullet scope per `plan.md` §3 (the slice components); master vision per `.skillgrid/decisions/skillgrid-workflow-update-plan.md`.

**Findings:** `.skillgrid/specs/2026-10-02-skillgrid-workflow-updates/findings.md` — GSD mechanism borrow (deterministic state + plan-check + Nyquist) and the BMAD script contract (JSON stdout, atomic write, `--dry-run`, tests beside the script, "agent decides *which*; script owns everything after").

## Terms

None beyond the spec's. The Release/2 term list in `briefing.md` §15 (Execution Packet, Verification Receipt, …) is proposed, not locked; this slice uses only plain "artifact", "spec folder", "validation matrix".

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- Running `skillgrid validate <topic>` in a temp project tree with a complete, in-order spec folder (briefing + blueprint + acceptance.feature + tasks) exits 0 and prints a JSON object with `{"status":"pass","findings":[]}`.
- Running `skillgrid validate <topic>` on a tree where `tasks.md` exists but `acceptance.feature` does not (with briefing + blueprint present) exits 1 and the JSON contains a finding naming the missing artifact and the phase order it violates (the cucumber file canonically precedes tasks).
- Running `skillgrid validate <topic>` on a mid-phase tree with `briefing.md` + `blueprint.md` only exits 0 (a valid in-flight state, not a false positive).
- Running `skillgrid validate <topic>` on a nonexistent topic exits 2 with a usage message on stderr.
- Running `node scripts/validation-matrix-check.mjs <spec-dir>` on a temp tree where every locked requirement in `briefing.md` has a row in `blueprint.md`'s `## Validation` table exits 0.
- Running the same script on a tree where a locked requirement (e.g. `#### Req 19`) has no row in the `## Validation` table exits 1 and names the missing requirement ID on stdout.
- The `writing-blueprints` SKILL.md instructs running the validation-matrix script before declaring the blueprint validated, and fails the validation stage when the script exits 1.
- **backstop:** a planner that drops a validation row cannot pass slicing — verified by running both checkers against this change's own spec folder (held-out: a real tree, not a fixture). Negative half (Task 4): the JS coverage checker on the real folder exits 1 naming reqs 19 and 21, before Task 7 fills the matrix. Positive half (Task 7 Step 3, post-slicing): after the `## Validation` rows + `acceptance.feature` + `tasks.md` exist, `skillgrid validate 2026-10-02-skillgrid-workflow-updates` MUST exit 0 — self-validation is a must-have.

**Artifacts** (files that must exist with real implementation, not stubs):
- [`skillgrid-cli/internal/validate/schemas.go` — typed Go structs for Briefing, Validation matrix rows, Result/Finding]
- [`skillgrid-cli/internal/validate/parse.go` — markdown extraction (briefing headings + `## Validation` table)]
- [`skillgrid-cli/internal/validate/root.go` — `FindRoot` project-root discovery (walk-up)]
- [`skillgrid-cli/internal/validate/validate.go` — spec-folder phase-order + validation-row check, `Validate(root, topic string) (Result, error)`]
- [`skillgrid-cli/cmd/skillgrid/validate.go` — the `validate` subcommand wiring (JSON stdout)]
- [`scripts/validation-matrix-check.mjs` — the JS validation-matrix checker (coverage-only, cell split mirrors Go)]
- [`scripts/test-validation-matrix.mjs` — fixture-driven tests for the JS checker + Go/JS contract case (same shape as `scripts/test-state-drift.mjs`)]
- [`scripts/test-skill-wiring.mjs` — grep tests for the SKILL.md binding (committed with the skill edit, never red between tasks)]
- [`.skillgrid/decisions/04-adr-0046-release2-runtime-home.md` — ADR: Go runtime lives in `skillgrid-cli` as `internal/<engine>/` packages (0046, not plan.md's stale 0032)]

**Key links** (critical connections between artifacts that must work together):
- [`validate.Validate` must read the spec folder from the project root (the dir containing `.skillgrid/`), and the CLI must pass the topic through unchanged]
- [The JS checker must use the same requirement-ID extraction and table parsing as the Go checker (heading `#### Req <N> — <title>` in `briefing.md`; `## Validation` rows split identically), pinned by a contract fixture test that feeds the JS checker the EXACT bytes of the Go-pinned `briefFixture` (reqs 19/21/12) and asserts the same covered decision — positive (all rows → exit 0) and negative (drop req 21's row → exit 1 naming exactly "21")]
- [The `writing-blueprints` skill's validation gate must invoke `node scripts/validation-matrix-check.mjs .skillgrid/specs/<topic>` and treat exit 1 as a failed blueprint validation (not a warning)]

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- None. The runtime-home ADR is a decision record, not a migration; no schema is published, no contract is broken. (If the Go runtime later grows beyond `internal/validate`, a superseding ADR moves it — that is the designed exit, not a door.)

## Global Constraints

- Go 1.22+ minimum to build (per AGENTS.md § Rules, ADR-0031). The `skillgrid-cli` module declares `go 1.26.0`; the new package must not raise it.
- No new dependencies without an ADR (per AGENTS.md § Rules). YAML parsing uses the already-required `gopkg.in/yaml.v3` in `skillgrid-cli/go.mod`.
- JS helpers follow the BMAD contract locked in the briefing: structured JSON stdout including errors, atomic writes, `--dry-run` for mutating ops (this checker is read-only: no writes, `--dry-run` N/A), exit 0 pass / 1 findings / 2 usage, tests beside the script, no interactive prompts (per `findings.md`, GSD §mechanism 2 + BMAD borrow).
- Conventional commits only, no AI-attribution trailers (per AGENTS.md § Rules).
- Spec-zone changes commit before code-zone changes (per AGENTS.md § Rules).
- New hooks/checkers fail open: a checker that cannot parse an artifact reports a finding, it never panics the agent (ADR-0021 posture applied to plan-time checks).

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Shared-convention drift (edit to `_shared/` or skill SKILL.md) | Applicable: Task 7 edits `.agents/skills/planning/writing-blueprints/SKILL.md`, binding the skill to the new script path + exit-code contract | One-line impact statement: `writing-blueprints` is now the source of truth for the validation-matrix gate — any skill that references the script by old path breaks. Named skills: none (new script, new path). | Grep check (in `scripts/test-skill-wiring.mjs`, committed with the skill edit so the test is never red between tasks): `writing-blueprints/SKILL.md` contains `validation-matrix-check.mjs` and the exit-1-as-fail contract; link-resolution: the named path exists |
| Git repository selection | N/A: `skillgrid validate` reads the spec folder relative to the project root it is invoked from; no `git -C`, no repo resolution in this slice | — | — |
| Commit state / Push state / PR commands | N/A: the slice neither commits, pushes, nor opens PRs; it only reads spec artifacts | — | — |
| Documentation-like paths | N/A: no executable-file classification; artifacts are read as markdown/YAML text | — | — |
| Mnemonic tool surface (`mem_*` / `code_*` / `web_cache_*`) | N/A: no Mnemonic tool contract changes in this slice (Mnemonic integration is Release/2 Phase 1 item 12, not in the tracer bullet) | — | — |

## File Structure

- `skillgrid-cli/internal/validate/schemas.go` — typed structs: `Briefing` (status, topic, requirement IDs + locked flags), `ValidationRow` (req ID, happy, edge, failure, check, falsifier), `Result` (status, findings), `Finding` (kind, subject, message).
- `skillgrid-cli/internal/validate/parse.go` — markdown extraction: `ExtractBriefing(path string) (Briefing, error)` parses the `> **STATUS:**` banner, the `> **Topic:**` line, and `#### Req N — title` headings (`(locked)` marker in the title suffix); `ExtractValidationRows(path string) ([]ValidationRow, error)` parses the `## Validation` table in `blueprint.md` (columns Req | Happy | Edge | Failure | Check | Falsifier; rows whose first cell is not numeric are skipped).
- `skillgrid-cli/internal/validate/validate.go` — `Validate(root, topic string) (Result, error)`: phase-order check (an artifact implies only the artifacts that canonically precede it, per `_shared/rules/sdd-structure.md` line 13: `briefing → blueprint → acceptance.feature → tasks`) + coverage check (every locked req ID in `briefing.md` must have a row in `blueprint.md`'s `## Validation`); returns findings, never panics on unreadable files (fail open: unreadable → finding).
- `skillgrid-cli/internal/validate/root.go` — `FindRoot(start string) (string, error)`: walk up from `start` until a directory containing `.skillgrid/specs/` is found; error if none (tested in isolation).
- `skillgrid-cli/internal/validate/validate_test.go` — table-driven Go tests over temp trees (complete folder → pass; `briefing + blueprint` mid-phase → **pass**; `briefing + tasks` → two phase-order findings; locked req without row → coverage finding; missing topic → error; unreadable file → parse finding; em-dash heading fixture pinned).
- `skillgrid-cli/cmd/skillgrid/validate.go` — `runValidate(args []string) int`: always JSON `Result` to stdout, one-line human summary to stderr; exit 0 pass / 1 findings / 2 usage or root-not-found. Root discovery via `validate.FindRoot(os.Getwd())`.
- `scripts/validation-matrix-check.mjs` — JS twin of the coverage check (the same two extraction functions in JS, cell split mirroring the Go implementation), read-only, JSON stdout, exit 0/1/2.
- `scripts/test-validation-matrix.mjs` — fixture tests (temp trees: complete matrix → 0; missing row → 1 + named req; usage error → 2; **Go/JS contract fixture**: the same `briefing.md` + `blueprint.md` parsed by both implementations yields identical req-ID sets).
- `scripts/test-skill-wiring.mjs` — grep tests for the `writing-blueprints` SKILL.md binding (script path named + exit-1-as-fail contract); committed in the same task as the skill edit so no task leaves a red test behind.
- `.skillgrid/decisions/04-adr-0046-release2-runtime-home.md` — ADR body (frontmatter `status: accepted`, `date: 2026-10-17`): context (Release/2 needs a Go runtime home; open question in briefing §13; plan.md §5 named this ADR-0032, which is in force for the plugin architecture, so this decision records as 0046), decision (`skillgrid-cli/internal/<engine>` packages in the existing module; no new module; `mnemonic/` untouched), consequences (module boundary unchanged; ADR-0033's three-module workspace is unaffected — it governs the `ctx/` library module, this adds packages inside `skillgrid-cli/`).
- `.agents/skills/planning/writing-blueprints/SKILL.md` — validation section: add the script invocation + exit-code contract to the "Blueprint validation (mandatory, two-stage)" block (modify only that block; no other edits).

---

### Task 1: Go artifact schemas + extraction (TDD)

**Files:**
- Create: `skillgrid-cli/internal/validate/schemas.go`
- Create: `skillgrid-cli/internal/validate/parse.go`
- Test: `skillgrid-cli/internal/validate/parse_test.go`

**Interfaces:**
- Consumes: nothing (leaf package; extraction is stdlib markdown regex + string parsing, per the BMAD contract that parsing helpers own parse/validate)
- Produces: `type Briefing struct { Status, Topic string; ReqIDs, LockedReqs []string }`; `func ExtractBriefing(path string) (Briefing, error)`; `type ValidationRow struct { ReqID string; Happy, Edge, Failure, Check, Falsifier string }`; `func ExtractValidationRows(path string) ([]ValidationRow, error)`
- Seam: the `validate` package API (`Validate` in Task 2 consumes these)
- Deletion test: if deleted, Task 2's checker has no typed input — it re-derives parsing inline
- Adapters: 2 (production markdown files + test fixtures)

**SATISFIES:** validate-extracts-briefing-reqs (acceptance scenario: extractor returns the locked requirement IDs from a real-shaped `briefing.md` fixture)

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/validate/parse_test.go
package validate

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// briefFixture uses the EXACT heading form of the real briefing.md
// (em-dash, "(locked)" in the title suffix) — the coupling is pinned here.
const briefFixture = `# Briefing

> **STATUS:** \`draft\` (2026-10-09)
> **Topic:** 2026-10-02-skillgrid-workflow-updates

#### Req 19 — Technical validation plan (locked)

Some text.

#### Req 21 — Scope pushback (locked)

Some text.

#### Req 12 — Ablate always-loaded rules (named follow-up)

Follow-up text.
`

func TestExtractBriefingLockedReqs(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "briefing.md")
	if err := os.WriteFile(p, []byte(briefFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := ExtractBriefing(p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b.ReqIDs, []string{"19", "21", "12"}) {
		t.Fatalf("ReqIDs = %v, want [19 21 12]", b.ReqIDs)
	}
	if !reflect.DeepEqual(b.LockedReqs, []string{"19", "21"}) {
		t.Fatalf("LockedReqs = %v, want [19 21]", b.LockedReqs)
	}
	if b.Status != "draft" {
		t.Fatalf("Status = %q, want draft", b.Status)
	}
	if b.Topic != "2026-10-02-skillgrid-workflow-updates" {
		t.Fatalf("Topic = %q, want 2026-10-02-skillgrid-workflow-updates", b.Topic)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/validate/ -run TestExtractBriefingLockedReqs -v`
Expected: FAIL (package does not exist / undefined `ExtractBriefing`)
(BDD: the `[validate-extracts-briefing-reqs]` scenario in `acceptance.feature` is RED — it cannot pass until this exists.)

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/validate/schemas.go
package validate

// Briefing is the typed extraction of a spec-folder briefing.md.
type Briefing struct {
	Status     string   // the > **STATUS:** banner value (e.g. "draft")
	Topic      string   // the > **Topic:** line value
	ReqIDs     []string // heading IDs in document order, e.g. ["19","21","12"]
	LockedReqs []string // subset of ReqIDs whose heading title carries "(locked)"
}

// ValidationRow is one row of a blueprint's ## Validation table.
type ValidationRow struct {
	ReqID     string
	Happy     string
	Edge      string
	Failure   string
	Check     string
	Falsifier string
}

// Result is the checker's output (printed as JSON by the CLI).
type Result struct {
	Status   string    `json:"status"` // "pass" | "fail"
	Findings []Finding `json:"findings"`
}

// Finding is one named, locatable problem.
type Finding struct {
	Kind    string `json:"kind"`    // "phase-order" | "coverage" | "parse"
	Subject string `json:"subject"` // topic, artifact name, or req ID
	Message string `json:"message"`
}
```

```go
// skillgrid-cli/internal/validate/parse.go
package validate

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Em-dash per the real briefing.md heading form (pinned by test fixture).
	reqHeadingRe = regexp.MustCompile(`^####\s+Req\s+(\d+)\s+—\s+(.+)$`)
	statusRe     = regexp.MustCompile("^>\\s*\\*\\*STATUS:\\*\\*\\s*`" + "([a-z][a-z0-9-]+)" + "`")
	topicRe      = regexp.MustCompile(`^>\s*\*\*Topic:\*\*\s*(.+)$`)
)

// ExtractBriefing parses briefing.md headings. It returns an error only for
// I/O failures; a missing heading section yields an empty slice, not an error
// (fail open: the checker turns emptiness into a finding, not a panic).
func ExtractBriefing(path string) (Briefing, error) {
	var b Briefing
	f, err := os.Open(path)
	if err != nil {
		return b, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if m := reqHeadingRe.FindStringSubmatch(line); m != nil {
			id := m[1]
			title := m[2]
			b.ReqIDs = append(b.ReqIDs, id)
			if strings.Contains(strings.ToLower(title), "(locked)") {
				b.LockedReqs = append(b.LockedReqs, id)
			}
			continue
		}
		if m := statusRe.FindStringSubmatch(line); m != nil {
			b.Status = m[1]
			continue
		}
		if m := topicRe.FindStringSubmatch(line); m != nil {
			b.Topic = strings.TrimSpace(m[1])
		}
	}
	return b, sc.Err()
}

// ExtractValidationRows parses the "## Validation" markdown table in
// blueprint.md. Column order is fixed: Req | Happy | Edge | Failure | Check |
// Falsifier (per the briefing's starter matrix). Rows whose Req cell is not a
// known numeric ID are skipped (e.g. the matrix's own legend rows).
func ExtractValidationRows(path string) ([]ValidationRow, error) {
	var rows []ValidationRow
	f, err := os.Open(path)
	if err != nil {
		return rows, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	inSection := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "## Validation") {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(line, "## ") {
			break // next H2 ends the section
		}
		if !inSection || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitRow(line)
		if len(cells) < 6 {
			continue
		}
		id := strings.TrimSpace(cells[0])
		if _, err := strconv.Atoi(id); err != nil {
			continue // header row ("Req") or separator
		}
		rows = append(rows, ValidationRow{
			ReqID:     id,
			Happy:     strings.TrimSpace(cells[1]),
			Edge:      strings.TrimSpace(cells[2]),
			Failure:   strings.TrimSpace(cells[3]),
			Check:     strings.TrimSpace(cells[4]),
			Falsifier: strings.TrimSpace(cells[5]),
		})
	}
	return rows, sc.Err()
}

func splitRow(line string) []string {
	cells := strings.Split(line, "|")
	if len(cells) > 0 && cells[0] == "" {
		cells = cells[1:] // leading pipe
	}
	if len(cells) > 0 && cells[len(cells)-1] == "" {
		cells = cells[:len(cells)-1] // trailing pipe
	}
	return cells
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/validate/ -v`
Expected: PASS
Also add the `ExtractValidationRows` fixture test (a `## Validation` table with one numeric row → parsed; a row with `Req` header → skipped) and re-run.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/validate/
git commit -m "feat(validate): typed artifact extraction for briefing + validation matrix"
```

### Task 2: `Validate` phase-order + coverage checker (TDD)

**Files:**
- Create: `skillgrid-cli/internal/validate/validate.go`
- Test: `skillgrid-cli/internal/validate/validate_test.go`

**Interfaces:**
- Consumes: Task 1's `ExtractBriefing`, `ExtractValidationRows`
- Produces: `func Validate(root, topic string) (Result, error)` — `root` is the project root (containing `.skillgrid/`), `topic` is the `specs/<topic>` directory name. `error` is returned only for usage-class problems (topic dir absent); everything else is a `Finding`.
- Seam: the `validate` package API
- Deletion test: the CLI (Task 3) and the JS twin (Task 4) both depend on this exact contract — deleting it forces both to re-implement phase order
- Adapters: 2 (CLI + JS twin; JS twin re-implements the same logic in JS, tested against the same fixtures)

**SATISFIES:** validate-passes-complete-folder / validate-fails-missing-acceptance / validate-fails-uncovered-req (acceptance scenarios: complete in-order folder → pass; `tasks.md` without `acceptance.feature` → phase-order finding; locked req with no validation row → coverage finding)

Phase order — the canonical creation sequence per `.agents/skills/_shared/rules/sdd-structure.md` line 13 ("briefing → blueprint → blueprint validation → cucumber test file → tasks"), mapped to files: `briefing.md` → `blueprint.md` → `acceptance.feature` → `tasks.md`. The check is directional, not a deepest-implication: **a present artifact requires only the artifacts that canonically precede it.** So `briefing + blueprint` (no `acceptance.feature`) is a normal, valid mid-phase state (derived phase `blueprint`) and PASSES; `tasks.md` without `acceptance.feature` is a finding, because the cucumber file canonically precedes tasks.

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/validate/validate_test.go
package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidatePassesCompleteFolder(t *testing.T) {
	root := t.TempDir()
	topic := "2026-10-02-demo"
	writeTree(t, root, map[string]string{
		".skillgrid/specs/" + topic + "/briefing.md":       "# B\n\n#### Req 1 — one (locked)\n",
		".skillgrid/specs/" + topic + "/acceptance.feature": "Feature: demo\n",
		".skillgrid/specs/" + topic + "/blueprint.md":      "# BP\n\n## Validation\n\n| Req | Happy | Edge | Failure | Check | Falsifier |\n|-----|-------|------|---------|-------|-----------|\n| 1 | h | e | f | go test ./... | wrong output |\n",
		".skillgrid/specs/" + topic + "/tasks.md":          "# T\n",
	})
	res, err := Validate(root, topic)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "pass" || len(res.Findings) != 0 {
		t.Fatalf("got %+v, want pass/0 findings", res)
	}
}

func TestValidatePassesMidPhaseBlueprint(t *testing.T) {
	// briefing + blueprint (no acceptance.feature, no tasks) is a valid
	// mid-phase state per sdd-structure.md line 13 — it must PASS.
	root := t.TempDir()
	topic := "2026-10-02-demo"
	writeTree(t, root, map[string]string{
		".skillgrid/specs/" + topic + "/briefing.md": "# B\n",
		".skillgrid/specs/" + topic + "/blueprint.md": "# BP\n",
	})
	res, err := Validate(root, topic)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "pass" {
		t.Fatalf("mid-phase folder must pass, got %+v", res)
	}
}

func TestValidateFailsTasksWithoutAcceptance(t *testing.T) {
	// tasks.md canonically comes AFTER acceptance.feature (line 13) —
	// tasks present without the cucumber file is a phase-order finding.
	root := t.TempDir()
	topic := "2026-10-02-demo"
	writeTree(t, root, map[string]string{
		".skillgrid/specs/" + topic + "/briefing.md": "# B\n\n#### Req 1 — one (locked)\n",
		".skillgrid/specs/" + topic + "/blueprint.md": "# BP\n",
		// acceptance.feature missing
		".skillgrid/specs/" + topic + "/tasks.md": "# T\n",
	})
	res, err := Validate(root, topic)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "fail" {
		t.Fatalf("want fail, got %+v", res)
	}
	found := false
	for _, f := range res.Findings {
		if f.Kind == "phase-order" && strings.Contains(f.Message, "acceptance.feature") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no phase-order finding naming acceptance.feature: %+v", res.Findings)
	}
}

func TestValidateMultiFinding(t *testing.T) {
	// briefing + tasks (no blueprint, no feature) → two phase-order findings.
	root := t.TempDir()
	topic := "2026-10-02-demo"
	writeTree(t, root, map[string]string{
		".skillgrid/specs/" + topic + "/briefing.md": "# B\n",
		".skillgrid/specs/" + topic + "/tasks.md":     "# T\n",
	})
	res, err := Validate(root, topic)
	if err != nil {
		t.Fatal(err)
	}
	subjects := map[string]bool{}
	for _, f := range res.Findings {
		if f.Kind == "phase-order" {
			subjects[f.Subject] = true
		}
	}
	// Directional semantics pinned: tasks.md (index 3) implies blueprint.md
	// (1) AND acceptance.feature (2) — never briefing.md (0).
	if !subjects["blueprint.md"] || !subjects["acceptance.feature"] {
		t.Fatalf("want findings for blueprint.md + acceptance.feature, got %+v", res.Findings)
	}
	if subjects["briefing.md"] {
		t.Fatalf("briefing.md is present — it must not be flagged: %+v", res.Findings)
	}
}

func TestValidateFailsUncoveredLockedReq(t *testing.T) {
	root := t.TempDir()
	topic := "2026-10-02-demo"
	writeTree(t, root, map[string]string{
		".skillgrid/specs/" + topic + "/briefing.md":      "# B\n\n#### Req 1 — one (locked)\n\n#### Req 2 — two (locked)\n",
		".skillgrid/specs/" + topic + "/acceptance.feature": "Feature: demo\n",
		".skillgrid/specs/" + topic + "/blueprint.md": "# BP\n\n## Validation\n\n| Req | Happy | Edge | Failure | Check | Falsifier |\n|-----|-------|------|---------|-------|-----------|\n| 1 | h | e | f | cmd | fals |\n",
		".skillgrid/specs/" + topic + "/tasks.md": "# T\n",
	})
	res, err := Validate(root, topic)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range res.Findings {
		if f.Kind == "coverage" && f.Subject == "2" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no coverage finding for req 2: %+v", res.Findings)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/validate/ -run TestValidate -v`
Expected: FAIL (undefined `Validate`)

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/validate/validate.go
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// phaseOrder is the canonical artifact creation order, mapped from
// .agents/skills/_shared/rules/sdd-structure.md line 13 ("briefing →
// blueprint → blueprint validation → cucumber test file → tasks"). A file at
// index i requires every file at index < i to exist. Mid-phase folders
// (e.g. briefing + blueprint, no feature) pass: only PRESENT artifacts
// trigger the requirement, so the deepest-present rule never flags a valid
// in-flight state.
var phaseOrder = []string{"briefing.md", "blueprint.md", "acceptance.feature", "tasks.md"}

// Validate checks a spec folder for (1) phase-order violations and (2)
// validation-matrix coverage of locked requirements. It never panics on
// missing/unreadable artifacts: those become "parse" findings (fail open,
// ADR-0021 posture).
func Validate(root, topic string) (Result, error) {
	dir := filepath.Join(root, ".skillgrid", "specs", topic)
	if _, err := os.Stat(dir); err != nil {
		return Result{}, fmt.Errorf("topic %q not found under %s/.skillgrid/specs", topic, root)
	}
	var res Result
	res.Status = "pass"

	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}

	// 1. Phase order: the deepest present artifact implies all shallower ones.
	deepest := -1
	for i, name := range phaseOrder {
		if exists(name) {
			deepest = i
		}
	}
	for i := 0; i <= deepest; i++ {
		if !exists(phaseOrder[i]) {
			res.Findings = append(res.Findings, Finding{
				Kind:    "phase-order",
				Subject: phaseOrder[i],
				Message: fmt.Sprintf("%s present but %s missing — artifact order is briefing.md -> blueprint.md -> acceptance.feature -> tasks.md",
					phaseOrder[deepest], phaseOrder[i]),
			})
		}
	}

	// 2. Coverage: every locked req in briefing.md needs a row in blueprint.md.
	if exists("briefing.md") && exists("blueprint.md") {
		brief, err := ExtractBriefing(filepath.Join(dir, "briefing.md"))
		if err != nil {
			res.Findings = append(res.Findings, Finding{Kind: "parse", Subject: "briefing.md", Message: err.Error()})
		} else if brief.Status == "" {
			// Fail-open, not fail-silent: a missing STATUS banner (required by
			// sdd-structure.md's Status Contract) is a parse finding, not a
			// silent success.
			res.Findings = append(res.Findings, Finding{
				Kind: "parse", Subject: "briefing.md",
				Message: "no > **STATUS:** banner found (required by the Status Contract); parsing may be incomplete",
			})
		} else if len(brief.LockedReqs) == 0 {
			res.Findings = append(res.Findings, Finding{
				Kind: "parse", Subject: "briefing.md",
				Message: "no locked requirements found (#### Req N — title (locked)); coverage check skipped",
			})
		}
		rows, err := ExtractValidationRows(filepath.Join(dir, "blueprint.md"))
		if err != nil {
			res.Findings = append(res.Findings, Finding{Kind: "parse", Subject: "blueprint.md", Message: err.Error()})
		} else {
			covered := map[string]bool{}
			for _, r := range rows {
				covered[r.ReqID] = true
			}
			sort.Strings(brief.LockedReqs)
			for _, id := range brief.LockedReqs {
				if !covered[id] {
					res.Findings = append(res.Findings, Finding{
						Kind:    "coverage",
						Subject: id,
						Message: fmt.Sprintf("locked requirement %s has no row in blueprint.md's ## Validation table", id),
					})
				}
			}
		}
	}

	if len(res.Findings) > 0 {
		res.Status = "fail"
	}
	return res, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/validate/ -v`
Expected: PASS (all of Task 1's + Task 2's tests)

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/validate/
git commit -m "feat(validate): phase-order + validation-matrix coverage checker"
```

### Task 3: `skillgrid validate` CLI subcommand

**Files:**
- Create: `skillgrid-cli/internal/validate/root.go`
- Create: `skillgrid-cli/internal/validate/root_test.go`
- Create: `skillgrid-cli/cmd/skillgrid/validate.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go` (the `switch pos[0]` at ~line 138 gains a `case "validate"`; the `fs.Usage` string gains one line)
- Test: `skillgrid-cli/cmd/skillgrid/validate_test.go`

**Interfaces:**
- Consumes: `validate.Validate(root, topic) (validate.Result, error)` from Task 2; `validate.FindRoot(start string) (string, error)` (this task)
- Produces: `skillgrid validate <topic>` — JSON `Result` on stdout, one-line human summary on stderr, exit 0 pass / 1 findings / 2 usage or root-not-found or topic-not-found
- Seam: the CLI entrypoint
- Deletion test: the checker has no invocation path — it is dead code
- Adapters: 1 (production CLI); the JS twin is a separate runtime, not an adapter of this seam

**Note on `reorderArgs`:** `main()` calls `fs.Parse(reorderArgs(rest))` before the `switch pos[0]` dispatch. `reorderArgs` (main.go:191) is an install-arg helper; this task adds a unit test pinning that it passes `["validate", "2026-10-02-demo"]` through unchanged, so the subcommand's args are never mangled.

**SATISFIES:** validate-cli-exit-codes (acceptance scenario: CLI prints JSON, exit 0 on pass, exit 1 on findings, exit 2 on unknown topic / bad usage)

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/cmd/skillgrid/validate_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunValidateExitCodes(t *testing.T) {
	// pass case
	root := t.TempDir()
	topic := "2026-10-02-demo"
	dir := filepath.Join(root, ".skillgrid", "specs", topic)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"briefing.md":        "# B\n\n#### Req 1 — one (locked)\n",
		"blueprint.md":       "# BP\n\n## Validation\n\n| Req | Happy | Edge | Failure | Check | Falsifier |\n|-----|-------|------|---------|-------|-----------|\n| 1 | h | e | f | go test ./... | wrong |\n",
		"acceptance.feature": "Feature: demo\n",
		"tasks.md":           "# T\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if code := runValidate([]string{topic}, root); code != 0 {
		t.Fatalf("pass case: exit %d, want 0", code)
	}
	// findings case — the one code path this change exists to produce
	failDir := filepath.Join(root, ".skillgrid", "specs", "2026-10-02-fail")
	if err := os.MkdirAll(failDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(failDir, "briefing.md"), []byte("# B\n\n#### Req 1 — one (locked)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(failDir, "blueprint.md"), []byte("# BP\n\n## Validation\n\n| Req | Happy | Edge | Failure | Check | Falsifier |\n|-----|-------|------|---------|-------|-----------|\n| 2 | h | e | f | go test ./... | wrong |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runValidate([]string{"2026-10-02-fail"}, root); code != 1 {
		t.Fatalf("findings case: exit %d, want 1", code)
	}
	if code := runValidate([]string{"no-such-topic"}, root); code != 2 {
		t.Fatalf("missing topic: exit %d, want 2", code)
	}
	if code := runValidate([]string{}, root); code != 2 {
		t.Fatalf("no args: exit %d, want 2", code)
	}
}

func TestReorderArgsLeavesValidateIntact(t *testing.T) {
	got := reorderArgs([]string{"validate", "2026-10-02-demo"})
	if len(got) != 2 || got[0] != "validate" || got[1] != "2026-10-02-demo" {
		t.Fatalf("reorderArgs mangled validate args: %v", got)
	}
}

// TestRunValidateJSONStdout pins the BMAD contract: the JSON Result goes to
// STDOUT (parseable, correct status), the human summary goes to STDERR.
// Captures os.Stdout via a pipe so a regression that swaps the streams fails.
func TestRunValidateJSONStdout(t *testing.T) {
	root := t.TempDir()
	topic := "2026-10-02-demo"
	dir := filepath.Join(root, ".skillgrid", "specs", topic)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"briefing.md":        "# B\n\n#### Req 1 — one (locked)\n",
		"blueprint.md":       "# BP\n\n## Validation\n\n| Req | Happy | Edge | Failure | Check | Falsifier |\n|-----|-------|------|---------|-------|-----------|\n| 1 | h | e | f | go test ./... | wrong |\n",
		"acceptance.feature": "Feature: demo\n",
		"tasks.md":           "# T\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := runValidate([]string{topic}, root)
	w.Close()
	os.Stdout = orig
	out, _ := io.ReadAll(r)

	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	var got struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("stdout is not parseable JSON: %v (%q)", err, out)
	}
	if got.Status != "pass" {
		t.Fatalf("status = %q, want pass", got.Status)
	}
}
```
(imports for this test file: add `"encoding/json"` and `"io"`)

And `skillgrid-cli/internal/validate/root_test.go` (written first, per TDD):

```go
package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRootWalksUp(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".skillgrid", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := FindRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("FindRoot = %q, want %q", got, root)
	}
}

func TestFindRootNotProject(t *testing.T) {
	if _, err := FindRoot(t.TempDir()); err == nil {
		t.Fatal("want error for a dir without .skillgrid/specs")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/cmd/skillgrid/ -run TestRunValidateExitCodes -v`
Expected: FAIL (undefined `runValidate`)

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/validate/root.go
package validate

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRoot walks up from start until it finds a directory containing
// .skillgrid/specs/ (the project root). It returns an error if no ancestor
// qualifies — the CLI turns that into exit 2.
func FindRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".skillgrid", "specs")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .skillgrid/specs found at or above %s", start)
		}
		dir = parent
	}
}
```

```go
// skillgrid-cli/cmd/skillgrid/validate.go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/validate"
)

// runValidate implements `skillgrid validate <topic>`. It returns the process
// exit code: 0 pass, 1 findings, 2 usage / root-not-found / topic-not-found.
// JSON Result goes to stdout; a one-line human summary goes to stderr (the
// BMAD contract: structured stdout including errors). root is explicit (the
// test injects a temp tree); production calls it with FindRoot(os.Getwd()).
func runValidate(args []string, root string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: skillgrid validate <topic>")
		return 2
	}
	res, err := validate.Validate(root, args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	out, err := json.Marshal(res)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal result:", err)
		return 2
	}
	fmt.Println(string(out))
	if res.Status == "pass" {
		fmt.Fprintf(os.Stderr, "validate: %s — pass\n", args[0])
		return 0
	}
	fmt.Fprintf(os.Stderr, "validate: %s — %d finding(s)\n", args[0], len(res.Findings))
	for _, f := range res.Findings {
		fmt.Fprintf(os.Stderr, "  [%s] %s: %s\n", f.Kind, f.Subject, f.Message)
	}
	return 1
}
```

Then in `main.go`, in the `switch pos[0]` (at ~line 138, alongside the `install`/`in` cases), add:

```go
	case "validate":
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		root, err := validate.FindRoot(cwd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		os.Exit(runValidate(pos[1:], root))
```

(`os.Getwd()` is called inline — main.go has no `cwd` variable today; the import is `"github.com/devopstales/skillgrid/skillgrid-cli/internal/validate"`. No `--root` flag: discovery is walked, and a subtopic invocation is a usage error, exit 2.)

and add the usage line to `fs.Usage`:

```go
		fmt.Fprintln(w, `  validate      Validate a spec folder's phase order + validation matrix`)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/cmd/skillgrid/ ./skillgrid-cli/internal/validate/ -v`
Expected: PASS (all of Tasks 1–3). Then build: `go build ./...` from the repo root and run the real binary once against this change's own spec folder as a smoke:
`go run ./skillgrid-cli/cmd/skillgrid validate 2026-10-02-skillgrid-workflow-updates`
Expected: **exit 0 pass** — this change's folder currently has briefing + blueprint only, which is the canonical `blueprint` phase (a valid mid-phase state under the directional phase-order rule; the deepest present artifact is `blueprint.md`, and nothing present implies a missing artifact). The backstop's negative half (exit 1 on real coverage failure) is proven by the JS checker in Task 4 Step 4.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/cmd/skillgrid/ skillgrid-cli/internal/validate/
git commit -m "feat(validate): skillgrid validate subcommand (JSON stdout, exit 0/1/2)"
```
(`git add` includes `skillgrid-cli/internal/validate/` because this task creates `root.go` + `root_test.go` there — one logical change, one commit.)

### Task 4: JS validation-matrix checker + fixture tests

**Files:**
- Create: `scripts/validation-matrix-check.mjs`
- Test: `scripts/test-validation-matrix.mjs`

**Interfaces:**
- Consumes: the same extraction semantics as Task 1 (`#### Req N — title (locked)` headings; `## Validation` table with columns Req | Happy | Edge | Failure | Check | Falsifier; identical cell split)
- Produces: `node scripts/validation-matrix-check.mjs <spec-dir>` — JSON on stdout (`{"status":"pass"|"fail","findings":[{kind,subject,message}]}`), exit 0 pass / 1 findings / 2 usage
- Seam: none (in-process script; the seam is the exit-code contract the skill consumes)
- Deletion test: the skill gate (Task 7) has no oracle — validation is prose again
- Adapters: 2 (real spec folders + test fixtures)

**SATISFIES:** js-checker-passes-complete-matrix / js-checker-fails-missing-row / js-checker-usage-error / go-js-contract-agree (acceptance scenarios: complete matrix → 0; locked req without row → 1 naming the req; no spec dir → 2; JS over the Go-pinned fixture bytes yields the same covered decision — positive and negative)

- [ ] **Step 1: Write the failing test**

```js
// scripts/test-validation-matrix.mjs — fixture-driven tests for
// validation-matrix-check.mjs (same shape as test-state-drift.mjs).
// Usage: node scripts/test-validation-matrix.mjs — Exit: 0 all pass, 1 any fail.
// (The SKILL.md wiring checks live in test-skill-wiring.mjs — committed with
// the skill edit in Task 7, so this file is never red between tasks.)

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const CHECKER = join(HERE, "validation-matrix-check.mjs");

let passed = 0, failed = 0;
const failures = [];

function specDir({ briefing, blueprint }) {
  const root = mkdtempSync(join(tmpdir(), "sgvmatrix-"));
  mkdirSync(join(root, "specs"), { recursive: true });
  writeFileSync(join(root, "specs", "briefing.md"), briefing);
  writeFileSync(join(root, "specs", "blueprint.md"), blueprint);
  return join(root, "specs");
}

function run(spec) {
  try {
    const out = execFileSync("node", [CHECKER, spec], { encoding: "utf8" });
    return { code: 0, out };
  } catch (e) {
    return { code: e.status, out: (e.stdout || "") + (e.stderr || "") };
  }
}

function check(name, cond) {
  if (cond) { passed++; }
  else { failed++; failures.push(name); }
}

const goodBriefing = `# B\n\n#### Req 1 — one (locked)\n\n#### Req 2 — two (locked)\n`;
const goodBlueprint = `# BP\n\n## Validation\n\n| Req | Happy | Edge | Failure | Check | Falsifier |\n|-----|-------|------|---------|-------|-----------|\n| 1 | h | e | f | go test ./... | wrong |\n| 2 | h | e | f | go test ./... | wrong |\n`;

// 1. complete matrix -> exit 0
let d = specDir({ briefing: goodBriefing, blueprint: goodBlueprint });
let r = run(d);
check("complete matrix exits 0", r.code === 0);
check("complete matrix status pass", r.out.includes('"status":"pass"') || r.out.includes('"status": "pass"'));
rmSync(d, { recursive: true, force: true });

// 2. missing row for req 2 -> exit 1, names req 2
const noRow2 = goodBlueprint.replace("| 2 | h | e | f | go test ./... | wrong |\n", "");
d = specDir({ briefing: goodBriefing, blueprint: noRow2 });
r = run(d);
check("missing row exits 1", r.code === 1);
check("missing row names req 2", r.out.includes('"2"'));
rmSync(d, { recursive: true, force: true });

// 3. no spec dir -> exit 2
r = run(join(tmpdir(), "no-such-spec-dir-xyz"));
check("missing spec dir exits 2", r.code === 2);

// 4. Go/JS contract: same INPUT BYTES on both sides, same covered set.
//    contractFixture is the LITERAL text of parse_test.go's briefFixture plus
//    the matching Validation rows (reqs 19, 21, 12) — the JS side must see
//    exactly the locked IDs the Go side pins (19, 21), and the covered rows
//    must satisfy them. If the two fixture texts drift, this test fails.
const contractBriefing = `# Briefing

> **STATUS:** \`draft\` (2026-10-09)
> **Topic:** 2026-10-02-skillgrid-workflow-updates

#### Req 19 — Technical validation plan (locked)

Some text.

#### Req 21 — Scope pushback (locked)

Some text.

#### Req 12 — Ablate always-loaded rules (named follow-up)

Follow-up text.
`;
const contractBlueprint = `# BP

## Validation

| Req | Happy | Edge | Failure | Check | Falsifier |
|-----|-------|------|---------|-------|-----------|
| 19 | h | e | f | go test ./... | wrong |
| 21 | h | e | f | go test ./... | wrong |
| 12 | h | e | f | go test ./... | wrong |
`;
d = specDir({ briefing: contractBriefing, blueprint: contractBlueprint });
r = run(d);
check("contract: JS exits 0 on the Go-pinned fixture", r.code === 0);
check("contract: JS covered set satisfies locked reqs 19+21 (not 12)", r.code === 0 && !r.out.includes('"coverage"'));
rmSync(d, { recursive: true, force: true });

// 4b. negative contract: drop req 21's row → JS must flag exactly "21".
d = specDir({ briefing: contractBriefing, blueprint: contractBlueprint.replace("| 21 | h | e | f | go test ./... | wrong |\n", "") });
r = run(d);
check("contract: JS flags exactly req 21 when its row is dropped", r.code === 1 && r.out.includes('"21"') && !r.out.includes('"19"'));
rmSync(d, { recursive: true, force: true });

console.log(`\nvalidation-matrix tests: ${passed} passed, ${failed} failed`);
if (failed > 0) { console.error(failures.join("\n")); process.exit(1); }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `node scripts/test-validation-matrix.mjs`
Expected: FAIL (checker file does not exist — all four cases fail)

- [ ] **Step 3: Write minimal implementation**

```js
// scripts/validation-matrix-check.mjs — read-only validation-matrix checker.
// Usage: node scripts/validation-matrix-check.mjs <spec-dir>
// Exit: 0 pass, 1 findings, 2 usage/missing dir.
// Structured JSON on stdout (the BMAD contract, per findings.md GSD §mechanism 2).

import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

function extractLockedReqs(briefing) {
  const re = /^####\s+Req\s+(\d+)\s+—\s+(.+)$/gm;
  const ids = [];
  let m;
  while ((m = re.exec(briefing)) !== null) {
    if (m[2].toLowerCase().includes("(locked)")) ids.push(m[1]);
  }
  return ids;
}

// extractValidationRows mirrors the Go ExtractValidationRows cell split
// EXACTLY (split on "|", drop the leading/trailing empty cells from the outer
// pipes, require >= 6 cells, numeric first cell). The Go/JS contract test in
// test-validation-matrix.mjs case 4 pins the agreement.
function extractValidationRows(blueprint) {
  const lines = blueprint.split("\n");
  const rows = new Set();
  let inSection = false;
  for (const raw of lines) {
    const line = raw.trim();
    if (line.startsWith("## Validation")) { inSection = true; continue; }
    if (inSection && line.startsWith("## ")) break;
    if (!inSection || !line.startsWith("|")) continue;
    let cells = line.split("|");
    if (cells[0] === "") cells = cells.slice(1);
    if (cells[cells.length - 1] === "") cells = cells.slice(0, -1);
    if (cells.length < 6) continue;
    const id = cells[0].trim();
    if (/^\d+$/.test(id)) rows.add(id);
  }
  return rows;
}

function main() {
  const specDir = process.argv[2];
  if (!specDir || !existsSync(specDir)) {
    console.error("usage: node scripts/validation-matrix-check.mjs <spec-dir>");
    process.exit(2);
  }
  const findings = [];
  const briefingPath = join(specDir, "briefing.md");
  const blueprintPath = join(specDir, "blueprint.md");
  if (!existsSync(briefingPath) || !existsSync(blueprintPath)) {
    findings.push({ kind: "phase-order", subject: "briefing.md|blueprint.md", message: "both briefing.md and blueprint.md must exist" });
  } else {
    const locked = extractLockedReqs(readFileSync(briefingPath, "utf8"));
    if (locked.length === 0) {
      findings.push({ kind: "parse", subject: "briefing.md", message: "no locked requirements found; coverage check skipped" });
    }
    const rows = extractValidationRows(readFileSync(blueprintPath, "utf8"));
    for (const id of locked) {
      if (!rows.has(id)) {
        findings.push({ kind: "coverage", subject: id, message: `locked requirement ${id} has no row in blueprint.md's ## Validation table` });
      }
    }
  }
  const result = { status: findings.length ? "fail" : "pass", findings };
  console.log(JSON.stringify(result));
  process.exit(findings.length ? 1 : 0);
}

main();
```

- [ ] **Step 4: Run test to verify it passes**

Run: `node scripts/test-validation-matrix.mjs`
Expected: all 4 cases PASS (the SKILL.md wiring checks are NOT here — they live in `test-skill-wiring.mjs`, committed in Task 7, so this file is never red after its own commit).
Then run the checker against this change's real spec folder: `node scripts/validation-matrix-check.mjs .skillgrid/specs/2026-10-02-skillgrid-workflow-updates`
Expected: exit 1 with coverage findings naming reqs 19 and 21 (the real `briefing.md` marks both `(locked)`; the real `blueprint.md`'s `## Validation` table has no row for either yet — the backstop truth, a real tree, not a fixture).

- [ ] **Step 5: Commit**

```bash
git add scripts/validation-matrix-check.mjs scripts/test-validation-matrix.mjs
git commit -m "feat(scripts): validation-matrix-check.mjs with fixture tests"
```

### Task 5: ADR-0046 — Go runtime home

**Files:**
- Create: `.skillgrid/decisions/04-adr-0046-release2-runtime-home.md`
- Modify: `.skillgrid/ASSUMPTIONS.md` (append one index row under `## LOCKED` → ADR index; update "Highest sequence in use" to 0046)

**Interfaces:**
- Consumes: the ADR format used by every file in `.skillgrid/decisions/` (frontmatter `status: accepted`, `date`, `supersedes:` empty; body: Context / Decision / Consequences — one decision per file, per ADR-0019)
- Produces: the in-force decision that Release/2 Go runtime packages live in `skillgrid-cli/internal/<engine>/`
- Seam: none (documentation artifact)
- Deletion test: future engine packages have no named home — the briefing's open question ("mnemonic/ module or new skillgrid-runtime/ module?") re-opens
- Adapters: n/a

**SATISFIES:** adr-0046-records-runtime-home (acceptance scenario: the ADR file exists with frontmatter, names the three options considered, and is indexed in ASSUMPTIONS.md with In-force = yes)

- [ ] **Step 1: Write the ADR body**

```markdown
---
status: accepted
date: 2026-10-17
supersedes: []
---

# ADR-0046: Release/2 Go runtime home — `skillgrid-cli/internal/` packages

## Context

Release/2 (`.skillgrid/specs/2026-10-02-skillgrid-workflow-updates/`) adds a Go
runtime: workflow-state validation, artifact schemas, and (later) the policy,
execution, and verification engines. `briefing.md` §13 left the home open:
"Does the Go runtime land in the `mnemonic/` module or a new `skillgrid-runtime/`
module?" The tracer bullet (plan.md §3) needs the answer before the first
package is written.

Note on numbering: `plan.md` §5 named this decision "Create ADR-0032: Go Runtime
Architecture for Release/2." ADR-0032 is already in force for a different
decision (opencode + Kilo native TS plugin architecture), so this decision
records as **0046** — the next free sequence — and plan.md §5 is stale on that
line (the ADR's purpose is unchanged, only its number).

## Decision

The runtime lives in the **existing `skillgrid-cli` module** as
`skillgrid-cli/internal/<engine>/` packages (`internal/validate` first).
No new Go module; `mnemonic/` is untouched.

Options considered:

1. **New `skillgrid-runtime/` module** — clean boundary, but a third module
   with one package, a `go.work` change, and an install story for a binary
   nobody invokes yet. Rejected for the tracer bullet; revisit when the
   runtime has its own CLI surface.
2. **`mnemonic/` module** — wrong owner: Mnemonic is the memory + code-index
   authority (ADR-0012/0016); workflow-state validation is not memory.
3. **`skillgrid-cli/internal/`** (chosen) — the `skillgrid` binary already
   exists (`cmd/skillgrid`), already parses YAML (`gopkg.in/yaml.v3` is a
   direct dep), and `validate` is a natural sibling of `install`.
   ADR-0033's three-module workspace is unaffected: it governs the `ctx/`
   library module, which still imports `mnemonic`.

## Consequences

- The `skillgrid` binary gains subcommands as engines land (`validate` now;
  `doctor`/`explain` later per plan §38 item 2–4).
- `internal/validate` is importable only within `skillgrid-cli`; if a future
  engine must be consumed by another module, that is a superseding ADR moving
  it — the boundary is designed to be crossed deliberately.
- ADR-0033 (three-module workspace) is **unaffected**: it governs the `ctx/`
  library module importing `mnemonic`; adding packages inside `skillgrid-cli/`
  does not change the workspace shape.
- ASSUMPTIONS.md § LOCKED gains one index row (path only, per ADR-0019).
```

- [ ] **Step 2: Verify format**

Run: `grep -c "^status: accepted" .skillgrid/decisions/04-adr-0046-release2-runtime-home.md`
Expected: `1`. (The ADR invariants script `scripts/check-adr-invariants.mjs` runs in QA; it is the held-out check.)

- [ ] **Step 3: Update the index**

In `.skillgrid/ASSUMPTIONS.md`, under the ADR index table, add the row — **exactly matching the on-file row shape** (4 cells: `# | Description | In force | File`, backticked path last), the same shape as rows 0001–0045:

```
| 0046 | Release/2 Go runtime home: `skillgrid-cli/internal/<engine>/` packages, no new module, `mnemonic/` untouched | yes | `.skillgrid/decisions/04-adr-0046-release2-runtime-home.md` |
```

and update `**Highest sequence in use:** 0045` → `0046` (next ADR is `04-adr-0047-slug.md`). Do not "correct" the shape to the script's stale 6-column header comment — the parser matches the 4-column rows (its `.*?` absorbs Description + In force); a wrong-shaped row silently breaks the pre-commit bijection check.

- [ ] **Step 4: Run the invariants check (baseline-aware)**

Run: `node scripts/check-adr-invariants.mjs`
Expected: **no NEW violations** beyond the pre-existing baseline. Known baseline drift (verify before starting — this predates the change): `04-adr-0042-skill-pi-dependencies.md` reports as an orphan (no in-force table row) — that ADR's row shape was changed by another change; it is OUT OF SCOPE here. The 0046 row must resolve cleanly (no new orphan, no duplicate). Note: the pre-commit hook runs this same script — if the baseline drift blocks the commit, that is a pre-existing hook failure, not this task's; record it in the commit's `[skillgrid-context]` block and do NOT fix 0042's row here (out of scope, different change).

- [ ] **Step 5: Commit**

```bash
git add .skillgrid/decisions/04-adr-0046-release2-runtime-home.md .skillgrid/ASSUMPTIONS.md
git commit -m "docs(adr): 0046 Release/2 Go runtime home — skillgrid-cli internal packages"
```

### Task 6: Wire the checker into `writing-blueprints` (+ wiring test)

**Files:**
- Create: `scripts/test-skill-wiring.mjs`
- Modify: `.agents/skills/planning/writing-blueprints/SKILL.md` (the "Blueprint validation (mandatory, two-stage)" block only)

**Interfaces:**
- Consumes: `scripts/validation-matrix-check.mjs` exit-code contract (Task 4)
- Produces: the skill's validation gate fails on exit 1
- Seam: the skill text (the threat-matrix shared-convention-drift row)
- Deletion test: the checker is never run — blueprints with dropped validation rows pass again
- Adapters: n/a (prose binding)

The wiring test is committed **in this same task** as the skill edit (threat-matrix response to the ordering hazard: a test that is red between tasks blocks the gate that this change creates).

**SATISFIES:** skill-gate-fails-dropped-row (acceptance scenario: the SKILL.md text invokes the script and treats exit 1 as a failed blueprint validation — the grep case in `scripts/test-skill-wiring.mjs` turns green)

- [ ] **Step 1: Write the wiring test (RED)**

```js
// scripts/test-skill-wiring.mjs — grep tests for the writing-blueprints
// SKILL.md binding to validation-matrix-check.mjs (threat matrix:
// shared-convention drift). Committed WITH the skill edit (Task 6), so it is
// never red between tasks.
// Usage: node scripts/test-skill-wiring.mjs — Exit: 0 all pass, 1 any fail.

import { readFileSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
const SKILL_MD = join(HERE, "..", ".agents", "skills", "planning", "writing-blueprints", "SKILL.md");

let passed = 0, failed = 0;
const failures = [];
function check(name, cond) {
  if (cond) passed++;
  else { failed++; failures.push(name); }
}

const skill = readFileSync(SKILL_MD, "utf8");
check("SKILL.md names the checker", skill.includes("validation-matrix-check.mjs"));
check("SKILL.md binds exit-1 to failed validation",
  /validation-matrix-check\.mjs[\s\S]{0,400}exit.*1|exit.*1[\s\S]{0,400}validation-matrix-check\.mjs/.test(skill));

console.log(`\nskill-wiring tests: ${passed} passed, ${failed} failed`);
if (failed > 0) { console.error(failures.join("\n")); process.exit(1); }
```

Run: `node scripts/test-skill-wiring.mjs`
Expected: FAIL (both cases — the script is not named in the skill yet)

- [ ] **Step 2: Edit the skill**

In the "Blueprint validation (mandatory, two-stage)" block, before the critic-subagent item (item 1), add a mechanical item 0:

```markdown
0. **Validation matrix (mechanical, before anything else):** run
   `node scripts/validation-matrix-check.mjs .skillgrid/specs/YYYY-MM-DD-<topic>`.
   Exit 1 (a locked requirement has no `## Validation` row, or a phase-order
   gap) is a **failed** blueprint validation — fix the blueprint (add the
   row, or fill the missing artifact) and re-run. Exit 2 is a usage error —
   fix the invocation. Exit 0 is the only path to the critic stage.
```

- [ ] **Step 3: Run test to verify it passes**

Run: `node scripts/test-skill-wiring.mjs`
Expected: both cases PASS

- [ ] **Step 4: Run the full check suite**

Run: `node scripts/test-validation-matrix.mjs && node scripts/test-skill-wiring.mjs && go test ./skillgrid-cli/...`
Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add .agents/skills/planning/writing-blueprints/SKILL.md scripts/test-skill-wiring.mjs
git commit -m "feat(skills): writing-blueprints gates on validation-matrix-check.mjs"
```

### Task 7: This blueprint's own `## Validation` section + self-validation

**Files:**
- Modify: `.skillgrid/specs/2026-10-02-skillgrid-workflow-updates/blueprint.md` (add the `## Validation` section below, before `## Execution Handoff`)

**Interfaces:**
- Consumes: both checkers (Tasks 2–4)
- Produces: the backstop truth's positive half — this change self-validates
- Seam: n/a (artifact content)
- Deletion test: the change's own matrix is empty — the must-have "self-validation exit 0" is unproven
- Adapters: n/a

**SATISFIES:** validate-passes-complete-folder (re-run on the real tree — the backstop truth)

The rows (one per in-scope locked requirement — req 19, the validation matrix itself, and req 21, scope pushback, the two locked reqs this change's briefing names; the tracer-bullet tasks are the check column):

```markdown
## Validation

| Req | Happy | Edge | Failure | Check | Falsifier |
|-----|-------|------|---------|-------|-----------|
| 19 | `skillgrid validate 2026-10-02-skillgrid-workflow-updates` exits 0 after this section + acceptance.feature + tasks.md exist | mid-phase folder (briefing + blueprint only) exits 0 (not a false positive); `briefing + tasks` exits 1 with two phase-order findings | locked req with no row exits 1 naming the req | `go test ./skillgrid-cli/internal/validate/ -v` + `node scripts/test-validation-matrix.mjs` + the real-folder re-run in Task 7 Step 3 | a planner that drops a row passes slicing |
| 21 | the recorded first-slice pick is `plan.md` §3 (the tracer bullet: schemas + `skillgrid validate` + matrix script + skill wiring + ADR); the other 32 plan items are explicitly deferred | the wedge touches 3 independent subsystems (Go CLI, JS scripts, skill text) but produces one demoable behavior; provenance: interview round 9 cut 21+19+8, round 13 re-opened the cut, plan.md §3 re-cut it as the tracer bullet (req 8's `.mjs` contract is realized by Task 4's tested script) | a `blueprint.md` exists while no first-slice pick is recorded | `plan.md` §3 (the pick) + `briefing.md` §12 interview log rounds 9 + 13 (provenance) | blueprint scope > one demoable behavior |
```

- [ ] **Step 1: Add the section** to `blueprint.md` (the markdown block above), placed after Task 6 and before `## Execution Handoff`, so it reads as the change's own matrix.

- [ ] **Step 2: Verify the real tree passes the JS checker**

Run: `node scripts/validation-matrix-check.mjs .skillgrid/specs/2026-10-02-skillgrid-workflow-updates`
Expected: still exit 1 — but now only because `acceptance.feature` and `tasks.md` do not exist yet (phase order) OR coverage is now satisfied (the rows exist). If the checker's phase-order logic (which the Go side implements; the JS side checks coverage only) means this passes coverage: expect exit 0 from the JS checker once both locked reqs have rows. **The JS checker is coverage-only** (its `main` checks briefing+blueprint existence + coverage; phase order is the Go checker's job) — so after this step, the JS checker MUST exit 0 on the real folder.

- [ ] **Step 3: (post-slicing) Re-run the Go CLI on the complete tree — owner: the slicing ticket that writes `tasks.md`**

This re-run happens once `acceptance.feature` (authored by `skillgrid:acceptance-test-authoring`, which runs between blueprint validation and slicing per sdd-structure line 13) and `tasks.md` (slicing) exist. **Ownership is explicit:** the slicing ticket's brief carries this as its final step — it is the ticket that makes the folder complete, so it proves completeness.
Run: `go run ./skillgrid-cli/cmd/skillgrid validate 2026-10-02-skillgrid-workflow-updates`
Expected: **exit 0, `{"status":"pass","findings":[]}`** — the backstop truth, proven on the real tree. Record the output in `report.md`'s evidence section at QA time.

**Contract with `acceptance.feature` (C3 resolution):** no task in this plan authors `acceptance.feature` — `skillgrid:acceptance-test-authoring` owns it, and it runs after blueprint validation and before slicing (sdd-structure line 13). The `SATISFIES:` labels in Tasks 1–6 (`validate-extracts-briefing-reqs`, `validate-passes-complete-folder`, `validate-fails-missing-acceptance`, `validate-fails-uncovered-req`, `validate-cli-exit-codes`, `js-checker-passes-complete-matrix`, `js-checker-fails-missing-row`, `js-checker-usage-error`, `go-js-contract-agree`, `adr-0046-records-runtime-home`, `skill-gate-fails-dropped-row`) are the **committed contract** that `acceptance.feature` must implement — one scenario per label, named exactly. The slicing pass maps each ticket's `SATISFIES` onto those scenario names; a label with no scenario is a slicing failure.

- [ ] **Step 4: Commit**

```bash
git add .skillgrid/specs/2026-10-02-skillgrid-workflow-updates/blueprint.md
git commit -m "docs(spec): add this change's Validation matrix (reqs 19, 21)"
```

## Plan Review

- Verdict: READY FOR EXECUTION (after revision pass)
- Findings: 3 Critical (fixed: C1 Task 5 baselines the pre-existing 0042 ADR-invariants drift + names the pre-commit-hook handling; C2 pins the 0046 index row to the on-file 4-cell shape; C3 names `acceptance-test-authoring` as the `acceptance.feature` owner and the slicing ticket as owner of Task 7 Step 3's re-run, with the SATISFIES labels as the committed scenario contract), 3 Important (fixed: I1 Go/JS contract test now feeds the JS checker the exact Go-pinned fixture bytes, positive + negative; I2 adds `TestRunValidateJSONStdout` pinning JSON-to-stdout; I3 adds a missing-banner parse finding so parsing fails open, not silent; I4 `os.Getwd()` is called inline — no phantom `cwd`), 5 Minor (fixed: M1 Task 3 commit adds `internal/validate/`; M2 multi-finding test pins subjects; M4 req 21 row leads with plan.md §3 as the pick. Deferred: M3 commit-scope vocabulary — executor matches `git log` at commit time.)
- Reviewed: 2026-10-17 (critic pass + plan-review pass, both subagent)

## Execution Handoff

- **Slicing:** 7 tasks → invoke `skillgrid:slicing` after blueprint validation (two stages: critic subagent + user approval). Task 7's Step 3 (the Go CLI re-run on the complete tree) lands after `acceptance.feature` and `tasks.md` exist — slicing schedules it accordingly.
- **One-way doors:** none.
- **Backstop truth (held-out, two halves):**
  1. **Negative (proven in Task 4):** the JS checker on this change's real folder exits 1 with coverage findings naming reqs 19 and 21 — before Task 7 fills the matrix.
  2. **Positive (proven in Task 7 Step 3, post-slicing):** after the `## Validation` rows + `acceptance.feature` + `tasks.md` exist, `skillgrid validate 2026-10-02-skillgrid-workflow-updates` exits 0 — the change self-validates under the gate it built.
