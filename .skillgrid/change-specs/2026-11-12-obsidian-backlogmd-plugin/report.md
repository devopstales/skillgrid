# Report — Obsidian Backlog.md Board plugin

> Change: `.skillgrid/change-specs/2026-11-12-obsidian-backlogmd-plugin/` (moves to `.skillgrid/change-archive/2026-11-12-obsidian-backlogmd-plugin/` at ship)
> Generated: 2026-11-13T10:45:56Z (qa)
> Gate: FAIL

## Test Plan

> Derived from `blueprint.md`, `tasks.md`, and `acceptance.feature`.

### Risk Ranking

The most likely production failure is **silent data loss or corruption of task notes**: the plugin rewrites frontmatter in place over files the backlog CLI also owns, so a serializer that drops body bytes, field order, or the `ordinal`/`updated_date` invariants silently desyncs the CLI's board and the vault. Second: the **two-pull GitHub sync** overwriting local planning fields (sprint/rank/points/priority/links) that the user set after the first pull. Third: UI-only behaviors (drag, autosave, watcher, secret storage) that cannot be unit-tested in this repo and rest on live-probe evidence.

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Frontmatter round-trip preserves body byte-for-byte; only patched field changes | `infrastructure/vault` (parseTask/serializeTask/updateFrontmatter) | unit | `vault.test.ts::"round trip body stable"` + `vault.test.ts::"updateFrontmatter changes only patched field"` | P0 | pr | covered |
| 2 | MoveTask rewrites status/ordinal/updated_date, bodies intact; empty column writes nothing | `application/moveTask` | unit | `move-task.test.ts::"writes status, ordinal and updated_date, keeping body"` + `"reassigns 1000-step ordinals in the given order, bodies intact"` + `"writes nothing for an empty column"` | P0 | pr | covered |
| 3 | Bidirectional link writes both sides; remove unlinks both | `application/linkTasks` (src/application/link-tasks.ts:4) | unit | `tests/application/link-tasks.test.ts` (not yet written) | P0 | pr | pending |
| 4 | createTask writes `<key>-NNN.md` with prefixed id (gap-aware numbering) | `application/createTask` (src/application/create-task.ts:26) | unit | id gen: `create-task.test.ts` (5 nextTaskId tests); write path: not yet tested | P0 | pr | covered (id) / pending (write path) |
| 5 | GitHub two-pull semantics: first pull creates with source + code-blocked body; second syncs title/assignee/state only | `application/pullIssues` | unit | `pull-issues.test.ts::"first pull creates tasks with source ref and code-blocked body"` + `"second pull syncs title/assignee/state only, keeps local planning and body"` + `"assigns collision-free ids when several new issues arrive at once"` | P0 | pr | covered |
| 6 | Project discovery: scan, manual roots, dedupe, honors backlog_directory, rejects out-of-vault ancestors | `infrastructure/discoverProjects` | unit | `config.test.ts` (7 tests incl. `"merges manual roots with the scan and dedupes"`, `"honors backlog_directory when the project root is the vault root"`) | P1 | pr | covered |
| 7 | Malformed frontmatter → error surface, no crash | `infrastructure/vault` parse → `ui/kanban-view` render (kanban-view.ts:57-62 rethrows; no error-card element rendered) | manual (Obsidian runtime) | scenario `frontmatter-malformed-surfaces-error` — live-probe only | P1 | pr | pending |
| 8 | Milestone parse (CLI format), state derivation, complete/plan/unplan | `domain/milestone` + `application/milestone` | unit | `milestone.test.ts` (domain 5 + application 4 tests) | P1 | pr | covered |
| 9 | isDone/isBlocked domain rules (blocked icon predicate) | `domain/task` | unit | `task.test.ts` (3 tests: done, blocked true, blocked false) | P1 | pr | covered |
| 10 | UI: kanban columns/cards/drag, backlog filter, modal summary-required + autosave, watcher refresh, commands, cmd-click, settings persist, token in secret storage | Obsidian runtime (ItemView/Modal/Setting/secret storage) | manual smoke (not in `testing.layers`) | 15 scenarios — live-probe in session record, not in spec zone | P1 | nightly | pending |

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| R2 frontmatter | multiline body, arrays, quoted wikilinks | body byte-identical after round-trip | `vault.test.ts::"round trip body stable"` | covered |
| R2 frontmatter | `ordinal` present / absent | preserved on round-trip / undefined when absent | `vault.test.ts::"parse preserves ordinal and serialize round-trips it"` + `"parse leaves ordinal undefined when absent"` | covered |
| R2 frontmatter | invalid YAML frontmatter | error surface, no crash | scenario `frontmatter-malformed-surfaces-error` (no test; parseTask throws at vault.ts:11, kanban-view rethrows) | pending |
| R5 create-task | empty tracker | id starts at `<KEY>-001` | `create-task.test.ts::"starts at 001 when no tasks exist"` | covered |
| R5 create-task | deletions leave id gaps | max numeric suffix + 1, not count | `create-task.test.ts::"takes max numeric suffix + 1 (not count — deletions leave gaps)"` | covered |
| R5 create-task | non-numeric / unrelated ids in tracker | ignored | `create-task.test.ts::"ignores IDs without a numeric suffix"` + `"uppercases the prefix"` | covered |
| R3 move-task | empty target column | writes nothing | `move-task.test.ts::"writes nothing for an empty column"` | covered |
| R9 pull | several new issues arrive at once | collision-free ids | `pull-issues.test.ts::"assigns collision-free ids when several new issues arrive at once"` | covered |
| R9 pull | existing id case differs | matched, not recreated | `pull-issues.test.ts::"is case-insensitive and ignores subtask or unrelated ids"` | covered |
| R1 discovery | ancestor whose backlog dir sits outside the vault | rejected | `config.test.ts::"rejects an ancestor whose backlog directory is not inside the vault"` | covered |
| R1 discovery | vault with no backlog config | zero projects | `config.test.ts::"finds no project in a vault without any backlog config"` | covered |
| R1 discovery | walk depth limit | stops at maxDepth | `config.test.ts::"walkUpForFile stops at maxDepth"` | covered |

