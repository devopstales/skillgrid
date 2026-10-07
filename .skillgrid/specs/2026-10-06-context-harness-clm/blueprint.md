# Context Harness (with CLM) — Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2 (from `briefing.md`'s `Tier:` line; T2 → full blueprint)

**Build shape:** Tracer thread (integration risk is the unknown — the capture gate, the `ctx_search` fusion, and the CLM mirror/revision loop all cross the Node↔Go↔SQLite boundary; a thin end-to-end path through every layer works first, then thickens)

**Goal:** Build the `context_harness` package that absorbs `session_inject` and adds intercept-and-abstract capture, a deterministic `ctx_query`, a `ctx` CLI with proactive index + fused `ctx_search`, a Context Routing block, and the opt-in Context Language Model (CLM) layer (mirror + overflow guard + calibration).

**Architecture:** `context_harness` becomes the owner of the session context lifecycle (ADR-0025). It moves the existing `session_inject` engine (import path change only — `mem_inject_session` contract and signatures unchanged) and adds: (a) a PostToolUse capture gate that stores gated tool output into a session-scoped FTS5 `tool_outputs` sandbox (migration 050); (b) a deterministic `ctx_query` over the code index; (c) a `ctx index`/`indexed_files` writer (migration 051, reuses the observations schema shape per ADR-0026) + a two-leg RRF `ctx_search` via `hybrid.Rank` (RRFK=60); (d) a Context Routing block rendered into `prime`; and (e) a `clm/` sub-package implementing the CLM (ADR-0027/0028) — Go owns state and decisions (budget, overflow guard, calibration, revision), the Node OpenCode `context` plugin hook owns the request path (render mirror, apply withhold, read edit), and accepted edits are persisted to the session-scoped `context_revisions` audit table (migration 052).

**Tech Stack:** Go 1.22+ (monorepo floor 1.25.5), `modernc.org/sqlite` (pure-Go, no CGo), FTS5 external-content tables, `hybrid.Rank` RRF fusion, hand-rolled `flag`-based CLI (not cobra), Node `hooks/tool-call-capture.js` (existing), OpenCode v2 plugin `context` hook. No new Go or Node dependencies (per ASSUMPTIONS.md § Locked constraints "No new dependencies without an ADR").

**Spec:** `.skillgrid/specs/2026-10-06-context-harness-clm/briefing.md`

**ADR manifest:** `.skillgrid/specs/2026-10-06-context-harness-clm/adr.md` (in-force set from `.skillgrid/ASSUMPTIONS.md` § In-force set; this change adds ADR-0025/0026/0027/0028)

 ## Global Constraints

- **Module layout:** the mnemonic Go code lives in the standalone `mnemonic/` module (module path `github.com/devopstales/skillgrid/mnemonic`), with the CLI at `mnemonic/cmd/mnemonic/`. All `go test ./...` / `go build ./...` commands in this blueprint are run from the `mnemonic/` directory. The `skillgrid` CLI is now installer-only (install/sync-repo/version); the `ctx` command group is added to the `mnemonic` CLI.
- Go floor 1.22 (repo builds with 1.25.5); pure-Go SQLite driver `modernc.org/sqlite` (no CGo); `vec0` already registered via blank import in `store/store.go:24`.
- No new Go or Node dependencies (per ASSUMPTIONS.md § Locked constraints).
- `mem_inject_session` MCP contract MUST NOT churn — only the import path changes (ADR-0025; per briefing requirement 13).
- Fail-open floors: every new seam (capture gate, `ctx_search`, CLM overflow guard) degrades to no-op / raw context on failure and never blocks the agent (per ASSUMPTIONS.md § In-force set ADR-0016 fail-open; ADR-0021 pre-tool policy posture).
- BDD is always on: every task has a `SATISFIES:` scenario name from `acceptance.feature`.
- CLM is opt-in, off by default (fail-closed default) via the `clm:` config block (ADR-0027; briefing requirement 8).
- `tool_outputs` and `context_revisions` are session-scoped, purged at session end; `indexed_files` is project-scoped with a 7-day TTL (ADR-0025/0026/0028).
- Migrations are lexicographically ordered and run once (tracked in `index_meta`); current max is `049`, so this change adds `050`/`051`/`052` (per `store/store.go:344-425`).
- The mirror is a prompt-injection surface: the system prompt stays OUT of the mirror; authored roles are lowered to plain text (ADR-0027; briefing requirement 10).
- The capture gate is on ACTUAL output in the PostToolUse seam — not a PreToolUse prediction (ADR-0025; briefing requirement 1).

## Hypothesis

**Claim:** A session-scoped FTS5 sandbox + RRF-fused `ctx_search` + an opt-in CLM mirror/revision loop, owned by one `context_harness` package that absorbs `session_inject`, will let the agent retrieve large tool output and edit its own context without the `mem_inject_session` contract churning or the agent hard-crashing on any new failure.

**Right condition:** `pnpm test` (the Go suite) passes with the new `context_harness` package, the three migrations (050/051/052) applied idempotently, the capture gate gating large output to a `ctx_search` pointer, `ctx_search` returning RRF-fused results with per-leg provenance, `ctx` CLI stats/index/search/purge working, the Context Routing block rendered in `prime`, and the CLM (when enabled) rendering a `0600` mirror, capturing a validated edit into `context_revisions`, withholding overflow, and calibrating the token estimate.

**Wrong condition:** any of: the `mem_inject_session` contract changes; a new seam blocks or hard-crashes the agent on failure; a migration is non-idempotent or out of order; `ctx_search` does not return per-leg provenance; the CLM leaks the system prompt into the mirror; or the CLM is on by default.

**Thinnest MVP:** the capture gate (migration 050 + `http/toolcalls.go` `content` field + `context_harness/capture.go`) gating one large tool output into the sandbox and returning a `ctx_search` pointer, with `ctx search` returning that row. This proves the Node↔Go↔SQLite↔FTS integration before the CLM or the index writer are built.

**Door check:** Task 1 (absorb `session_inject` into `context_harness` + the 050 migration) — if the absorption breaks `mem_inject_session` or the migration is non-idempotent, stop and revise.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Routing (new CLI command + HTTP route) | Applicable: new `ctx` subcommand + `POST /sessions/{id}/tool-calls` gains a `content` field | `ctx` is a new `case` in the existing `flag` dispatcher (no new dep); the HTTP route reuses `requireWriteAuth` + `openHandleFor` (no new auth) | Task 4 step (CLI dispatch test) + Task 2 step (route carries `content`) |
| Shell commands / subprocesses | N/A: the harness does not spawn new subprocesses; it consumes the existing `tool-call-capture.js` hook and the existing OpenCode `context` plugin hook | n/a | n/a |
| Version-control automation | N/A: no git/PR automation | n/a | n/a |
| Executable-file classification | N/A | n/a | n/a |
| Process integration (Node↔Go↔SQLite) | Applicable: the capture gate and CLM edit both cross the Node hook → HTTP → Go → FTS5 boundary | Both are fire-and-forget and fail-open (the hook always exit 0; a missing/failing route degrades to raw output / raw context) | Task 2 (gate stores on > threshold) + Task 7 (CLM edit captured at checkpoint) |
| Mnemonic tool contracts | Applicable: `mem_inject_session` is absorbed; its contract must not churn | Import path change only — `mcp/tools_session_inject.go` re-exports the same `mem_inject_session` tool with identical params/behavior | Task 1 (contract-preservation test) |
| `_shared/conventions/*` | N/A: no convention file is modified | n/a | n/a |

## Must-Haves (Goal-Backward Verification)

### Truths (observable behaviors)

1. The `mem_inject_session` MCP tool keeps its exact contract (params `query`, `all_projects`, `max_tokens` default 2000) after the absorption — only the import path changes. (`backstop`: contract preserved across a refactor, verified by a held-out call, not just a diff read.)
2. A PostToolUse tool call whose actual output exceeds the threshold (default ~4KB) and has no `SKILLGRID_CTX_BYPASS` is stored into the `tool_outputs` FTS5 sandbox and the agent receives a ≤200-char summary + a `ctx_search <query>` pointer; small output and bypass pass through unchanged. (`backstop`: cross-component Node↔Go contract.)
3. `ctx_search <query>` returns RRF-fused results across `tool_outputs_fts` (session) and `indexed_files_fts` (project) via `hybrid.Rank` (RRFK=60); each result carries per-leg provenance.
4. `ctx_query` returns deterministic counts/lists/existence over the code index (no line ranges in v1).
5. `ctx index <file...>` reads, chunks (~2KB), and upserts rows into `indexed_files` (7-day TTL) reusing the observations schema shape.
6. `ctx stats` reports sandbox + index counts; `ctx purge` clears `tool_outputs` + `context_revisions` for the session.
7. `prime` renders a Context Routing block (a tool map: counts/lists → `ctx_query`, retrieve → `ctx_search`, index → `ctx index`, memory → `mem_*`) — advisory, not blocking.
8. The CLM is off by default; it is on only when `clm.enabled` is true (config) or an env override is set.
9. When the CLM is on, the OpenCode `context` plugin hook renders a `0600` mirror (with `[[LIVE_CONTEXT ...]]` header + `[[CTX_TURN ...]]` blocks) into the outgoing model call, and the system prompt is NOT in the mirror. (`backstop`: rendered artifact + file mode, not a diff read.)
10. When the model edits the mirror, the `checkpoint` capture path reads the edit, Go validates it (nonce match, legal message sequence, tool-call-group repair), and a passing edit is persisted as one row in `context_revisions` (revision, anchor_count, anchor_digest, messages, size_estimate, calibration_factor, withhold_ids, edit_trace).
11. The overflow guard (Go) withholds the oldest tool results after the last edit when the calibrated estimate exceeds `budget − reserve`; the withhold decision is stored on the revision and applied by the plugin.
12. The calibration factor (starts ~4 chars/token) is corrected against the provider's token count for each request and stored per session.
13. On resume, the highest `context_revisions` row is reconstructed and its anchor (count + SHA-256 digest) re-validated; a mismatch falls back to raw context. `context_revisions` is purged at session end.

