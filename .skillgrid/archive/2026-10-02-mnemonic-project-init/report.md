# Report — mnemonic-project-init

> Change: `.skillgrid/specs/2026-10-02-mnemonic-project-init/` (moves to `.skillgrid/archive/2026-10-02-mnemonic-project-init/` at ship)
> Generated: 2026-10-03T07:40:00Z (qa)
> Gate: PASS
>
> Two phases, one file: **qa** writes the QA half (the sections above `## Final-State Facts`,
> through `## Gate Decision` + `## Human Override`) into the spec folder. **ship** reads the
> `## Gate Decision` verdict PRE-MOVE, then moves the folder. **reflect** completes the retro
> half (from `## Final-State Facts` onward) IN PLACE in the archive folder.

**Tier:** T2 · **Floor:** L2 (classification standard) · verification scope: COMPLETE (focused `TestInit*|TestOnboardingSkill*` + `go build ./cmd/skillgrid` + `skillgrid init -h` smoke)

## Test Plan

### Risk Ranking

Most likely production break: boot-file write succeeds but ingest/index silently no-ops, leaving a “initialized” repo with empty memory/index. Second: sentinel duplication on re-init polluting every prompt. Tests hit those first.

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Boot file written + counts | CLI `projectInit` | unit | `TestInitWritesBootFileAndReportsCounts` | P0 | pr | covered |
| 2 | Help lists `--force`/`--docs` | CLI flags | unit | `TestInitHelpListsFlags` | P1 | pr | covered |
| 3 | Non-dir boot failure fatal | CLI `projectInit` | unit | `TestInitBootFileWriteFailureIsFatal` | P0 | pr | covered |
| 4 | Preamble + single sentinel | `writeBootFile` | unit | `TestInitUpsertsPreambleAndSentinel` | P0 | pr | covered |
| 5 | `--force` preamble only | `writeBootFile` | unit | `TestInitForceRewritesPreamble` | P0 | pr | covered |
| 6 | Index ≥1 on `.go` fixture | `RunCodeIndex` | unit | `TestInitIndexesTheProject` | P0 | pr | covered |
| 7 | Index failure non-fatal | `initRunIndex` stub | unit | `TestInitIndexFailureIsNonFatal` | P0 | pr | covered |
| 8 | Default path ingest | `ingestPaths` | unit | `TestInitIngestsDefaultPaths` | P0 | pr | covered |
| 9 | Missing `docs/` skipped | `ingestPaths` | unit | `TestInitSkipsMissingDocs` | P1 | pr | covered |
| 10 | Extra `--docs` ingest | `ingestPaths` | unit | `TestInitIngestsExtraDocs` | P0 | pr | covered |
| 11 | Repeated `--docs` | `ingestPaths` | unit | `TestInitIngestsRepeatedDocsFlags` | P1 | pr | covered |
| 12 | Missing extra non-fatal | `ingestPaths` | unit | `TestInitMissingExtraDocsIsNonFatal` | P0 | pr | covered |
| 13 | Docs jail | `ingestPaths` | unit | `TestInitRejectsDocsOutsideProject` | P0 | pr | covered |
| 14 | Onboarding names init | SKILL.md | unit | `TestOnboardingSkillCallsSkillgridInit` | P0 | pr | covered |
| 15 | Skill passes `--docs` | SKILL.md | unit | `TestOnboardingSkillPassesDocsThrough` | P1 | pr | covered |
| 16 | No second ingest in skill | SKILL.md | unit | `TestOnboardingSkillHasNoSecondIngest` | P0 | pr | covered |

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| Boot file | target path is a file | error, non-zero | `TestInitBootFileWriteFailureIsFatal` | covered |
| Index | pipeline fails after boot | exit 0, Errors listed | `TestInitIndexFailureIsNonFatal` | covered |
| Default ingest | no `docs/` | Skipped, still exit 0 | `TestInitSkipsMissingDocs` | covered |
| Extra docs | path outside project | Errors, non-fatal | `TestInitRejectsDocsOutsideProject` | covered |
| Extra docs | missing path | Errors, boot exists | `TestInitMissingExtraDocsIsNonFatal` | covered |

### Out of Scope