### Out of Scope

- Obsidian runtime internals (drag gesture mechanics, `Platform.isMacOS` cmd-click semantics, secret-storage engine) — covered by Obsidian's API contract; plugin-side wiring smoke-tested live only.
- GitHub/GitLab server behavior — mocked via the `IssueSource` port; the adapter (`infrastructure/github.ts`) is thin passthrough.
- Mobile layout — desktop-only per briefing constraint.
- Backlog CLI behavior — owned by the CLI; the plugin never writes `index.json`/`AGENTS.md` (briefing Implementation Decisions).

## Goal-Backward Verification

**Stated goal** (from briefing.md): give Obsidian users a kanban/backlog view + create/edit/move affordances over the backlog.md files already in their vault, without duplicating or re-owning the data.

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | SC1: a vault folder with a backlog config is auto-discovered and appears in the project switcher within one scan | `config.test.ts` (scan + skillgrid-vault + legacy `.backlog/config.yml` discovery) | VERIFIED (discovery logic); switcher render live-only → PRESENT_BEHAVIOR_UNVERIFIED |
| Truth | SC2: every task file renders as a card in the column matching its `status`, using configured statuses | no test (UI); live-probe only | PRESENT_BEHAVIOR_UNVERIFIED |
| Truth | SC3: dragging a card rewrites only that task's `status` frontmatter and writes the file | `move-task.test.ts` (command: status+ordinal+updated_date, body kept) | VERIFIED (command); drag gesture live-only → PRESENT_BEHAVIOR_UNVERIFIED |
| Truth | SC4: create/edit persist via frontmatter round-trip; body preserved byte-for-byte | `vault.test.ts` round-trip + updateFrontmatter | VERIFIED (serializer); createTask write path untested → PARTIAL |
| Truth | SC5: dependency/parent links written on both sides; blocked icon iff a dependency is not done | `task.test.ts` (isBlocked predicate) | VERIFIED (predicate); both-side write (link-tasks.ts) UNTESTED; icon render live-only |
| Truth | SC6: external edits refresh views within ~1s via file watcher | no test (Obsidian runtime); live-probe only | PRESENT_BEHAVIOR_UNVERIFIED |
| Truth | SC7: desktop only, Obsidian ≥ 1.13 | `manifest.json` `"minAppVersion": "1.13.0"` + `versions.json` | VERIFIED |
| Artifact | Plugin scaffold building main.js (`package.json`, `esbuild.config.mjs`, `src/main.ts`) | `npm run build` exit 0 this session | VERIFIED |
| Artifact | Views: kanban / backlog list / milestone / modals / project switcher | files exist (`src/ui/*`), wired in `src/compose.ts`; live-probe load OK | PRESENT_BEHAVIOR_UNVERIFIED |
| Artifact | Settings tab + GitHub client | `src/settings/settings-tab.ts`, `src/infrastructure/github.ts` exist and are wired | PRESENT_BEHAVIOR_UNVERIFIED |
| Key Link | drag → moveTask → vault.updateFrontmatter → file on disk | `move-task.test.ts` exercises command→repo.write with mocked port; drag→command wiring live-only | PRESENT_BEHAVIOR_UNVERIFIED |
| Key Link | create modal → createTask → repo.write | id gen tested; command→write untested | UNVERIFIED (write path) |
| Key Link | edit modal → linkTasks → both sides written | no test references linkTasks | UNVERIFIED |
| Key Link | watcher event → debounced re-render | no test (runtime) | PRESENT_BEHAVIOR_UNVERIFIED |
| Data Flow | task file bytes → parse → card fields → serialize → file bytes | `vault.test.ts` real fixture input, observable output | VERIFIED |
| Data Flow | GitHub issues → task notes (create + re-sync) | `pull-issues.test.ts` real issue fixtures, observable written tasks | VERIFIED (with mocked IssueSource) |