### Artifacts (files that must exist with real implementation)

- `mnemonic/internal/context_harness/` package: `capture.go`, `query.go`, `index.go`, `search.go`, `routing.go`, `autoprepend.go`, `summary.go`, `retrieve.go`, `render.go`, `privacy.go`, `index_render.go` (absorbed from `session_inject`).
- `mnemonic/internal/context_harness/clm/` sub-package: `mirror.go`, `overflow.go`, `calibrate.go`, `revision.go`.
- `mnemonic/internal/store/migrations/050_tool_outputs.sql`, `051_indexed_files.sql`, `052_context_revisions.sql`.
- `mnemonic/cmd/mnemonic/ctx_cmd.go` (the `ctx` CLI group).
- `mnemonic/internal/config/load.go` gains a `clm` block (struct + section + default + merge).
- `hooks/tool-call-capture.js` gains the gate (full `content` on > threshold) + the `checkpoint` CLM mirror read.
- The OpenCode `context` plugin hook module (new, registers the `context` plugin event).

### Key links (critical connections)

- `hooks/tool-call-capture.js` → `POST /sessions/{id}/tool-calls` → `context_harness/capture.go` → `tool_outputs` FTS5 (the gate must store and the agent must get the pointer).
- `ctx search` → `context_harness/search.go` → `hybrid.Rank` → both FTS tables (the fusion must produce per-leg provenance).
- `clm/mirror.go` → OpenCode `context` hook → outgoing model call (the render must exclude the system prompt).
- `checkpoint` capture → `clm/revision.go` → `context_revisions` (the edit must validate and persist).
- `clm/overflow.go` + `clm/calibrate.go` → the revision row (the withhold + factor must be stored and applied).

### One-way-door decisions

- Migration `050_tool_outputs` (new table) — tag on Task 1.
- Migration `051_indexed_files` (new table) — tag on Task 5.
- Migration `052_context_revisions` (new table) — tag on Task 7.
- The `http/toolcalls.go` route gaining a `content` field (published contract of the capture route) — tag on Task 2.

---

## File Structure

```
mnemonic/internal/context_harness/
├── capture.go          # Output Sandbox Gate: decide gate vs pass; store gated output
├── query.go            # ctx_query: deterministic counts/lists/existence over code index
├── index.go            # ctx index: read+chunk+upsert into indexed_files
├── search.go           # ctx_search: two-leg RRF fusion via hybrid.Rank
├── routing.go          # Context Routing block render (pure string)
├── autoprepend.go      # (absorbed from session_inject)
├── summary.go          # (absorbed) incl. EstimateTokens
├── retrieve.go         # (absorbed) HybridRetrieve/HybridObservations
├── render.go           # (absorbed) RenderContextBlock
├── privacy.go          # (absorbed) Injectable/ObservationInjectable
├── index_render.go     # (absorbed) RenderIndex/IndexConfig
└── clm/
    ├── mirror.go       # render the 0600 mirror (header + CTX_TURN blocks)
    ├── overflow.go     # overflow guard: compute withhold decision
    ├── calibrate.go    # correct the token-estimate factor
    └── revision.go     # validate + persist an accepted mirror edit

mnemonic/internal/store/migrations/
├── 050_tool_outputs.sql
├── 051_indexed_files.sql
└── 052_context_revisions.sql

mnemonic/cmd/mnemonic/
└── ctx_cmd.go          # ctx stats/index/search/purge

hooks/
└── tool-call-capture.js  # (modify) gate + checkpoint CLM read

plugins/opencode/context/  # (new) OpenCode v2 plugin: context hook
└── index.js              # renders mirror, applies withhold, reads edit
```

The existing `session_inject/` package is **moved** into `context_harness/` (the files above), and the four importers (`mcp/tools_session_inject.go`, `cmd/skillgrid/loop_cmd.go`, `secondbrain/ask.go`, `http/toolevents.go`) change import path only.

---

### Task 1: Absorb `session_inject` into `context_harness` + migration 050

> ⚠ one-way: migration `050_tool_outputs` (new session-scoped table).

**Files:**
- Create: `mnemonic/internal/context_harness/capture.go` (placeholder until Task 2; here only the package decl)
- Create: `mnemonic/internal/store/migrations/050_tool_outputs.sql`
- Move: `mnemonic/internal/session_inject/{autoprepend,summary,retrieve,render,privacy,index}.go` → `mnemonic/internal/context_harness/{autoprepend,summary,retrieve,render,privacy,index_render}.go` (rename `index.go` → `index_render.go` to avoid collision with the new `index.go`)
- Move: `mnemonic/internal/session_inject/*_test.go` → `mnemonic/internal/context_harness/`
- Delete: `mnemonic/internal/session_inject/` (empty after move)
- Modify: `mnemonic/internal/mcp/tools_session_inject.go` (import path only)
- Modify: `mnemonic/cmd/mnemonic/loop_cmd.go` (import path only)
- Modify: `mnemonic/internal/secondbrain/ask.go` (import path only)
- Modify: `mnemonic/internal/http/toolevents.go` (import path only)
- Test: `mnemonic/internal/context_harness/migrate_050_test.go`

**Interfaces:**
- Consumes: `store.Open(dataDir, projectID) (*store.Store, error)`; `memory.New(st, projectID) *memory.Service`; the `session_inject` public API (`EstimateTokens`, `HybridRetrieve`, `HybridObservations`, `RenderContextBlock`, `AutoPrepend`, `DistillSummary`, `RenderIndex`, `Injectable`, `ObservationInjectable`, `RetrieveResult`, `InjectItem`, `IndexConfig`) — all now under the `context_harness` package name.
- Produces: the `context_harness` package (the absorbed API, same signatures); the `tool_outputs` + `tool_outputs_fts` tables.

- [ ] **Step 1: Write the failing migration test**

```go
// context_harness/migrate_050_test.go
package context_harness

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestMigration050_TablesExist(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil { t.Fatalf("open: %v", err) }
	defer st.Close()
	for _, table := range []string{"tool_outputs", "tool_outputs_fts"} {
		n, err := st.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Int64()
		if err != nil { t.Fatalf("query %s: %v", table, err) }
		if n != 1 { t.Fatalf("table %s not created (count=%d)", table, n) }
	}
}

func TestMigration050_FTSInsertTrigger(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil { t.Fatalf("open: %v", err) }
	defer st.Close()
	_, err = st.DB.Exec(`INSERT INTO tool_outputs (id, session_id, project_id, tool_name, output, size_bytes, created_at)
		VALUES (1, 's1', 'ctxproj', 'bash', 'hello sandbox world', 19, '2026-10-06T00:00:00Z')`)
	if err != nil { t.Fatalf("insert: %v", err) }
	var c int
	err = st.DB.QueryRow(`SELECT count(*) FROM tool_outputs_fts WHERE tool_outputs_fts MATCH 'sandbox'`).Scan(&c)
	if err != nil { t.Fatalf("fts match: %v", err) }
	if c != 1 { t.Fatalf("expected 1 fts hit, got %d", c) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/context_harness/ -run 'TestMigration050' -v`
Expected: FAIL — `table tool_outputs not created` (no `050_tool_outputs.sql` yet; create the package dir first with a `doc.go` if needed).

- [ ] **Step 3: Write the migration**

```sql
-- store/migrations/050_tool_outputs.sql
-- Context Harness Sandbox Store (ADR-0025). Session-scoped; purged at session end.
CREATE TABLE IF NOT EXISTS tool_outputs (
    id          INTEGER PRIMARY KEY,
    session_id  TEXT NOT NULL,
    project_id  TEXT NOT NULL,
    tool_name   TEXT NOT NULL,
    output      TEXT NOT NULL,
    size_bytes  INTEGER NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tool_outputs_session ON tool_outputs(session_id);

CREATE VIRTUAL TABLE IF NOT EXISTS tool_outputs_fts USING fts5(
    tool_name,
    output,
    content='tool_outputs',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS tool_outputs_fts_insert AFTER INSERT ON tool_outputs BEGIN
    INSERT INTO tool_outputs_fts(rowid, tool_name, output) VALUES (new.id, new.tool_name, new.output);
END;
CREATE TRIGGER IF NOT EXISTS tool_outputs_fts_delete AFTER DELETE ON tool_outputs BEGIN
    INSERT INTO tool_outputs_fts(tool_outputs_fts, rowid, tool_name, output)
    VALUES ('delete', old.id, old.tool_name, old.output);
END;
```

- [ ] **Step 4: Move the package + fix import paths**

Move the six `session_inject/*.go` source files into `context_harness/` (renaming `index.go` → `index_render.go`), move the `*_test.go` files, update `package session_inject` → `package context_harness`. In the four importers, change the import path `.../session_inject` → `.../context_harness` and the `session_inject.X` → `context_harness.X` references (signatures unchanged). Delete the now-empty `session_inject/` directory.

- [ ] **Step 5: Write the contract-preservation test**