- Live IDE onboarding interview (skill prose only; CLI contract tested)
- Shared LLM provider / KBs (explicitly deferred)
- `--json` result shape (not this slice)

## Goal-Backward Verification

**Stated goal:** Run project init and get a lean boot file, forced code index, and local docs in the existing store.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | `skillgrid init` sequences boot + index + ingest | focused suite PASS + `init -h` | VERIFIED |
| Truth | Single sentinel; `--force` preamble only | Upsert + Force tests | VERIFIED |
| Truth | Index non-empty on Go fixture; index fail non-fatal | Indexes + IndexFailure tests | VERIFIED |
| Truth | Default/extra ingest with stable topic keys | Ingest* + Skips + Rejects tests | VERIFIED |
| Truth | Onboarding finishes via CLI, no second ingest | OnboardingSkill* tests | VERIFIED |
| Artifact | `init_cmd.go`, `init_boot.go`, `init_ingest.go`, main dispatch | present + wired | VERIFIED |
| Key Link | `projectInit` → writeBootFile → initRunIndex → ingestPaths | integration via TestInit* | VERIFIED |
| Data Flow | files → `memory.Save` topic `init/docs/<relpath>` | IngestsDefault/Extra/Repeated | VERIFIED |

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario | Ran | Result |
|--------------------------------------|-----------------|-----|--------|
| happy path init writes boot file and reports counts | `TestInitWritesBootFileAndReportsCounts` | yes | pass |
| init help lists force and docs flags | `TestInitHelpListsFlags` | yes | pass |
| boot file write failure is the only fatal error | `TestInitBootFileWriteFailureIsFatal` | yes | pass |
| happy path init upserts preamble and sentinel | `TestInitUpsertsPreambleAndSentinel` | yes | pass |
| second init merges without duplicating the sentinel | `TestInitUpsertsPreambleAndSentinel` (second call) | yes | pass |
| force rewrites the preamble only | `TestInitForceRewritesPreamble` | yes | pass |
| happy path init indexes the project | `TestInitIndexesTheProject` | yes | pass |
| init reuses the existing indexer | `TestInitIndexesTheProject` (`svc.RunCodeIndex`) | yes | pass |
| index failure after boot file still exits 0 | `TestInitIndexFailureIsNonFatal` | yes | pass |
| happy path init ingests default paths | `TestInitIngestsDefaultPaths` | yes | pass |
| missing docs directory is skipped | `TestInitSkipsMissingDocs` | yes | pass |
| second init upserts the same topic keys | `TestInitIngestsDefaultPaths` (re-run count) | yes | pass |
| happy path init ingests extra --docs path | `TestInitIngestsExtraDocs` | yes | pass |
| repeated --docs flags ingest each path | `TestInitIngestsRepeatedDocsFlags` | yes | pass |
| missing extra docs path is listed and non-fatal | `TestInitMissingExtraDocsIsNonFatal` | yes | pass |
| happy path onboarding skill calls skillgrid init | `TestOnboardingSkillCallsSkillgridInit` | yes | pass |
| skill passes extra docs through | `TestOnboardingSkillPassesDocsThrough` | yes | pass |
| skill does not contain a second ingest procedure | `TestOnboardingSkillHasNoSecondIngest` | yes | pass |

**Coverage:** 18/18 scenarios covered by a test that ran and passed.

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| _(none open)_ | | | | |

QA added `TestInitIngestsRepeatedDocsFlags` and `TestOnboardingSkillPassesDocsThrough` before the gate to clear two acceptance scenarios that had no dedicated test.

## TDD Evidence Audit

| Task | SATISFIES Scenario | RED Evidence | GREEN Evidence | Verdict |
|------|--------------------|--------------|----------------|---------|
| T01 | boot/help/fatal | report: undefined `projectInit` | b5239413 | OK |
| T02 | preamble/sentinel/force | report RED | 9dd840f5 | OK |
| T03 | index + non-fatal | report RED | 03439ce7 | OK |
| T04 | ingest defaults/extras | report RED ingest | 8763f9b6 | OK |
| T05 | onboarding init | RED skillgrid init absent | 2324c38e | OK |

## Security Audit

No new network/auth surface. Path jail for `--docs` covered by `TestInitRejectsDocsOutsideProject`. Trivy advisory-only (config). No CRITICAL.

## Code-Quality Gates

