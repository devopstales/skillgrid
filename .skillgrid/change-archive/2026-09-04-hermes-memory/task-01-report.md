# TICKET-01 — Fact Memory schema + module + fact_add tracer

**Change:** 2026-09-04-hermes-memory · **Work unit:** 1 (PR 1) · **Status:** DONE

## Implementation

### 1. Migration `011_facts_skills.sql` (Create)
`skillgrid-cli/internal/mnemonic/store/migrations/011_facts_skills.sql`

- `facts` — per-project store table mirroring the 014 importance columns
  (`importance_score`, `recency_decay`, `maturity_tier`, `retrieval_usage`) and the
  soft-delete `deleted_at` (NULL = alive) convention.
- `facts_fts` — FTS5 (porter tokenizer, external-content) + 3 sync triggers
  (after insert / update / delete) so the BM25 leg works with no vec0 present.
- `skills`, `skills_fts` (+ 3 triggers), `skill_usage` — created now for the
  shared migration file; populated by TICKET-03/04 (skills package).
- Indexes: `idx_facts_project_deleted`, `idx_facts_importance` (importance DESC,
  partial on `deleted_at IS NULL`), FTS rowids, skills indexes, skill_usage index.
- Additive only — no ALTER of existing tables; 003/014/040/042 left intact.

Note: `idx_facts_importance` keys on `(importance_score DESC)` without a
`project` column (unlike the 025 observations pattern) because `facts` is a
per-project store table with no project column — documented inline in the SQL.

### 2. `facts` package (Create)
`skillgrid-cli/internal/mnemonic/facts/facts.go` + `facts_test.go`

- `Store{db, project}`, `New(db, project)`.
- `Add(ctx, sessionID, content) (int64, error)` — **transactional**: inserts the
  fact row then a `session_events` row (action_type=`fact_add`, payload
  `{"fact_id":<id>}`). An unknown `session_id` (FK → `sessions(id)`) fails the
  whole transaction, so no orphan fact row is left behind.
- Empty content returns an error before touching the DB.

### 3. MCP `fact_add` tool (Create)
`skillgrid-cli/internal/mnemonic/mcp/tools_facts.go` + `tools_facts_test.go`

- `registerFactTools(s *server.MCPServer)` → `factAddTool` → `handleFactAdd`.
- `handleFactAdd` resolves the project store via `svc.Open(projectID)`, then
  `h.Store().DB` → `facts.New(db, projectID).Add(...)`. Returns the new fact id.
- Blank content → tool error before store open.

### 4. Wiring (Modify)
`skillgrid-cli/internal/mnemonic/mcp/server.go` — `registerFactTools(s)` added in
both `Start()` and `NewServer()`. Tool surface 80 → 81; assertions bumped in 7
test files (`server_test.go`, `tools_code_affected_test.go`,
`tools_memory_governance_test.go`, `tools_code_knowledge_test.go`,
`tools_memory_layer_args_test.go`, `tools_memory_retrieval_test.go`,
`tools_memory_layer_test.go`).

### 5. Migration tests (Create)
`skillgrid-cli/internal/mnemonic/store/migrations_011_test.go` — verifies
tables exist after open, 014 importance columns present on facts, prior
observations unchanged, and re-open is idempotent (no duplicate tables).

## Acceptance mapping

| Acceptance criterion | Evidence |
|---|---|
| Store open after 040+042 creates facts/facts_fts/skills/skills_fts/skill_usage | `TestMigration011FactMemoryTables` |
| Re-open idempotent | `TestMigration011ReopenIdempotent` |
| vec0 absent → FTS works, vector leg degrades | FTS5 (non-vec0) created; search leg not in scope this ticket |
| 014 importance columns present on facts | `TestMigration011FactMemoryTables` (PRAGMA column check) |
| `fact_add` MCP tool registered, returns fact id | `TestFactAddRegistered`, `TestFactAddMCP` |
| session_events row records action_type + fact id | `TestFactAddMCP` (asserts trail row + payload fact_id) |
| `mem_save` still works / prior observations unchanged | full `mcp` package green; migration test asserts observations rows intact |

## TDD evidence (RED → GREEN)

**RED** (before implementation):
```
# ./internal/mnemonic/facts/
facts_test.go: undefined: New
# ./internal/mnemonic/mcp/
tools_facts_test.go: undefined: handleFactAdd
# ./internal/mnemonic/store/
TestMigration011FactMemoryTables: facts table count = 0, want 1
```

**GREEN** (after implementation, `-count=1 -run 'Fact|Migration011'`):
```
ok  .../internal/mnemonic/facts   (TestAddCreatesFactRowAndEvent)
ok  .../internal/mnemonic/store   (TestMigration011FactMemoryTables, TestMigration011ReopenIdempotent)
ok  .../internal/mnemonic/mcp     (TestFactAddRegistered, TestFactAddMCP, TestFactAddRequiresContent)
```

Fails-when gate (`go test ./internal/mnemonic/facts/ ./internal/mnemonic/store/
./internal/mnemonic/mcp/ -count=1 -run 'Fact|Migration011'`) **PASS**, exit 0.

## 4-pass self-review

1. **Completeness** — every acceptance bullet has a named test; no stubs.
2. **Re-read** — `tools_facts.go` alignment quirk on import line fixed via `gofmt -w`.
3. **Defects** — `RequireString` does NOT trim (mcp-go v0.58.0 `tools.go:174`),
   so the empty-content test was corrected from `"   "` to `""` to match the
   real guard (`facts.Add` checks `strings.TrimSpace` internally).
4. **Full run** — `gofmt -l` clean on changed files; focused gate re-run GREEN.

## Lazy / gate evidence

- Additive SQL, no new dependencies (no ADR needed).
- Reused existing `store` embed/migrate infra (011 auto-picked by
  `//go:embed migrations/*.sql`), existing `session_events` append pattern,
  and existing MCP `registerXxxTools` convention — no new abstractions.
- `//go:embed` already sorts and dedupes; no migration runner changes.

## Files

- `skillgrid-cli/internal/mnemonic/store/migrations/011_facts_skills.sql` (Create)
- `skillgrid-cli/internal/mnemonic/facts/facts.go` (Create)
- `skillgrid-cli/internal/mnemonic/facts/facts_test.go` (Create)
- `skillgrid-cli/internal/mnemonic/mcp/tools_facts.go` (Create)
- `skillgrid-cli/internal/mnemonic/mcp/tools_facts_test.go` (Create)
- `skillgrid-cli/internal/mnemonic/mcp/server.go` (Modify)
- `skillgrid-cli/internal/mnemonic/store/migrations_011_test.go` (Create)
- 7 test files: tool-count 80→81 (Modify)

## Concerns

- **Pre-existing `go vet` finding** at `internal/mnemonic/mcp/tools_code_pdg.go:241`
  (`self-assignment of callable[0]`, from commit `fbb0351b`) — not introduced by
  this change, left untouched; worth a follow-up chore.
- Tracker test failures seen under `./...` are env/network (gh CLI timeouts under
  parallel load), not from this change; they pass in isolation.