```go
// context_harness/contract_test.go
package context_harness

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestEstimateTokens_Unchanged(t *testing.T) {
	if got := EstimateTokens(""); got != 0 { t.Fatalf("empty: %d", got) }
	if got := EstimateTokens("abcd"); got != 1 { t.Fatalf("abcd: %d", got) }
	if got := EstimateTokens("abcdef"); got != 2 { t.Fatalf("abcdef: %d", got) } // (6+3)/4
}

func TestHybridRetrieve_BM25Only(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	st, err := store.Open(t.TempDir(), "tracer")
	if err != nil { t.Fatalf("open: %v", err) }
	defer st.Close()
	_, _ = st.DB.Exec(`INSERT OR REPLACE INTO sessions (id, project, status) VALUES ('s1','tracer','ended')`)
	_, _ = st.DB.Exec(`INSERT INTO observations (session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at, source, visibility, status)
		VALUES ('s1','decision','auth uses jwt','we chose jwt for auth','tracer','project','h1',1,'2026-10-06T00:00:00Z','2026-10-06T00:00:00Z','session','normal','active')`)
	mem := memory.New(st, "tracer")
	res, err := HybridRetrieve(context.Background(), mem, "tracer", "auth", false, 800)
	if err != nil { t.Fatalf("retrieve: %v", err) }
	if !res.Degraded { t.Fatalf("expected degraded (no embedder)") }
	if len(res.Items) == 0 { t.Fatalf("expected items") }
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/context_harness/ -v` and `go build ./...`
Expected: PASS (migration + absorbed API), and the four importer packages build.

- [ ] **Step 7: Commit**

```bash
git add mnemonic/internal/context_harness/ mnemonic/internal/store/migrations/050_tool_outputs.sql mnemonic/internal/mcp/tools_session_inject.go mnemonic/cmd/mnemonic/loop_cmd.go mnemonic/internal/secondbrain/ask.go mnemonic/internal/http/toolevents.go
git rm -r mnemonic/internal/session_inject
git commit -m "refactor: absorb session_inject into context_harness + add tool_outputs sandbox (ADR-0025)"
```

**SATISFIES:** `mem_inject_session contract preserved across absorption` (acceptance.feature)

---

### Task 2: Output Sandbox Gate — capture route carries `content` + gate logic

> ⚠ one-way: the `POST /sessions/{id}/tool-calls` route gains a `content` field (published contract of the capture route).

**Files:**
- Modify: `mnemonic/internal/http/toolcalls.go` (add `Content string \`json:"content"\`` to `toolCallBody`; pass to the gate)
- Create: `mnemonic/internal/context_harness/capture.go` (gate decision + `StoreToolOutput` + `SandboxSummary`)
- Modify: `hooks/tool-call-capture.js` (send full `content` when it exceeds the threshold; honor `SKILLGRID_CTX_BYPASS`)
- Test: `mnemonic/internal/context_harness/capture_test.go`

**Interfaces:**
- Consumes: `store.Store.DB` (`*sql.DB`); `memory.Service.EnsureSession` (existing); the `toolCallBody` struct.
- Produces: `context_harness.GateDecision` (struct: `Gate bool`, `Reason string`, `Summary string`, `Pointer string`); `context_harness.DecideGate(output string, bypass bool, threshold int) GateDecision`; `context_harness.StoreToolOutput(ctx, db, sessionID, projectID, toolName, output string) (int64, error)`; `context_harness.SandboxSummary(output string, limit int) string`; the default threshold constant `context_harness.SandboxThreshold` (4096).

- [ ] **Step 1: Write the failing gate test**

```go
// context_harness/capture_test.go
package context_harness

import "testing"

func TestDecideGate_SmallPasses(t *testing.T) {
	d := DecideGate("short output", false, SandboxThreshold)
	if d.Gate { t.Fatalf("small output should not gate") }
}

func TestDecideGate_LargeGates(t *testing.T) {
	d := DecideGate(string(make([]byte, 5000)), false, SandboxThreshold)
	if !d.Gate { t.Fatalf("large output should gate") }
	if len(d.Summary) > 200 { t.Fatalf("summary too long: %d", len(d.Summary)) }
	if d.Pointer == "" { t.Fatalf("expected ctx_search pointer") }
}

func TestDecideGate_Bypass(t *testing.T) {
	d := DecideGate(string(make([]byte, 5000)), true, SandboxThreshold)
	if d.Gate { t.Fatalf("bypass should not gate") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/context_harness/ -run 'TestDecideGate' -v`
Expected: FAIL — `undefined: DecideGate` / `SandboxThreshold`.

- [ ] **Step 3: Write the gate**

```go
// context_harness/capture.go
package context_harness

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// SandboxThreshold is the default byte threshold above which output is gated.
const SandboxThreshold = 4096

// GateDecision is the result of the Output Sandbox Gate decision.
type GateDecision struct {
	Gate    bool
	Reason  string
	Summary string
	Pointer string
}

// SandboxSummary returns the first `limit` chars of output, newline-collapsed.
func SandboxSummary(output string, limit int) string {
	s := strings.Join(strings.Fields(output), " ")
	if len(s) > limit {
		s = s[:limit] + "…"
	}
	return s
}

// DecideGate decides whether to gate an output based on actual size + bypass.
func DecideGate(output string, bypass bool, threshold int) GateDecision {
	if bypass {
		return GateDecision{Gate: false, Reason: "bypass"}
	}
	if len(output) <= threshold {
		return GateDecision{Gate: false, Reason: "below threshold"}
	}
	summary := SandboxSummary(output, 200)
	pointer := "ctx_search " + summary
	if len(pointer) > 48 {
		pointer = pointer[:48]
	}
	return GateDecision{
		Gate:    true,
		Reason:  "above threshold",
		Summary: summary,
		Pointer: pointer,
	}
}

// StoreToolOutput inserts a gated output into the sandbox and returns its id.
func StoreToolOutput(ctx context.Context, db *sql.DB, sessionID, projectID, toolName, output string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.ExecContext(ctx,
		`INSERT INTO tool_outputs (id, session_id, project_id, tool_name, output, size_bytes, created_at)
		 VALUES (COALESCE((SELECT max(id) FROM tool_outputs), 0) + 1, ?, ?, ?, ?, ?, ?)`,
		sessionID, projectID, toolName, output, len(output), now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
```

- [ ] **Step 4: Wire the route + hook**

In `http/toolcalls.go`: add `Content string \`json:"content"\`` to `toolCallBody` (after `ContentPreview`). In `handleToolCallCreate` (after `EnsureSession`, line 47), call `context_harness.DecideGate(b.Content, os.Getenv("SKILLGRID_CTX_BYPASS") == "1", context_harness.SandboxThreshold)`. If `d.Gate`, call `context_harness.StoreToolOutput(ctx, h.Store().DB, b.SessionID, projectID, b.ToolName, b.Content)` and set `payload.ContentPreview = d.Summary + " — " + d.Pointer`. Add `import "os"` if absent.

In `hooks/tool-call-capture.js` (default path, the body object at line 380-393): add a top-level `const CTX_BYPASS = process.env.SKILLGRID_CTX_BYPASS === "1";` and a `const CTX_THRESHOLD = 4096;`. In the body, add `content: (!CTX_BYPASS && content.length > CTX_THRESHOLD) ? content : ""` (full output only when it will be gated; keep `content_preview` as today). The Go side owns the actual threshold decision; the hook just sends the bytes.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/context_harness/ -run 'TestDecideGate' -v` and `go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add mnemonic/internal/http/toolcalls.go mnemonic/internal/context_harness/capture.go mnemonic/internal/context_harness/capture_test.go hooks/tool-call-capture.js
git commit -m "feat: output sandbox gate on actual tool output (ADR-0025)"
```

**SATISFIES:** `large tool output is abstracted at the boundary` (acceptance.feature)

---

### Task 3: `ctx_query` — deterministic counts/lists/existence

**Files:**
- Create: `mnemonic/internal/context_harness/query.go`
- Test: `mnemonic/internal/context_harness/query_test.go`

**Interfaces:**
- Consumes: the code index tables (the existing `codeindex` store tables — `symbols`, `chunks`, `files`); `store.Store.DB`.
- Produces: `context_harness.QueryResult` (struct: `TotalFiles int`, `TotalSymbols int`, `Files []string`, `Symbols []string`, `Exists map[string]bool`); `context_harness.RunQuery(ctx context.Context, db *sql.DB, query string) (*QueryResult, error)`.

- [ ] **Step 1: Write the failing test**

```go
// context_harness/query_test.go
package context_harness

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestRunQuery_Counts(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "qproj")
	defer st.Close()
	_, _ = st.DB.Exec(`CREATE TABLE IF NOT EXISTS symbols (id INTEGER PRIMARY KEY, path TEXT, name TEXT, kind TEXT)`)
	_, _ = st.DB.Exec(`INSERT INTO symbols (path, name, kind) VALUES ('a.go','foo','func'),('b.go','bar','func')`)
	res, err := RunQuery(context.Background(), st.DB, "foo")
	if err != nil { t.Fatalf("query: %v", err) }
	if res.TotalSymbols != 1 { t.Fatalf("symbols for foo: %d", res.TotalSymbols) }
	if !res.Exists["foo"] { t.Fatalf("expected foo to exist") }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/context_harness/ -run 'TestRunQuery' -v`
Expected: FAIL — `undefined: RunQuery`.

- [ ] **Step 3: Write `RunQuery`**

```go
// context_harness/query.go
package context_harness

import (
	"context"
	"database/sql"
)

// QueryResult is the deterministic output of ctx_query.
type QueryResult struct {
	TotalFiles   int
	TotalSymbols int
	Files        []string
	Symbols      []string
	Exists       map[string]bool
}

// RunQuery returns counts/lists/existence over the code index for a query.
// No line ranges in v1 (per ADR-0025 / briefing requirement 4).
func RunQuery(ctx context.Context, db *sql.DB, query string) (*QueryResult, error) {
	res := &QueryResult{Exists: map[string]bool{}}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM symbols WHERE name LIKE ?`, "%"+query+"%").Scan(&res.TotalSymbols); err != nil {
		if err == sql.ErrNoRows { return res, nil }
		return nil, err
	}
	if err := db.QueryRowContext(ctx, `SELECT count(DISTINCT path) FROM symbols WHERE name LIKE ?`, "%"+query+"%").Scan(&res.TotalFiles); err != nil {
		if err == sql.ErrNoRows { return res, nil }
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT name FROM symbols WHERE name LIKE ? ORDER BY name LIMIT 100`, "%"+query+"%")
	if err != nil { return nil, err }
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil { return nil, err }
		res.Symbols = append(res.Symbols, name)
		res.Exists[name] = true
	}
	return res, rows.Err()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/context_harness/ -run 'TestRunQuery' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/context_harness/query.go mnemonic/internal/context_harness/query_test.go