| Gate | Result |
|------|--------|
| Typecheck / build | PASS — `go build ./cmd/skillgrid` |
| Focused suite | PASS — `go test ./cmd/skillgrid -run 'TestInit\|TestOnboardingSkill'` |
| Coverage / mutation / lint | N/A (thresholds 0 / unset) |
| P0 / P1 pass rate | 100% of planned rows covered |

## Floor

Weakest dimension: skill contract tests are source-text assertions (acceptable for SKILL.md handoff). Does not drop below PASS.

## Gate Decision

**PASS**

**Reasoning:** All acceptance scenarios mapped to green focused tests; goal-backward truths VERIFIED; no CRITICAL/MISSING_RED; ADR-0012 (one store) preserved. Smoke: `skillgrid init -h` names both flags.

## Human Override

_(none)_

## Final-State Facts

**Shipped:** `skillgrid init [--force] [--docs path]...` on `release/2`. The command writes a lean AGENTS/CLAUDE preamble above a single Skillgrid sentinel (`b5239413`, `9dd840f5`), runs the existing `RunCodeIndex` pipeline (`03439ce7`), and upserts default + extra docs at `init/docs/<relpath>` (`8763f9b6`). Onboarding Step 9 calls that CLI (`2324c38e`). Review jail wave: symlink boot refuse, `EvalSymlinks` + `outsideJail` for `--docs`, 512KiB + NUL skip, help bound to `newInitFlagSet`, extra-walk error accumulation (`cb077edf`). Size/relative-cwd regressions in `3e2794bf`. Spec zone committed in `c32ef909` (D21).
**Base branch:** release/2 · **Chain strategy:** stay-on-release/2 (option 3; `tasks.md` chain strategy was `pending`; serial constraint forbids a parallel branch)
**Integration:** kept branch `release/2` (option 3, not pushed). Archive commit `e76fbb4e`. Focused suite `TestInit*|TestOnboardingSkill*` green at ship; `go build ./...` ok. Known flakes (`TestTTLBoundaryExactSecond`, `TestSearchObservationsAllParallelManyStores`, `TestExecuteGoSkill`, `TestSkillSearchHybridDegraded`) passed in isolation and are outside this change.

## Gates

