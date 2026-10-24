## 18-memory-types

### Goal

Typed memory categories + LLM dedup + async two-phase commit.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-18` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 04, 05

**Files:**
- `internal/mnemonic/memory/types.go` — CREATE: memory types + LLM dedup + async two-phase commit
- `internal/mnemonic/memory/types_test.go` — CREATE: memory types tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `memory-type` subcommand

**Interfaces:**
- Consumes: TTL defaults from step 04; LLM extraction from step 05; `Save()` pipeline
- Produces: `memory_type` column with 9 typed categories; LLM dedup before write; async two-phase commit; `memory_diff.json` audit

### Tasks

- [ ] 18.1 `[RED]` memory_type column supports 9 typed categories
  - [ ] 18.1.a Write failing test (`TestMemoryTypeCategories`): save observations with each of the 9 types (`profile`, `preferences`, `entities`, `events`, `identity`, `soul`, `cases`, `trajectories`, `experiences`); verify each is stored correctly; verify `mem list --type preferences` returns only preference-typed observations; verify invalid type is rejected
  - [ ] 18.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeCategories'` — Expected: FAIL
  - [ ] 18.1.c Minimal implementation — add `memory_type` (TEXT) column to observations via migration; define the 9 valid type constants; validate on save (reject unknown types with a clear error); add `--type` filter to `mem list` CLI
  - [ ] 18.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeCategories'` — Expected: PASS
  - [ ] 18.1.e Commit — `feat(mnemonic): add 9 typed memory categories`
- [ ] 18.2 `[RED]` LLM dedup detects semantic duplicates before write
  - [ ] 18.2.a Write failing test (`TestLLMDedupDetectsSemanticDuplicates`): save an observation with content "The build fails on macOS because of the missing SDK"; attempt to save a similar observation "macOS build broken due to absent SDK package"; verify the LLM dedup flags the second as a duplicate; verify the second is not stored (or is merged with the first); verify a genuinely different observation is not flagged
  - [ ] 18.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestLLMDedupDetectsSemanticDuplicates'` — Expected: FAIL
  - [ ] 18.2.c Minimal implementation — in the `Save()` path, before writing, call an LLM dedup check: query existing observations with similar embeddings (vector pre-filter), then ask the LLM if the new observation is a semantic duplicate; if duplicate, skip the write (or update the existing); hash dedup as fallback when LLM is unavailable
  - [ ] 18.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestLLMDedupDetectsSemanticDuplicates'` — Expected: PASS
  - [ ] 18.2.e Commit — `feat(mnemonic): LLM dedup before write with hash fallback`
- [ ] 18.3 `[RED]` Async two-phase commit: sync write + async LLM extraction
  - [ ] 18.3.a Write failing test (`TestAsyncTwoPhaseCommit`): call `session.commit()`; verify the sync phase completes immediately (messages written, compression_index incremented); verify the async phase (LLM extraction, dedup) runs in the background; verify `memory_diff.json` is written after the async phase completes; verify the main write path is not blocked by the async phase
  - [ ] 18.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestAsyncTwoPhaseCommit'` — Expected: FAIL
  - [ ] 18.3.c Minimal implementation — in `session.commit()`, split into sync phase (write messages, increment compression_index, return) and async phase (launch goroutine for LLM extraction, vector pre-filtering, dedup, write `memory_diff.json`); ensure the sync phase is durable before returning; the async phase writes to `memory_diff.json` for auditing
  - [ ] 18.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestAsyncTwoPhaseCommit'` — Expected: PASS
  - [ ] 18.3.e Commit — `feat(mnemonic): async two-phase session commit`
- [ ] 18.4 `[AFK]` Fallback to auto-classification when LLM type assignment fails
  - [ ] 18.4.a Write failing test (`TestMemoryTypeFallbackToAutoClassification`): mock the LLM to return an error during type classification; save an observation; verify it falls back to auto-classification (keyword-based type inference); verify the observation is stored with a best-guess type; verify no error is propagated
  - [ ] 18.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeFallbackToAutoClassification'` — Expected: FAIL
  - [ ] 18.4.c Minimal implementation — in the type assignment path, if the LLM returns an error, fall back to keyword-based auto-classification (map keywords to types); log a warning; ensure the observation is still saved with the fallback type
  - [ ] 18.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeFallbackToAutoClassification'` — Expected: PASS
  - [ ] 18.4.e Commit — `feat(mnemonic): fallback to auto-classification when LLM type fails`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeCategories\|TestLLMDedupDetectsSemanticDuplicates\|TestAsyncTwoPhaseCommit'` | PASS | | |
| Acceptance `@step-18` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify observations without memory_type still work (default) | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): typed memory categories with LLM dedup and async two-phase commit`

---