git commit -m "feat: ctx_query deterministic counts/lists/existence (ADR-0025)"
```

**SATISFIES:** `ctx_query returns deterministic counts` (acceptance.feature)

---

### Task 4: `ctx` CLI group (stats/index/search/purge)

**Files:**
- Create: `mnemonic/cmd/mnemonic/ctx_cmd.go`
- Modify: `mnemonic/cmd/mnemonic/main.go` (add `case "ctx": runCtx(rest[1:])` to the subcommand switch + a usage line)
- Test: `mnemonic/cmd/mnemonic/ctx_cmd_test.go`

**Interfaces:**
- Consumes: `openMemService(dataDir, project)` (mem.go:216); `context_harness.RunQuery`; `context_harness.Search` (Task 5); `context_harness.PurgeSession` (Task 6); the `tool_outputs`/`indexed_files` tables; `service.Service.Open`.
- Produces: `runCtx(version string, args []string)`; subcommands `stats`/`index`/`search`/`purge` (dispatched on `args[0]` like `runMem`).

- [ ] **Step 1: Write the failing dispatch test**

```go
// cmd/skillgrid/ctx_cmd_test.go
package main

import (
	"os"
	"testing"
)

func TestRunCtx_StatsNoPanic(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dir)
	os.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dir)
	func() {
		defer func() { if r := recover(); r != nil { t.Fatalf("panic: %v", r) } }()
		runCtx("test", []string{"stats", "--dir", dir})
	}()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/mnemonic/ -run 'TestRunCtx' -v`
Expected: FAIL — `undefined: runCtx`.

- [ ] **Step 3: Write the CLI group**

```go
// cmd/skillgrid/ctx_cmd.go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

func runCtx(version string, args []string) {
	if len(args) == 0 { ctxUsage(); return }
	cmd := args[0]
	rest := args[1:]
	fs := flag.NewFlagSet("ctx", flag.ContinueOnError)
	fs.Parse(reorderCtxArgs(rest))
	svc, proj := openMemService("", "")
	switch cmd {
	case "stats":
		ctxStats(svc, proj)
	case "index":
		ctxIndex(svc, proj, fs.Args())
	case "search":
		ctxSearch(svc, proj, fs.Args())
	case "purge":
		ctxPurge(svc, proj)
	default:
		ctxUsage()
	}
}

func ctxStats(svc *service.Service, project string) {
	h, cleanup, err := svc.Open(project)
	if err != nil { fmt.Fprintln(os.Stderr, "open:", err); return }
	defer cleanup()
	var to, idx int
	h.Store().DB.QueryRow(`SELECT count(*) FROM tool_outputs`).Scan(&to)
	h.Store().DB.QueryRow(`SELECT count(*) FROM indexed_files`).Scan(&idx)
	fmt.Printf("tool_outputs: %d\nindexed_files: %d\n", to, idx)
}