| Gate | Result |
|------|--------|
| Ship gate | ✅ success + `diff -r` empty (`/usr/bin/diff -r` against the pre-move tree; this shell's `diff` is delta. Git records `R100`.) |
| QA gate | ✅ PASS (18/18 scenarios; QA half 2026-10-03T07:40:00Z) |
| Verdict gate (advisory) | accepted-with-open-items — recorded, not enforced |

## Decisions

**Every entry MUST cite a source** — a file:line, a commit, a ticket ID, or a scenario name. No source = undiagnosed, not a learning.

| Decision | Tradeoff | Why | Source |
|----------|----------|-----|--------|
| Orchestrator CLI (`skillgrid init`) sequences boot + existing index + ingest; onboarding interviews then runs the CLI | No skill-only last mile; no second store | Deterministic floor; ADR-0012 one SQLite store; interview already works | briefing Approaches A; `adr.md`; `2324c38e` |
| Boot-file write is the only fatal error; index/ingest failures list in `Errors` and still exit 0 | A “success” exit can hide a failed index | Briefing error policy: a written boot file is enough to continue | briefing req 1; `TestInitIndexFailureIsNonFatal`; D12 noise |
| `--docs` jail uses `EvalSymlinks` + `outsideJail` (`.` / `..` / `../`), not lexical `HasPrefix` | Extra syscalls; `/var` → `/private/var` must use `realDir` for topic Rel | Lexical jail lost to symlink escape (D1 High) | `init_ingest.go` `outsideJail` / `EvalSymlinks`; `cb077edf` |
| `Lstat` refuses a symlink boot target | Cannot init through a linked AGENTS.md | `WriteFile` would overwrite the host file (D2 High) | `init_boot.go`; `cb077edf` |
| Ingest skips files >512KiB or with a NUL in the first 512 bytes | Large/binary docs stay out of the store | Unbounded WalkDir + binary blob (D3) | `init_ingest.go` `maxIngestBytes`; `3e2794bf` |
| Keep `release/2` (option 3) | No merge to `main`, no PR | Work was already on `release/2`; merging it would merge the whole branch | ship Return Envelope, 2026-10-03; `e76fbb4e` |
| Defer D17 (no `ExpiresAt` on ingest Save) | Docs inherit the store 7-day TTL | Not a new store default; durable-ingest follow-up | `review.md` D17; `saveIngestFile` `SaveInput` |

## Lessons

| Lesson | Root Cause | Do Differently | Source |
|--------|-----------|----------------|--------|
| Spec-zone files must be on the branch before the review floor, not after | `acceptance.feature` / briefing / blueprint / tasks / adr were untracked at first review (D21) | Commit the spec folder as soon as it is accepted, before dispatch | `review.md` D21; `c32ef909` |
| Ship `diff -r` on this host is delta unless you call `/usr/bin/diff` | `diff` is aliased to `delta`; `delta` is not recursive directory compare | Always `/usr/bin/diff -r` for archive readback | ship context, 2026-10-03; memory-improvements `report.md` Gates |
| Help and flag tests must construct the production flag set | `TestInitHelpListsFlags` asserted a hand-built usage string (D4) | Call `newInitFlagSet` (or the real command) so flag drift fails the test | `cb077edf`; `TestInitHelpListsFlags` |
| Stay-on-release/2 is the default when `tasks.md` chain strategy is still `pending` | Serial constraint + a long-lived release branch | Record option 3 in the ship envelope; do not invent a merge | `tasks.md` Delivery Strategy; `e76fbb4e` |

## Patterns

| Pattern | Reuse | Source |
|---------|-------|--------|
| Path jail: `EvalSymlinks` both sides, then `Rel`, then reject `.` / `..` / `../` | Any CLI that walks a user-supplied path inside a project root | `init_ingest.go` `outsideJail`; `cb077edf` |
| `Lstat` before `WriteFile` on a boot/config target | Any writer that must not follow a symlink to a host file | `init_boot.go`; `TestInitBootFileWriteFailureIsFatal` / symlink test in `cb077edf` |
| Size + NUL probe before `memory.Save` of walked files | Any future ingest (init extras, onboarding, second-brain capture of local docs) | `saveIngestFile`; `3e2794bf` |
| Stay-on-release/2 (option 3) when the change already lives on the release branch | Every later ship on this branch until an explicit merge is requested | `e76fbb4e`; memory-improvements ship |

## Surprises

| Surprise | Signal | Evidence |
|----------|--------|----------|
| `Save` without `ExpiresAt` inherits the store’s 7-day TTL | Ingested README/docs can vanish after a week even with stable `topic_key` | `review.md` D17; `saveIngestFile` omits `ExpiresAt` |
| Blank `Owner` falls back to the session UUID, so later sessions may not see the rows | `Scope: "project"` is not enough for cross-session search | `review.md` D18; `mem_save` owner fallback |
| Hash floor runs before `topic_key` upsert | Identical title+content+type within 24h noops a second key | `review.md` D19; second-brain `Save()` sha256 lesson |
| `MNEMONIC_PROJECT` can split boot-file cwd from the opened store | Boot writes in one tree; ingest lands in another project id | `review.md` D20 |
| Review.md cited `9fa312fe` for D3/D6 tests; that SHA is not on `release/2` | The regression commit on this branch is `3e2794bf` | `git log` `3e2794bf`; `review.md` Verdict (stale SHA at re-render) |

## Acceptance Verdict

**Verdict:** accepted-with-open-items

**Grounding:** Briefing goal is “run project init and get a lean boot file, a forced code index, and local docs in the existing store.” QA half: 18/18 scenarios passed, goal-backward truths VERIFIED, verdict PASS (`report.md` Gate Decision, 2026-10-03). Review floor met-with-fixes after `cb077edf` (D1–D7, D16, D21 closed).

**Reasoning:** The in-scope sequencer shipped and the focused suite is green. Open items are the deferred review notes (TTL, owner, hash-floor, `MNEMONIC_PROJECT`, mutable `initRunIndex`, inlined sentinel). They do not reopen a failed scenario. The verdict is accepted-with-open-items rather than accepted because D17 remains the worst leftover and will drop ingested docs after seven days.

## Open Items (→ next change)

- D17: `saveIngestFile` does not set `ExpiresAt`, so ingested docs inherit the store 7-day TTL. Path: durable ingest (zero/`ExpiresAt` far-future, or a store durable-type) in a follow-up, likely with `2026-10-02-mnemonic-llm-provider` only if that change already touches Save; otherwise its own ticket.
- D18: ingest `Owner` is the session UUID → later sessions may miss the rows. Path: set a project-stable owner or document the share/search contract.
- D19: hash floor before `topic_key` can noop a second key for identical content. Path: confirm AUDN/topic upsert order; add a test with two keys, same body.
- D20: `MNEMONIC_PROJECT` can split boot vs store. Path: resolve project before writing the boot file, or refuse a mismatched env.
- D8: package-level mutable `initRunIndex` for the test stub. Path: inject the indexer.
- D9: inlined `sentinelTemplate` vs `agent-config/block.md`. Path: share the block file.
- D10: double store Open + N+1 Save. Path: one handle, batch or reuse.
- D11: warm index can report `indexed: 0` after a successful run. Path: count from the store, not the walk.
- D13 / D14: flags-after-dir dropped; onboarding commit-then-init leaves a dirty tree. Path: document or parse argv; skill says commit after init.
- D22: several Then clauses still count-only. Path: assert not-ingested / skip reasons where cheap.
- FOLLOWUP task-029 shared LLM client (bitemporal / memory-improvements carry-forward). Path: queued `2026-10-02-mnemonic-llm-provider`.

## Prior-Change Follow-Through

| Prior open item (from previous archived change) | Addressed by this change? | Evidence |
|--------------------------------------------------|---------------------------|----------|
| Keyword/FTS `expires_at` filter (memory-improvements) | no | still open; init does not edit `search_blend.go` |
| Non-decay config sections replace home layer (memory-improvements) | no | still open |
| `entity_aliases` prune / `query_cache` purge / `hybrid_search` cache / C6 observer (memory-improvements) | no | still open |
| `lifecycle_log` `failed` status unreachable (second-brain) | no | still open |
| Scoped `mem_search` first-call health cost (second-brain) | no | still open |
| FOLLOWUP task-029 shared LLM client (bitemporal / state) | no | still needs-triage; next queued change owns it |
| Five deferred bi-temporal read-path items (state) | no | still open |

## Move Evidence (from ship context)

**From:** `.skillgrid/specs/2026-10-02-mnemonic-project-init/` → **To:** `.skillgrid/archive/2026-10-02-mnemonic-project-init/`
**`diff -r` readback:** empty → PASS (`/usr/bin/diff -r` of the pre-move tree against the archive; `git` name-status `R100` on the seven files). Commit `e76fbb4e`.

## Overrides / Waivers / Contradictions

- None. Size-exception stayed on one branch. No QA waiver, no human CONCERNS override.
- Ranked contradiction: `review.md` Verdict cites `9fa312fe` for D3/D6 tests. That SHA is not on `release/2`. Final-state evidence is `3e2794bf` (`git log` at close). The review line is a stale snapshot at re-render, not a second commit.

## Lineage (observation IDs)

- briefing: in-repo only (`.skillgrid/archive/2026-10-02-mnemonic-project-init/briefing.md`) — Cursor MCP cwd `/Users/paladm` is ambiguous; lineage saves used repo-local `skillgrid mcp` with `MNEMONIC_PROJECT=skillgrid`
- blueprint: in-repo only (`blueprint.md`)
- tasks: in-repo only (`tasks.md`)
- ship: observation `39` (`topic_key` `skillgrid/2026-10-02-mnemonic-project-init/ship`, session `b0e0ccf8-26db-4cec-8d14-d81c86497c02`) · commit `e76fbb4e`
- report (QA half): in-repo only (this file, sections above `## Final-State Facts`)
- report (retro): observation saved at reflect close (`topic_key` `skillgrid/2026-10-02-mnemonic-project-init/report`)
- research / findings / ADRs: `adr.md` (no new ADR; restates ADR-0012); durable lifts appended to `.skillgrid/artifacts/06-research-findings.md`
- Review: `review.md` floor **met-with-fixes**; all axes Grade B; Highs D1/D2 closed in `cb077edf`; D21 closed in `c32ef909`; worst remaining D17

