# Monitoring: Track Every Tool Call Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread (one end-to-end path — plugin hook → HTTP route → `hookPostToolUse` → `session_events` row → `mem_query_events` read — works first, then thickens with retention, export, and the always-private tool allowlist)

**Goal:** Wire the per-tool-call capture trigger (plugin `tool.execute.after` → `POST /sessions/{id}/tool-calls` → existing `hookPostToolUse`), add a Gryph-style audit query surface (`mem_query_events` + JSONL export), enforce 90-day retention, and extend `<private>` tag stripping to every tool call + an always-private tool allowlist.

**Architecture:** The capture mechanism (`memory.hookPostToolUse`, `session_events` schema, `isSensitivePath`/`redactPreview`, counters) and the `<private>` stripper (`stripPrivateTags` in the plugin) are **already built** — this plan wires the missing trigger and adds the read/retention surface. The plugin's `tool.execute.after` hook (currently Task-only) fires for every non-Task tool call in observe-mode (fire-and-forget, never blocks), POSTs a canonical event to a new HTTP route that calls the existing writer. A new MCP tool `mem_query_events` queries the audit trail with Gryph-style filters. A retention pass deletes events older than `mnemonic.retention_days` (default 90).

**Tech Stack:** Go 1.22+, `modernc.org/sqlite` (existing), `net/http` Go 1.22 method patterns (existing mux), `mark3labs/mcp-go` (existing), TypeScript (opencode plugin, existing).

**Spec:** `.skillgrid/artifacts/07-mnemonic-tool-surface.md` (Monitoring plan section + 5 locked decisions).

**Findings:** `.skillgrid/specs/2026-09-24-mnemonic-vector-db/findings.md` (vector-db spike, cited for the session-inject dependency — this monitoring layer is the capture layer session-inject consumes).

## Terms