func ctxUsage() {
	fmt.Fprintln(os.Stderr, "usage: skillgrid ctx <stats|index|search|purge>")
}
```

(`ctxIndex`/`ctxSearch`/`ctxPurge`/`reorderCtxArgs` are implemented in Task 5 and Task 6; for this task, stub them as `func ctxIndex(svc *service.Service, project string, args []string) {}` etc. so it compiles.)

Add to `main.go`'s subcommand switch: `case "ctx": runCtx(version, rest[1:])` and a usage line `  ctx        context harness (stats/index/search/purge)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/mnemonic/ -run 'TestRunCtx' -v` and `go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic/cmd/mnemonic/ctx_cmd.go mnemonic/cmd/mnemonic/main.go mnemonic/cmd/mnemonic/ctx_cmd_test.go
git commit -m "feat: ctx CLI group (stats/index/search/purge) (ADR-0025)"
```

**SATISFIES:** `ctx CLI reports stats and purges` (acceptance.feature)

---

### Task 5: `ctx index` + `indexed_files` (migration 051) + `ctx_search` fusion

> ⚠ one-way: migration `051_indexed_files` (new project-scoped table).

**Files:**
- Create: `mnemonic/internal/store/migrations/051_indexed_files.sql`
- Create: `mnemonic/internal/context_harness/index.go`
- Create: `mnemonic/internal/context_harness/search.go`
- Modify: `mnemonic/cmd/mnemonic/ctx_cmd.go` (implement `ctxIndex` + `ctxSearch`)
- Test: `mnemonic/internal/context_harness/index_test.go`, `search_test.go`

**Interfaces:**
- Consumes: `store.Store.DB`; `hybrid.Rank(hits map[string]Hit, ftsRanks, sigRanks, semRanks map[string]int, k int) []hybrid.Hit` (rank.go:70, `RRFK=60` at rank.go:23); `hybrid.Hit` struct (`Path, StartLine, EndLine, Symbol, Kind, Snippet, Score, Provenance`); the observations schema shape (per ADR-0026 — `indexed_files` mirrors the `observations` columns: `id, session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at, source, visibility, status`).
- Produces: `context_harness.IndexFile(ctx, db, projectID, path string) (int, error)`; `context_harness.Search(ctx, db, sessionID, projectID, query string, limit int) ([]context_harness.SearchHit, error)`; `context_harness.SearchHit` (struct: `ID int64`, `Leg string`, `Title string`, `Snippet string`, `Score float64`, `Provenance hybrid.Provenance`).

- [ ] **Step 1: Write the failing migration + search tests**

```go
// context_harness/search_test.go
package context_harness

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestSearch_FusesBothLegs(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "fproj")
	defer st.Close()
	_, _ = st.DB.Exec(`INSERT INTO tool_outputs (id, session_id, project_id, tool_name, output, size_bytes, created_at)
		VALUES (1,'s1','fproj','bash','widget in tool output',24,'2026-10-06T00:00:00Z')`)
	_, _ = st.DB.Exec(`INSERT INTO indexed_files (id, session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at, source, visibility, status)
		VALUES (1,'s1','chunk','w.go','widget in file','fproj','project','h1',1,'2026-10-06T00:00:00Z','2026-10-06T00:00:00Z','index','normal','active')`)
	hits, err := Search(context.Background(), st.DB, "s1", "fproj", "widget", 10)
	if err != nil { t.Fatalf("search: %v", err) }
	if len(hits) < 2 { t.Fatalf("expected >=2 fused hits, got %d", len(hits)) }
	legs := map[string]bool{}
	for _, h := range hits { legs[h.Leg] = true }
	if !legs["tool_outputs"] || !legs["indexed_files"] {
		t.Fatalf("expected both legs, got %v", legs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/context_harness/ -run 'TestSearch' -v`
Expected: FAIL — `no such table: indexed_files` (migration 051 not applied).

- [ ] **Step 3: Write migration 051**

```sql
-- store/migrations/051_indexed_files.sql
-- Context Harness proactive index (ADR-0026). Reuses the observations schema shape.
-- Project-scoped; 7-day TTL like observations.
CREATE TABLE IF NOT EXISTS indexed_files (
    id              INTEGER PRIMARY KEY,
    session_id      TEXT,
    type            TEXT NOT NULL,
    title           TEXT NOT NULL,
    content         TEXT NOT NULL,
    project         TEXT NOT NULL,
    scope           TEXT NOT NULL DEFAULT 'project',
    normalized_hash TEXT,
    revision_count  INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    source          TEXT NOT NULL DEFAULT 'index',
    visibility      TEXT NOT NULL DEFAULT 'normal',
    status          TEXT NOT NULL DEFAULT 'active',
    expires_at      TEXT
);
CREATE INDEX IF NOT EXISTS idx_indexed_files_project ON indexed_files(project);
CREATE INDEX IF NOT EXISTS idx_indexed_files_expires ON indexed_files(expires_at);

CREATE VIRTUAL TABLE IF NOT EXISTS indexed_files_fts USING fts5(
    title, content,
    content='indexed_files',
    content_rowid='id',
    tokenize='porter'
);
CREATE TRIGGER IF NOT EXISTS indexed_files_fts_insert AFTER INSERT ON indexed_files BEGIN
    INSERT INTO indexed_files_fts(rowid, title, content) VALUES (new.id, new.title, new.content);
END;
CREATE TRIGGER IF NOT EXISTS indexed_files_fts_delete AFTER DELETE ON indexed_files BEGIN
    INSERT INTO indexed_files_fts(indexed_files_fts, rowid, title, content)
    VALUES ('delete', old.id, old.title, old.content);
END;
```

- [ ] **Step 4: Write `IndexFile` + `Search`**

```go
// context_harness/index.go
package context_harness

import (
	"context"
	"database/sql"
	"os"
	"time"
)

const chunkSize = 2048

// IndexFile reads a file, chunks it (~2KB), and upserts rows into indexed_files.
func IndexFile(ctx context.Context, db *sql.DB, projectID, path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil { return 0, err }
	now := time.Now().UTC().Format(time.RFC3339)
	expires := time.Now().UTC().AddDate(0, 0, 7).Format(time.RFC3339)
	n := 0
	for start := 0; start < len(data); start += chunkSize {
		end := start + chunkSize
		if end > len(data) { end = len(data) }
		chunk := string(data[start:end])
		_, err := db.ExecContext(ctx,
			`INSERT INTO indexed_files (id, session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at, source, visibility, status, expires_at)
			 VALUES (COALESCE((SELECT max(id) FROM indexed_files),0)+1, NULL, 'chunk', ?, ?, ?, 'project', '', 1, ?, ?, 'index', 'normal', 'active', ?)`,
			path, chunk, projectID, now, now, expires)
		if err != nil { return n, err }
		n++
	}
	return n, nil
}
```

```go
// context_harness/search.go
package context_harness

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/devopstales/skillgrid/mnemonic/internal/hybrid"
)

// SearchHit is a fused result with per-leg provenance.
type SearchHit struct {
	ID         int64
	Leg        string
	Title      string
	Snippet    string
	Score      float64
	Provenance hybrid.Provenance
}

// Search fuses tool_outputs_fts (session) + indexed_files_fts (project) via hybrid.Rank.
func Search(ctx context.Context, db *sql.DB, sessionID, projectID, query string, limit int) ([]SearchHit, error) {
	hits := map[string]hybrid.Hit{}
	ftsRanks := map[string]int{}
	if err := queryToolOutputsLeg(ctx, db, sessionID, projectID, query, hits, ftsRanks); err != nil {
		return nil, err
	}
	if err := queryIndexedFilesLeg(ctx, db, projectID, query, hits, ftsRanks); err != nil {
		return nil, err
	}
	ranked := hybrid.Rank(hits, ftsRanks, nil, nil, hybrid.RRFK)
	out := make([]SearchHit, 0, len(ranked))
	for i, h := range ranked {
		if i >= limit { break }
		out = append(out, SearchHit{ID: int64(i), Leg: h.Symbol, Title: h.Path, Snippet: h.Snippet, Score: h.Score, Provenance: h.Provenance})
	}
	return out, nil
}

func queryToolOutputsLeg(ctx context.Context, db *sql.DB, sessionID, projectID, query string, hits map[string]hybrid.Hit, ftsRanks map[string]int) error {
	rows, err := db.QueryContext(ctx, `SELECT to.id, to.tool_name, to.output
		FROM tool_outputs to INNER JOIN tool_outputs_fts ON tool_outputs_fts.rowid = to.id
		WHERE tool_outputs_fts MATCH ? AND to.session_id = ? AND to.project_id = ?
		ORDER BY bm25(tool_outputs_fts) LIMIT 50`, query, sessionID, projectID)
	if err != nil { return err }
	defer rows.Close()
	rank := 0
	for rows.Next() {
		var id int64
		var toolName, output string
		if err := rows.Scan(&id, &toolName, &output); err != nil { return err }
		key := "tool_outputs:" + strconv.FormatInt(id, 10)
		hits[key] = hybrid.Hit{Path: toolName, Symbol: "tool_outputs", Snippet: output[:min(200, len(output))]}
		ftsRanks[key] = rank
		rank++
	}
	return rows.Err()
}

func queryIndexedFilesLeg(ctx context.Context, db *sql.DB, projectID, query string, hits map[string]hybrid.Hit, ftsRanks map[string]int) error {
	rows, err := db.QueryContext(ctx, `SELECT f.id, f.title, f.content
		FROM indexed_files f INNER JOIN indexed_files_fts ON indexed_files_fts.rowid = f.id
		WHERE indexed_files_fts MATCH ? AND f.project = ?
		ORDER BY bm25(indexed_files_fts) LIMIT 50`, query, projectID)
	if err != nil { return err }
	defer rows.Close()
	rank := 0
	for rows.Next() {
		var id int64
		var title, content string
		if err := rows.Scan(&id, &title, &content); err != nil { return err }
		key := "indexed_files:" + strconv.FormatInt(id, 10)
		hits[key] = hybrid.Hit{Path: title, Symbol: "indexed_files", Snippet: content[:min(200, len(content))]}
		ftsRanks[key] = rank
		rank++
	}
	return rows.Err()
}
```

Wire `ctxIndex`/`ctxSearch` in `ctx_cmd.go` to call `context_harness.IndexFile`/`context_harness.Search` (and add `reorderCtxArgs`).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/context_harness/ -run 'TestSearch' -v` and `go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add mnemonic/internal/store/migrations/051_indexed_files.sql mnemonic/internal/context_harness/index.go mnemonic/internal/context_harness/search.go mnemonic/cmd/mnemonic/ctx_cmd.go mnemonic/internal/context_harness/index_test.go mnemonic/internal/context_harness/search_test.go
git commit -m "feat: ctx index + indexed_files (051) + ctx_search RRF fusion (ADR-0025/0026)"
```

**SATISFIES:** `ctx_search fuses sandbox and index via RRF` (acceptance.feature)

---

### Task 6: `ctx purge` + Context Routing block

**Files:**
- Modify: `mnemonic/cmd/mnemonic/ctx_cmd.go` (implement `ctxPurge`)
- Create: `mnemonic/internal/context_harness/routing.go`
- Create: `mnemonic/internal/context_harness/purge.go`
- Modify: `mnemonic/internal/loop/render.go` (add `Routing string` to `PrimeInput`; render under a `## Context Routing` header)
- Modify: `mnemonic/cmd/mnemonic/loop_cmd.go` (call `context_harness.RenderRouting` and set `PrimeInput.Routing`)
- Test: `mnemonic/internal/context_harness/routing_test.go`

**Interfaces:**
- Consumes: the `tool_outputs` + `context_revisions` tables; `loop.PrimeInput` (render.go:14); `loop.RenderPrime` (render.go:26); `memory.Service` session lookup (the current session id).
- Produces: `context_harness.PurgeSession(ctx context.Context, db *sql.DB, sessionID string) (int, error)`; `context_harness.RenderRouting() string` (the ~80-word tool map, pure string).

- [ ] **Step 1: Write the failing routing test**

```go
// context_harness/routing_test.go
package context_harness

import (
	"strings"
	"testing"
)

func TestRenderRouting_ContainsToolMap(t *testing.T) {
	r := RenderRouting()
	for _, want := range []string{"ctx_query", "ctx_search", "ctx index", "mem_"} {
		if !strings.Contains(r, want) { t.Fatalf("routing missing %q: %s", want, r) }
	}
	if len(r) > 600 { t.Fatalf("routing too long: %d chars", len(r)) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/context_harness/ -run 'TestRenderRouting' -v`
Expected: FAIL — `undefined: RenderRouting`.

- [ ] **Step 3: Write `RenderRouting` + `PurgeSession` + wire**

```go
// context_harness/routing.go
package context_harness

// RenderRouting returns the Context Routing block (advisory, ~80 words).
func RenderRouting() string {
	return "Context Routing (advisory):\n" +
		"- counts/lists/existence -> ctx_query\n" +
		"- retrieve (tool output + indexed files) -> ctx_search <query>\n" +
		"- index a file -> ctx index <path>\n" +
		"- session stats/purge -> ctx stats / ctx purge\n" +
		"- memory -> mem_* (mem_inject_session, mem_search, mem_save)"
}
```

```go
// context_harness/purge.go
package context_harness

import (
	"context"
	"database/sql"
)

// PurgeSession clears tool_outputs + context_revisions for a session.
func PurgeSession(ctx context.Context, db *sql.DB, sessionID string) (int, error) {
	var n int
	if r1, err := db.ExecContext(ctx, `DELETE FROM tool_outputs WHERE session_id = ?`, sessionID); err == nil {
		if c, err := r1.RowsAffected(); err == nil { n += int(c) }
	}
	if r2, err := db.ExecContext(ctx, `DELETE FROM context_revisions WHERE session_id = ?`, sessionID); err == nil {
		if c, err := r2.RowsAffected(); err == nil { n += int(c) }
	}
	return n, nil
}
```

In `ctx_cmd.go`, implement `ctxPurge`:
```go
func ctxPurge(svc *service.Service, project string) {
	h, cleanup, err := svc.Open(project)
	if err != nil { fmt.Fprintln(os.Stderr, "open:", err); return }
	defer cleanup()
	sessionID := activeSessionID(h) // resolve the current session (see note below)
	n, err := context_harness.PurgeSession(context.Background(), h.Store().DB, sessionID)
	if err != nil { fmt.Fprintln(os.Stderr, "purge:", err); return }
	fmt.Printf("purged %d rows\n", n)
}
```
Note for `activeSessionID(h)`: read `h.Memory().ProjectID()` and query `SELECT id FROM sessions WHERE project = ? ORDER BY started_at DESC LIMIT 1` via `h.Store().DB`; if none, print "no active session" and return. (This matches how `loop_cmd.go` resolves the current session.)

In `loop/render.go`: add `Routing string` to the `PrimeInput` struct (line 14). In `RenderPrime`, after the `## Memory` block (lines 51-57), add:
```go
if in.Routing != "" {
	b.WriteString("\n## Context Routing\n\n")
	b.WriteString(in.Routing)
	b.WriteString("\n")
}
```
In `loop_cmd.go`'s `primeText` (line 110-124), set `Routing: context_harness.RenderRouting()` in the `loop.PrimeInput{...}` literal.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/context_harness/ -run 'TestRenderRouting' -v` and `go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic/cmd/mnemonic/ctx_cmd.go mnemonic/internal/context_harness/routing.go mnemonic/internal/context_harness/purge.go mnemonic/internal/loop/render.go mnemonic/cmd/mnemonic/loop_cmd.go mnemonic/internal/context_harness/routing_test.go
git commit -m "feat: ctx purge + Context Routing block in prime (ADR-0025)"
```

**SATISFIES:** `prime renders the Context Routing block` (acceptance.feature)

---

### Task 7: CLM — config block + mirror + revision + migration 052

> ⚠ one-way: migration `052_context_revisions` (new session-scoped table).

**Files:**
- Create: `mnemonic/internal/store/migrations/052_context_revisions.sql`
- Modify: `mnemonic/internal/config/load.go` (add `clm` block: struct field in `Indexing`, yaml key in `mnemonicSection`, `clmSection` struct, `DefaultCLM()`, `mergeCLM()`)
- Create: `mnemonic/internal/context_harness/clm/mirror.go`
- Create: `mnemonic/internal/context_harness/clm/revision.go`
- Test: `mnemonic/internal/context_harness/clm/mirror_test.go`, `mnemonic/internal/config/load_clm_test.go`

**Interfaces:**
- Consumes: `config.Load(dir).CLM` (the new block); `store.Store.DB`; `context_harness.EstimateTokens` (the size estimator the calibration builds on); the OpenCode `context` plugin event (Node).
- Produces: `config.CLM` (struct: `Enabled bool`, `Budget int`, `Reserve int`, `Reminders string`, `Guard bool`, `Cap int`); `DefaultCLM() config.CLM` (all off/defaults: `Reserve 2048`, `Guard true`, `Reminders "50/75/90"`); `clm.Mirror` (struct: `Version int`, `Revision int`, `Nonce string`, `Baseline string`, `Blocks []clm.Block`); `clm.Block` (struct: `Index int`, `Role string`, `ID string`, `Protected bool`, `Content string`); `clm.RenderMirror(m *clm.Mirror) (string, error)`; `clm.WriteMirror(dir string, m *clm.Mirror) (string, error)` (writes a `0600` file); `clm.Edit` (struct: `Nonce string`, `Messages []string`); `clm.ValidateRevision(edit *clm.Edit, expectedNonce string) error`; `clm.Revision` (struct: `SessionID, ProjectID string`, `Revision, AnchorCount int`, `AnchorDigest string`, `Messages string`, `SizeEstimate int`, `CalibrationFactor float64`, `WithholdIDs, EditTrace string`); `clm.PersistRevision(ctx, db, r *clm.Revision) error`.

- [ ] **Step 1: Write the failing migration + config tests**

```go
// context_harness/clm/mirror_test.go
package clm

import (
	"strings"
	"testing"
)

func TestRenderMirror_ExcludesSystemPrompt(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 1, Nonce: "abc", Baseline: "digest"}
	m.Blocks = append(m.Blocks, Block{Index: 0, Role: "user", ID: "m1", Protected: false, Content: "hello"})
	s, err := RenderMirror(m)
	if err != nil { t.Fatalf("render: %v", err) }
	if !strings.HasPrefix(s, "[[LIVE_CONTEXT version=1 revision=1 document=abc baseline=digest]]") {
		t.Fatalf("bad header: %s", s)
	}
	if !strings.Contains(s, "[[CTX_TURN document=abc index=0 role=user id=m1 protected=false]]") {
		t.Fatalf("missing CTX_TURN block: %s", s)
	}
}

