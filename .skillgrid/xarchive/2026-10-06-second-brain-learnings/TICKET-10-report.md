# TICKET-10 report — MCP scope block + descriptions (envelope-additive)

## What I implemented

**Scope block (`mnemonic/internal/mcp/tools_memory.go`):**
- New `scopeBlockForHits(hits []memory.Observation) []map[string]any` — pure helper that tallies per-bucket counts over the FINAL merged result, in first-seen bucket order, and returns `nil` for a single-bucket (or empty) result.
- `handleMemSearch` all-projects branch: when `len(buckets_hit) ≥ 2`, the additive `buckets_hit[]` envelope field is attached (shape `[{bucket, count}]`). It is a LABEL — counts are derived from the post-truncation merged result; it never reorders or re-ranks. Single-bucket → no key (default shape byte-identical).
- The scoped (default) branch is UNTOUCHED — no scope block path there at all, so the default shape is byte-identical by construction.

**Progressive-disclosure descriptions (descriptions only, no behavior/param change):**
- `tools_memory.go` `mem_search` — high-level "ranked, budgeted recall" + what the envelope carries (merged `observations[]`, `count`, `project`, `buckets_hit[]` scope block) → detailed per-hit fields (corpus/score/matched_via/signals) → param semantics.
- `tools_secondbrain.go` `mem_ask` + `mem_lifecycle` — high-level returns, then the action/field shape.
- `tools_session_inject.go` `mem_inject_session` — high-level `block` + `items[]`, then envelope fields (block/items/total_tokens/degraded).

