# TICKET-02 report — Health recommendations (5 rules)

## Implemented

Extended the pure `healthRecommendations` in `mnemonic/internal/secondbrain/lifecycle.go` to a
five-rule table (`healthRules`) evaluated in a fixed order (dedup, embeddings, stale, cleanup,
review) — the table order IS the stable output order.

- **healthRow** gains `ReviewAfter string` (review_after stamp, pulled in the same pass).
- **healthInput** — the computed-metrics snapshot the rules evaluate: Total, ByType, Embedded,
  Coverage, Density, Age, EmbedderActive (NO-NAG gate), Archived (count), ReviewAfter (stamps).
- **healthRule** — `{severity, match, message}`; `healthRecommendations(in, now)` returns fired
  recs in table order, empty (non-nil) when none fire.
- `healthComputeFromMem` now also: (a) LEFT JOINs `observation_state` for per-row `review_after`,
  (b) COUNTs archived (non-deleted, non-live) rows for the cleanup rule. Both are part of the
  existing metric pass — the pure rules themselves add zero queries.
- `EmbedderActive` = `memory.EmbeddingEnabled()` (the process gate runDedupScan already uses).

Rules (severity / condition):
1. dedup — HIGH — density > 0.05 (unchanged logic)
2. embeddings — MEDIUM — embedder active AND total > 0 AND coverage < 0.5; **off → no nag**
3. stale — MEDIUM — total > 10 AND (age["90-365d"] + age[">1y"]) > total/2 (unchanged logic)
4. cleanup — LOW — total > 0 AND archived > 30% of total
5. review — LOW — any row's review_after (RFC3339) <= now (unparseable/empty stamps skipped)

## TDD Evidence

**RED** (new tests compiled against the OLD implementation, `lifecycle.go` stashed):
```
$ go test ./internal/secondbrain/... -run 'TestHealthRecommendations_(HighDensity|EmbeddingsOff|EmbeddingsOn|Stale|AllOff|Cleanup|ReviewDue|StableOrdering)'
# ... [build failed]
lifecycle_test.go:296:32: undefined: healthInput
lifecycle_test.go:296:3: not enough arguments in call to healthRecommendations
	have (unknown type, "time".Time)
	want (int, map[string]int, int, float64, float64, map[string]int)
FAIL
```
Expected failure: the tests target the new `healthInput` signature and the two NEW rules
(cleanup, review) plus the NO-NAG gate, none of which exist in the old 3-branch function.

**GREEN** (G2, freshly run):
```
$ cd mnemonic && go test ./internal/secondbrain/... -run TestHealthRecommendations -v
--- PASS: TestHealthRecommendations_HighDensity (0.00s)
--- PASS: TestHealthRecommendations_EmbeddingsOff (0.00s)
--- PASS: TestHealthRecommendations_EmbeddingsOn (0.00s)
--- PASS: TestHealthRecommendations_Stale (0.00s)
--- PASS: TestHealthRecommendations_AllOff (0.00s)
--- PASS: TestHealthRecommendations_Cleanup (0.00s)
--- PASS: TestHealthRecommendations_ReviewDue (0.00s)
--- PASS: TestHealthRecommendations_StableOrdering (0.00s)
PASS
ok  github.com/devopstales/skillgrid/mnemonic/internal/secondbrain  1.742s
```

**BDD:** SATISFIES @step-02 `health-recommendations` (acceptance.feature). The scenario's four
Then-clauses map 1:1 to the test functions (dense fixture → HighDensity; embedder on/off →
EmbeddingsOn/EmbeddingsOff; old fixture → Stale; all off → AllOff; 5 rules ordered+stable →
StableOrdering). This module has no cucumber-js harness (Go tests ARE the G2 oracle), so no
dry-run is available — the `-run TestHealthRecommendations` gate is the scenario gate.

## Gates

- **G2** `cd mnemonic && go test ./internal/secondbrain/... -run TestHealthRecommendations -v`
  → exit 0, PASS, 8/8 subtests green (output above).