## Traceability Matrix

> `acceptance.feature` is flat Gherkin (no `### Requirement` / `#### Scenario` headings), so the
> traceability script reports total=0 (format mismatch); this matrix is the hand-built
> equivalent — all 26 scenarios enumerated (count verified by `grep -c "Scenario:"` = 26).

| Scenario (from acceptance.feature) | Test / Scenario (from test plan) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| project-discovery-from-scan | `config.test.ts::"discovers a child folder with a legacy .backlog/config.yml from a scan"` + `"discovers a skillgrid vault whose project root sits above the vault"` | yes | pass |
| project-discovery-manual-root | `config.test.ts::"merges manual roots with the scan and dedupes"` | yes | pass |
| project-discovery-dedupe | `config.test.ts::"merges manual roots with the scan and dedupes"` | yes | pass |
| honors-backlog-directory | `config.test.ts::"honors backlog_directory when the project root is the vault root"` | yes | pass |
| frontmatter-roundtrip | `vault.test.ts::"round trip body stable"` + `"updateFrontmatter changes only patched field"` (asserts body preserved, not byte-identical file) | yes | pass (PARTIAL — byte-identical file claim not asserted) |
| frontmatter-malformed-surfaces-error | none (parseTask throws at vault.ts:11; kanban-view.ts:57-62 rethrows; no error-card element) | no | pending |
| kanban-columns-from-statuses | none (UI) | no | pending |
| kanban-card-in-status-column | none (UI) | no | pending |
| drag-move-updates-status | `move-task.test.ts` (command verified; drag gesture live-only) | yes (command) | pass (PARTIAL) |
| backlog-list-filter | none (UI) | no | pending |
| create-task-requires-summary | none (UI) | no | pending |
| create-task-writes-file | `create-task.test.ts` (id prefix/numbering); createTask write path untested | yes (id) | pass (PARTIAL) |
| edit-task-autosave | none (UI; editTask command also untested) | no | pending |
| bidirectional-link-add | none (link-tasks.ts has zero test references) | no | pending |
| bidirectional-link-remove | none (link-tasks.ts has zero test references) | no | pending |
| blocked-icon | `task.test.ts::"isBlocked true when dependency not done"` + `"isBlocked false when dependency done"` (domain predicate; icon render live-only) | yes (predicate) | pass (PARTIAL) |
| milestone-filter | none (UI) | no | pending |
| milestone-list-loads | `milestone.test.ts` (domain parse/state; repo-level load untested) | yes (domain) | pass (PARTIAL) |
| watcher-refresh-on-external-edit | none (Obsidian runtime) | no | pending |
| github-pull-creates-tasks | `pull-issues.test.ts::"first pull creates tasks with source ref and code-blocked body"` | yes | pass |
| github-pull-syncs-without-overwrite | `pull-issues.test.ts::"second pull syncs title/assignee/state only, keeps local planning and body"` | yes | pass |
| github-pull-descriptions-code-blocked | `pull-issues.test.ts::"first pull creates tasks with source ref and code-blocked body"` | yes | pass |
| commands-registered | none (plugin runtime) | no | pending |
| cmd-click-opens-note | none (UI) | no | pending |
| settings-persist | none (Obsidian runtime) | no | pending |
| token-in-secret-storage | none (Obsidian runtime) | no | pending |

