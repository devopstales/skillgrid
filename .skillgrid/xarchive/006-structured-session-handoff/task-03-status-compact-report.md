# 03-status-compact — Implementation Report

Change `006-structured-session-handoff`, step 03. Resumed from step 02 (commits
53cc95b, 29feb6e, b72ecb8, 174eeb4, ef53f61). Single commit:
`3cad7e7 feat(mnemonic): session status and thin knowledge compact`.

## Per sub-task

### 03.1 `[RED]` Mnemonic tool surface
- **RED test** written in `server_test.go`: `TestSessionStatus`,
  `TestKnowledgeCompact`, `TestMemSave`. `TestSessionStatus` asserts
  `session_status` is registered and returns `handoff_count:0` on a fresh store
  (no crash). `TestKnowledgeCompact` seeds a handoff via the `session_handoff`
  MCP call (bundle on disk + a `session_handoffs` row, **no Fact Memory
  written**), then asserts `knowledge_compact` succeeds (no error) and writes a
  `KNOWLEDGE.md`. `TestMemSave` seeds a real session (`mem_session_start`) and
  asserts `mem_save` still dispatches + returns an id (additive surface).
- **RED confirmed FAIL** (tools not registered / handlers undefined):
  `undefined: handleSessionStatus`, `undefined: handleKnowledgeCompact`.
- **GREEN confirmed PASS** after implementing via the step-02 registrar hook:
  `ok .../internal/mnemonic/mcp 2.432s`.

### 03.2 `status.go` — Status aggregation
- New `Snapshot` struct `{ HandoffCount int; ContextUsagePercent *float64; CostUSD *float64 }`
  and `Stats` (caller-supplied optional cost/context, pointer = absent).
- `Status(ctx, db, projectID, stats *Stats) (Snapshot, error)`:
  - Calls the step-02 `HandoffCount` helper (counts `session_handoffs` rows).
  - **Only** populates cost/context from `stats` when the caller supplied them.
    `stats == nil` (or nil field) → output omits that field; the relay never
    invents cost/context it doesn't have.
  - Empty store → `HandoffCount 0`, no error (warn + continue).
  - Nil store / blank project id → error (fail closed).

### 03.3 `compact.go` — thin CompactKnowledge
- `CompactKnowledge(ctx, db, projectID, projectRoot) (CompactResult, error)`:
  **NO Fact Memory dependency** — reads only the cleave bundle + `session_handoffs`.
- Knowledge body priority (handoff inputs / session notes ONLY):
  1. existing `.cleave/KNOWLEDGE.md` (bundle knowledge), if non-empty;
  2. the `session_handoffs.context_summary` notes, joined (session notes);
  3. a minimal placeholder.
- Empty/missing inputs → `warn + continue`: writes a minimal `KNOWLEDGE.md`
  (`# KNOWLEDGE` + a note), `CompactResult.Empty=true`, **no error**.
- Returns `{ KnowledgePath }` (absolute path under `.skillgrid/.cleave/`).

### 03.4 `[AFK]` happy — `Status`/`Compact` (relay)
- `TestStatus`: a store with one handoff reports `HandoffCount 1`;
  caller-supplied `context_usage_percent=42`/`cost_usd=1.5` surface; caller
  supplies nothing → both `nil` (not invented).
- `TestCompact`: handoff with non-empty `KNOWLEDGE.md` → compact preserves the
  handoff knowledge, `Empty=false`, file exists.
- `TestCompactFromContextNotes`: empty bundle `KNOWLEDGE.md` but a
  `context_summary` note → compact folds the note in (no Fact Memory),
  `Empty=false`.
- `Run: go test ./.../relay/ ./.../mcp/ -run 'Status|Compact'` → PASS.

### 03.5 `[AFK]` edge — no handoffs yet
- `TestStatusEmpty`: `Status` on an empty store → `HandoffCount 0`, no error.
- `TestCompactEmpty`: `CompactKnowledge` with no handoff rows / no `.cleave/`
  bundle → writes a minimal (non-empty) `KNOWLEDGE.md` under
  `.skillgrid/.cleave/`, `Empty=true`, no error.
- `Run: go test ./.../relay/ -run 'Status|Compact|Empty'` → PASS.

## RED→GREEN evidence (03.1)

| Step | Command | Result |
|------|---------|--------|
| RED (before impl) | `go test ./internal/mnemonic/mcp/ -run 'SessionStatus\|KnowledgeCompact\|MemSave' -count=1` | `FAIL [build failed]` — `undefined: handleSessionStatus`, `undefined: handleKnowledgeCompact` |
| GREEN (after impl) | same | `ok .../internal/mnemonic/mcp 2.432s` |

## How registration used the step-02 hook WITHOUT re-editing `server.go`