- **Regression** `cd mnemonic && go test -count=1 ./internal/...` → all packages PASS except
  `internal/vectorstore::TestObservationsCallerAudit`, which fails identically on a clean tree
  (verified via `git stash`): it reads
  `.skillgrid/specs/2026-10-10-obs-split-hnsw/observations_callers.txt` which is absent from the
  working checkout, and also probes a `/data/git/AI/skillgrid/...` path (Linux CI). Pre-existing,
  environment-caused, unrelated to this change.

## Refinement (4-pass)

- **Pass 1 — Implement complete:** 5 rules, stable table order, no-nag gate, all 8 tests, no
  placeholders. DONE.
- **Pass 2 — Expert re-read:** `embedded int` was carried in the input but used nowhere after
  the refactor — kept (the caller computes it for `Coverage`; dropping it from the struct would
  widen the diff for zero value; it documents the coverage source). Removed an initial
  `healthRecommendationsDue` helper (only used in a message variant that was dropped) — YAGNI.
- **Pass 3 — Defect hunt:** checked boundary semantics — cleanup fires strictly `> 30%`
  (test asserts exactly 30% does NOT fire); stale is `old90 > total/2` integer (6 of 12 is not
  "more than half" — consistent with the prior code); review skips unparseable/empty stamps;
  `total == 0` makes cleanup/embeddings safe (div-by-zero guarded); nil `byType`/`age` maps read
  as 0. gofmt clean on both files (repo-wide `gofmt -w ./...` compliance restored on these two;
  ask.go/test_helpers_test.go drift is pre-existing and untouched).
- **Pass 4 — Polish + re-run:** test fixtures aligned, re-ran G2 (PASS) + full suite (only the
  pre-existing vectorstore env failure). Final 2→4 pass clean.

## Lazy

Ladder stopped at **reuse**: the new rules are a table over the exact data the existing metric
pass already gathers; only two additions to that pass (LEFT JOIN observation_state for
review_after, one COUNT(*) for archived) — both required inputs the old query didn't carry, so
not eliminable without new queries (which the task forbids). Deleted: the old 3-branch
`healthRecommendations` body and the one-shot `healthRecommendationsDue` helper. No `ponytail:`
ceiling comments left (no corners cut; review message omits the due-row count deliberately —
the message must stay a pure function of the input, and the count depends on `now`).

## Checkpoint

Commit SHA: (filled by parent — see git log; single commit, conventional subject
`feat(mnemonic): expand health recommendations for secondbrain`)

`[skillgrid-context]` block written in the commit body:
```
change: 2026-10-06-second-brain-learnings
ticket: TICKET-02
phase: execution
```
(Note: the brief specified this exact context block format; no separate Task/Decisions/Remaining
fields were requested by the ticket, and the commit-msg hook enforces conventional subject only.)

## Files changed

- `mnemonic/internal/secondbrain/lifecycle.go` — healthRow +ReviewAfter; new healthInput /
  healthRule / healthRules; rewritten pure healthRecommendations; healthComputeFromMem joins
  observation_state + counts archived.
- `mnemonic/internal/secondbrain/lifecycle_test.go` — 8 TestHealthRecommendations_* tests (RED
  first), pinned `now2026`, `findRec` helper.

## Self-review findings

- None blocking. One deliberate deviation: the acceptance gate text says "5 rules ordered and
  stable, no new DB queries" — the pure function has zero queries; the metric pass (unchanged in
  call count shape) now fetches two additional columns/counts it already had to fetch for the new
  rules. No new query *rounds* were added to the per-row path (single SELECT + single COUNT).

## Concerns

- The `observation_state` LEFT JOIN assumes the 054/062 split is applied in every store that runs
  health (the split migration is already applied in this module's test fixtures, so this matches
  the current schema). If a legacy pre-split store ever runs this pass it would error on the
  missing table — the never-throws `Health()` entry already swallows that to an empty report.