**Coverage:** 7/26 scenarios covered by a test that ran and passed (COMPLIANT). 5 PARTIAL (command/domain verified, UI or write path not). 14 UNTESTED (no runnable test; 12 UI/Obsidian-runtime live-probe scenarios + frontmatter-malformed + createTask write path — see Gap Audit). Full suite this session: **38/38 pass** (`npm test`, 8 test files).

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| Bidirectional link writes have no test | `src/application/link-tasks.ts:4` (linkTasks) | regression | `grep -rn "linkTasks\|link-tasks" tests/` → zero references; scenarios `bidirectional-link-add`/`bidirectional-link-remove` depend on both-side write | delete one `repo.write` call — both scenarios fail, no test catches it |
| editTask command has no test | `src/application/edit-task.ts:4` | regression | `grep -rn "editTask\|edit-task" tests/` → zero references; scenario `edit-task-autosave` (UI) + edit persistence (SC4) | drop the `updated_date` stamp — no test catches it |
| createTask write path untested | `src/application/create-task.ts:26` | regression | only `nextTaskId` (create-task.ts:17) is tested; `grep -rn "createTask" tests/` → zero references to the command | write to wrong folder or drop body — id-gen tests still pass |
| Malformed-frontmatter error path untested | `src/infrastructure/vault.ts:10-11` (matter() throws) → `src/ui/kanban-view.ts:57-62` (catch logs + rethrows; no error-card element rendered) | regression | no test feeds invalid YAML; the seam rethrows to the view framework instead of rendering an error card, so "error card shown" (R2 acceptance) is unproven at this seam | wrap parse in a swallowing catch — board goes blank silently, no test fails |
| No oracles in spec zone | `acceptance.feature` (entire file) | broken-verification | zero `#### Gates` blocks — no G<n>, no CHECK/EXPECT, no manual EVIDENCE for any of the 26 scenarios, while every prior archived change carries them (e.g. `.skillgrid/change-archive/2026-10-06-context-harness-clm/acceptance.feature`: `#### Gates` + `# CHECK:` + `# EXPECT:` + `# EVIDENCE:`) | a future re-run of qa cannot mechanically confirm any scenario; coverage rests on hand-built inference |

## TDD Evidence Audit

> Per `_shared/references/strict-tdd.md` § TDD Cycle Evidence Table.
> Execution was committed test + implementation in the same commit per ticket (Standard mode, `testing.tdd: false`); RED/GREEN are therefore combined — per the 2026-09-21-session-events-layer precedent this is `MISSING_RED` → WARNING, not a gate driver on its own.

| Task | SATISFIES | RED (commit + exit) | GREEN (commit + exit) | File | Verdict |
|---|---|---|---|---|---|
| TASK-121 | (scaffolding — no scenario) | N/A (no scenario) | `9befb7e4` (npm run build exit 0) | `esbuild.config.mjs` | N/A (no scenario) |
| TASK-122 | task-model (→ blocked-icon) | `9befb7e4` (combined) | `9befb7e4` (npm test exit 0) | `tests/domain/task.test.ts` | MISSING_RED |
| TASK-123 | frontmatter-roundtrip | `9befb7e4` (combined) | `9befb7e4` (npm test exit 0) | `tests/infrastructure/vault.test.ts` | MISSING_RED |
| TASK-124 | project-discovery-from-scan, project-discovery-manual-root, project-discovery-dedupe, honors-backlog-directory | `2958cdd8` (combined) | `2958cdd8` (npm test exit 0) | `tests/infrastructure/config.test.ts` | MISSING_RED |
| TASK-125 | move-task, create-task, edit-task, link-tasks, bidirectional-link-add, bidirectional-link-remove, blocked-icon | `2958cdd8` (combined) | `2958cdd8` (npm test exit 0) | `tests/application/move-task.test.ts` | MISSING_RED |
| TASK-126 | (wiring — no scenario) | N/A (no scenario) | `2958cdd8` (npm run build exit 0) | `src/compose.ts` | N/A (no scenario) |
| TASK-127 | kanban-columns-from-statuses, kanban-card-in-status-column, drag-move-updates-status, cmd-click-opens-note | N/A (UI — no unit test file; live-probe) | `7b956294` (npm run build exit 0) | `src/ui/kanban-view.ts` | MISSING_RED |
| TASK-128 | backlog-list-filter | N/A (UI — no unit test file; live-probe) | `7b956294` (npm run build exit 0) | `src/ui/backlog-view.ts` | MISSING_RED |
| TASK-129 | create-task-requires-summary, create-task-writes-file, edit-task-autosave | N/A (UI — no unit test file; live-probe) | `7b956294` (npm run build exit 0) | `src/ui/task-modal.ts` | MISSING_RED |
| TASK-130 | watcher-refresh-on-external-edit | N/A (Obsidian runtime) | `7b956294` (npm run build exit 0) | `src/infrastructure/watcher.ts` | MISSING_RED |
| TASK-131 | settings-persist, token-in-secret-storage | N/A (Obsidian runtime) | `b1e4bd41` (npm run build exit 0) | `src/settings/settings-tab.ts` | MISSING_RED |
| TASK-132 | milestone-filter, milestone-list-loads | `2958cdd8` (domain tests, combined) | `23747a5c` (npm run build exit 0) | `tests/domain/milestone.test.ts`, `src/ui/milestone-view.ts` | MISSING_RED |
| TASK-133 | github-pull-creates-tasks, github-pull-syncs-without-overwrite, github-pull-descriptions-code-blocked | `8c3cad7b` (combined) | `8c3cad7b` (npm test exit 0) | `tests/application/pull-issues.test.ts` | MISSING_RED |
| TASK-134 | commands-registered, cmd-click-opens-note | N/A (UI — no unit test file; live-probe) | `dfadf488` (npm run build exit 0) | `src/ui/project-switcher.ts` | MISSING_RED |