`server.go`'s `Start()`/`NewServer()` already call `registerSessionTools(s)`
(added in step 02). Step 03 appended two entries to the `tools` slice inside
`registerSessionTools` in `tools_session_handoff.go`:
`{sessionStatusTool(), handleSessionStatus}` and
`{knowledgeCompactTool(), handleKnowledgeCompact}`. The new tool definitions +
handlers live in the new `tools_session_status.go`. **`server.go` was NOT
re-edited** (confirmed: `server.go` absent from the commit's diff; `git
diff --cached --name-only | grep server\.go$` → none). No new registrar call was
needed — the step-02 hook already wires every session tool.

## How Status aggregates cost/context (zero counts on empty, no crash)

`Status` delegates the count to the existing `HandoffCount` (step 02), which
`SELECT COUNT(*) FROM session_handoffs WHERE project = ?` → 0 on an empty store.
Cost/context are **caller-supplied optional** MCP params
(`context_usage_percent?` / `cost_usd?`). The handler uses a `-1` sentinel
default from `GetFloat`; a value ≥ 0 → pointer set, otherwise omitted. So the
result JSON includes `context_usage_percent`/`cost_usd` only when the caller
passed them — never invented. Empty store → `{"handoff_count":0}`.

## How CompactKnowledge refreshes KNOWLEDGE.md from handoff/session notes ONLY

`gatherKnowledge` reads, in order: (1) the existing bundle `KNOWLEDGE.md`
(`os.ReadFile` on `.skillgrid/.cleave/KNOWLEDGE.md`), (2) `session_handoffs`
`context_summary` notes via `db.QueryContext` (added `QueryContext` to the
`relay.Store` interface — additive; `*sql.DB` satisfies it), (3) a minimal
placeholder. There is **no Fact Memory store access** anywhere in `compact.go`,
so a compact succeeds on a session with handoff inputs but no Fact Memory.
Empty/missing inputs → minimal file + `Empty=true`, warn + continue, no error.

## Full relevant suite + output

`go test ./internal/mnemonic/relay/ ./internal/mnemonic/mcp/ -count=1`:

```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay	2.360s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp	33.889s
```

## 005/008/010/011/013 baselines still pass

`go test ./internal/mnemonic/memory/... ./internal/mnemonic/route/...
./internal/mnemonic/affected/... ./internal/mnemonic/community/...
./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1`:

```
ok  	.../mnemonic/memory	8.472s
ok  	.../mnemonic/memory/layer	2.519s
ok  	.../mnemonic/route	2.162s
ok  	.../mnemonic/affected	10.530s
ok  	.../mnemonic/community	2.938s
ok  	.../mnemonic/pdg	4.385s
ok  	.../mnemonic/codeindex	14.907s
```

## MCP tool-surface lock: 80 → 82 (additive)

The baseline lock was spread across the `want` list in `server_test.go` (now
82 entries) plus seven tests that each asserted `len(tools) != 80` (the step-02
value). All were updated additively to 82 with accurate breakdowns
(`+ 2 status/compact`). The step-02 `tools_session_handoff_test.go` count moved
78 baseline + 2 session = 80 → 78 baseline + 2 session + 2 status/compact = 82.
`NewServer().ListTools()` measured at **82** (verified by a throwaway count test).
The existing `expectedMemToolSurface` checks confirm `mem_*` names + required
params are unchanged (additive — `session_status`/`knowledge_compact` have no
required params, so they add to the count without altering any `mem_*` tool).

Files updated for the count: `server_test.go`, `tools_session_handoff_test.go`,
`tools_code_affected_test.go`, `tools_code_knowledge_test.go`,
`tools_memory_governance_test.go`, `tools_memory_layer_args_test.go`,
`tools_memory_layer_test.go`, `tools_memory_retrieval_test.go`.

## Commits

- `3cad7e7` feat(mnemonic): session status and thin knowledge compact
- `95a280b` fix(mnemonic): thread caller ctx through knowledge compact query

## Post-review fix: ctx threading (95a280b)

**Finding (MINOR):** `CompactKnowledge` dropped its `ctx` (`_ = ctx`) and the
`contextNotes(db, projectID)` helper called `db.QueryContext(context.Background(),
...)`, so the caller's deadline/cancellation never reached that query.

**Fix:** threaded the caller's `ctx` through the whole chain —
`CompactKnowledge` → `gatherKnowledge(ctx, ...)` → `contextNotes(ctx, db,
projectID)` → `db.QueryContext(ctx, ...)`. Removed the `_ = ctx` discard. No
change to the `KNOWLEDGE.md` output, the no-Fact-Memory behavior, the tool
schemas, or the 82-tool surface — only `compact.go` was modified (1 file,
+8/−8).

**Verification:** existing relay + mcp tests still pass (tests are the only
verification; no `go build`/`go vet`/`gofmt -l` run).

`go test ./internal/mnemonic/relay/ ./internal/mnemonic/mcp/ -count=1`:

```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay	2.652s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp	39.716s
```

## Concerns / decisions

- **Cost/context are truly caller-supplied**: yes. The relay has no access to
  token/cost data (that is the agent host's); they are optional MCP params and
  are omitted from the result when not supplied. This is the intended
  "the relay doesn't invent cost/context it doesn't have" behavior.
- **`relay.Store` interface gained `QueryContext`**: additive; `*sql.DB`
  already satisfies it (the handler passes `h.Store().DB`). No existing
  implementation broken.
- **KNOWLEDGE.md format** (minimal, empty-input case):
  `# KNOWLEDGE` + an italic "no knowledge recorded yet (thin compact, no Fact
  Memory)" line + an `compacted at <UTC>` timestamp. Deliberately minimal; not
  an error. If the operator later wants a fixed template, it is a one-function
  change (`minimalKnowledge`).
- **`Compact` preserves the existing bundle `KNOWLEDGE.md` when non-empty**
  (priority 1) — i.e. it refreshes from the bundle first, then falls back to
  session notes, then a minimal placeholder. The "handoff inputs / session
  notes ONLY" reading is honored: no Fact Memory is ever read.
- **No re-edit of `server.go`**: confirmed (absent from the commit diff).
- **Unrelated pre-existing working-tree changes** (`docs/skillgrid/...`,
  `openspec/`, `.backlog/...` etc.) were **not** staged; the commit contains
  only the 14 step-03 files.
- Per the rules, `go build ./...` / `go vet` / `gofmt -l` were not run (tests
  are the only verification; the test compiler exercised all new code).