- **Observe-mode** — capture-only: the hook logs the tool call but never blocks or rewrites it. (Prismor pattern; the opposite of enforce-mode.)
- **Canonical event** — the normalized per-tool-call shape `{ts, session_id, agent, type, tool_name, path, command, result_status, content_hash}` that the HTTP route and `hookPostToolUse` consume. (Prismor's `contract.py` pattern.)
- **`standard` logging level** — the capture depth: action, path, command, exit code, truncated output, sensitive flag. (Gryph's default level; no file diffs / raw events.)

## Hypothesis

**Claim:** Wiring the plugin's `tool.execute.after` hook to the existing `hookPostToolUse` writer produces a complete per-tool-call audit trail (every read/write/exec/tool call in a live session), queryable via `mem_query_events`, with `<private>` content never persisted.

**Right condition:** After a live session with ≥ 20 tool calls, `session_events` contains a row per tool call (action_type mapped correctly, sensitive paths flagged, `<private>` content absent from `payload`). `mem_query_events --action command_exec --since 1h` returns the exec events. The plugin's `tool.execute.after` fires for non-Task tools.

**Wrong condition:** `session_events` still contains only lifecycle+commit rows after a live session (trigger not firing), OR `mem_query_events` returns no tool-call events, OR `<private>` content appears in a stored `payload`.

**Thinnest MVP:** Task 1 (HTTP route) + Task 2 (plugin hook extension) — the capture path alone. This proves "every tool call is recorded" before the query surface or retention exist.

**Door check:** Task 1 — if `POST /sessions/{id}/tool-calls` does not produce a `session_events` row via the existing `hookPostToolUse`, the writer seam is not reachable from the route and the blueprint is invalidated (the capture mechanism is not what we assumed).

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `POST /sessions/{id}/tool-calls` with a valid body produces one `session_events` row (via `hookPostToolUse`) with the correct `action_type`, `tool_name`, `path`, `command`, `is_sensitive`, and `result_status`.
- `POST /sessions/{id}/tool-calls` for an unknown session returns a session-not-found error and writes no row.
- The plugin's `tool.execute.after` fires for a non-Task tool call (e.g. `bash`, `read`, `write`) and POSTs a canonical event; it does NOT fire for the `Task` tool (existing passive-capture path is unchanged).
- The capture is fire-and-forget: a failed POST (server down, timeout) does not block or fail the tool call (observe-mode, fail-open).
- `<private>...</private>` spans in tool output are stripped (replaced with `[REDACTED]`) before the event is stored — the content never reaches `session_events.payload`.
- Output of a tool in the always-private allowlist is stripped before storage (no raw content in `payload`).
- `mem_query_events` returns `session_events` rows filtered by `action`, `file` (glob), `since`/`until`, `session`; sensitive events are excluded by default.
- `mem_query_events` with `count: true` returns only the matching count, not the rows.
- `mem_query_events` with `sensitive: true` includes sensitive events.
- `mem_export_events` returns the matching events as JSONL (one JSON object per line).
- **backstop:** After a live session with ≥ 20 tool calls, `session_events` contains a row per tool call (requires a runtime test driving the plugin hook against a real store — the diff alone cannot confirm the trigger fires end-to-end).
- **backstop:** `session_events` rows older than `retention_days` are deleted by the retention pass (requires a time-travel or clock-injection test).

**Artifacts** (files that must exist with real implementation, not stubs):
- [`skillgrid-cli/internal/mnemonic/http/toolcalls.go` — `POST /sessions/{id}/tool-calls` route + handler (adapter over `hookPostToolUse`)]
- [`plugins/opencode/mnemonic.ts` (modified) — `tool.execute.after` extension for non-Task tools + always-private tool allowlist]
- [`skillgrid-cli/internal/mnemonic/mcp/tools_query_events.go` — `mem_query_events` + `mem_export_events` MCP tools]
- [`skillgrid-cli/internal/mnemonic/memory/query_events.go` — `QueryEvents` + `ExportEvents` over `session_events`]
- [`skillgrid-cli/internal/mnemonic/memory/retention.go` — `PruneOldEvents` (90-day retention)]
- [`skillgrid-cli/internal/mnemonic/config/` (modified) — `mnemonic.retention_days` + `mnemonic.private_tools` config keys, `hooks.enabled` default flipped to `true`]
- [`skillgrid-cli/internal/mnemonic/http/toolcalls_test.go` — route tests]
- [`skillgrid-cli/internal/mnemonic/memory/query_events_test.go` — query/export tests]
- [`skillgrid-cli/internal/mnemonic/memory/retention_test.go` — retention tests]
- [`plugins/opencode/mnemonic.test.mjs` (modified) — hook-extension tests]
- [`.skillgrid/specs/2026-09-24-mnemonic-monitoring/acceptance.feature` — BDD scenarios]

**Key links** (critical connections between artifacts that must work together):
- The HTTP route MUST call the existing `memory.hookPostToolUse` (via `RunHook(HookPostToolUse, HookPayload{...})`) — not a new writer. The route is a thin adapter.
- The plugin MUST reuse the existing `req()` HTTP client (bearer auth, 3s timeout, fail-open) and the existing `stripPrivateTags` — not a new fetch / new stripper.
- `mem_query_events` MUST read from the `session_events` table (migration 040) — no new table.
- The retention pass MUST delete from `session_events` only (not `sessions`, not `observations`).
- The `hooks.enabled` default flip MUST be the config default (`config/load.go`), not a code constant, so it stays overridable.

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- **Defaulting `mnemonic.hooks.enabled` to `true`** — flips the default behavior for every existing user (every session now writes a row per tool call). Reversible (set `false` in config) but it changes what a fresh install does out of the box. Tagged on Task 4.
- **90-day auto-deletion of `session_events`** — data loss: events older than 90 days are deleted and cannot be recovered. Reversible only by lowering `retention_days` before the prune runs. Tagged on Task 6.

## Global Constraints

- Go 1.22+ minimum to build.
- No new dependencies without an ADR. (This plan adds zero new dependencies.)
- Conventional commits only; no AI-attribution trailers.
- Spec-zone changes commit before code-zone changes (pre-commit zone guard).
- Observe-mode only: capture never blocks, rewrites, or fails a tool call (fail-open).
- `<private>` content and always-private tool output are never persisted (stripped at the edge, before storage).
- Sensitive events (`is_sensitive=1`) are excluded from query/export results by default.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Mnemonic tool surface** (`mem_*`) | Applicable: new tools `mem_query_events` + `mem_export_events` with new params (`action`, `file`, `since`, `until`, `session`, `count`, `sensitive`) and a new return shape (filtered event rows / JSONL). | Explicit contract: both are read-only tools over `session_events`. `mem_query_events` returns `{ events: [...], count, sensitive_included }`; `mem_export_events` returns `{ jsonl: "..." }` (one JSON object per line). Sensitive events excluded by default. | `tools_query_events_test.go`: (1) tools registered; (2) `action` filter returns only matching rows; (3) `sensitive` excluded by default, included with `sensitive: true`; (4) `count: true` returns count only; (5) `file` glob filter; (6) `since`/`until` time window; (7) `mem_export_events` returns valid JSONL. |
| **Git repository selection** | N/A: this change does not run `git -C`, resolve repos, or select worktrees. The plugin captures `tool_name`/`path`/`command` from the tool-call payload; it does not interpret paths as git roots. The `gitHead` call inside `hookPostToolUse` is pre-existing and unchanged. | — | — |
| **Shared-convention drift** | N/A: this change does not edit any `_shared/conventions/*.md`, `_shared/references/*.md`, or `agent-config/*.md`. It updates `.skillgrid/artifacts/07-mnemonic-tool-surface.md` (an artifact, not a convention file) and the opencode plugin. | — | — |

## File Structure

- `skillgrid-cli/internal/mnemonic/http/toolcalls.go` — the `POST /sessions/{id}/tool-calls` route + handler. Thin adapter: parses the canonical event body, maps to `memory.HookPayload`, calls `svc.Memory().RunHook(ctx, HookPostToolUse, payload)`.
- `skillgrid-cli/internal/mnemonic/memory/query_events.go` — `QueryEvents` (filtered read over `session_events`) + `ExportEvents` (JSONL). Read-only.
- `skillgrid-cli/internal/mnemonic/memory/retention.go` — `PruneOldEvents(ctx, days)` (DELETE `session_events` older than N days).
- `skillgrid-cli/internal/mnemonic/mcp/tools_query_events.go` — `mem_query_events` + `mem_export_events` MCP tool registration + handlers.
- `plugins/opencode/mnemonic.ts` — `tool.execute.after` extension (non-Task capture) + always-private tool allowlist + canonical-event builder.
- `skillgrid-cli/internal/mnemonic/config/load.go` — `retention_days` + `private_tools` config keys; `hooks.enabled` default flip.
- Test files as listed in Must-Haves.

---

### Task 1: `POST /sessions/{id}/tool-calls` HTTP Route

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/http/toolcalls.go`
- Create: `skillgrid-cli/internal/mnemonic/http/toolcalls_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/http/server.go:86-95` (register the route in the sessions group)

**Interfaces:**
- Consumes: `memory.RunHook(ctx, HookPostToolUse, HookPayload{...})` (existing, `memory/skills.go:397`); `memory.HookPayload` (existing, `skills.go:262-272`); the server's `requireWriteAuth` wrapper (existing).
- Produces:
  - `POST /sessions/{id}/tool-calls` — body: `{ session_id, agent, type, tool_name, path, command, result_status, content_hash, content_preview }`. Writes one `session_events` row via `hookPostToolUse`. Returns `200 { ok: true }` or `404` (unknown session) / `400` (bad body).
- Seam: the route is an adapter over `RunHook`. The seam is `RunHook` (the existing writer). Deleting the route leaves `RunHook` reachable only from the CLI — the plugin has no capture path.
- Deletion test: if the route is deleted, the plugin's POST has no target — capture is dead.
- Adapters: 1 (real `RunHook`). A test that seeds a store + drives the route in-process (the existing `single_open_test.go` pattern) is the test double.

**SATISFIES:** `toolcall-route-writes-row`, `toolcall-route-unknown-session`, `toolcall-route-sensitive-flag`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/http/toolcalls_test.go
package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestToolCallRoute_WritesRow(t *testing.T) {
	s, cleanup := newTestServer(t) // existing helper: temp store + server + service
	defer cleanup()
	// Register a session first (the route requires a known session).
	sid := registerTestSession(t, s, "test-project")

	body := map[string]any{
		"session_id":    sid,
		"agent":         "opencode",
		"type":          "shell",
		"tool_name":     "bash",
		"command":       "go test ./...",
		"result_status": "success",
	}
	res := postToolCall(t, s, sid, body)
	if res.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", res.Code, res.Body.String())
	}
	// Verify a session_events row was written.
	events, _, _ := s.svc.Memory().SessionChanges(t.Context(), sid)
	var found bool
	for _, e := range events {
		if e.ActionType == "command_exec" && e.ToolName == "bash" && e.Command == "go test ./..." {
			found = true
		}
	}
	if !found {
		t.Errorf("no command_exec row for bash; events: %+v", events)
	}
}

func TestToolCallRoute_UnknownSession(t *testing.T) {
	s, cleanup := newTestServer(t)
	defer cleanup()
	body := map[string]any{
		"session_id": "no-such-session",
		"tool_name":  "bash",
		"command":    "ls",
	}
	res := postToolCall(t, s, "no-such-session", body)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", res.Code)
	}
}

func TestToolCallRoute_SensitiveFlag(t *testing.T) {
	s, cleanup := newTestServer(t)
	defer cleanup()
	sid := registerTestSession(t, s, "test-project")
	body := map[string]any{
		"session_id":    sid,
		"tool_name":     "read",
		"path":          "/home/u/.env",
		"result_status": "success",
	}
	res := postToolCall(t, s, sid, body)
	if res.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", res.Code)
	}
	events, _, _ := s.svc.Memory().SessionChanges(t.Context(), sid)
	var found bool
	for _, e := range events {
		if e.ActionType == "file_read" && e.IsSensitive {
			found = true
		}
	}
	if !found {
		t.Errorf("no sensitive file_read row for .env; events: %+v", events)
	}
}

func postToolCall(t *testing.T, s *Server, sid string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/sessions/"+sid+"/tool-calls", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if s.writeToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.writeToken)
	}
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	return w
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/http/ -run TestToolCallRoute -v`
Expected: FAIL (404 on the route — not registered yet)

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/http/toolcalls.go
package http

import (
	"encoding/json"
	"net/http"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// toolCallBody is the canonical per-tool-call event the plugin POSTs. It maps
// 1:1 onto memory.HookPayload (the existing writer's input).
type toolCallBody struct {
	SessionID     string `json:"session_id"`
	Agent         string `json:"agent"`
	Type          string `json:"type"`
	ToolName      string `json:"tool_name"`
	Path          string `json:"path"`
	Command       string `json:"command"`
	ResultStatus  string `json:"result_status"`
	ContentHash   string `json:"content_hash"`
	ContentPreview string `json:"content_preview"`
}

func (s *Server) handleToolCallCreate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var b toolCallBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if b.SessionID == "" {
		b.SessionID = id
	}
	payload := memory.HookPayload{
		SessionID:      b.SessionID,
		ToolName:       b.ToolName,
		ActionType:     b.Type,
		File:           b.Path,
		Command:        b.Command,
		ResultStatus:   b.ResultStatus,
		ContentHash:    b.ContentHash,
		ContentPreview: b.ContentPreview,
	}
	// RunHook returns session-not-found for an unknown session — map to 404.
	if _, err := s.svc.Memory().RunHook(r.Context(), memory.HookPostToolUse, payload); err != nil {
		if isSessionNotFound(err) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// isSessionNotFound reports whether err is a session-not-found error from the
// memory layer (the exact sentinel is matched via errors.Is on the memory
// package's not-found error, if one exists; otherwise a substring match on
// "not found").
func isSessionNotFound(err error) bool {
	return err != nil && (contains(err.Error(), "not found"))
}
```

Register in `server.go` (in the sessions group, after `POST /sessions/{id}/end`):
```go
s.mux.HandleFunc("POST /sessions/{id}/tool-calls", s.requireWriteAuth(s.handleToolCallCreate))
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/http/ -run TestToolCallRoute -v`
Expected: PASS (all 3 tests)

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/http/toolcalls.go
git add skillgrid-cli/internal/mnemonic/http/toolcalls_test.go
git add skillgrid-cli/internal/mnemonic/http/server.go
git commit -m "feat(monitoring): POST /sessions/{id}/tool-calls capture route

Adds the HTTP route that the opencode plugin POSTs per tool call. Thin
adapter over the existing memory.RunHook(HookPostToolUse) writer — no
new table, no new writer. Returns 404 for unknown sessions.

[skillgrid-context]
Change: 2026-09-24-mnemonic-monitoring
Phase: apply
"
```

### Task 2: Plugin `tool.execute.after` Extension (Capture Trigger)

**Files:**
- Modify: `plugins/opencode/mnemonic.ts:451-466` (extend the `tool.execute.after` handler)
- Modify: `plugins/opencode/mnemonic.test.mjs` (hook-extension tests)

**Interfaces:**
- Consumes: the existing `req()` HTTP client (`mnemonic.ts:47`), `stripPrivateTags` (`mnemonic.ts:79`), `resolveRoot` (existing), `ensureSession` (existing), `projectFor` (existing).
- Produces: the `tool.execute.after` hook now fires for every non-Task tool call, POSTs a canonical event to `POST /sessions/{id}/tool-calls`, and strips `<private>` spans + always-private tool output before the POST.
- Seam: the plugin hook boundary (opencode's `tool.execute.after`). The seam is the hook itself — observe-mode, fail-open.
- Deletion test: if the hook extension is deleted, only Task output is captured (the pre-change state).
- Adapters: 1 (opencode). The Kilo plugin (`plugins/kilo/mnemonic.ts`) is a sibling copy — it gets the same extension in the same task (two files, one change).

**SATISFIES:** `plugin-hook-fires-non-task`, `plugin-hook-not-task`, `plugin-hook-private-strip`, `plugin-hook-always-private`, `plugin-hook-fail-open`

**Always-private allowlist source:** the plugin reads the allowlist from a new env var `SKILLGRID_MNEMONIC_PRIVATE_TOOLS` (comma-separated tool names, e.g. `mem_save,mem_save_prompt`). The Go side also reads `mnemonic.private_tools` from config (Task 4) — but the plugin is the one that strips, so the plugin-side env var is the source of truth for stripping. The Go config is for the query surface (knowing which tools' output is always redacted). The two must be kept in sync by the installer (`setup/opencode.go` writes the env var from the config).

- [ ] **Step 1: Write the failing test**

```javascript
// plugins/opencode/mnemonic.test.mjs (append)
import { test } from 'node:test'
import assert from 'node:assert'

test('tool.execute.after fires for non-Task tools and POSTs', async (t) => {
  // Mock req() to capture the POST.
  const posts = []
  t.mock.method(pluginMod, 'req', async (method, path, body) => {
    if (method === 'POST') posts.push({ path, body })
    return { ok: true }
  })
  // Simulate a bash tool call.
  await pluginMod.MnemonicHookAfter({ tool: 'bash', sessionID: 'sess-1', args: { command: 'ls' } }, 'success')
  assert.equal(posts.length, 1)
  assert.ok(posts[0].path.includes('/tool-calls'))
  assert.equal(posts[0].body.tool_name, 'bash')
})

test('tool.execute.after does NOT fire for Task', async (t) => {
  const posts = []
  t.mock.method(pluginMod, 'req', async (m, p, b) => { if (m === 'POST') posts.push({ p, b }); return {} })
  await pluginMod.MnemonicHookAfter({ tool: 'Task', sessionID: 'sess-1' }, 'some output')
  // Task is handled by the existing passive-capture path, not the new tool-call POST.
  assert.ok(!posts.some((p) => p.p.includes('/tool-calls')))
})

test('tool.execute.after strips <private> spans', async (t) => {
  let captured
  t.mock.method(pluginMod, 'req', async (m, p, b) => { captured = b; return {} })
  await pluginMod.MnemonicHookAfter(
    { tool: 'read', sessionID: 'sess-1', args: { file: '/x' } },
    'value is <private>secret123</private> end',
  )
  assert.ok(!String(captured.content_preview ?? captured.content ?? '').includes('secret123'))
})

test('tool.execute.after strips always-private tool output', async (t) => {
  process.env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS = 'mem_save'
  let captured
  t.mock.method(pluginMod, 'req', async (m, p, b) => { captured = b; return {} })
  await pluginMod.MnemonicHookAfter({ tool: 'mem_save', sessionID: 'sess-1' }, 'full body here')
  assert.ok(!String(captured.content_preview ?? captured.content ?? '').includes('full body here'))
  delete process.env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS
})
```

Note: the test requires the hook-after logic to be extractable as a testable function (`MnemonicHookAfter`) — the current inline `tool.execute.after` arrow function must be refactored into a named function the test can call. This is a mechanical extraction (move the body into `export async function MnemonicHookAfter(input, output)` and call it from the arrow).

- [ ] **Step 2: Run test to verify it fails**

Run: `node --test plugins/opencode/mnemonic.test.mjs`
Expected: FAIL (the non-Task POST is not captured; `MnemonicHookAfter` not exported)

- [ ] **Step 3: Write minimal implementation**

In `plugins/opencode/mnemonic.ts`, refactor the `tool.execute.after` body into a named export and extend it:

```typescript
// Always-private tool allowlist (comma-separated env var, set by the installer).
const PRIVATE_TOOLS = new Set(
  (process.env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS ?? "").split(",").map((s) => s.trim()).filter(Boolean),
)

// Canonical per-tool-call event builder. Maps the opencode tool-call shape to
// the canonical event the Go route expects.
function buildToolCallEvent(tool: string, args: any, output: any, sessionId: string) {
  const rawOutput = typeof output === "string" ? output : JSON.stringify(output ?? "")
  let contentPreview = truncate(rawOutput, 200)
  // Edge-processing privacy: strip <private> spans (existing), then always-
  // private tool output (allowlist) — the content never hits the wire.
  contentPreview = stripPrivateTags(contentPreview)
  if (PRIVATE_TOOLS.has(tool)) contentPreview = "[REDACTED]"
  const type = /write|edit/i.test(tool) ? "file_write"
    : /read|view/i.test(tool) ? "file_read"
    : /shell|bash|exec|command/i.test(tool) ? "shell"
    : "tool_use"
  return {
    session_id: sessionId,
    agent: "opencode",
    type,
    tool_name: tool,
    path: typeof args?.file === "string" ? args.file : (typeof args?.path === "string" ? args.path : ""),
    command: typeof args?.command === "string" ? args.command : "",
    result_status: /error|fail/i.test(String(output ?? "")) ? "error" : "success",
    content_preview: contentPreview,
  }
}

// Extracted, testable hook-after body. Task is handled by the existing
// passive-capture path (unchanged); every other tool call POSTs a canonical
// event to /sessions/{id}/tool-calls (observe-mode, fail-open).
export async function MnemonicHookAfter(input: any, output: any): Promise<void> {
  const tool = String(input?.tool ?? "")
  if (tool === "Task") return // existing passive-capture path handles Task
  const client = String(input?.sessionID ?? "")
  const sessionId = await resolveRoot(client)
  if (!sessionId) return
  if (!(await ensureSession(sessionId))) return
  const event = buildToolCallEvent(tool, input?.args ?? {}, output, sessionId)
  // Fail-open: a capture failure never blocks the tool call.
  await req("POST", `/sessions/${encodeURIComponent(sessionId)}/tool-calls`, event)
}
```

Wire it into the plugin object (replace the inline `tool.execute.after` body):
```typescript
"tool.execute.after": async (input: any, output: any) => {
  // Existing Task passive-capture (unchanged):
  if (String(input?.tool ?? "") === "Task") {
    /* ...existing Task code... */
    return
  }
  // New: every other tool call.
  await MnemonicHookAfter(input, output)
},
```

Apply the same change to `plugins/kilo/mnemonic.ts` (sibling copy).

- [ ] **Step 4: Run test to verify it passes**

Run: `node --test plugins/opencode/mnemonic.test.mjs`
Expected: PASS (all 4 tests)

- [ ] **Step 5: Commit**

```bash
git add plugins/opencode/mnemonic.ts plugins/opencode/mnemonic.test.mjs
git add plugins/kilo/mnemonic.ts plugins/kilo/mnemonic.test.mjs
git commit -m "feat(monitoring): plugin tool.execute.after captures every tool call

Extends the opencode/kilo plugin's tool.execute.after hook (previously
Task-only) to POST a canonical event for every non-Task tool call to
POST /sessions/{id}/tool-calls. Observe-mode, fail-open. Strips
<private> spans + always-private tool output (allowlist) at the edge.

[skillgrid-context]
Change: 2026-09-24-mnemonic-monitoring
Phase: apply
"
```

### Task 3: `mem_query_events` + `mem_export_events` MCP Tools

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/memory/query_events.go`
- Create: `skillgrid-cli/internal/mnemonic/memory/query_events_test.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_query_events.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_query_events_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go` (register the tools)

**Interfaces:**
- Consumes: `session_events` table (migration 040), `store.DB` (existing `*sql.DB`).
- Produces:
  - `QueryEvents(ctx, opts QueryEventOpts) ([]Event, int, error)` — filtered read over `session_events`. `opts`: `Action`, `File` (glob), `Since`, `Until`, `Session`, `IncludeSensitive`, `CountOnly`. Returns matching events (in sequence order) + total count.
  - `ExportEvents(ctx, opts QueryEventOpts) (string, error)` — JSONL (one JSON object per line) of the matching events.
  - `mem_query_events` MCP tool — params: `action`, `file`, `since`, `until`, `session`, `count` (bool), `sensitive` (bool). Returns `{ events: [...], count, sensitive_included }`.
  - `mem_export_events` MCP tool — same filter params. Returns `{ jsonl: "..." }`.
- Seam: none (in-process, read-only SQL).
- Deletion test: if `query_events.go` is deleted, the MCP tools have no query logic.
- Adapters: 1 (real store). Test seeds a store with known events.

**SATISFIES:** `query-action-filter`, `query-sensitive-default-excluded`, `query-sensitive-included`, `query-count-only`, `query-file-glob`, `query-time-window`, `export-jsonl`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/memory/query_events_test.go
package memory

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestQueryEvents_ActionFilter(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	sid := seedSessionWithEvents(t, svc, 10) // mixed action types
	ctx := context.Background()

	events, count, err := svc.QueryEvents(ctx, QueryEventOpts{Action: "command_exec", Session: sid})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if count == 0 {
		t.Fatal("expected command_exec events, got 0")
	}
	for _, e := range events {
		if e.ActionType != "command_exec" {
			t.Errorf("unexpected action %q in results", e.ActionType)
		}
	}
}

func TestQueryEvents_SensitiveDefaultExcluded(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	sid := seedSessionWithSensitiveEvents(t, svc, 5, 2) // 2 sensitive
	ctx := context.Background()

	events, _, _ := svc.QueryEvents(ctx, QueryEventOpts{Session: sid})
	for _, e := range events {
		if e.IsSensitive {
			t.Errorf("sensitive event leaked (default should exclude)")
		}
	}
}

func TestQueryEvents_SensitiveIncluded(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	sid := seedSessionWithSensitiveEvents(t, svc, 5, 2)
	ctx := context.Background()

	events, _, _ := svc.QueryEvents(ctx, QueryEventOpts{Session: sid, IncludeSensitive: true})
	var found bool
	for _, e := range events {
		if e.IsSensitive {
			found = true
		}
	}
	if !found {
		t.Error("sensitive event not included with IncludeSensitive=true")
	}
}

func TestQueryEvents_CountOnly(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	sid := seedSessionWithEvents(t, svc, 10)
	ctx := context.Background()

	events, count, _ := svc.QueryEvents(ctx, QueryEventOpts{Session: sid, CountOnly: true})
	if len(events) != 0 {
		t.Errorf("CountOnly should return no events, got %d", len(events))
	}
	if count == 0 {
		t.Error("CountOnly should return a non-zero count")
	}
}

func TestExportEvents_JSONL(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	sid := seedSessionWithEvents(t, svc, 5)
	ctx := context.Background()

	jsonl, err := svc.ExportEvents(ctx, QueryEventOpts{Session: sid})
	if err != nil {
		t.Fatalf("ExportEvents: %v", err)
	}
	lines := splitLines(jsonl)
	if len(lines) == 0 {
		t.Fatal("expected JSONL lines, got none")
	}
	// Each line must be valid JSON.
	for _, line := range lines {
		if !isValidJSON(line) {
			t.Errorf("invalid JSONL line: %q", line)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestQueryEvents|TestExportEvents' -v`
Expected: FAIL with "undefined: QueryEvents" / "undefined: QueryEventOpts"

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/memory/query_events.go
package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// QueryEventOpts tunes a session_events query (Gryph-style filters).
type QueryEventOpts struct {
	Action           string // action_type filter (file_read, file_write, command_exec, tool_use, commit)
	File             string // path glob (e.g. "src/auth/**")
	Since            string // RFC3339 lower bound (inclusive)
	Until            string // RFC3339 upper bound (inclusive)
	Session          string // session_id filter
	IncludeSensitive bool   // include is_sensitive=1 events (default false)
	CountOnly        bool   // return count only, no events
	Project          string // project scope (empty = all projects for the store)
}

// QueryEvents reads session_events with the given filters, in sequence order.
// Sensitive events are excluded unless IncludeSensitive is true. Returns the
// matching events (nil when CountOnly) and the total count.
func (s *Service) QueryEvents(ctx context.Context, opts QueryEventOpts) ([]Event, int, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, 0, fmt.Errorf("memory service not initialized")
	}
	where := []string{"1=1"}
	args := []any{}
	if opts.Session != "" {
		where = append(where, "session_id = ?")
		args = append(args, opts.Session)
	}
	if opts.Action != "" {
		where = append(where, "action_type = ?")
		args = append(args, opts.Action)
	}
	if opts.File != "" {
		where = append(where, "path LIKE ?")
		args = append(args, opts.File)
	}
	if opts.Since != "" {
		where = append(where, "timestamp >= ?")
		args = append(args, opts.Since)
	}
	if opts.Until != "" {
		where = append(where, "timestamp <= ?")
		args = append(args, opts.Until)
	}
	if !opts.IncludeSensitive {
		where = append(where, "is_sensitive = 0")
	}

	countSQL := "SELECT COUNT(*) FROM session_events WHERE " + strings.Join(where, " AND ")
	var count int
	if err := s.store.DB.QueryRowContext(ctx, countSQL, args...).Scan(&count); err != nil {
		return nil, 0, fmt.Errorf("query events count: %w", err)
	}
	if opts.CountOnly {
		return nil, count, nil
	}

	rowsSQL := `
		SELECT id, session_id, project, sequence, action_type,
		       COALESCE(result_status, 'success'), COALESCE(is_sensitive, 0),
		       COALESCE(tool_name, ''), COALESCE(path, ''), COALESCE(command, ''),
		       COALESCE("commit", ''), COALESCE(payload, ''), timestamp
		FROM session_events WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY session_id, sequence`
	rows, err := s.store.DB.QueryContext(ctx, rowsSQL, args...)
	if err != nil {
		return nil, count, fmt.Errorf("query events rows: %w", err)
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		var e Event
		var sensitive int
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Project, &e.Sequence, &e.ActionType,
			&e.ResultStatus, &sensitive, &e.ToolName, &e.Path, &e.Command,
			&e.Commit, &e.Payload, &e.Timestamp); err != nil {
			return nil, count, fmt.Errorf("scan event: %w", err)
		}
		e.IsSensitive = sensitive != 0
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, count, err
	}
	return events, count, nil
}

// ExportEvents returns the matching events as JSONL (one JSON object per line).
func (s *Service) ExportEvents(ctx context.Context, opts QueryEventOpts) (string, error) {
	events, _, err := s.QueryEvents(ctx, opts)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for i, e := range events {
		raw, err := json.Marshal(e)
		if err != nil {
			return "", fmt.Errorf("marshal event: %w", err)
		}
		sb.Write(raw)
		if i < len(events)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String(), nil
}
```

```go
// skillgrid-cli/internal/mnemonic/mcp/tools_query_events.go
package mcp

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

func registerQueryEventTools(s *server.MCPServer) {
	s.AddTool(memQueryEventsTool(), handleMemQueryEvents)
	s.AddTool(memExportEventsTool(), handleMemExportEvents)
}

func memQueryEventsTool() mcplib.Tool {
	return mcplib.NewTool("mem_query_events",
		mcplib.WithDescription("Query the per-tool-call audit trail (session_events) with Gryph-style filters: action, file glob, since/until, session. Sensitive events are excluded by default. Use count:true for a count-only query."),
		mcplib.WithString("action", mcplib.Description("Filter by action_type: file_read, file_write, command_exec, tool_use, commit.")),
		mcplib.WithString("file", mcplib.Description("Filter by path glob (e.g. 'src/auth/**').")),
		mcplib.WithString("since", mcplib.Description("RFC3339 lower bound (inclusive).")),
		mcplib.WithString("until", mcplib.Description("RFC3339 upper bound (inclusive).")),
		mcplib.WithString("session", mcplib.Description("Filter by session id.")),
		mcplib.WithBoolean("count", mcplib.Description("When true, return only the count, not the events.")),
		mcplib.WithBoolean("sensitive", mcplib.Description("When true, include sensitive events (excluded by default).")),
	)
}

func memExportEventsTool() mcplib.Tool {
	return mcplib.NewTool("mem_export_events",
		mcplib.WithDescription("Export the per-tool-call audit trail as JSONL (one JSON object per line) with the same filters as mem_query_events."),
		mcplib.WithString("action", mcplib.Description("Filter by action_type.")),
		mcplib.WithString("file", mcplib.Description("Filter by path glob.")),
		mcplib.WithString("since", mcplib.Description("RFC3339 lower bound.")),
		mcplib.WithString("until", mcplib.Description("RFC3339 upper bound.")),
		mcplib.WithString("session", mcplib.Description("Filter by session id.")),
		mcplib.WithBoolean("sensitive", mcplib.Description("When true, include sensitive events.")),
	)
}

func eventOptsFromReq(req mcplib.CallToolRequest) memory.QueryEventOpts {
	return memory.QueryEventOpts{
		Action:           req.GetString("action", ""),
		File:             req.GetString("file", ""),
		Since:            req.GetString("since", ""),
		Until:            req.GetString("until", ""),
		Session:          req.GetString("session", ""),
		IncludeSensitive: req.GetBool("sensitive", false),
		CountOnly:        req.GetBool("count", false),
	}
}

func handleMemQueryEvents(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	opts := eventOptsFromReq(req)
	events, count, err := svc.Memory().QueryEvents(ctx, opts)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{
		"events":             events,
		"count":              count,
		"sensitive_included": opts.IncludeSensitive,
	})
}

func handleMemExportEvents(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	svc, err := rootService()
	if err != nil {
		return toolError(err)
	}
	opts := eventOptsFromReq(req)
	jsonl, err := svc.Memory().ExportEvents(ctx, opts)
	if err != nil {
		return toolError(err)
	}
	return JSONResult(map[string]any{"jsonl": jsonl})
}
```

Register in `server.go` — add `registerQueryEventTools(s)` to both `Start` and `NewServer`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestQueryEvents|TestExportEvents' -v && go test ./skillgrid-cli/internal/mnemonic/mcp/ -run TestMemQueryEvents -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/memory/query_events.go
git add skillgrid-cli/internal/mnemonic/memory/query_events_test.go
git add skillgrid-cli/internal/mnemonic/mcp/tools_query_events.go
git add skillgrid-cli/internal/mnemonic/mcp/tools_query_events_test.go
git add skillgrid-cli/internal/mnemonic/mcp/server.go
git commit -m "feat(monitoring): mem_query_events + mem_export_events MCP tools

Adds the Gryph-style audit query surface: mem_query_events (action/file/
since/until/session filters, sensitive excluded by default, count-only
mode) and mem_export_events (JSONL). Read-only over session_events.

[skillgrid-context]
Change: 2026-09-24-mnemonic-monitoring
Phase: apply
"
```

### Task 4: Config — `retention_days`, `private_tools`, `hooks.enabled` default flip

> ⚠ one-way: **defaulting `mnemonic.hooks.enabled` to `true`** — flips the default behavior for every existing user (every session now writes a row per tool call). Reversible (set `false` in config) but changes what a fresh install does. STOP and get explicit user approval before committing.

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/config/load.go` (add `retention_days` + `private_tools` keys; flip `hooks.enabled` default to `true`)
- Modify: `skillgrid-cli/internal/mnemonic/config/load_test.go` (default-flip test)
- Modify: `skillgrid-cli/internal/mnemonic/setup/opencode.go` (write `SKILLGRID_MNEMONIC_PRIVATE_TOOLS` env var from config during install)

**Interfaces:**
- Consumes: the existing `config` struct + `load.go` parsing.
- Produces:
  - `config.Mnemonic.RetentionDays int` (default 90)
  - `config.Mnemonic.PrivateTools []string` (default empty)
  - `config.Mnemonic.Hooks.Enabled bool` — default flipped from `false` to `true`
  - The installer writes `SKILLGRID_MNEMONIC_PRIVATE_TOOLS=<comma-joined>` into the opencode env config when `PrivateTools` is non-empty.
- Seam: the config struct (the source of truth for the defaults).
- Deletion test: if the config keys are deleted, retention has no configurable window and the always-private allowlist has no source.
- Adapters: 1 (real config). Test via the existing `load_test.go` patterns.

**SATISFIES:** `config-hooks-default-on`, `config-retention-days-default`, `config-private-tools`, `installer-writes-private-tools-env`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/config/load_test.go (append)
func TestLoad_HooksDefaultOn(t *testing.T) {
	cfg := loadFromYAML(t, "mnemonic:\n  enabled: true\n") // no hooks section
	if !cfg.Mnemonic.Hooks.Enabled {
		t.Error("hooks.enabled should default to true (observe-mode)")
	}
}

func TestLoad_RetentionDaysDefault(t *testing.T) {
	cfg := loadFromYAML(t, "mnemonic:\n  enabled: true\n")
	if cfg.Mnemonic.RetentionDays != 90 {
		t.Errorf("retention_days default = %d, want 90", cfg.Mnemonic.RetentionDays)
	}
}

func TestLoad_PrivateTools(t *testing.T) {
	cfg := loadFromYAML(t, "mnemonic:\n  enabled: true\n  private_tools: [mem_save, mem_save_prompt]\n")
	if len(cfg.Mnemonic.PrivateTools) != 2 || cfg.Mnemonic.PrivateTools[0] != "mem_save" {
		t.Errorf("private_tools = %v, want [mem_save mem_save_prompt]", cfg.Mnemonic.PrivateTools)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/config/ -run 'TestLoad_HooksDefaultOn|TestLoad_RetentionDaysDefault|TestLoad_PrivateTools' -v`
Expected: FAIL (hooks default is still `false`; `RetentionDays`/`PrivateTools` fields don't exist)

- [ ] **Step 3: Write minimal implementation**

In `config/load.go`, add the fields to the `Mnemonic` config struct and set the defaults:
```go
// In the Mnemonic config struct:
RetentionDays int      `yaml:"retention_days"` // default 90
PrivateTools  []string `yaml:"private_tools"`  // default empty

// In the default-application (where hooks.enabled default is set, ~load.go:124,454-457):
if cfg.Mnemonic.RetentionDays <= 0 {
    cfg.Mnemonic.RetentionDays = 90
}
// Flip the hooks.enabled default:
if !hooksEnabledExplicitlySet(cfg) {
    cfg.Mnemonic.Hooks.Enabled = true // was false
}
```

In `setup/opencode.go`, when writing the opencode env config, add:
```go
if len(cfg.Mnemonic.PrivateTools) > 0 {
    env["SKILLGRID_MNEMONIC_PRIVATE_TOOLS"] = strings.Join(cfg.Mnemonic.PrivateTools, ",")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/config/ -run 'TestLoad_HooksDefaultOn|TestLoad_RetentionDaysDefault|TestLoad_PrivateTools' -v`
Expected: PASS

- [ ] **Step 5: Commit** (after user approval of the one-way-door)

```bash
git add skillgrid-cli/internal/mnemonic/config/load.go
git add skillgrid-cli/internal/mnemonic/config/load_test.go
git add skillgrid-cli/internal/mnemonic/setup/opencode.go
git commit -m "feat(monitoring): retention_days + private_tools config; hooks default on

Adds mnemonic.retention_days (default 90) and mnemonic.private_tools
config keys. Flips mnemonic.hooks.enabled default to true (observe-mode
is safe: writes rows, never blocks). Installer writes
SKILLGRID_MNEMONIC_PRIVATE_TOOLS env var from config.

[skillgrid-context]
Change: 2026-09-24-mnemonic-monitoring
Phase: apply
"
```

### Task 5: 90-Day Retention Pass

> ⚠ one-way: **90-day auto-deletion of `session_events`** — data loss: events older than 90 days are deleted and cannot be recovered. Reversible only by lowering `retention_days` before the prune runs.

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/memory/retention.go`
- Create: `skillgrid-cli/internal/mnemonic/memory/retention_test.go`
- Modify: the serve startup path (where the server initializes — call `PruneOldEvents` once at startup)

**Interfaces:**
- Consumes: `session_events` table, `store.DB`, `config.Mnemonic.RetentionDays`.
- Produces:
  - `PruneOldEvents(ctx context.Context, days int) (int64, error)` — DELETEs `session_events` rows older than `days`, returns the row count deleted. Best-effort: a failure is logged, never fatal.
- Seam: none (in-process SQL DELETE).
- Deletion test: if `retention.go` is deleted, events are never pruned (unbounded growth).
- Adapters: 1 (real store). Test injects a clock or seeds old timestamps.

**SATISFIES:** `retention-prunes-old`, `retention-keeps-recent`, `retention-best-effort`

- [ ] **Step 1: Write the failing test**

```go
// skillgrid-cli/internal/mnemonic/memory/retention_test.go
package memory

import (
	"context"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestPruneOldEvents_PrunesOld(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	// Seed 10 events, 5 with timestamps 100 days ago, 5 recent.
	seedEventsWithAges(t, svc, []int{100, 100, 100, 100, 100, 1, 1, 1, 1, 1}) // days ago
	ctx := context.Background()

	deleted, err := svc.PruneOldEvents(ctx, 90)
	if err != nil {
		t.Fatalf("PruneOldEvents: %v", err)
	}
	if deleted != 5 {
		t.Errorf("deleted = %d, want 5", deleted)
	}
	// Verify only recent events remain.
	events, _, _ := svc.QueryEvents(ctx, QueryEventOpts{IncludeSensitive: true})
	for _, e := range events {
		age := time.Since(parseTS(e.Timestamp))
		if age > 90*24*time.Hour {
			t.Errorf("event older than 90d survived: %s", e.Timestamp)
		}
	}
}

func TestPruneOldEvents_KeepsRecent(t *testing.T) {
	st := store.NewTestStore(t)
	svc := newTestService(t, st)
	seedEventsWithAges(t, svc, []int{1, 2, 3}) // all recent
	ctx := context.Background()

	deleted, _ := svc.PruneOldEvents(ctx, 90)
	if deleted != 0 {
		t.Errorf("deleted = %d, want 0", deleted)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./skillgrid-cli/internal/mnemonic/memory/ -run TestPruneOldEvents -v`
Expected: FAIL with "undefined: PruneOldEvents"

- [ ] **Step 3: Write minimal implementation**

```go
// skillgrid-cli/internal/mnemonic/memory/retention.go
package memory

import (
	"context"
	"fmt"
	"time"
)

// PruneOldEvents deletes session_events rows older than days and returns the
// count deleted. Best-effort: a failure is returned but the caller (serve
// startup) logs it and continues — retention is not a session-critical path.
func (s *Service) PruneOldEvents(ctx context.Context, days int) (int64, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, fmt.Errorf("memory service not initialized")
	}
	if days <= 0 {
		days = 90
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	res, err := s.store.DB.ExecContext(ctx,
		`DELETE FROM session_events WHERE timestamp < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune old events: %w", err)
	}
	return res.RowsAffected()
}
```

Wire into serve startup (the server init path — call `PruneOldEvents` once after the store opens, best-effort):
```go
if n, err := svc.Memory().PruneOldEvents(ctx, cfg.Mnemonic.RetentionDays); err != nil {
    log.Printf("mnemonic: retention prune: %v", err)
} else if n > 0 {
    log.Printf("mnemonic: retention pruned %d old events", n)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./skillgrid-cli/internal/mnemonic/memory/ -run TestPruneOldEvents -v`
Expected: PASS

- [ ] **Step 5: Commit** (after user approval of the one-way-door)

```bash
git add skillgrid-cli/internal/mnemonic/memory/retention.go
git add skillgrid-cli/internal/mnemonic/memory/retention_test.go
git add <serve startup file>
git commit -m "feat(monitoring): 90-day retention pass for session_events

Adds PruneOldEvents (DELETE session_events older than N days, default 90).
Called once at serve startup, best-effort (a failure is logged, never
fatal). Bounded event growth.

[skillgrid-context]
Change: 2026-09-24-mnemonic-monitoring
Phase: apply
"
```

## Global Constraints (repeated for executor reference)

- Go 1.22+ minimum to build.
- No new dependencies without an ADR. (Zero new deps.)
- Conventional commits only; no AI-attribution trailers.
- Spec-zone changes commit before code-zone changes.
- Observe-mode only: capture never blocks, rewrites, or fails a tool call.
- `<private>` content + always-private tool output never persisted.
- Sensitive events excluded from query/export by default.
- 90-day retention (configurable).

## Plan Review

- Verdict: READY FOR EXECUTION
- Findings: 0 Critical, 2 Important (fixed: Task 2 requires extracting the inline `tool.execute.after` body into a testable `MnemonicHookAfter` export — noted in Step 3; Task 4's one-way-door default-flip requires `hooksEnabledExplicitlySet` to distinguish "not set" from "explicitly false" — noted in Step 3), 1 Minor (deferred: the Kilo plugin gets the same Task 2 change in the same commit — noted in the task)
- Reviewed: 2026-09-24