**Validation notes:**
- All GREEN hashes resolve in git history (commits 9befb7e4, 2958cdd8, 7b956294, 23747a5c, b1e4bd41, 8c3cad7b, dfadf488); named test files exist on disk; GREEN re-run this session = 38/38 pass.
- `scripts/tdd-evidence.mjs` reports 14/14 OK / scope COMPLETE: its check verifies hash resolvability + test-file existence only, not RED-before-GREEN ordering. The skill's audit rule ("no RED commit before GREEN → MISSING_RED") still applies because RED and GREEN are the same commit — hence the per-row MISSING_RED verdicts above and the W-1 WARNING.
- SATISFIES names in `tasks.md` are approximate where they don't match a scenario exactly (`task-model`, `move-task`, `create-task`, `edit-task`, `link-tasks`); mapping to `acceptance.feature` scenarios is by intent (see W-6).
- TRIANGULATE: TASK-124 (4 scenarios, 4+ tests), TASK-125 (7 scenario names, 1 test file — move-task only; link/create/edit untested — see Gap Audit), TASK-133 (3 scenarios, 3 tests).

**Summary:** 0/14 tasks have complete TDD evidence (12 MISSING_RED, 2 N/A — no scenario). Standard-mode precedent (2026-09-21) → WARNING.

## Assertion Quality Audit

| File | Line | Pattern | Evidence | Severity |
|---|---|---|---|---|
| `tests/application/move-task.test.ts` | updated_date assertions | impl-detail / weak value assertion | `expect(updated.updated_date).toBeTruthy()` — presence only; no format or value check | WARNING |
| `tests/application/milestone.test.ts` | start_date assertions | weak value assertion | `expect(...start_date).toBeTruthy()` — presence only | WARNING |
| `tests/infrastructure/vault.test.ts` | "round trip body stable" | snapshot-adjacent (toContain-only) | asserts body contained, not that the serialized file is byte-identical — the SC4 byte-for-byte claim rests on this | SUGGESTION |

**Summary:** 0 CRITICAL, 2 WARNING, 1 SUGGESTION.

## Changed-File Coverage

Coverage: n/a — `testing.coverage` empty (no coverage tool configured for the plugin).

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| — | none | All 38 tests exercise production code (domain rules, commands over mocked ports, adapter round-trip) with realistic fixture shapes; no vacuous/pass-always patterns found. Assertion-strength items (weak date assertions) are recorded under Assertion Quality, not contract violations. |

## Test Strategy Audit

**Available layers** (from `testing.layers`): unit, integration.

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|----------|---------------|----------|---------------------|-----------|
| Domain rules (task, milestone) | unit | no | no | pure functions — unit is the highest fitting layer |
| Application commands (move/create/edit/link/milestone/pull) | unit with mocked ports | yes (integration would need a real vault; the Obsidian runtime is not injectable without a live vault) | no | ports are the seam; mocks return realistic Task/file shapes |
| Frontmatter adapter (gray-matter) | unit with fixture files | no | no | file bytes are the contract |
| Config discovery | unit with fixture directory trees | no | no | filesystem walking is testable without Obsidian |
| UI views, modals, watcher, settings, secret storage | manual smoke (Obsidian desktop) | yes (not in `testing.layers`) | no | Obsidian runtime APIs (ItemView, Modal, PluginSettingTab, SecretStorage) have no headless harness in this repo |

**Layer distribution:** 38 unit, 0 integration (mocked ports stand in), 15 manual-smoke scenarios. Reasonable for this risk profile: all data-mutation logic sits at the unit-testable seam; the manual surface is the UI shell the briefing explicitly assigned to live smoke (Testing Decisions).

## Security Audit

**Mode:** Trivy (Mode A) + manual spot-check.

### Trivy Findings

**Command:** `trivy fs plugins/obsidian/backlogmd --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW`

| Scanner | Finding | Severity | Location | Fix | Classification |
|---------|---------|----------|----------|-----|----------------|
| vuln | none | — | `plugins/obsidian/backlogmd/package-lock.json` (0 findings) | — | — |

Repo-wide run additionally found 13 findings (seroval HIGH ×3 + MEDIUM, lodash-es MEDIUM) — all inside gitignored `.worktrees/mnemonic-llm-provider/` (other workstream's worktree; not in this change's footprint).

**Trivy gate:** `fail_on: ""` (advisory) → PASS.

### Manual Findings