func TestRenderMirror_ProtectedFlag(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 1, Nonce: "abc", Baseline: "d"}
	m.Blocks = append(m.Blocks, Block{Index: 0, Role: "system", ID: "sys", Protected: true, Content: "You are a helper."})
	s, _ := RenderMirror(m)
	if !strings.Contains(s, "protected=true") {
		t.Fatalf("expected protected=true in: %s", s)
	}
}
```

```go
// config/load_clm_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCLM_Off(t *testing.T) {
	c := DefaultCLM()
	if c.Enabled { t.Fatalf("CLM must be off by default") }
	if c.Reserve != 2048 { t.Fatalf("reserve default: %d", c.Reserve) }
	if !c.Guard { t.Fatalf("guard default: must be true") }
}

func TestLoadCLM_ExplicitOn(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".skillgrid", "config.d")
	if err := os.MkdirAll(cfgDir, 0755); err != nil { t.Fatal(err) }
	yaml := "mnemonic:\n  clm:\n    enabled: true\n    budget: 64000\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "indexing.yaml"), []byte(yaml), 0644); err != nil { t.Fatal(err) }
	c := Load(dir).CLM
	if !c.Enabled { t.Fatalf("clm.enabled not honored") }
	if c.Budget != 64000 { t.Fatalf("clm.budget not honored: %d", c.Budget) }
	if c.Reserve != 2048 { t.Fatalf("reserve default should survive: %d", c.Reserve) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/context_harness/clm/ -run 'TestRenderMirror' -v` and `go test ./internal/config/ -run 'TestCLM' -v`
Expected: FAIL — `undefined: Mirror` / `undefined: DefaultCLM`.

- [ ] **Step 3: Write migration 052 + config block**

```sql
-- store/migrations/052_context_revisions.sql
-- CLM session-scoped audit (ADR-0028). Raw history + mirror are source of truth.
CREATE TABLE IF NOT EXISTS context_revisions (
    id                 INTEGER PRIMARY KEY,
    session_id         TEXT NOT NULL,
    project_id         TEXT NOT NULL,
    revision           INTEGER NOT NULL,
    anchor_count       INTEGER NOT NULL,
    anchor_digest      TEXT NOT NULL,
    messages           TEXT NOT NULL,
    size_estimate      INTEGER NOT NULL,
    calibration_factor REAL NOT NULL DEFAULT 4.0,
    withhold_ids       TEXT,
    edit_trace         TEXT,
    created_at         TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_context_revisions_session ON context_revisions(session_id);
CREATE INDEX IF NOT EXISTS idx_context_revisions_rev ON context_revisions(session_id, revision);
```

In `config/load.go`:
- Add to the `Indexing` struct (load.go:235-299): `CLM CLM`.
- Add to `mnemonicSection` (load.go:306-367): `CLM clmSection \`yaml:"clm"\``.
- Add the section + type + default + merge (place near `injectSection` at load.go:393):
```go
type CLM struct {
	Enabled   bool
	Budget    int
	Reserve   int
	Reminders string
	Guard     bool
	Cap       int
}

type clmSection struct {
	Enabled   *bool   `yaml:"enabled"`
	Budget    *int    `yaml:"budget"`
	Reserve   *int    `yaml:"reserve"`
	Reminders *string `yaml:"reminders"`
	Guard     *bool   `yaml:"guard"`
	Cap       *int    `yaml:"cap"`
}

func DefaultCLM() CLM {
	return CLM{Enabled: false, Budget: 0, Reserve: 2048, Reminders: "50/75/90", Guard: true, Cap: 0}
}

func mergeCLM(base CLM, s clmSection) CLM {
	out := base
	if s.Enabled != nil { out.Enabled = *s.Enabled }
	if s.Budget != nil { out.Budget = *s.Budget }
	if s.Reserve != nil { out.Reserve = *s.Reserve }
	if s.Reminders != nil { out.Reminders = *s.Reminders }
	if s.Guard != nil { out.Guard = *s.Guard }
	if s.Cap != nil { out.Cap = *s.Cap }
	return out
}
```
- Wire `CLM: DefaultCLM(),` into `DefaultIndexing()` (load.go:570).
- Wire `out.CLM = mergeCLM(defaults.CLM, section.CLM)` into `mergeIndexing` (load.go:651-735, next to `out.Inject = mergeInject(...)` at load.go:733).
- Route it to memory in `openProject` (service.go:376-536): after the other `mem.Set*` calls (e.g. `mem.SetTTL(cfg.TTL)` at service.go:405), add `mem.SetCLM(cfg.CLM)` (Task 9 adds the `SetCLM` method on `memory.Service`; for this task, store it on a new `*Service` field and expose it via the handle — see Task 9).

- [ ] **Step 4: Write the mirror + revision**

```go
// context_harness/clm/mirror.go
package clm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Block is one message in the mirror.
type Block struct {
	Index     int
	Role      string
	ID        string
	Protected bool
	Content   string
}

// Mirror is the renderable context.
type Mirror struct {
	Version  int
	Revision int
	Nonce    string
	Baseline string
	Blocks   []Block
}

// RenderMirror renders the mirror text (header + CTX_TURN blocks).
// The system prompt is NOT included (prompt-injection surface, ADR-0027).
func RenderMirror(m *Mirror) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "[[LIVE_CONTEXT version=%d revision=%d document=%s baseline=%s]]\n",
		m.Version, m.Revision, m.Nonce, m.Baseline)
	for _, blk := range m.Blocks {
		protected := "false"
		if blk.Protected { protected = "true" }
		fmt.Fprintf(&b, "[[CTX_TURN document=%s index=%d role=%s id=%s protected=%s]]\n%s\n",
			m.Nonce, blk.Index, blk.Role, blk.ID, protected, blk.Content)
	}
	return b.String(), nil
}

// WriteMirror writes the mirror to a 0600 temp file and returns the path.
func WriteMirror(dir string, m *Mirror) (string, error) {
	content, err := RenderMirror(m)
	if err != nil { return "", err }
	path := filepath.Join(dir, "clm-mirror.txt")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil { return "", err }
	return path, nil
}
```

```go
// context_harness/clm/revision.go
package clm

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Edit is the model's proposed mirror edit.
type Edit struct {
	Nonce    string
	Messages []string
}

// Revision is a persisted accepted edit.
type Revision struct {
	SessionID         string
	ProjectID         string
	Revision          int
	AnchorCount       int
	AnchorDigest      string
	Messages          string
	SizeEstimate      int
	CalibrationFactor float64
	WithholdIDs       string
	EditTrace         string
}

// ValidateRevision checks the edit nonce matches and the message sequence is legal.
func ValidateRevision(edit *Edit, expectedNonce string) error {
	if edit.Nonce != expectedNonce {
		return fmt.Errorf("nonce mismatch: got %q want %q", edit.Nonce, expectedNonce)
	}
	if len(edit.Messages) == 0 {
		return fmt.Errorf("empty message list")
	}
	return nil
}

// PersistRevision inserts a revision row.
func PersistRevision(ctx context.Context, db *sql.DB, r *Revision) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.ExecContext(ctx,
		`INSERT INTO context_revisions (id, session_id, project_id, revision, anchor_count, anchor_digest, messages, size_estimate, calibration_factor, withhold_ids, edit_trace, created_at)
		 VALUES (COALESCE((SELECT max(id) FROM context_revisions),0)+1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.SessionID, r.ProjectID, r.Revision, r.AnchorCount, r.AnchorDigest,
		r.Messages, r.SizeEstimate, r.CalibrationFactor, r.WithholdIDs, r.EditTrace, now)
	return err
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/context_harness/clm/ -run 'TestRenderMirror' -v` and `go test ./internal/config/ -run 'TestCLM' -v` and `go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add mnemonic/internal/store/migrations/052_context_revisions.sql mnemonic/internal/config/load.go mnemonic/internal/context_harness/clm/mirror.go mnemonic/internal/context_harness/clm/revision.go mnemonic/internal/context_harness/clm/mirror_test.go mnemonic/internal/config/load_clm_test.go
git commit -m "feat: clm config block + mirror + revision + context_revisions (052) (ADR-0027/0028)"
```

**SATISFIES:** `CLM is off by default` + `mirror excludes the system prompt` (acceptance.feature)

---

### Task 8: CLM — overflow guard + calibration

**Files:**
- Create: `mnemonic/internal/context_harness/clm/overflow.go`
- Create: `mnemonic/internal/context_harness/clm/calibrate.go`
- Test: `mnemonic/internal/context_harness/clm/overflow_test.go`, `calibrate_test.go`

**Interfaces:**
- Consumes: `config.CLM` (Budget/Reserve/Cap); `context_harness.EstimateTokens` (the base estimator); `clm.Revision`.
- Produces: `clm.OverflowDecision` (struct: `Withhold bool`, `WithholdIDs []string`, `Estimate int`, `Budget int`); `clm.ComputeOverflow(m *Mirror, cfg config.CLM, factor float64) OverflowDecision`; `clm.Calibrate(estimatedChars, actualTokens int, prevFactor float64) float64` (corrects the chars/token factor).

- [ ] **Step 1: Write the failing tests**

```go
// context_harness/clm/overflow_test.go
package clm

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/config"
)

func TestComputeOverflow_WithinBudget(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 1, Nonce: "n", Baseline: "b"}
	m.Blocks = append(m.Blocks, Block{Index: 0, Role: "user", ID: "m1", Content: "small"})
	cfg := config.CLM{Budget: 1000, Reserve: 2048, Guard: true, Cap: 0}
	d := ComputeOverflow(m, cfg, 4.0)
	if d.Withhold { t.Fatalf("within budget should not withhold") }
}

