## 07-triple-store-linkage

### Goal

Cross-link relational + vector + graph stores.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-07` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/codeindex/symbol_embeddings.go` — CREATE: bridge table: symbols ↔ embeddings
- `internal/mnemonic/memory/lifecycle.go` — MODIFY: add graph_ref to observations

**Interfaces:**
- Consumes: codeindex `symbols` table; existing `observations` and `long_term_memories` tables; vector embeddings from step 06
- Produces: `graph_ref` column on observations; `embedding_blob` column on long_term_memories; `symbol_embeddings` bridge table; cross-link query path

### Tasks

- [ ] 07.1 `[RED]` graph_ref column on observations defaults to NULL when symbol not found
  - [ ] 07.1.a Write failing test (`TestGraphRefDefaultsToNull`): save an observation with no matching codeindex symbol; verify `graph_ref` is NULL; save an observation with a matching symbol ID; verify `graph_ref` is set to that symbol ID; verify no broken references
  - [ ] 07.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestGraphRefDefaultsToNull'` — Expected: FAIL
  - [ ] 07.1.c Minimal implementation — add `graph_ref` column (INTEGER, nullable) to `observations` table via migration; in `Save()`, look up the codeindex symbol ID for the observation's source file; set `graph_ref` to the symbol ID if found, NULL otherwise; create index on `graph_ref`
  - [ ] 07.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestGraphRefDefaultsToNull'` — Expected: PASS
  - [ ] 07.1.e Commit — `feat(mnemonic): add graph_ref column to observations`
- [ ] 07.2 `[RED]` symbol_embeddings bridge table links codeindex symbols to embeddings
  - [ ] 07.2.a Write failing test (`TestSymbolEmbeddingsBridge`): create a codeindex symbol; create an embedding vector; insert a row in `symbol_embeddings` linking them; query the bridge table and verify the symbol ID maps to the correct embedding; verify the table has `symbol_id`, `embedding_blob`, `model`, `created_at` columns
  - [ ] 07.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestSymbolEmbeddingsBridge'` — Expected: FAIL
  - [ ] 07.2.c Minimal implementation — create `symbol_embeddings.go` with the `symbol_embeddings` table schema: `symbol_id INTEGER NOT NULL, embedding_blob BLOB NOT NULL, model TEXT NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (symbol_id)`; add CRUD functions for the bridge table
  - [ ] 07.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestSymbolEmbeddingsBridge'` — Expected: PASS
  - [ ] 07.2.e Commit — `feat(mnemonic): add symbol_embeddings bridge table`
- [ ] 07.3 `[RED]` Cross-link query traverses observation→symbol→embedding→related observations
  - [ ] 07.3.a Write failing test (`TestTripleStoreCrossLinkQuery`): create an observation with `graph_ref` set to a symbol; create a symbol embedding in `symbol_embeddings`; create another observation that references the same symbol; run the cross-link query from the first observation; verify it returns the related observation via the SQL JOIN path
  - [ ] 07.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTripleStoreCrossLinkQuery'` — Expected: FAIL
  - [ ] 07.3.c Minimal implementation — implement `CrossLinkQuery(ctx, observationID)` that runs a single SQL JOIN: `observations o JOIN symbol_embeddings se ON o.graph_ref = se.symbol_id JOIN observations o2 ON o2.graph_ref = se.symbol_id WHERE o.id = ? AND o2.id != ?`; add `embedding_blob` column to `long_term_memories`; ensure triple-store queries are opt-in (separate method, not in default search path)
  - [ ] 07.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTripleStoreCrossLinkQuery'` — Expected: PASS
  - [ ] 07.3.e Commit — `feat(mnemonic): triple-store cross-link query via SQL JOIN`
- [ ] 07.4 `[AFK]` No broken references — orphan detection
  - [ ] 07.4.a Write failing test (`TestTripleStoreOrphanDetection`): create an observation with `graph_ref` pointing to a non-existent symbol ID; create an observation with `graph_ref` pointing to a valid symbol; run an integrity check; verify the orphan is detected and reported; verify the valid reference passes
  - [ ] 07.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTripleStoreOrphanDetection'` — Expected: FAIL
  - [ ] 07.4.c Minimal implementation — add `CheckCrossLinkIntegrity(ctx)` that queries observations with non-NULL `graph_ref` and verifies each referenced symbol exists in the codeindex `symbols` table; return a list of orphan references; run on a schedule or on demand
  - [ ] 07.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestTripleStoreOrphanDetection'` — Expected: PASS
  - [ ] 07.4.e Commit — `feat(mnemonic): add cross-link integrity check for orphan detection`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestGraphRefDefaultsToNull\|TestTripleStoreCrossLinkQuery'` | PASS | | |
| Acceptance `@step-07` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/codeindex/ ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify existing query paths unchanged (triple-store is opt-in) | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): cross-link relational, vector, and graph stores via graph_ref and symbol_embeddings`

---