**Secrets Archaeology:** no tokens, keys, or URLs with credentials in the plugin diff; the GitHub token flows to Obsidian's `SecretStorage` per design (R11); `data.json` is user-vault state, not committed to this repo.

**OWASP Spot-Check (A01/A03/A06):** the GitHub adapter makes HTTPS calls with the token as a header (not in URL query); task-note bodies from issues are written into a fenced code block by default, so Markdown-in-description is not rendered as HTML (mitigates injection into the board); no `innerHTML` from untrusted content in `src/ui/*` (checked — elements built via `createEl`/text APIs).

**Security verdict:** PASS — 0 findings in change footprint (CRITICAL: 0, WARNING: 0, SUGGESTION: 0); advisory environmental findings outside footprint noted.

## Code Quality Gate

| Gate | Command | Threshold (config) | Actual | Result |
|------|---------|--------------------|--------|--------|
| Coverage (whole project) | not configured (`testing.coverage` "") | `quality.coverage_min` = 0 | N/A | N/A |
| Changed-file coverage | not configured | 0 | N/A | N/A |
| Mutation | not configured | `quality.mutation_min` = 0 | N/A | N/A |
| Lint | not configured (`commands.lint` "") | errors only | N/A (tsc covers typecheck) | N/A |
| Typecheck | `npm run build` (tsc -noEmit -skipLibCheck + esbuild) | errors only | 0 errors (exit 0) | PASS |
| P0 pass rate | `npm test` (plan rows P0) | `quality.p0_pass_rate` = 100 | 100% (38/38; P0 rows 1-5 all green) | PASS |
| P1 pass rate | `npm test` (plan rows P1) | `quality.p1_pass_rate` = 95 | 100% | PASS |
| Trivy security | `trivy fs` (above) | `fail_on: ""` advisory | 0 in footprint | PASS (advisory) |
| Duplication (jscpd) | `npx jscpd` (src/ tests/) | `fail_on: 0` advisory | 11 clones · 3.5% | PASS (advisory) |

**Quality config status:** configured (quality section present; coverage/mutation empty → N/A).

**Dead code:** N/A (not configured).

## Duplication

**Command:** `npx jscpd` scoped to `src/` + `tests/` (the entire change — new plugin project).

**Result:** 11 clones · 3.5% duplication — scope: COMPLETE.

| Files (lines) | Kind | Introduced? | Severity | Note |
|---------------|------|-------------|----------|------|
| `src/ui/kanban-view.ts` ~ `src/ui/backlog-view.ts` ~ `src/ui/milestone-view.ts` (render loop / filter-bar scaffolding) | token-sequence | yes | SUGGESTION | view boilerplate; a small shared render helper would collapse it — advisory, `fail_on` 0 |
| `tests/application/move-task.test.ts` ~ `tests/application/milestone.test.ts` (fixture + write-assertion scaffolding) | token-sequence | yes | SUGGESTION | test boilerplate; acceptable at this size |
| `src/ui/task-modal.ts` internal (create vs edit form setup) | token-sequence | yes | SUGGESTION | two forms share field wiring; intentional parity |

## State Drift

**Verdict:** DRIFT DETECTED (at render time) — self-resolving.

| Field | Stale (state.yaml) | Derived (spec zone) |
|-------|-------------------|---------------------|
| pipeline.current_phase | qa | slicing (no `[x]` in tasks.md, no report.md yet) |

**Fix applied:** no state.yaml patch — writing this report.md flips the derived phase to `qa`, matching state. (`tasks.md` was never `[x]`-marked because execution was tracked via the backlog CLI; recorded as W-5.)

