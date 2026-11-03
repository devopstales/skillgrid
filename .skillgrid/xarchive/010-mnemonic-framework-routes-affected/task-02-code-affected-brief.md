## 02-code-affected

### Goal

`code_affected` (read) + `code_rename` (write) MCP/CLI tools — "which tests do I run after this change?" and "what does renaming this symbol touch?" — both grounded in the existing graph. `code_affected` now supports **`--base <ref>`**: it derives the changed set from the merge-base diff, groups the result into affected areas, and names git-history owners ("who to tag") via `git blame` — a ready PR comment / CI gate. Blast-radius math traverses only resolved edges (step 01's drop policy), so a dropped/ambiguous reference can never inflate the radius.

### Out of scope / Non-Goals

- Route edges + the drop-not-guess policy (step 01); watcher / banner / fingerprint gate / warm embedder (step 03)
- Recomputing the dependency graph; storing a precomputed impact matrix

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-framework-routes

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/affected/affected.go`
- Create: `skillgrid-cli/internal/mnemonic/affected/rename.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_affected.go`
- Modify: `skillgrid-cli/cmd/skillgrid/code_intel.go` (`skillgrid search affected` CLI `--stdin`/`--base <ref>`/`--depth`/`--filter`/`--json`/`--quiet` + `skillgrid search rename`)
- Test: `skillgrid-cli/internal/mnemonic/affected/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/cmd/skillgrid/...`

**Interfaces:**
- Consumes: 005's import + `tests-for` edges and symbol disambiguation; step 01's drop-not-guess resolved-edge set (blast-radius math); 005/010 `symbols`/`edges` tables; `git` (merge-base diff + `git blame`) for `--base`
- Produces: `code_affected` traversal (depth-capped, `--stdin`/`--base <ref>`/`--filter`/`--json`/`--quiet`); `--base` merge-base diff → affected-area grouping + git-history owners ("who to tag"); `code_rename` (graph vs. text-search edit buckets, `dry_run` default true, per-edit confidence); MCP tools + CLI parity

### Tasks

- [ ] 02.1 `[RED]` `code_affected` traverses changed files to test files (Scenario: code_affected returns affected test files)
  - [ ] 02.1.a Write failing test — fixture repo where changed sources are transitively imported; `code_affected src/utils.ts src/api.ts` returns the test files that (transitively) import the changed symbols via import + `tests-for` edges; `--depth` caps traversal (default 5); `--filter "e2e/*"` restricts reported test files; `--json` / `--quiet` emit scriptable output; result is paths/relationships from the existing graph only (no invented edges)
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeAffected -count=1` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation — `affected/affected.go` traversal (import + `tests-for`, depth cap, filter) + service/CLI facade
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeAffected -count=1` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(mnemonic): code_affected test-impact traversal`
- [ ] 02.2 `[RED]` PR commands — `code_affected --stdin` on a fixture diff (Scenario: code_affected consumes a diff file list from stdin) — threat: PR commands
  - [ ] 02.2.a Write failing test — `git diff --name-only`-style stdin (fixture diff) returns the correct test files; an empty diff (empty stdin) returns an empty result with a clear message, not an error
  - [ ] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/cmd/skillgrid/... -run CodeAffectedStdin -count=1` — Expected: FAIL
  - [ ] 02.2.c Minimal implementation — `--stdin` file-list ingestion in traversal + `code_intel.go` `search affected` CLI
  - [ ] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/cmd/skillgrid/... -run CodeAffectedStdin -count=1` — Expected: PASS
  - [ ] 02.2.e Commit — `feat(mnemonic): code_affected --stdin for CI and hooks`
- [ ] 02.3 `[RED]` PR commands — `code_affected --base <ref>` merge-base diff + affected areas + git-history owners (Scenario: code_affected --base derives areas and owners) — threat: PR commands; Business rule "`code_affected --base <ref>` PR mode"
  - [ ] 02.3.a Write failing test — fixture branch with a merge-base diff against a base ref; `code_affected --base origin/main` derives the changed file set from `git merge-base <base> HEAD` + diff (no `--stdin` plumbing), walks the same import/`tests-for` traversal, groups the result into affected **areas** (per-symbol detail collapsed under each area), and names **git-history owners** ("who to tag") for each touched area via `git blame` on the changed lines; `--quiet`/`--json` shape the output as a ready PR comment / CI gate
  - [ ] 02.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeAffectedBase -count=1` — Expected: FAIL
  - [ ] 02.3.c Minimal implementation — `--base` merge-base diff changed-set source + affected-area grouping + `git blame` owner attribution in `affected/affected.go` + `code_intel.go`
  - [ ] 02.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeAffectedBase -count=1` — Expected: PASS
  - [ ] 02.3.e Commit — `feat(mnemonic): code_affected --base merge-base diff with owners`
- [ ] 02.4 `[RED]` `code_rename` splits edits into graph vs text-search buckets (Scenario: code_rename returns graph and text-search edit buckets)
  - [ ] 02.4.a Write failing test — `code_rename validateUser verifyUser` on a fixture returns `files_affected` / `total_edits` split into **graph edits** (high-confidence: definition + typed references from the edges table) and **text-search edits** (lower-confidence: flagged "review carefully"); every edit carries a confidence label; `--dry-run` (default) returns the plan without writing any file
  - [ ] 02.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeRename -count=1` — Expected: FAIL
  - [ ] 02.4.c Minimal implementation — `affected/rename.go` (resolve via 005 disambiguation; graph edges → graph bucket; string matches → text-search bucket; per-edit confidence; `dry_run` default true)
  - [ ] 02.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeRename -count=1` — Expected: PASS
  - [ ] 02.4.e Commit — `feat(mnemonic): code_rename graph and text-search buckets`
- [ ] 02.5 `[RED]` PR commands — non-dry `code_rename` edits only listed files, no commit/push (Scenario: code_rename applies only listed files without committing) — threat: PR commands
  - [ ] 02.5.a Write failing test — a non-dry `code_rename` against a fixture edits exactly the files in its plan (no other file touched); no commit or push is performed
  - [ ] 02.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeRenameApply -count=1` — Expected: FAIL
  - [ ] 02.5.c Minimal implementation — apply path limited to planned files; no git side effects
  - [ ] 02.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeRenameApply -count=1` — Expected: PASS
  - [ ] 02.5.e Commit — `feat(mnemonic): code_rename applies only planned files`
- [ ] 02.6 `[AFK]` Ambiguous rename target returns a ranked candidate list (Scenario: Ambiguous rename target returns candidates) — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeRenameAmbiguous -count=1` — Expected: PASS
- [ ] 02.7 `[AFK]` No changed files gives an empty result not an error (Scenario: code_affected with no changed files returns empty) — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -count=1` — Expected: PASS
- [ ] 02.8 `[AFK]` Blast-radius math traverses only resolved edges (Scenario: Dropped references are excluded from blast radius) — Business rule "Drop rather than guess" inherited by `code_affected`
  - [ ] 02.8.a Write failing test — on a fixture with a reference dropped at extraction (step 01 drop policy) and one resolved, `code_affected` traverses only the resolved edges: the dropped/ambiguous reference contributes nothing, so a false positive can never inflate the reported test-file radius
  - [ ] 02.8.b Run to confirm — `Run: go test ./skillgrid-cli/internal/mnemonic/affected/... -run BlastRadius -count=1` — Expected: PASS
- [ ] 02.9 `[RED]` Mnemonic tool surface — `code_affected` + `code_rename` register, rename disambiguation, dry_run touches nothing, 005/008 tools stable (Scenario: affected and rename tools register and stay stable) — threat: Mnemonic tool surface
  - [ ] 02.9.a Write failing test — assert `code_affected` + `code_rename` register with distinct `code_*` names; `code_affected` exposes `--stdin`/`--base <ref>`/`--depth`/`--filter`/`--json`/`--quiet`; rename disambiguation returns a candidate list (no silent pick); `dry_run` touches nothing; 005/008 tools (incl. `code_search`) still stable; bad affected/rename args rejected with a clear validation error
  - [ ] 02.9.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run AffectedTools -count=1` — Expected: FAIL
  - [ ] 02.9.c Minimal implementation — `tools_code_affected.go` + server registration without dropping existing `code_*`
  - [ ] 02.9.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run AffectedTools -count=1` — Expected: PASS
  - [ ] 02.9.e Commit — `feat(mnemonic): register code_affected and code_rename tools`

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/cmd/skillgrid/... -count=1` | PASS | | |
| Acceptance `@step-02` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `git diff --name-only \| skillgrid search affected --stdin` on fixture repo; `skillgrid search affected --base <ref>` on a fixture branch (areas + owners); `skillgrid search rename` dry-run | PASS | | |
| Rollback boundary | Drop `affected/` + `tools_code_affected.go`; revert `code_intel.go` | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): code_affected (--base) and code_rename tools`

---

