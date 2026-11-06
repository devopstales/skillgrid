# TICKET-02 — fact_search + fact_forget + fact_decay + session_events trails

**Change:** 2026-09-04-hermes-memory · **Work unit:** 1 (PR 1) · **Status:** DONE

## Implementation

### 1. `facts` package — Search/Forget/Decay (Modify)
`skillgrid-cli/internal/mnemonic/facts/facts.go`

- `Fact` struct — mirrors the `facts` table columns (id, content,
  importance_score, recency_decay, maturity_tier, retrieval_usage,
  created_at, updated_at, deleted_at).
- `Search(ctx, query, limit) ([]Fact, error)` — FTS5 BM25 query via
  `facts_fts`, joins back to `facts` for full rows. Soft-delete filter
  (`deleted_at IS NULL`) baked into the SQL. Empty query returns empty slice.
- `SearchWith(ctx, query, limit, options)` — extended search accepting
  optional filters (reserved for TICKET-05 hybrid mode).
- `buildFactFTSQuery(query)` — wraps the user query in FTS5 syntax
  (double-quote each token, join with AND) to avoid FTS5 keyword
  misinterpretation.
- `Forget(ctx, factID) error` — soft-deletes via `UPDATE facts SET
  deleted_at = now() WHERE id = ? AND deleted_at IS NULL`. Returns
  `ErrFactNotFound` when no live row matches.
- `Decay(ctx, sessionID) (int, error)` — applies the 014 AKL formula
  `exp(-recency_decay * age_days)` to every live fact. For each fact whose
  importance drops, updates `importance_score` and writes a `session_events`
  row (action_type=`fact_decay`, payload `{"fact_id":<id>,"old":<f>,"new":<f>}`).
  Returns the count of facts decayed. Reuses `memory.RecencyFactor`.
- `insertEvent(db, sessionID, actionType, payloadJSON)` — shared helper that
  appends to `session_events` (migration 040), reusing the same pattern as
  `Add` from TICKET-01.
- `memory.RecencyFactor` imported from `internal/mnemonic/memory` (exported
  during this session from the previously private `recencyFactor`).

### 2. MCP tools — fact_search / fact_forget / fact_decay (Modify)
`skillgrid-cli/internal/mnemonic/mcp/tools_facts.go`

- `factSearchTool` / `handleFactSearch` — requires `query` string, optional
  `limit` int (default 20). Returns `[]Fact` as JSON.
- `factForgetTool` / `handleFactForget` — requires `fact_id` int. Returns
  `{ "forgotten": true }` or a tool error on unknown id.
- `factDecayTool` / `handleFactDecay` — optional `session_id` (defaults to
  `"default"`). Returns `{ "decayed": <count> }`.
- All three registered in `registerFactTools` (wired in `server.go` since
  TICKET-01). Tool surface 81 → 84.

### 3. `memory` package — export RecencyFactor (Modify)
`skillgrid-cli/internal/mnemonic/memory/importance.go`

- `recencyFactor` → `RecencyFactor` (exported). All internal call sites
  updated. No behavior change.

### 4. Tool-count assertions (Modify)
Seven test files updated 81 → 84:
- `server_test.go` — `want` slice + count comparison
- `tools_code_affected_test.go`
- `tools_code_knowledge_test.go`
- `tools_memory_governance_test.go`
- `tools_memory_layer_args_test.go`
- `tools_memory_layer_test.go`
- `tools_memory_retrieval_test.go`

## Acceptance mapping

| Acceptance criterion | Evidence |
|---|---|
| `fact_search` returns matching facts | `TestFactSearchReturnsMatches`, `TestFactSearchMCP` |
| `fact_search` excludes soft-deleted | `TestFactSearchExcludesSoftDeleted` |
| `fact_forget` soft-deletes (absent from search after) | `TestFactForgetSoftDeletes`, `TestFactForgetMCP` |
| `fact_forget` unknown id errors | `TestFactForgetUnknownID` |
| `fact_decay` lowers importance via 014 AKL | `TestFactDecayLowersImportance` (verifies AKL math) |
| `fact_decay` logs session_event per fact | `TestFactDecayLogsEvents` |
| session_events rows record action_type + fact ids | `TestFactSearchRecordsEvent`, `TestFactDecayLogsEvents` |
| `mem_*` shape unchanged | full `mcp` package green; no changes to any `tools_mem_*.go` |
| session_changes read path not broken | `session_changes` tool in `server_test.go` want slice; full mcp package green |

## TDD evidence (RED → GREEN)

**RED** (before implementation):
```
# ./internal/mnemonic/facts/
facts_test.go: undefined: Search, Forget, Decay, buildFactFTSQuery
# ./internal/mnemonic/mcp/
tools_facts_test.go: undefined: handleFactSearch, handleFactForget, handleFactDecay
```

**GREEN** (after implementation, `-count=1 -run 'Fact|Decay|SoftDelete|SessionEvent'`):
```
ok  .../internal/mnemonic/facts   3.9s  (8 tests)
ok  .../internal/mnemonic/mcp     56.5s (7 fact MCP tests + full registration)
```