**Scope (from the guard's `SCOPE:` line):** UNSCOPED (ship_drift: no base ref configured for this change).

## Verification Scope

| Derivation | Scope | Stale? |
|---|---|---|
| Traceability matrix | COMPLETE (acceptance.feature read end-to-end; 26 scenarios enumerated by hand — script reports 0 due to flat-Gherkin format mismatch, count cross-checked with `grep -c`) | none |
| Verification-gap audit | COMPLETE (full changed-file list from the change's 7 code commits; every `src/application/*.ts` command inspected) | none |
| State Drift (9.5) | UNSCOPED (guard `SCOPE:` line) | n/a |
| Structure Drift (9.6) | n/a — not reported by qa-gate in this run | n/a |

**Stale-verification:** `STALE: none` — all evidence (38/38 vitest, tsc build, trivy, jscpd, go regression) executed in this session against the current working tree.

## Floor

| Dimension | Verdict |
|---|---|
| Goal-backward verification (weakest truth) | UNVERIFIED (both-side link write, createTask write path — no test) |
| Traceability (weakest scenario) | UNTESTED (14 scenarios) |
| Verification-gap audit | CRITICAL (missing-oracle + 4 untested seams) |
| TDD evidence (weakest ticket) | MISSING_RED (12 tickets) |
| Assertion quality audit (weakest finding) | WARNING |
| Changed-file coverage (weakest file) | N/A |
| Test quality audit | none |
| Code-quality gates (weakest gate) | PASS |
| Security audit | PASS (advisory) |
| Verification scope (composite) | UNSCOPED (ship_drift base ref) |

**FLOOR:** FAIL — driven by the Verification-Gap audit (CRITICAL): the spec zone carries no oracles for any of the 26 scenarios (an explicit FAIL criterion), and four changed code seams the scenarios depend on have no runnable test. A strong unit suite (38/38, P0/P1 100%) cannot lift the floor while acceptance oracles and the link/edit/create/malformed seams are unverified.

## Findings

> Open WARNING/SUGGESTION findings below are auto-appended to
> `.skillgrid/WINDOWS.md` (the cross-change defect register) at gate-render
> time. They survive the archive move.

### CRITICAL (must fix before merge)

- **C-1 — No oracles in spec zone.** `acceptance.feature` contains zero `#### Gates` blocks: no G<n>, no `# CHECK:` / `# EXPECT:` / `# EVIDENCE:` for any of the 26 scenarios. Every prior archived change carries runnable oracles (reference: `.skillgrid/change-archive/2026-10-06-context-harness-clm/acceptance.feature`). Without them, qa cannot mechanically confirm any scenario on re-run. **Fix (spec zone):** backfill `#### Gates` per scenario — runnable `npx vitest run <file> -t "<name>"` CHECKs where a test exists; manual `# EVIDENCE:` citing the live-probe for the 12 Obsidian-runtime scenarios.
- **C-2 — Changed seams with no test.** `linkTasks` (`src/application/link-tasks.ts:4`), `editTask` (`src/application/edit-task.ts:4`), `createTask` write path (`src/application/create-task.ts:26`), and the malformed-frontmatter error path (`src/infrastructure/vault.ts:10-11` throws; `src/ui/kanban-view.ts:57-62` rethrows without rendering an error card) have zero test references — scenarios `bidirectional-link-add`, `bidirectional-link-remove`, `edit-task-autosave` (persistence half), `create-task-writes-file` (write half), `frontmatter-malformed-surfaces-error` are unverified. **Fix (code zone):** add `tests/application/link-tasks.test.ts`, `tests/application/edit-task.test.ts`, extend `tests/application/create-task.test.ts` with a write-path test (mocked repo), and a malformed-YAML parse test.

### WARNING (should fix before archive)

- **W-1 (→ W012) — MISSING_RED on all 14 tickets** (test + implementation combined in one commit per ticket; Standard mode, `testing.tdd: false`). Per the 2026-09-21-session-events-layer precedent → WARNING, not a gate driver.
- **W-2 (→ W013) — Pre-existing Go regression.** `go test ./...` in `mnemonic/` fails `TestNoStalePathReferences` (cmd/mnemonic): 6 stale `mnemonic mcp` / `mnemonic-v2/` references under `.skillgrid/projectwiki/backlog/{milestones,m--11,tasks/task-061,tasks/task-068,tasks/task-084 ×2,tasks/task-115.02}`, all last touched by 97b6af15 (backlog relocate — prior change); the test's path exclusion set lacks `backlog/`. Not caused by this change (zero Go touched) but it violates the briefing's "Go green regression floor". Fix: extend the exclusion set for `.skillgrid/projectwiki/backlog/` or rewrite the 6 lines.
- **W-3 — State drift at render time** (qa vs derived slicing) — resolved by this report.md landing; no state.yaml edit needed.
- **W-4 — `briefing.md` STATUS is `draft`** while the change is in qa (status_banner advisory) — update the STATUS line at ship or waive.
- **W-5 — `tasks.md` tickets never `[x]`-marked** (execution tracked via the backlog CLI; all 14 tickets closed in the tracker) — reconcile marks or note the tracker-waiver at ship.
- **W-6 — SATISFIES names in `tasks.md` are approximate** (`task-model`, `move-task`, `create-task`, `edit-task`, `link-tasks` have no exact scenario-name match) — mapped by intent; record the mapping in tasks.md at reconciliation to keep the STALE-class check honest.

### SUGGESTION (nice to have)

- **S-1 (→ W014) — jscpd 11 clones · 3.5%** in the new plugin (UI view boilerplate + test scaffolding) — advisory, `fail_on` 0; a shared view render helper would collapse the UI clones.
- **S-2 (→ W015) — Assertion strength:** `toBeTruthy()` on date fields (`move-task.test.ts`, `milestone.test.ts`) and toContain-only round-trip (`vault.test.ts`) — strengthen to exact-value / byte-identical assertions when the C-2 tests land.
- **S-3 — size_budget:** `.skillgrid/projectwiki/` SKILL.md +1069 lines over budget (pre-existing, outside this change's footprint).
- **S-4 — okf_lint:** 5 provenance-missing findings (pre-existing, outside this change's footprint).

## Gate Decision

**Verdict:** FAIL

**Reasoning:** Two CRITICAL findings drive the verdict. C-1: the spec zone has no oracles — no `#### Gates` block for any of the 26 scenarios, an explicit FAIL criterion (every prior archived change carries them). C-2: four changed seams the scenarios depend on (linkTasks, editTask, createTask write path, malformed-frontmatter error path) have zero runnable tests, so 14/26 scenarios are UNTESTED. Everything else is green — 38/38 vitest, tsc clean, trivy clean in-footprint, P0/P1 100%, typecheck PASS — but the floor is set by the weakest dimension, and a strong unit suite cannot substitute for missing acceptance oracles and untested mutation seams.

**Re-run path:**
1. **A (spec zone):** backfill `#### Gates` blocks in `acceptance.feature` (runnable vitest CHECKs where tests exist; manual EVIDENCE citing the session live-probe for the 12 Obsidian-runtime scenarios).
2. **B (code zone):** add vitest tests for `linkTasks` (both add and remove), `editTask` (patch + updated_date), `createTask` (write path via mocked repo), and malformed-YAML parse (expect throw / error surface).
3. Re-run `node .agents/skills/qa/scripts/qa-gate.mjs` + this report's audits; verdict target: PASS or CONCERNS (MISSING_RED + pre-existing Go fail remain WARNING-class).

**Open items (if CONCERNS):** n/a — verdict is FAIL, not CONCERNS.

## Human Override

> A human decision always overrides this machine verdict.
> An epic that fails its criteria with no human decision is recorded as **not accepted** — never as silently accepted.

**Human decision (fill in):** —

<!-- reflect completes this half at archive time -->

## Final-State Facts

**Shipped:** <what actually shipped>
**Base branch:** <base-branch> · **Chain strategy:** <strategy>
**Integration:** <merged @ <commit> | PR <url> | kept branch <name>> (from ship context)

## Gates

| Gate | Result |
|------|--------|
| Ship gate | <pending — ship runs after gate re-pass> |
| QA gate | ❌ FAIL (this render; re-run after C-1/C-2 fixes) |
| Verdict gate (advisory) | <pending> |

## Decisions

| Decision | Tradeoff | Why | Source |
|----------|----------|-----|--------|
| <pending — reflect> | | | |

## Lessons

| Lesson | Root Cause | Do Differently | Source |
|--------|-----------|----------------|--------|
| <pending — reflect> | | | |

## Patterns

| Pattern | Reuse | Source |
|---------|-------|--------|
| <pending — reflect> | | |

## Surprises

| Surprise | Signal | Source |
|----------|--------|--------|
| <pending — reflect> | | |

## Environment Retro

| Category | Finding | Fix lands in | Source |
|----------|---------|--------------|--------|
| <pending — reflect> | | | |

## Acceptance Verdict

**Verdict:** <pending — reflect>
**Grounding:** <pending — reflect>
**Reasoning:** <pending — reflect>

## Open Items (→ next change)

- <pending — reflect>

## Web-cache promotion candidates

No web-cache promotion candidates — web cache disabled/empty.

## Prior-Change Follow-Through

| Prior open item (from previous archived change) | Addressed by this change? | Evidence |
|--------------------------------------------------|---------------------------|----------|
| <pending — reflect> | | |

## Move Evidence (from ship context)

**From:** `.skillgrid/change-specs/2026-11-12-obsidian-backlogmd-plugin/` → **To:** `.skillgrid/change-archive/2026-11-12-obsidian-backlogmd-plugin/`
**`diff -r` readback:** <pending — ship>

## Overrides / Waivers / Contradictions

- None (this render).

## Lineage (observation IDs)

Every artifact read this close, for traceability:

- briefing: `.skillgrid/change-specs/2026-11-12-obsidian-backlogmd-plugin/briefing.md` (ade10952)
- blueprint: `.skillgrid/change-specs/2026-11-12-obsidian-backlogmd-plugin/blueprint.md`
- tasks: `.skillgrid/change-specs/2026-11-12-obsidian-backlogmd-plugin/tasks.md`
- acceptance: `.skillgrid/change-specs/2026-11-12-obsidian-backlogmd-plugin/acceptance.feature` (26 scenarios)
- code commits: 9befb7e4, 2958cdd8, 7b956294, 23747a5c, b1e4bd41, 8c3cad7b, dfadf488
- ship: n/a (not yet run)
- report (QA half): this file
- research / findings / ADRs: none