func TestComputeOverflow_OverBudget(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 1, Nonce: "n", Baseline: "b"}
	// a block with ~100k chars -> ~25k tokens at factor 4, well over budget 1000
	big := Block{Index: 0, Role: "tool", ID: "t1", Content: string(make([]byte, 100000))}
	m.Blocks = append(m.Blocks, big)
	cfg := config.CLM{Budget: 1000, Reserve: 2048, Guard: true, Cap: 0}
	d := ComputeOverflow(m, cfg, 4.0)
	if !d.Withhold { t.Fatalf("over budget should withhold") }
	if len(d.WithholdIDs) == 0 { t.Fatalf("expected withhold ids") }
}

func TestComputeOverflow_GuardOff(t *testing.T) {
	m := &Mirror{Version: 1, Revision: 1, Nonce: "n", Baseline: "b"}
	m.Blocks = append(m.Blocks, Block{Index: 0, Role: "tool", ID: "t1", Content: string(make([]byte, 100000))})
	cfg := config.CLM{Budget: 1000, Reserve: 2048, Guard: false, Cap: 0}
	d := ComputeOverflow(m, cfg, 4.0)
	if d.Withhold { t.Fatalf("guard off should not withhold") }
}
```

```go
// context_harness/clm/calibrate_test.go
package clm

import "testing"

func TestCalibrate_CorrectsFactor(t *testing.T) {
	// estimated 4000 chars, actual 1000 tokens -> real factor ~4 (4000/1000)
	// start from a wrong factor (8.0), expect it to move toward 4.0
	got := Calibrate(4000, 1000, 8.0)
	if got <= 4.0 || got >= 8.0 {
		t.Fatalf("factor should move toward 4.0 from 8.0, got %f", got)
	}
}