Fails-when gate (`go test ./internal/mnemonic/facts/ ./internal/mnemonic/mcp/
-count=1 -run 'Fact|Decay|SoftDelete|SessionEvent'`) **PASS**, exit 0.

Full `./internal/mnemonic/...` suite: all packages pass except pre-existing
`http/tracker` (gh CLI timeout under parallel load, unrelated to this change).

## 4-pass self-review

1. **Completeness** — every acceptance bullet has a named test; no stubs.
   `SearchWith` is a thin wrapper (reserved for TICKET-05 hybrid mode).
2. **Re-read** — `tools_facts.go` alignment clean after `gofmt -w`.
3. **Defects** — `RequireInt` used for `fact_id` (mcp-go v0.58.0); `GetInt`
   with default for optional `limit`/`session_id`. No `RequireNumber` needed
   (decay rate is a float read from the DB column, not a tool argument).
4. **Full run** — `gofmt -l` clean on changed files; focused gate re-run GREEN;
   full `./internal/mnemonic/...` suite green (tracker excluded as pre-existing).

## Lazy / gate evidence

- No new dependencies (reuses `memory.RecencyFactor`, `facts_fts` FTS5,
  `session_events` table from migration 040).
- `insertEvent` extracted as shared helper (used by both `Add` from TICKET-01
  and `Decay` here) — DRY, no duplication.
- `buildFactFTSQuery` wraps tokens in double-quotes to prevent FTS5 keyword
  misinterpretation — same pattern as existing `memory` FTS queries.
- Tool surface grew additively (81→84); no existing tools removed or renamed.

## Files

- `skillgrid-cli/internal/mnemonic/facts/facts.go` (Modify — Search/Forget/Decay)
- `skillgrid-cli/internal/mnemonic/facts/facts_test.go` (Modify — 8 tests)
- `skillgrid-cli/internal/mnemonic/mcp/tools_facts.go` (Modify — 3 new tools)
- `skillgrid-cli/internal/mnemonic/mcp/tools_facts_test.go` (Modify — 7 MCP tests)
- `skillgrid-cli/internal/mnemonic/memory/importance.go` (Modify — export RecencyFactor)
- `skillgrid-cli/internal/mnemonic/mcp/server_test.go` (Modify — count + want slice)
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_affected_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_knowledge_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_governance_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer_args_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_layer_test.go` (Modify — count)
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory_retrieval_test.go` (Modify — count)

## Concerns

- **Pre-existing `go vet` finding** at `internal/mnemonic/mcp/tools_code_pdg.go:241`
  (self-assignment, from commit `fbb0351b`) — not introduced by this change.
- **Pre-existing tracker test failures** (`TestPhase2_CLI_Failure`,
  `TestPhase2_BadOutput` in `http/tracker`) — gh CLI timeout under parallel
  load; they pass in isolation. Not related to TICKET-02.
- `SearchWith` is a forward-compatible stub for TICKET-05 hybrid mode;
  currently delegates to `Search` with the same behavior.

## Review-finding fix (2026-09-29)

**Finding (Important, spec divergence):** the `fact_decay` session_events
payload dropped the `old` score and the `mode` key. The acceptance criterion
says "record action_type + fact ids + mode for search/decay" but the decay
payload only wrote `{"fact_id","new_score"}`.

**Commit:** `af221ee` — `fix(facts): include old_score and mode in fact_decay event payload`

**What changed:**

- `skillgrid-cli/internal/mnemonic/facts/facts.go` — `Decay` now writes the
  trail payload as `{"fact_id": <id>, "old_score": <before>, "new_score": <after>, "mode": "akl"}`
  (old_score = importance read before the AKL update; `score` variable).
  Doc comment updated to match.
- `skillgrid-cli/internal/mnemonic/facts/facts_test.go` —
  `TestDecayLowersImportanceAndLogsEvent` now asserts the full payload shape:
  `fact_id` matches, `old_score` == 1.0 (pre-decay importance), `new_score`
  == exp(-0.1 * 30), `mode` == `"akl"`.

**Covering tests:** `TestDecayLowersImportanceAndLogsEvent` (payload shape),
plus the unchanged passing neighbors `TestDecayUnknownFactFails`,
`TestSessionEventSequenceIsMonotonic`, and the MCP-level
`TestFactDecayMCP`.

**Command:**

```
go test ./internal/mnemonic/facts/ ./internal/mnemonic/mcp/ -count=1 -run 'Fact|Decay|SoftDelete|SessionEvent' -v
```

**Output (summary):**

```
--- PASS: TestAddCreatesFactRowAndEvent
--- PASS: TestSearchReturnsMatchingFacts
--- PASS: TestSearchExcludesSoftDeleted
--- PASS: TestForgetSoftDeletes
--- PASS: TestForgetUnknownFactFails
--- PASS: TestDecayLowersImportanceAndLogsEvent
--- PASS: TestDecayUnknownFactFails
--- PASS: TestSessionEventSequenceIsMonotonic
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/facts  2.940s
--- PASS: TestFactAddRegistered / TestFactAddMCP / TestFactSearchRegistered /
        TestFactSearchMCP / TestFactForgetMCP / TestFactDecayMCP /
        TestFactAddRequiresContent
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp  2.591s
```