**New test file (`mnemonic/internal/mcp/tools_memory_scope_block_test.go`):**
- `TestMemSearch_ScopeBlock` — 2 buckets (scope-a×2 + scope-b×1) sharing a term; all_projects=true → `buckets_hit` present with both buckets + correct counts; counts == merged-rank per-project row counts (ranking untouched). Scoped default (all_projects=false) → NO `buckets_hit`, NO `all_projects` key, project = bucket id.
- `TestMemSearch_ScopeBlock_SingleBucket` — 1 bucket; all_projects=true → NO scope block; the hit itself unaffected (1 observation).
- `TestMemSearch_ScopeBlock_ByteIdentical` — default scoped search → `buckets_hit` ABSENT (not present-empty) + pre-existing keys (project/observations/count) intact, count correct.
- Fixture helpers: `newScopeBlockService`, `seedObservationInBucket` (opens the BUCKET's own store, starts a session IN that store — the observations FK is per-store — then saves), `seedSessionForBucket` (reads the first session id from the bucket store), `scopedSearchJSON`.

## What I tested and results

- Focused: `cd mnemonic && go test ./internal/mcp/... -run TestMemSearch_ScopeBlock -v` → **3/3 PASS**.
- Full mcp package: `cd mnemonic && go test ./internal/mcp/...` → **ok, 78.461s** (all green).
- Federated/shape floor (the critical constraint), re-run explicitly: `TestMemSearchAllProjectsFederated`, `TestMemSearchContractShape`, `TestMemSearchSingleOpen`, `TestMemSearchCorpusLabeled`, `TestFusedWebHitBudget`, `TestMemSearchAllProjectsSignalsAreKeywordOnly` → **6/6 PASS**. `TestBudgetedRetrievalMCP` (pins the 114-tool surface + required params) → **PASS** (descriptions don't change the tool surface).
- `go build ./...` → clean. `gofmt` clean on all my files (`tools_secondbrain.go` has PRE-EXISTING gofmt drift on lines I didn't touch — verified present on HEAD via stash; left alone, descriptions-only edit).
- `git diff --stat HEAD` → 4 files, +332/-5 (the test file is the bulk).

## TDD Evidence

Scenario made green: `explicit-search-scope` (@step-11, G12) in `.skillgrid/specs/2026-10-06-second-brain-learnings/acceptance.feature` (lines 135–143).

- **RED:** `TestMemSearch_ScopeBlock` written first, run before any implementation → `FAIL: expected the scope block to name 2 non-empty buckets, got 0` (the failure is the missing `buckets_hit` field — feature absent, not a typo; the fixture proves the merged ranks already span both buckets with count=3). The other two tests are absence-assertions (they pass at RED by design; they pin that the default shape never drifts).
- **GREEN:** after `scopeBlockForHits` + the all-projects branch attach → `3/3 PASS`, then full `./internal/mcp/...` green (78s).
- **BDD note:** this repo's acceptance.feature has no cucumber-js wiring (G-gates are shell `CHECK:` comments in tasks.md), so the scenario is driven by the named `TestMemSearch_ScopeBlock*` tests. The G12 gate itself (`go test ./internal/mcp/... -run TestMemSearch_ScopeBlock -v` EXPECT PASS) ran green; the EXPECT clause "existing federated tests green unmodified" verified by re-running the federated/shape floor (6/6 PASS, `git diff` shows no edits to them).

## Lazy

- Ladder rung: **one-line core / reuse** — the scope block is a pure tally over a result the handler already computed (`res.Hits`); no new service method, no new query, no store open. Reused the existing `SetService`/`seedObservation` fixture shape from the federated tests.
- One deliberate corner: the block counts the BUDGETED (post-truncation) merged result, not the pre-truncation service output — so its counts always match what the caller sees in `observations`. Marked with a `ponytail:` ceiling comment in `scopeBlockForHits` (upgrade path: pre-truncation counts if the item cap ever hides a bucket the caller should know about).
- `scopeBlockForHits` lives in `tools_memory.go` (the handler's file) rather than a new file — single clear responsibility (the one envelope-additive field this ticket adds), no file split.

## Gates

- **G12:** `cd mnemonic && go test ./internal/mcp/... -run TestMemSearch_ScopeBlock -v` → exit 0, 3/3 PASS. **FRESH, run by me.**
- **G12 (unmodified federated):** `cd mnemonic && go test ./internal/mcp/...` → `ok` (full package green; federated/shape tests 6/6 PASS on explicit re-run, none edited). **FRESH, run by me.**
- **Build:** `cd mnemonic && go build ./...` → clean. **FRESH, run by me.**

## Refinement

- **Pass 1 — Implement complete:** scope block + 3 description rewrites + 3 tests, all present; no placeholders.
- **Pass 2 — Expert re-read:** renamed the fixture helper from a `newScopeBlockService` alias hack to a direct `SetService(service.New(dataDir))`; fixed the FK seed (session must be created IN the bucket's own store — first draft's `SessionStart` on the shared service resolved to the CWD bucket, so scope-b's seed failed its FK). Names now say what they do (`seedObservationInBucket`, `seedSessionForBucket`). The "ahead of merged ranks" comment was softened to "labels the merged ranks" — the JSON key lands after `observations`; what matters is the ranks are untouched, and that's what the test pins.
- **Pass 3 — Defect hunt:** (a) Confirmed `Project` is a NOT-NULL-ish column on every scanned observation (the block can't silently drop a bucket by empty project). (b) Confirmed the single-bucket all_projects case is genuinely single-bucket in the fixture (temp dataDir, one store) so the absence assertion is meaningful, not a fluke of an empty second store. (c) gofmt drift in `tools_secondbrain.go` verified pre-existing on HEAD (stash + gofmt -l) — not my edit, left alone. (d) Verified the 114-tool count test still passes (descriptions don't change the surface). Final pass 2→4 clean.
- **Pass 4 — Polish:** gofmt on the new test file (struct tag alignment); comment hygiene (TICKET/scenario/ADR refs consistent with the sibling tests' convention). Re-ran the focused + full suite after the polish. Clean.

## Checkpoint

- **Commit:** `da3e6474` — `feat(mnemonic): add explicit scope block to mem_search envelope`
- **`[skillgrid-context]` block:**
  ```
  change: 2026-10-06-second-brain-learnings
  ticket: TICKET-10
  phase: execution
  ```
- Conventional commit, no AI-attribution trailer, atomic (one feature: the scope block + its description wording + its tests). The spec zone (`.skillgrid/`) was NOT touched, so no zone-order commit was needed.

## Files changed

- `mnemonic/internal/mcp/tools_memory.go` — scope block helper + all-projects attach + `mem_search` description.
- `mnemonic/internal/mcp/tools_secondbrain.go` — `mem_ask` + `mem_lifecycle` descriptions only.
- `mnemonic/internal/mcp/tools_session_inject.go` — `mem_inject_session` description only.
- `mnemonic/internal/mcp/tools_memory_scope_block_test.go` — NEW, 3 tests + fixture helpers.

## Self-review findings

- The scope block is keyed on `Observation.Project`, which is the same field the DTOs surface as `project` — so the block's bucket names are byte-for-byte the bucket ids the caller sees in each hit's `project`. Consistent by construction.
- The block is attached to the all-projects branch only. The ticket + acceptance scenario are scoped to the multi-bucket (all_projects) case; the scoped default branch has no scope-block path, which is exactly the byte-identity guarantee.
- No behavior or param change in any of the three description edits — only the `WithDescription` strings.

## Issues / concerns

- None blocking. One pre-existing note carried from TICKET-06 (not this ticket's regression): the handler-boundary observation-leg fusion gap (1-obs+1-web → 0 hits at the handler). This ticket's fixture uses all-observations multi-bucket, so it's unaffected.