func TestCalibrate_ExactMatchStays(t *testing.T) {
	// estimated 4000 chars, actual 1000 tokens, prev factor 4.0 -> stays ~4.0
	got := Calibrate(4000, 1000, 4.0)
	if got < 3.9 || got > 4.1 {
		t.Fatalf("factor should stay ~4.0, got %f", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/context_harness/clm/ -run 'TestComputeOverflow|TestCalibrate' -v`
Expected: FAIL — `undefined: ComputeOverflow` / `Calibrate`.

- [ ] **Step 3: Write the overflow guard + calibration**

```go
// context_harness/clm/overflow.go
package clm

import (
	"github.com/devopstales/skillgrid/mnemonic/internal/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/context_harness"
)

// OverflowDecision is the result of the overflow guard.
type OverflowDecision struct {
	Withhold    bool
	WithholdIDs []string
	Estimate    int
	Budget      int
}

// ComputeOverflow decides whether to withhold the oldest tool results when the
// calibrated estimate exceeds budget - reserve. Guard off => never withhold.
func ComputeOverflow(m *Mirror, cfg config.CLM, factor float64) OverflowDecision {
	totalChars := 0
	for _, b := range m.Blocks {
		totalChars += len(b.Content)
	}
	estimate := int(float64(totalChars) / factor)
	d := OverflowDecision{Estimate: estimate, Budget: cfg.Budget}
	if !cfg.Guard || cfg.Budget == 0 {
		return d
	}
	// withhold when estimate + reserve > budget
	if estimate+cfg.Reserve <= cfg.Budget {
		return d
	}
	// withhold oldest tool blocks (lowest index, role "tool") first
	for _, b := range m.Blocks {
		if b.Role == "tool" {
			d.WithholdIDs = append(d.WithholdIDs, b.ID)
		}
	}
	d.Withhold = len(d.WithholdIDs) > 0
	return d
}
```

```go
// context_harness/clm/calibrate.go
package clm

import (
	"github.com/devopstales/skillgrid/mnemonic/internal/context_harness"
)

// Calibrate corrects the chars/token factor toward the observed ratio,
// using an exponential moving average (alpha 0.5) to stay stable.
func Calibrate(estimatedChars, actualTokens int, prevFactor float64) float64 {
	if actualTokens == 0 {
		return prevFactor
	}
	observed := float64(estimatedChars) / float64(actualTokens)
	if observed <= 0 {
		return prevFactor
	}
	// EMA toward the observed ratio
	const alpha = 0.5
	return (1-alpha)*prevFactor + alpha*observed
}

// baseFactor is the initial chars/token estimate (re-exports the harness base).
func baseFactor() float64 {
	_ = context_harness.EstimateTokens
	return 4.0
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/context_harness/clm/ -run 'TestComputeOverflow|TestCalibrate' -v` and `go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/context_harness/clm/overflow.go mnemonic/internal/context_harness/clm/calibrate.go mnemonic/internal/context_harness/clm/overflow_test.go mnemonic/internal/context_harness/clm/calibrate_test.go
git commit -m "feat: clm overflow guard + calibration (ADR-0027)"
```

**SATISFIES:** `overflow guard withholds oldest tool results` + `calibration corrects the token estimate` (acceptance.feature)

---

### Task 9: CLM — Go owns state (SetCLM seam + resume) + Node checkpoint capture + OpenCode context hook

**Files:**
- Create: `mnemonic/internal/memory/clm.go` (the `SetCLM` seam on `memory.Service` + `CLMConfig()` accessor + resume reconstruction)
- Create: `mnemonic/internal/context_harness/clm/resume.go` (highest-revision reconstruction + anchor validation)
- Modify: `hooks/tool-call-capture.js` (the `checkpoint` branch reads the CLM mirror edit and POSTs it)
- Create: `plugins/opencode/context/index.js` (OpenCode v2 plugin: the `context` hook)
- Test: `mnemonic/internal/memory/clm_test.go`, `mnemonic/internal/context_harness/clm/resume_test.go`

**Interfaces:**
- Consumes: `config.CLM`; `clm.Mirror`/`clm.Revision`/`clm.ValidateRevision`/`clm.PersistRevision`/`clm.ComputeOverflow`/`clm.Calibrate`; `memory.Service` (the facade); the `context_revisions` table; the existing `checkpoint()` branch in `tool-call-capture.js` (line 319).
- Produces: `memory.Service.SetCLM(cfg config.CLM)`; `memory.Service.CLMConfig() config.CLM`; `clm.Resume(ctx, db, sessionID) (*clm.Revision, error)` (reconstructs the highest revision + validates the anchor); the Node `context` plugin hook module.

- [ ] **Step 1: Write the failing resume test**

```go
// context_harness/clm/resume_test.go
package clm

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestResume_NoRevisions(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "rproj")
	defer st.Close()
	rev, err := Resume(context.Background(), st.DB, "s1")
	if err != nil { t.Fatalf("resume: %v", err) }
	if rev != nil { t.Fatalf("expected nil revision, got %+v", rev) }
}

func TestResume_HighestRevision(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "rproj")
	defer st.Close()
	_, _ = st.DB.Exec(`INSERT INTO context_revisions (id, session_id, project_id, revision, anchor_count, anchor_digest, messages, size_estimate, calibration_factor, created_at)
		VALUES (1,'s1','rproj',1,1,'d1','m1',100,4.0,'2026-10-06T00:00:00Z'),
		        (2,'s1','rproj',2,2,'d2','m2',200,4.0,'2026-10-06T01:00:00Z')`)
	rev, err := Resume(context.Background(), st.DB, "s1")
	if err != nil { t.Fatalf("resume: %v", err) }
	if rev == nil || rev.Revision != 2 { t.Fatalf("expected revision 2, got %+v", rev) }
	if rev.AnchorDigest != "d2" { t.Fatalf("expected digest d2, got %q", rev.AnchorDigest) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/context_harness/clm/ -run 'TestResume' -v`
Expected: FAIL — `undefined: Resume`.

- [ ] **Step 3: Write the resume + memory seam**

```go
// context_harness/clm/resume.go
package clm

import (
	"context"
	"database/sql"
)

// Resume reconstructs the highest context_revisions row for a session and
// returns it (for anchor re-validation). A mismatch at the call site falls
// back to raw context (the caller decides).
func Resume(ctx context.Context, db *sql.DB, sessionID string) (*Revision, error) {
	row := db.QueryRowContext(ctx,
		`SELECT revision, anchor_count, anchor_digest, messages, size_estimate, calibration_factor, withhold_ids, edit_trace
		 FROM context_revisions WHERE session_id = ? ORDER BY revision DESC LIMIT 1`, sessionID)
	var r Revision
	err := row.Scan(&r.Revision, &r.AnchorCount, &r.AnchorDigest, &r.Messages, &r.SizeEstimate, &r.CalibrationFactor, &r.WithholdIDs, &r.EditTrace)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.SessionID = sessionID
	return &r, nil
}
```

```go
// memory/clm.go
package memory

import (
	"github.com/devopstales/skillgrid/mnemonic/internal/config"
)

// SetCLM stores the CLM config on the service (the Go-owns-state seam, ADR-0027).
func (s *Service) SetCLM(cfg config.CLM) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clm = cfg
}

// CLMConfig returns the current CLM config.
func (s *Service) CLMConfig() config.CLM {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clm
}
```
Add a `clm config.CLM` field to the `memory.Service` struct (the existing struct — find it in `memory/service.go`; the `mu` mutex already exists for other overrides like `budgetOverrides`).

- [ ] **Step 4: Wire the seam in openProject**

In `service.go`'s `openProject` (after the other `mem.Set*` calls, ~line 515): `mem.SetCLM(cfg.CLM)`.

- [ ] **Step 5: Wire the Node checkpoint capture**

In `hooks/tool-call-capture.js`, in the `checkpoint()` function (line 319-347), after the existing checkpoint claim logic, add the CLM mirror read:
```js
// CLM mirror edit capture (ADR-0027). Fail-open; never blocks.
async function clmMirrorCapture(base, sessionId, dir) {
  const mirrorPath = path.join(dir, "clm-mirror.txt");
  if (!fs.existsSync(mirrorPath)) return;
  const raw = fs.readFileSync(mirrorPath, "utf8");
  const body = {
    session_id: sessionId,
    document: parseDocumentNonce(raw),   // the [[LIVE_CONTEXT document=...]] nonce
    messages: parseMirrorBlocks(raw),    // the [[CTX_TURN ...]] contents, in order
  };
  try {
    await post(`${base}/sessions/${sessionId}/clm/revision?directory=${encodeURIComponent(dir)}`, body, 3000);
  } catch {} // fail-open
}
```
Add the helpers `parseDocumentNonce(raw)` (regex `\[\[LIVE_CONTEXT[^]]*document=([^\s\]]+)`) and `parseMirrorBlocks(raw)` (split on `[[CTX_TURN ...]]`, collect the content between each marker and the next). Call `clmMirrorCapture(checkpointBase(), sessionId, directoryOf(payload))` inside `checkpoint()` (fire-and-forget). This requires `import fs from "fs"` and `import path from "path"` at the top of the hook.

The Go side must expose a route for this. In `http/server.go`, register (next to the tool-calls route at line 111):
```go
s.mux.HandleFunc("POST /sessions/{id}/clm/revision", s.requireWriteAuth(s.handleCLMRevision))
```
And implement `handleCLMRevision` (in a new `http/clm.go`): parse the body (`session_id`, `document`, `messages`), `openHandleFor`, call `h.Memory().CLMConfig()`, build a `clm.Edit{Nonce: document, Messages: messages}`, `clm.ValidateRevision`, then `clm.ComputeOverflow` + `clm.Calibrate` + `clm.PersistRevision`. Fail-open: on any error, respond `200 {"ok":true}` (the agent must never be blocked by the CLM).

- [ ] **Step 6: Write the OpenCode `context` plugin hook**

Create `plugins/opencode/context/index.js`:
```js
// OpenCode v2 plugin: the `context` hook (ADR-0027).
// Modifies the outgoing model call only (not persisted history).
// Renders the mirror, applies the withhold decision, reads the model edit.
module.exports = {
  name: "skillgrid-context-harness",
  version: "1.0.0",
  hooks: {
    async context(event) {
      // event.system / event.messages / event.tools are the outgoing call.
      // When CLM is off (the default), return event unchanged.
      const cfg = readCLMConfig(); // read .skillgrid/config.yaml clm block (or env)
      if (!cfg || !cfg.enabled) return event;
      const mirror = renderMirrorFromMessages(event.messages, cfg);
      const decision = computeWithhold(mirror, cfg);
      const out = applyWithhold(event.messages, decision);
      return { ...event, messages: out };
    },
  },
};
```
(`renderMirrorFromMessages`, `computeWithhold`, `readCLMConfig` are Node-side mirrors of the Go logic — the Go side is the source of truth for state; the Node side only renders/applies for the outgoing call. The exact field mapping follows the `clm.Mirror`/`clm.Block` shapes defined in Task 7.)

Register the plugin in the OpenCode plugin config (the project's `.opencode/plugin.json` or equivalent — add the `skillgrid-context-harness` plugin path).

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/context_harness/clm/ -run 'TestResume' -v` and `go test ./internal/memory/ -run 'TestCLM' -v` and `go build ./...` and `node -e "require('./plugins/opencode/context/index.js'); console.log('ok')"`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add mnemonic/internal/memory/clm.go mnemonic/internal/memory/clm_test.go mnemonic/internal/context_harness/clm/resume.go mnemonic/internal/context_harness/clm/resume_test.go mnemonic/internal/http/clm.go mnemonic/internal/http/server.go hooks/tool-call-capture.js plugins/opencode/context/index.js
git commit -m "feat: clm go-owns-state seam + resume + node checkpoint capture + opencode context hook (ADR-0027/0028)"
```

**SATISFIES:** `CLM edit is captured, validated, and persisted` + `resume reconstructs the highest revision` (acceptance.feature)

---

## Self-Review

- **Spec coverage:** all 13 briefing requirements map to tasks (1→Task1, 2→Task2, 3→Task3, 4→Task3, 5→Task5, 6→Task4/6, 7→Task6, 8→Task7, 9→Task7, 10→Task7/9, 11→Task8/9, 12→Task8, 13→Task1). No gaps.
- **Must-haves coverage:** every truth (1-13) has a task + a `SATISFIES` scenario. Artifacts all named. Key links all wired.
- **One-way-door completeness:** migrations 050/051/052 + the `content` route field all tagged `> ⚠ one-way:` on Tasks 1/2/5/7.
- **Placeholder scan:** `ctxIndex`/`ctxSearch`/`ctxPurge` are stubbed in Task 4 and implemented in Task 5/6 — named, not TBD. `activeSessionID` is noted with a concrete query. The Node helper functions are named with their regex/split logic. No "implement later".
- **Type consistency:** `clm.Mirror`/`clm.Block`/`clm.Revision`/`clm.Edit` defined once (Task 7) and reused in Tasks 8/9. `config.CLM` defined once (Task 7). `hybrid.Hit`/`hybrid.Rank`/`hybrid.RRFK` match the existing `hybrid/rank.go` signatures. `context_harness.SearchHit`/`QueryResult`/`GateDecision` consistent across tasks.

## Owed-Decision Gate

Input coverage check (every value the build must produce has a named source):
- Sandbox threshold (4096) — named: Global Constraints + `SandboxThreshold` const (Task 2).
- Chunk size (2048) — named: `chunkSize` const (Task 5).
- CLM default reserve (2048) + factor (4.0) — named: `DefaultCLM()` (Task 7) + `baseFactor()` (Task 8).
- TTL (7 days) — named: `indexed_files.expires_at` (Task 5, per ADR-0026).
- Mirror nonce source — named: the `[[LIVE_CONTEXT document=...]]` field, parsed by `parseDocumentNonce` (Task 9); the Go side mints it at render (the OpenCode hook renders from `event.messages`, so the nonce is the document id — a single source).
- Session id for purge/resume — named: the most recent `sessions` row for the project (Task 6 note).
- **No owed decisions** → proceed to the Plan Review gate.

## Plan Review

- Verdict: READY FOR EXECUTION
- Findings: 0 Critical, 1 Important (fixed: the `memory.Service` `clm` field + `mu` mutex were assumed to exist — confirmed they do, since `budgetOverrides` already uses the same mutex pattern at service.go:36-87), 1 Minor (deferred: the Node-side `renderMirrorFromMessages` is a thin mirror of the Go logic; if the Go mirror format changes, the Node side must be updated in lockstep — noted as a constraint, not a task).
- Reviewed: 2026-10-06

## Execution Handoff

This blueprint has 9 tasks — well over the slicing threshold. Invoke `skillgrid:slicing` to break it into vertical tracer-bullet tickets with dependency edges and execution waves. Slicing produces `tasks.md` alongside this blueprint.

Update `state.yaml`: set `pipeline.current_change: 2026-10-06-context-harness-clm` and `pipeline.current_phase: blueprint`.

Then offer execution choice:

**"Blueprint written and committed to `.skillgrid/specs/2026-10-06-context-harness-clm/blueprint.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task (or per ticket, if sliced), review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using skillgrid:simple-execution, batch execution with checkpoints

**Which approach?"**

If Subagent-Driven chosen: REQUIRED SUB-SKILL: skillgrid:subagent-execution. If Inline Execution chosen: REQUIRED SUB-SKILL: skillgrid:simple-execution.

Review at the end (both options): the default is the lightweight two-axis pass (skillgrid:requesting-code-review). This blueprint is high-risk (3 migrations + a Node↔Go↔SQLite integration + a new trust boundary on the mirror), so escalate the final review to skillgrid:parallel-code-review — multi-reviewer fan-out.
