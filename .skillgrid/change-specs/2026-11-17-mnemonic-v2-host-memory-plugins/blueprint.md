# Mnemonic-v2 5-Host Memory Plugins — Implementation Blueprint (steps, files, verification)

**Status:** PROPOSED

**Tier:** T2

**Classification:** standard (L2: full suite + build) — a normal feature across TS plugins, one Go verb, and the installer; no new trust boundary, no engine migration (the v1 data import is a separate manual script, outside the verification floor).

**Build shape:** Tracer thread — a thin end-to-end path (installer → MCP entry → host capture → prompt store) is made real on one host before fanning out to the other four. Why: the integration unknowns (do pi/omp spawn `mcp.json` servers at session cwd? does the JSONC merge preserve each dialect? do the hooks fire with the expected payloads?) are only answerable with something real and connected.

**Goal:** Give pi, omp, opencode, kilo, and cursor the mnemonic-v2 memory surface (22 agent-profile tools + prompt capture + session registration + compaction recovery) through one shared installer, a clean v1 cutover, and exactly one new Go verb — per ADR-0067.

**Architecture:** All five hosts dispatch memory tools through a host-owned `mnemonic-memory` MCP entry (`mnemonic memory mcp --profile agent`, absolute binary path). pi/omp and opencode/kilo additionally get thin hooks-only TS extensions (one shared source per pair; node builtins only; no persistent engine connection) doing session registration (subagent guard), per-prompt capture via the new `memory save-prompt` verb, compaction recovery, protocol host-shell injection, and a session-start liveness probe. Cursor gets `hooks.json` entries + an alwaysApply rule. The adapter owns `session_id` on every host (inject / rewrite / pin). The installer extends skillgrid-cli with (host, family)-keyed tracked-set manifests, no-clobber JSON(C) merge, and the v1 cutover rewrites.

**Tech Stack:** Go 1.22+ (the `save-prompt` verb in `mnemonic/cmd/mnemonic`; the installer in `skillgrid-cli/internal/install/`; stdlib + existing dependencies only); TypeScript ESM (pi-ExtensionAPI for pi/omp, `@opencode-ai/plugin` types for opencode/kilo; node builtins only — no new npm dependencies); bash + sqlite3 CLI (the one-shot v1 import script); JSON/JSONC (host config files, three dialects).

**Spec:** `.skillgrid/change-specs/2026-11-17-mnemonic-v2-host-memory-plugins/briefing.md`

**In-force constraints:** ADR-0067 (this change's architecture — the blueprint must follow it); ADR-0061 (MCP family split — the entry shape is `mnemonic <family> mcp`, key `mnemonic-memory`); AGENTS.md § Rules (Go 1.22+ floor; no new dependencies without an ADR; serial development; spec-zone commits before code-zone).

**Rejected (from the briefing, for the record):**
- Engram literal port for pi/omp (spawned child + `registerTool` + hand-rolled wire client + generated contract projection) — the projection/self-heal machinery targets an independently-released HTTP server this local-binary design lacks; omp already runs a host-spawned MCP child.
- Second persistent MCP child for opencode/kilo capture — ~30–80 MB RSS per session for calls a one-shot CLI verb serves fail-open.
- `mnemonic memory migrate` Go verb for v1 data — rejected to hold the zero-Go lock; one-shot script instead.

## Hypothesis

**Claim:** all five hosts get a working v2 memory surface from one shared installer, two thin TS sources, and the cursor hooks — with exactly one new Go verb.
**Right condition:** a smoke session on every host shows the `mem_*` tools callable, N prompts → N prompt rows, exactly one session row (subagents add zero rows), and the compaction notice fires; `skillgrid uninstall --family memory` returns a fixture HOME byte-identical.
**Wrong condition:** any host where tools are absent/broken (e.g. pi/omp does not spawn `mcp.json` servers at session cwd), capture creates phantom or duplicate sessions, or the installer touches a foreign entry — any one falsifies the unified transport for that host.
**Thinnest MVP:** one host end-to-end — cursor (hooks + rule + MCP entry via the installer) with the `save-prompt` verb storing prompts.
**Door check:** Task 2 — a live omp session (the pi/omp representative) with the installed MCP entry: tools present, `mem_save` → `mem_search` round-trip at session cwd. If it fails: the affected host falls back to a per-session spawned child (ADR-0067 revisit criterion 1), recorded as a deviation in the task.

## Terms

Pointers: `.skillgrid/projectwiki/glossary/01-business-terms.md` / `02-technical-terms.md` (canonical; rows added by this change).

- **Host Shell** — the per-host protocol delta (tool-name mapping, `session_id` pin, repair command); the protocol body is authored only in the Go MCP initialize instructions. Sharpened during planning: the body/shell boundary is enforced by diff tests, not convention.
- **Adapter-Owns Session Identity** — the adapter is the single authority on `session_id`; the model never invents it.
- **Host-Registry Installer** — the skillgrid-cli installer keyed (host, family) with tracked-set-only manifests.
- Relies on existing rows: **Family Server**, **Tool Profile** (ADR-0061) — the entry shape and `--profile agent` vocabulary.

## Must-Haves (goal-backward verification)

Derived from the spec's requirements R1–R12 (`briefing.md`). The verifier checks the result against this list, not just that tasks exist.

**Truths** (observable behaviors that must hold):

- **T-verb:** `mnemonic memory save-prompt` reads a plain-text prompt from stdin, stores it in the cwd-resolved project's prompt store attached to the pinned session, prints nothing, exits 0. Re-firing with identical content + session within the turn window stores no second row.
- **T-verb-cursor:** the same verb accepts the cursor hook JSON payload on stdin (fixture-pinned shape) and extracts the prompt field.
- **T-omp-live** `backstop`: in a live omp session with the installed extension + MCP entry, the agent's N prompts produce exactly N prompt rows, the session table holds exactly 1 row for the conversation, and a subagent (task) session adds 0 session rows and 0 prompt rows. Held-out test: the Task 2 door-check smoke (steps recorded there).
- **T-omp-cwd:** in that session, `mem_save` → `mem_search` round-trips against the store of the session's cwd (project resolution correct).
- **T-opencode-live** `backstop`: same truths on live opencode and kilo; additionally, a `mem_save` call whose args carry a foreign or absent `session_id` is stored under the adapter's session (the `tool.execute.before` rewrite fired).
- **T-compaction** `backstop`: on compaction of a pi/omp/opencode/kilo session, the agent's context receives the FIRST ACTION REQUIRED memory re-injection notice.
- **T-cursor-live** `backstop`: cursor sessionStart → one `cursor` session row; beforeSubmitPrompt → prompt rows; the installed rule contains `session_id: "cursor"` and zero occurrences of `web_cache_`, `fact_`, or `serve`.
- **T-fail-open:** a failing or timing-out `save-prompt` verb never surfaces an error to the host session; the per-project drop counter increments; `skillgrid status --family memory` reports "N captures dropped (store busy)".
- **T-probe:** with the binary unresolvable (PATH case), a pi/omp session start injects a warning naming the cause and the exact repair command, writes the per-project scratch record, and the session continues; `status` surfaces the record.
- **T-installer-roundtrip:** `skillgrid install --family memory all` on each of the 5 fixture HOMEs creates exactly the tracked file set + entries; a re-run is byte-identical; foreign MCP entries and files are byte-untouched; `skillgrid uninstall --family memory all` returns each HOME byte-identical to pre-install.
- **T-cutover:** on a fixture HOME carrying the three stale v1 entry shapes (omp string-command, opencode/kilo JSONC arrays, cursor `mnemonic` key), install rewrites exactly the mnemonic entry to `[<abs-bin>,"memory","mcp","--profile","agent"]` with each host's dialect preserved (comments, sibling entries, `enabled`/`type` fields); the post-cutover installed set contains no functional reference to `:7438`, `skillgrid mcp`, `web_cache_`, or `fact_`.
- **T-protocol:** the rendered Go MCP `instructions` (agent profile) is ≤ 2,048 runes (Go guard test); the cursor rule file is diff-equal to the Go source (generated in the same task); a smoke session shows the protocol body exactly once.
- **T-import:** the import script moves a fixture v1 store's observations + sessions into the v2 store; `memory search` finds the v1 content after; a re-run adds 0 rows; `--dry-run` writes nothing.
- **T-engram:** install on a kilo fixture HOME with an engram file present prints the coexistence warning; `status` lists it.

**Artifacts** (files that must exist with real implementation, not stubs):

- `mnemonic/cmd/mnemonic/` — the `save-prompt` subcommand wired into `runMem` + its handler on the existing prompt-store path (shared with the `mem_save_prompt` MCP dispatch).
- `plugins/memory/pi-omp/` — the shared pi/omp extension: thin `.ts` host glue + plain-ESM `.js` logic modules (engram pattern: logic in `.js` so `node --test` runs it with zero dependencies; `.ts` glue loaded directly by pi/omp) + `package.json` + tests.
- `plugins/memory/opencode-kilo/` — the shared opencode/kilo plugin (same `.js`-logic + `.ts`-glue pattern) + `package.json` + tests; the kilo install is a byte-identical copy of this source (installer-enforced, identity asserted in a test).
- `plugins/memory/cursor/` — the `hooks.json` template + the generated `mnemonic.mdc` rule + the generation/diff test.
- `skillgrid-cli/internal/install/` — the memory family: `install`/`uninstall`/`status` wiring + the exported (host, family)-keyed manifest + no-clobber JSON(C) merge boundary + 5 fixture-HOME tests.
- `scripts/mnemonic-v1-import.sh` — the one-shot v1→v2 store import (sqlite3 CLI) + its fixture test.
- Cutover deletions/rewrites: `plugins/pi/mnemonic/{api.js, mnemonic-facts.ts, mnemonic-codeindex.ts, ctx-sandbox.ts, skillgrid-sdd-events.ts, skillgrid-events.ts, skillgrid-compaction.ts, mnemonic-memory.ts}` deleted, `index.ts` replaced by the new extension install path; `plugins/opencode/{mnemonic-memory.ts, skillgrid-events.ts, skillgrid-compaction.ts, mnemonic-codeindex.ts, skillgrid-squad.ts, skillgrid-context.ts, tools/, src/memory-prefetch.ts}` retired; `rules/mnemonic.mdc` rewritten; `.cursor-plugin/plugin.json` description updated; `.agents/skills/mnemonic/SKILL.md` updated to the v2 surface.
- Test artifacts: installer fixture HOMEs (5), `save-prompt` unit tests, extension/plugin unit tests, the protocol diff test + rune guard test, the import fixture store.

**Key links** (critical connections that must work together):

- The extension's capture must invoke `save-prompt` with the **same session id** it registered via `session start --id` — a mismatch orphans prompts from the session.
- The opencode/kilo rewrite must match the exact host-qualified tool names derived from the MCP entry key (`mnemonic-memory_mem_*`) — key/normalizer mismatch means the rewrite misses and phantom sessions appear.
- The installer's MCP entry must use the **absolute resolved binary path**, and the liveness probe must verify that same path — a bare `mnemonic` is the green-but-dead GUI-PATH case.
- The cursor rule's `session_id` pin must equal the hook's `--id cursor` value.
- The cursor rule is generated from the Go instructions source in the same task — the diff test is the drift detector.
- The import script targets the v2 `observations` **table** (v1's name is a VIEW) — verified by post-import search, not by schema inspection.
- The cutover matcher must recognize all three stale shapes (omp string-command; opencode/kilo arrays incl. `skillgrid` binary; cursor key) — a missed shape leaves a dead entry that `status` must then report.

**One-way-door decisions** (hard to reverse — flagged for explicit user approval before implementing):

- **D-1: the v1 cutover** — deleting the v1 plugin modules and rewriting the live host configs removes the `fact_*`/`code_*`/`ctx_*`/`squad` surfaces from all five hosts. Repo files revert via git; live configs via installer backups — but every host session's loaded plugin set changes. User-approved at the brainstorming gate (Q3/Q5 locks); the cutover task carries the `⚠ one-way:` tag and stops for a final confirm before the live rewrites run.
- **D-2: manifest tracked-set semantics** — `uninstall` deletes exactly the tracked set; a manifest that adopted a foreign file would delete another change's artifact on uninstall. Mitigated by tracked-set-only + backup; gate-approved (R7). The fixture tests are the held-out proof.

## Global Constraints

Every task's requirements implicitly include this section (values verbatim from the spec / AGENTS.md):

- **Go 1.22+ minimum** (ADR-0031; toolchain go1.26.6). **No new third-party dependencies without an ADR** (AGENTS.md § Rules) — this change adds zero: Go stdlib only; npm node builtins only (`node --test`, `node:child_process`, `node:fs`); import script = bash + sqlite3 CLI.
- **Zero Go engine changes except the `memory save-prompt` verb** (+ the ≤2,048-rune guard test on the MCP instructions; a text trim only if that test fails — recorded in the briefing's shootdown table).
- **MCP entry, all five hosts:** key `mnemonic-memory`, command `[<abs-bin>, "memory", "mcp", "--profile", "agent"]`; the omp dialect keeps its string-`command` + `args` shape. **Absolute resolved binary path in every installed config entry** (MCP entries, hook commands, nothing bare).
- **Protocol body is authored only in the Go MCP initialize instructions** (single source, ADR-0067); host surfaces carry the host shell only; rendered agent-profile instructions ≤ 2,048 runes.
- **Fail-open everywhere in the capture/probe path** — a memory failure never blocks or crashes the host session; drops are counted, not thrown.
- **Capture/probe budgets:** hard timeout 5 s on the `save-prompt` spawn; 10 s total for the session-start probe (transient `initialize` + one search); `save-prompt` stdin capped at 256 KiB (excess discarded, fail-open). Cursor hook commands are additionally bounded by the cursor harness's own limit (noted, not pinned).
- **Manifest semantics (tracked-set-only):** the manifest tracks only files/entries this change wrote; foreign content is never adopted, never clobbered, never deleted; uninstall removes exactly the tracked set; every write is atomic (tmp + rename) with a pre-write backup; re-runs are idempotent no-ops on matching state.
- **Session ID values:** pi/omp and opencode/kilo use an adapter-generated per-conversation id; cursor uses the literal `cursor` (pinned in the rule and the hooks). The protocol text on every host says "do not invent or override `session_id`".
- **Kilo install = byte-identical copy** of the opencode-kilo source (no substitution of any kind).
- **Opencode plugin entry shape:** default export `{ id, setup }` (ADR-0045 V2 form).
- **Ops:** serial development; spec-zone commits before code-zone; conventional commits, no AI-attribution trailers, `[skillgrid-context]` block naming the change.
- **Project resolution:** from the process cwd of the hook/extension/verb — no `--project` overrides in hook commands.

## File Structure

**Created:**

- `mnemonic/cmd/mnemonic/mem_prompt.go` — the `save-prompt` handler (stdin parse: plain text | cursor JSON fixture shape; prompt-store write on the same path as the `mem_save_prompt` MCP dispatch; idempotency under re-fire). Wired into `runMem`'s case list in `mem.go`.
- `mnemonic/cmd/mnemonic/save_prompt_test.go` — verb unit tests (stdin shapes, idempotency, cwd project, exit codes, fail-open timeout).
- `mnemonic/internal/transport/mcp/instructions_test.go` — the ≤2,048-rune guard test + the cursor-rule body diff test (reads the repo rule file, compares its body section to the rendered agent-profile instructions).
- `plugins/memory/pi-omp/` — the shared pi/omp extension (installed to `~/.skill-pi/agent/extensions/mnemonic/` and `~/.omp/agent/extensions/mnemonic/`):
  - `index.ts` — thin pi-ExtensionAPI glue: session lifecycle → logic modules.
  - `src/session.js` — session id generation + registration + subagent guard (pure).
  - `src/capture.js` — `save-prompt` spawn, fail-open, drop counter (pure command construction + result handling).
  - `src/compaction.js` — compaction recovery: notice text + `memory context` fetch (pure).
  - `src/shell.js` — protocol host shell assembly + the session-start probe (binary resolve → transient `initialize` spawn → `memory search --limit 1` store read) + scratch record (pure; the spawn is injected).
  - `test/{session,capture,compaction,shell}.test.mjs` — `node --test` suites over the `.js` modules.
  - `package.json` — `{ "name": "mnemonic-memory-piomp", "type": "module" }`, no dependencies.
- `plugins/memory/opencode-kilo/` — the shared opencode/kilo plugin, **single file** (opencode v2 auto-discovers bare top-level `.ts` in `plugins/`; a subdirectory package is unproven for discovery — the v1 + engram plugins are single files. The pi-omp `src/` split exists because pi/omp load extension *directories*; the two packages are installed independently, so the pure logic is duplicated, not shared):
  - `index.ts` — the `{ id, setup }` glue (ADR-0045) + the pure exports: session-event extraction + registry (the `parentID` subagent guard), inbox-prompt extraction + capture (fail-open, drop counter), the `mnemonic-memory_` tool-name normalizer + the `session_id` rewrite. Task 6 adds the compaction + shell exports to the same file.
  - `test/{rewrite,session,capture}.test.ts` — `node --test` suites (node ≥ 23.6 native TS; verified on v24.19.0).
  - `package.json` — no dependencies.
- `plugins/memory/cursor/` — the cursor surface:
  - `hooks.json` — hook template (sessionStart → `session start --id cursor`; beforeSubmitPrompt → `save-prompt`; stop → `session end --session cursor`); `<BIN>` placeholder resolved to the absolute path by the installer.
  - `mnemonic.mdc` — the alwaysApply rule (host shell: tool-name mapping `mcp__mnemonic-memory__mem_*`, `session_id: "cursor"` pin, repair command; body section mirrors the Go instructions — diff-tested).
  - `test/rule-shape.test.mjs` — rule shape checks (no `web_cache_`/`fact_`/`serve`; pin present; mapping line present).
- `skillgrid-cli/internal/install/memory.go` — memory family: `Install/Uninstall/Status(ctx, host, family)` + the per-host file/entry plans (extension dirs, plugin files, cursor rule + hooks, MCP entries, kilo AGENTS.md marker).
- `skillgrid-cli/internal/install/registry.go` — the exported (host, family) `Manifest` type (sha256 file map + entry list) + the no-clobber JSON(C) merge (dialect-preserving: string-vs-array command, comments, foreign keys) + atomic write/backup — the seam the queued obs change migrates onto.
- `skillgrid-cli/internal/install/memory_test.go` — the five fixture-HOME round-trip tests (install / re-run / clobber / uninstall / cutover rewrite per dialect).
- `scripts/mnemonic-v1-import.sh` — the one-shot v1→v2 store import (`--from`, `--to`, `--dry-run`, content-keyed idempotency, counts).
- `scripts/fixtures/v1-import/` — fixture v1 store (minimal v1 schema: `observations_core` + one satellite + `sessions`) + expected v2 rows.
- `plugins/memory/testdata/` — per-host config fixtures (real shapes: omp `mcp.json` string-command, opencode/kilo JSONC, cursor `mcp.json`, a kilo HOME with an engram file) shared by installer + extension tests.

**Modified:**

- `mnemonic/cmd/mnemonic/mem.go` — `save-prompt` added to the `runMem` case list + usage string.
- `skillgrid-cli/cmd/` (the `install` command wiring) — `--family` flag gained the `memory` value (agents: existing detection reused).
- `hooks/hooks-cursor.json` — cutover: v1 memory-domain entries (the ones referencing `:7438` memory routes / v1 memory verbs) replaced by the v2 entries from `plugins/memory/cursor/hooks.json`; all non-memory entries (ticketing, gates, precommit, checkpoint) untouched.
- `.cursor-plugin/plugin.json` — description updated off the removed `skillgrid mcp` reference.
- `.agents/skills/mnemonic/SKILL.md` — daemon / `:7438` / `skillgrid mcp` / `web_cache_*` sections rewritten to the v2 surface.

**Deleted (cutover inventory, per briefing R8):**

- `plugins/pi/mnemonic/` — the whole v1 package (`index.ts` replaced by the new install source; `api.js`, `mnemonic-facts.ts`, `mnemonic-codeindex.ts`, `mnemonic-memory.ts`, `ctx-sandbox.ts`, `skillgrid-events.ts`, `skillgrid-compaction.ts`, `skillgrid-sdd-events.ts` deleted; `package.json` moves to the new directory).
- `plugins/opencode/` — `mnemonic-memory.ts`, `skillgrid-events.ts`, `skillgrid-compaction.ts`, `mnemonic-codeindex.ts`, `skillgrid-squad.ts`, `skillgrid-context.ts`, `tools/` (v1 tool defs), `src/memory-prefetch.ts` deleted; `skillgrid-v2sse.ts` + logo files: verify at execution whether the new opencode package still consumes them — delete or keep with a one-line note (recorded in the cutover task).
- `plugins/kilo/` — the v1 install-time copy source (kilo now installs from `plugins/memory/opencode-kilo/`).

**Installed (runtime, installer-managed — not repo files):** `~/.skill-pi/agent/extensions/mnemonic/`, `~/.omp/agent/extensions/mnemonic/`, `~/.config/opencode/plugins/mnemonic-memory.ts` + `~/.config/kilo/plugins/mnemonic-memory.ts` (byte-identical), `~/.cursor/rules/mnemonic.mdc`, `~/.cursor/hooks.json` (merged entries), the four MCP config entries, `~/.skillgrid/mnemonic/memory-install.json` (manifest), `~/.skillgrid/mnemonic/<project>/.last-session-start.json` + drop-counter files (scratch state, read by `status`).

## Tasks

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

Tracer thread: Task 1 (verb) → Task 2 (pi/omp real end-to-end + the door check) → Tasks 3–7 (installer, fan-out, protocol) → Tasks 8–9 (cutover, import) → Task 10 (docs) → Task 11 (live smoke). Each task is an independently testable deliverable.

### Task 1: `memory save-prompt` verb (the host-capture stdin verb)

**Files:**
- Create: `mnemonic/cmd/mnemonic/mem_prompt.go`
- Create: `mnemonic/cmd/mnemonic/save_prompt_test.go`
- Create: `mnemonic/internal/memory/save_prompt_once_test.go`
- Modify: `mnemonic/cmd/mnemonic/mem.go` — `save-prompt` case in the `runMem` switch (after the `case "save":` block, ~line 80) + `save-prompt` added to the subcommand list in the `errExit` at line 77
- Modify: `mnemonic/cmd/mnemonic/main.go:311` — `save-prompt` added to the `memory` line of `printUsage`
- Modify: `mnemonic/cmd/mnemonic/run_test_helpers_test.go` — `installRunEnv` refactored over a new `installRunEnvDir(t) (dataDir string)` (tests that assert against per-project store files need the path; `installRunEnv` keeps its signature)
- Modify: `mnemonic/internal/memory/prompts.go` — `SavePromptOnce` + `promptDedupSeam` + `promptDedupWindow` (the verb's dedup probe; reuses the existing `SavePrompt` redact+insert path)
- Modify: `mnemonic/internal/transport/handler/handler.go` — `SavePromptOnce` wrapper next to `SavePrompt` (~line 354)
- Modify: `mnemonic/internal/core/store/sqlite.go` + `postgres.go` — `HasRecentPrompt` (the verb's dialect-rendered probe; the only store-surface addition this change makes)

**Interfaces:**
- Consumes: `openRuntime(cwd string, fl config.Flags, explicit string) (*handler.Handler, store.Store, error)` (main.go:101); the existing `(*Service).SavePrompt` (prompts.go:27 — redact + `promptSeam.InsertPrompt`, which auto-registers a missing session row via `ensureSessionTx`, sqlite.go:1831); `(*store.Store).Query` generic seam pattern (the new probe follows the store-native rendering of the `Exec`/`Query` seam); `installRunEnv`/`captureRun`/`newTestHandlerWithStore` test harness (run_test_helpers_test.go, cli_test.go); `store.Open(ctx, dataDir, project, store.SQLite, "")` keys one file per project at `dataDir/<project>.sqlite` (sqlite.go:138).
- Produces:
  - CLI: `mnemonic memory save-prompt --id <session> [--project <p>] [--source-inbox-id <s>]` — reads the prompt from stdin (256 KiB cap, excess discarded), accepts plain text or the cursor `{"prompt":"..."}` envelope (anything else is raw text, not an error), prints nothing on success, exits 0 (stored or dedup-skipped), 1 (store error; note on stderr), 2 (usage: missing `--id`). Never blocks the host beyond the store's own `busy_timeout(5000)`.
  - `(*handler.Handler).SavePromptOnce(ctx context.Context, sessionID, content, project, sourceInboxID string) (int64, error)` — resolves an empty project like `SavePrompt` does (`h.resolveProject(DefaultCWD, "")`), then `h.svc.SavePromptOnce`.
  - `(*Service).SavePromptOnce(ctx, sessionID, content, project, sourceInboxID string) (int64, error)` — redact (`redactPrompt`, the same `stripPrivate` + `ContentMaxBytes` path) → empty → (0, nil); dedup probe within `promptDedupWindow = 10 * time.Minute` (a hook re-fire in-turn is a retry, a human re-typing the same text after the window is a new prompt) → hit → (0, nil); else the existing `SavePrompt` insert. Returns the prompt's local id (0 = nothing stored).
  - `promptDedupSeam` (memory package): `HasRecentPrompt(ctx context.Context, sessionID, content string, since time.Time) (bool, error)` — implemented store-natively in `sqliteStore` (`created_at >= ?`, cutoff rendered `2006-01-02 15:04:05` UTC — the `datetime('now')` storage format) and `pgStore` (cutoff bound as `time.Time` — `TIMESTAMPTZ`); a store without the seam skips the probe (fail-open insert).
  - `stdinReader io.Reader` package var in cmd/mnemonic (default `os.Stdin`; the same test-seam pattern as `storeFor`/`handlerFor`/`gitToplevel`).
  - `readStdinCapped(r io.Reader) ([]byte, error)` in mem_prompt.go — `io.LimitReader(r, savePromptStdinCap+1)` + slice to exactly `savePromptStdinCap = 256 << 10`.

**SATISFIES:** verb-save-prompt-plain, verb-idempotent-refire, verb-cursor-json-payload, verb-stdin-cap, verb-resolves-cwd-store-like-tools, verb-auto-registers-missing-session

- [ ] **Step 1: Write the failing tests**

`mnemonic/cmd/mnemonic/save_prompt_test.go`:

```go
package main

// save_prompt_test.go — the host-capture verb contract (ADR-0067): stdin
// shapes, the dedup re-fire, the project-resolution chain (threat matrix:
// Git repository selection), exit codes, the fail-open cap.

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
)

// withStdin swaps the verb's stdin seam for one pass.
func withStdin(t *testing.T, s string) {
	t.Helper()
	prev := stdinReader
	stdinReader = strings.NewReader(s)
	t.Cleanup(func() { stdinReader = prev })
}

// promptRows counts user_prompts rows in the per-project store file.
func promptRows(t *testing.T, dataDir, project string) int {
	t.Helper()
	st, err := store.Open(context.Background(), dataDir, project, store.SQLite, "")
	if err != nil {
		t.Fatalf("open %s store: %v", project, err)
	}
	defer st.Close()
	rows, err := st.Query(context.Background(), `SELECT count(*) FROM user_prompts`)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	n, _ := rows[0][0].(int64)
	return int(n)
}

// TestSavePromptStoresPlainStdin: the contract — stdin plain text, exit 0,
// no stdout, one row under the adapter's session.
func TestSavePromptStoresPlainStdin(t *testing.T) {
	dataDir := installRunEnvDir(t)
	withStdin(t, "first prompt")
	out := runCLI(t, "memory", "save-prompt", "--id", "s1", "--project", "p")
	if out != "" {
		t.Fatalf("the verb prints nothing on success, got %q", out)
	}
	if n := promptRows(t, dataDir, "p"); n != 1 {
		t.Fatalf("want 1 user_prompts row, got %d", n)
	}
}

// TestSavePromptIdempotentRefire: identical content + session inside the
// window stores no second row; distinct content does.
func TestSavePromptIdempotentRefire(t *testing.T) {
	dataDir := installRunEnvDir(t)
	withStdin(t, "same prompt")
	runCLI(t, "memory", "save-prompt", "--id", "s1", "--project", "p")
	withStdin(t, "same prompt")
	runCLI(t, "memory", "save-prompt", "--id", "s1", "--project", "p")
	if n := promptRows(t, dataDir, "p"); n != 1 {
		t.Fatalf("re-fire must dedup, got %d rows", n)
	}
	withStdin(t, "different prompt")
	runCLI(t, "memory", "save-prompt", "--id", "s1", "--project", "p")
	if n := promptRows(t, dataDir, "p"); n != 2 {
		t.Fatalf("distinct content must store, got %d rows", n)
	}
}

// TestSavePromptCursorJSON: the cursor beforeSubmitPrompt envelope yields
// the "prompt" field; a malformed envelope is raw text (fail-open).
func TestSavePromptCursorJSON(t *testing.T) {
	dataDir := installRunEnvDir(t)
	withStdin(t, `{"prompt":"hello cursor","conversation_id":"c1"}`)
	runCLI(t, "memory", "save-prompt", "--id", "cursor", "--project", "p")
	withStdin(t, "{not json")
	runCLI(t, "memory", "save-prompt", "--id", "cursor", "--project", "p")
	if n := promptRows(t, dataDir, "p"); n != 2 {
		t.Fatalf("want 2 rows (json field + raw text), got %d", n)
	}
}

// TestSavePromptRequiresID: usage contract.
func TestSavePromptRequiresID(t *testing.T) {
	installRunEnv(t)
	withStdin(t, "prompt without id")
	err := runCLIError(t, "memory", "save-prompt", "--project", "p")
	if err == nil || !strings.Contains(err.Error(), "--id is required") {
		t.Fatalf("missing --id must be a usage error naming the flag: %v", err)
	}
}

// TestSavePromptAutoRegistersMissingSession: fail-open ordering — the
// store's auto-register means capture never fails on a session the
// adapter forgot to start (it would only create a phantom session row,
// which the adapter's guard prevents).
func TestSavePromptAutoRegistersMissingSession(t *testing.T) {
	dataDir := installRunEnvDir(t)
	withStdin(t, "prompt before session start")
	runCLI(t, "memory", "save-prompt", "--id", "s2", "--project", "p")
	st, _ := store.Open(context.Background(), dataDir, "p", store.SQLite, "")
	defer st.Close()
	rows, err := st.Query(context.Background(),
		`SELECT ownership FROM sessions WHERE id = 's2'`)
	if err != nil || len(rows) != 1 || rows[0][0] != "shared" {
		t.Fatalf("missing session must auto-register shared-ownership: rows=%v err=%v", rows, err)
	}
}

// TestSavePromptResolvesProjectChain: threat RED (Git repository
// selection) — the env-var leg of the chain (explicit > env > git >
// basename) routes the prompt to the named project's store and touches no
// other project's file.
func TestSavePromptResolvesProjectChain(t *testing.T) {
	dataDir := installRunEnvDir(t)
	t.Setenv(config.Default().Project.EnvVar, "wt")
	withStdin(t, "worktree prompt")
	runCLI(t, "memory", "save-prompt", "--id", "s1")
	if n := promptRows(t, dataDir, "wt"); n != 1 {
		t.Fatalf("env-var project must get the row, got %d", n)
	}
	if n := promptRows(t, dataDir, "p"); n != 0 {
		t.Fatalf("no other project's store may be touched, got %d", n)
	}
}

// TestReadStdinCapped: the 256 KiB cap (the subprocess threat line) —
// excess discarded, not an error.
func TestReadStdinCapped(t *testing.T) {
	big := strings.Repeat("a", savePromptStdinCap+1024)
	got, err := readStdinCapped(strings.NewReader(big))
	if err != nil {
		t.Fatalf("cap: %v", err)
	}
	if len(got) != savePromptStdinCap || string(got) != big[:savePromptStdinCap] {
		t.Fatalf("want exactly the first %d bytes, got %d", savePromptStdinCap, len(got))
	}
}
```

`mnemonic/internal/memory/save_prompt_once_test.go` — the service-level dedup window boundary (the CLI tests cover the in-window re-fire; this pins the window edge):

```go
package memory

// SavePromptOnce dedup window boundary: a stored prompt inside the
// window dedups; an identical prompt older than the window stores.

import (
	"context"
	"testing"
	"time"
)

func TestSavePromptOnceWindowBoundary(t *testing.T) {
	svc, st := newTestService(t) // the package's existing fixture (prompts_test.go)
	ctx := context.Background()
	id, err := svc.SavePromptOnce(ctx, "s1", "boundary prompt", "p", "")
	if err != nil || id == 0 {
		t.Fatalf("first store: id=%d err=%v", id, err)
	}
	if id, _ := svc.SavePromptOnce(ctx, "s1", "boundary prompt", "p", ""); id != 0 {
		t.Fatalf("in-window re-fire must dedup, got id %d", id)
	}
	// age the row past the window (the test-only raw seam).
	rawExec(t, st, `UPDATE user_prompts SET created_at = datetime('now', ?)`,
		"-11 minutes")
	if id, _ := svc.SavePromptOnce(ctx, "s1", "boundary prompt", "p", ""); id == 0 {
		t.Fatalf("an identical prompt past the window is a new capture")
	}
	_ = time.Now
}
```

(`rawExec` = the package's existing raw-SQL test helper; if `prompts_test.go` names it differently, use that name — the fixture is the package's standard `newTestService`.)

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd mnemonic && go test ./cmd/mnemonic/ -run 'TestSavePrompt|TestReadStdinCapped' -count=1 && go test ./internal/memory/ -run TestSavePromptOnceWindowBoundary -count=1`
Expected: FAIL — `undefined: stdinReader`, `undefined: readStdinCapped`, `undefined: installRunEnvDir`, `undefined: (*Service).SavePromptOnce`, etc. (compile errors are the RED)

- [ ] **Step 3: Write the implementation**

`mnemonic/internal/memory/prompts.go` (append):

```go
// promptDedupWindow bounds the SavePromptOnce dedup: a host hook re-fire
// within a turn is a retry (the harness may fire beforeSubmitPrompt more
// than once for the same prompt); a human re-typing the same text after
// the window is a legitimate new capture.
const promptDedupWindow = 10 * time.Minute

// promptDedupSeam is the store's dialect-rendered prompt-dedup probe
// (the verb's only store-surface addition; the SQLite leg renders the
// datetime('now') text format, the Postgres leg binds time.Time to
// TIMESTAMPTZ).
type promptDedupSeam interface {
	HasRecentPrompt(ctx context.Context, sessionID, content string, since time.Time) (bool, error)
}

// SavePromptOnce persists the turn's prompt exactly once per (session,
// content) within promptDedupWindow. Empty (post-redaction) content
// stores nothing. A store without the seam skips the probe (fail-open).
func (s *Service) SavePromptOnce(ctx context.Context, sessionID, content, project, sourceInboxID string) (int64, error) {
	if s == nil || s.st == nil {
		return 0, errNoStore
	}
	prompt := s.redactPrompt(content)
	if prompt == "" {
		return 0, nil
	}
	if se, ok := s.st.(promptDedupSeam); ok {
		since := time.Now().UTC().Add(-promptDedupWindow)
		dup, err := se.HasRecentPrompt(ctx, sessionID, prompt, since)
		if err != nil {
			return 0, err
		}
		if dup {
			return 0, nil
		}
	}
	return s.SavePrompt(ctx, sessionID, content, project, sourceInboxID)
}
```

`mnemonic/internal/core/store/sqlite.go` (next to `InsertPrompt`, ~line 1822):

```go
// HasRecentPrompt reports whether user_prompts holds an identical
// (session, content) row at/after since (the save-prompt verb's dedup
// probe). created_at stores datetime('now') — "YYYY-MM-DD HH:MM:SS" UTC.
func (s *sqliteStore) HasRecentPrompt(ctx context.Context, sessionID, content string, since time.Time) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM user_prompts
		WHERE session_id = ? AND content = ? AND created_at >= ? LIMIT 1`,
		sessionID, content, since.UTC().Format("2006-01-02 15:04:05")).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
```

`mnemonic/internal/core/store/postgres.go` (next to `InsertPrompt`, ~line 1282):

```go
// HasRecentPrompt — the Postgres leg of the save-prompt verb's dedup
// probe (created_at is TIMESTAMPTZ; the cutoff binds as time.Time).
func (s *pgStore) HasRecentPrompt(ctx context.Context, sessionID, content string, since time.Time) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM user_prompts
		WHERE session_id = $1 AND content = $2 AND created_at >= $3 LIMIT 1`,
		sessionID, content, since.UTC()).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
```

`mnemonic/internal/transport/handler/handler.go` (next to `SavePrompt`, ~line 354):

```go
// SavePromptOnce persists a user prompt exactly once per (session,
// content) within the dedup window — the host-capture verb's surface
// (the hook's re-fire is a retry, not a new prompt).
func (h *Handler) SavePromptOnce(ctx context.Context, sessionID, content, project, sourceInboxID string) (int64, error) {
	if strings.TrimSpace(project) == "" {
		p, err := h.resolveProject(DefaultCWD, "")
		if err != nil {
			return 0, err
		}
		project = p
	}
	return h.svc.SavePromptOnce(ctx, sessionID, content, project, sourceInboxID)
}
```

`mnemonic/cmd/mnemonic/mem_prompt.go` (new):

```go
package main

// mem_prompt.go — the host-capture verb (ADR-0067): the pi/omp extension
// and the opencode/kilo plugin spawn it on every prompt with the
// adapter's session id; cursor's beforeSubmitPrompt hook pipes its JSON
// envelope. Fail-open contract: prints nothing on success, never blocks
// the host beyond the store's busy_timeout, a non-zero exit is a drop
// the extension counts — host payload data never appears in the command
// line (it travels on stdin).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
)

// stdinReader is the verb's stdin seam (the tests swap it; production is
// os.Stdin — the same seam pattern as storeFor/handlerFor/gitToplevel).
var stdinReader io.Reader = os.Stdin

// savePromptStdinCap bounds the captured prompt; the excess is discarded
// (fail-open — the content ceiling inside redactPrompt bounds the row).
const savePromptStdinCap = 256 << 10

// readStdinCapped reads at most savePromptStdinCap bytes from r (reads
// one byte over to detect excess, then discards it).
func readStdinCapped(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, savePromptStdinCap+1))
	if len(b) > savePromptStdinCap {
		b = b[:savePromptStdinCap]
	}
	return b, err
}

// memSavePrompt runs `memory save-prompt`.
func memSavePrompt(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("memory save-prompt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var pf projectFlag
	pf.add(fs)
	var session, sourceInboxID string
	fs.StringVar(&session, "id", "", "the adapter's session id (required)")
	fs.StringVar(&sourceInboxID, "source-inbox-id", "", "optional source inbox id")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if session == "" {
		return errExit("memory save-prompt: --id is required")
	}
	raw, err := readStdinCapped(stdinReader)
	if err != nil {
		fmt.Fprintf(errW(), "mnemonic save-prompt: stdin: %v\n", err)
		return exitErr
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(errW(), "mnemonic save-prompt: %v\n", err)
		return exitErr
	}
	defer st.Close()
	if _, err := h.SavePromptOnce(context.Background(), session, extractPrompt(raw), pf.project, sourceInboxID); err != nil {
		fmt.Fprintf(errW(), "mnemonic save-prompt: %v\n", err)
		return exitErr
	}
	return exitOK // prints nothing
}

// extractPrompt pulls the prompt text out of the stdin payload: a JSON
// object with a non-empty string "prompt" field (cursor's
// beforeSubmitPrompt shape) yields that field; anything else is the raw
// text, trimmed. A malformed envelope is text, not an error (fail-open).
func extractPrompt(raw []byte) string {
	t := strings.TrimSpace(string(raw))
	if strings.HasPrefix(t, "{") {
		var env struct {
			Prompt string `json:"prompt"`
		}
		if json.Unmarshal([]byte(t), &env) == nil && env.Prompt != "" {
			return env.Prompt
		}
	}
	return t
}
```

Wiring in `mnemonic/cmd/mnemonic/mem.go` — in the `runMem` switch after `case "save":` (line 80):

```go
	case "save-prompt":
		return memSavePrompt(fl, cwd, args[1:])
```

and `save-prompt` added to both usage lists: the `errExit` subcommand list at mem.go:77 and the `memory` line in `printUsage` (main.go:311).

`mnemonic/cmd/mnemonic/run_test_helpers_test.go` — expose the data dir (installRunEnv's body moves into installRunEnvDir, which returns it):

```go
// installRunEnvDir is installRunEnv + the fixture data dir (tests that
// assert against the per-project store files need the path —
// dataDir/<project>.sqlite).
func installRunEnvDir(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	// …(the existing installRunEnv body verbatim, using dataDir)
	return dataDir
}

func installRunEnv(t *testing.T) {
	installRunEnvDir(t)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd mnemonic && go test ./cmd/mnemonic/ ./internal/memory/ ./internal/transport/handler/ ./internal/core/store/ -count=1`
Expected: PASS (the verb tests + the full existing suites — the store/handler/service changes are additive; a regression here is the RED signal the seam touched a shared path)

- [ ] **Step 5: Commit**

```bash
git add mnemonic/cmd/mnemonic/mem_prompt.go mnemonic/cmd/mnemonic/save_prompt_test.go \
        mnemonic/cmd/mnemonic/mem.go mnemonic/cmd/mnemonic/main.go mnemonic/cmd/mnemonic/run_test_helpers_test.go \
        mnemonic/internal/memory/prompts.go mnemonic/internal/memory/save_prompt_once_test.go \
        mnemonic/internal/transport/handler/handler.go \
        mnemonic/internal/core/store/sqlite.go mnemonic/internal/core/store/postgres.go
git commit -m "feat: add the memory save-prompt host-capture verb

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 1"
```

### Task 2: pi-omp extension v0 + live OMP door check

> The tracer thread: make the pi/omp path real end-to-end (extension → verb → store) with a manual install, then run the hypothesis door check on a live omp session. Fallback on door failure: the affected host falls back to a per-session spawned MCP child (ADR-0067 revisit criterion 1), recorded as a deviation in this task's evidence.

**Files:**
- Create: `plugins/memory/pi-omp/index.ts` — thin pi-ExtensionAPI glue (v0: session + capture only; protocol injection, probe, compaction land in Task 6)
- Create: `plugins/memory/pi-omp/src/session.js` — adapter-owned session identity + the project-resolution mirror (pure; spawn/git injected)
- Create: `plugins/memory/pi-omp/src/capture.js` — `save-prompt` spawn + drop counter (pure; spawn injected)
- Create: `plugins/memory/pi-omp/test/session.test.mjs`, `plugins/memory/pi-omp/test/capture.test.mjs`
- Create: `plugins/memory/pi-omp/package.json` — manifest (`pi` + `omp` extension keys, the obs-change shape)
- (runtime, manual for the door check) `~/.omp/agent/extensions/mnemonic/{index.ts,src/,package.json}` + the `mnemonic-memory` entry in `~/.omp/agent/mcp.json`

**Interfaces:**
- Consumes: the pi-ExtensionAPI surface proven by the v1 extension (`plugins/pi/mnemonic/index.ts` + `skillgrid-events.ts`): `pi.on("session_start" | "before_agent_start" | "session_shutdown", async (event, ctx) => …)`; per-user-run capture = `event.messageType === "user"` + `event.prompt` (after input expansion); quit-only end = `event.reason === "quit"`; `ctx.cwd`. **Subagent identity** (oh-my-pi Extension System, verified 2026-11-17): `ctx.agent.kind` is `"main" | "sub"` (`ExtensionAgentIdentity`: kind, id, name, depth, parentId) — guard on `kind`, NOT `depth` (`/tan` clones are subagents at depth 0); skill-pi (no identity field) must read as main. The same factory is rebound per subagent session, so all per-conversation state lives in the factory closure, never at module level. Task 1's verb (`memory save-prompt --id`); existing `memory session start --id` / `memory session end --session` verbs.
- Produces:
  - `createSession({ bin, spawn }) → { sessionId(), start(cwd), end(cwd) }` (session.js): mints `mem-<ts36>-<hex6>` per factory closure; `start` spawns `[bin, "memory", "session", "start", "--id", sid]` at `cwd` (fail-open); `end` spawns `[bin, "memory", "session", "end", "--session", sid]` (fail-open, clears the id).
  - `createCapture({ bin, session, spawn, dataDir, projectOf }) → { capture(cwd, prompt) }` (capture.js): no-op on absent session id or blank prompt; else spawns `[bin, "memory", "save-prompt", "--id", sid]` with the prompt on **stdin**; a rejecting spawn (non-zero exit, ENOENT, 5 s timeout — the production spawn resolves only on exit 0) increments the per-project drop file `<dataDir()>/<project>/ .capture-drops.json` (`{n, lastAt}`), best-effort, never throws.
  - `resolveProject(cwd, { env, gitToplevel })` (session.js) — the extension-side **mirror** of the CLI chain, env leg first (hooks never pass `--project`): `MNEMONIC_PROJECT` > `git rev-parse --show-toplevel` basename > `basename(cwd)`. Used for **scratch placement only** (drop file); store resolution stays in the verb (the Go chain — single source, threat-matrix row). `dataDir()` = `~/.skillgrid/mnemonic` (the Go default; a config-file `data_dir` override moves the store, the scratch follows the default — documented edge).
  - Production spawn (index.ts glue, not unit-tested): `spawnFile(bin, args, { cwd, stdio: ["pipe","ignore","ignore"] })`, 5 s hard timeout (`SIGKILL`), no shell, resolves only on exit 0 — command strings are the absolute binary + fixed verb words only (subprocess threat line).
  - `isSubagent(ctx)`: `ctx?.agent?.kind` present and `!== "main"` → subagent (skip); absent → main (fail-open for skill-pi).
- SATISFIES: live-omp-door-check, fail-open-drop-counter (capture side; the status read side is Task 3)

- [ ] **Step 1: Write the failing tests**

`plugins/memory/pi-omp/test/session.test.mjs`:

```js
import test from "node:test"
import assert from "node:assert/strict"
import { createSession, resolveProject } from "../src/session.js"

const calls = []
const okSpawn = async (args, stdin, cwd) => { calls.push({ args, stdin, cwd }) }

test("session: registers a minted id and ends the same id", async () => {
  const s = createSession({ bin: "mnemonic", spawn: okSpawn })
  const sid = await s.start("/repo")
  assert.match(sid, /^mem-[0-9a-z]+-[0-9a-f]+$/)
  assert.deepEqual(calls[0].args, ["mnemonic", "memory", "session", "start", "--id", sid])
  assert.equal(calls[0].cwd, "/repo")
  await s.end("/repo")
  assert.deepEqual(calls[1].args, ["mnemonic", "memory", "session", "end", "--session", sid])
})

test("session: spawn failure is fail-open (no throw, id retained for capture)", async () => {
  const s = createSession({ bin: "mnemonic", spawn: async () => { throw new Error("ENOENT") } })
  assert.ok(await s.start("/repo"))
  await s.end("/repo") // no throw
})

test("session: double start keeps one id (factory closure is per-conversation)", async () => {
  const s = createSession({ bin: "mnemonic", spawn: okSpawn })
  assert.equal(await s.start("/repo"), await s.start("/repo"))
  assert.equal(calls.length, 1)
})

test("resolveProject: MNEMONIC_PROJECT > git toplevel basename > cwd basename", () => {
  const git = (cwd) => (cwd === "/repo/wt" ? "/repo/wt" : null)
  assert.equal(resolveProject("/repo/wt", { env: {}, gitToplevel: git }), "wt")
  assert.equal(resolveProject("/repo/wt", { env: { MNEMONIC_PROJECT: "envp" }, gitToplevel: git }), "envp")
  assert.equal(resolveProject("/a/b/c", { env: {}, gitToplevel: () => null }), "c")
})
```

`plugins/memory/pi-omp/test/capture.test.mjs`:

```js
import test from "node:test"
import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import { mkdtemp } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { createCapture } from "../src/capture.js"

test("capture: blank prompt and missing session are no-ops (no spawn)", async () => {
  const calls = []
  const c = createCapture({ bin: "mnemonic", session: { sessionId: () => "s1" }, spawn: async (...a) => calls.push(a) })
  await c.capture("/repo", "   ")
  await c.capture("/repo", undefined)
  const c2 = createCapture({ bin: "mnemonic", session: { sessionId: () => null }, spawn: async () => { throw new Error("must not spawn") } })
  await c2.capture("/repo", "hello")
  assert.equal(calls.length, 0)
})

test("capture: prompt travels on stdin with the adapter's session id", async () => {
  const calls = []
  const c = createCapture({ bin: "mnemonic", session: { sessionId: () => "s1" }, spawn: async (...a) => calls.push(a), dataDir: () => tmpdir(), projectOf: () => "p" })
  await c.capture("/repo", "hello world")
  assert.deepEqual(calls[0][0], ["mnemonic", "memory", "save-prompt", "--id", "s1"])
  assert.equal(calls[0][1], "hello world")
  assert.equal(calls[0][2], "/repo")
})

test("capture: failing spawn increments the drop counter, never throws", async () => {
  const dataDir = await mkdtemp(join(tmpdir(), "mnemonic-drops-"))
  const c = createCapture({ bin: "mnemonic", session: { sessionId: () => "s1" }, spawn: async () => { throw new Error("exit 1") }, dataDir: () => dataDir, projectOf: () => "p" })
  await c.capture("/repo", "hello")
  await c.capture("/repo", "again")
  const rec = JSON.parse(readFileSync(join(dataDir, "p", ".capture-drops.json"), "utf8"))
  assert.equal(rec.n, 2)
  assert.ok(rec.lastAt)
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd plugins/memory/pi-omp && node --test test/`
Expected: FAIL — `Cannot find module '../src/session.js'` / `../src/capture.js`

- [ ] **Step 3: Write the implementation**

`plugins/memory/pi-omp/src/session.js`:

```js
// session.js — adapter-owned session identity (ADR-0067): the adapter
// mints the session id and is the single authority on it (the model
// never invents it). Pure logic: spawn is injected. Fail-open: a dead
// verb never blocks the host session.
import { randomBytes } from "node:crypto"
import { basename } from "node:path"

const mint = () => "mem-" + Date.now().toString(36) + "-" + randomBytes(3).toString("hex")

// Mirror of the CLI project chain (explicit > env > git toplevel >
// basename), env leg first — hooks never pass --project. Scratch-file
// placement ONLY: store resolution stays in the verb (the Go chain).
export function resolveProject(cwd, { env = process.env, gitToplevel } = {}) {
  if (env.MNEMONIC_PROJECT) return env.MNEMONIC_PROJECT
  if (typeof gitToplevel === "function") {
    const top = gitToplevel(cwd || process.cwd())
    if (top) return basename(top)
  }
  return basename(cwd || process.cwd())
}

export function createSession({ bin, spawn }) {
  let sid = null
  return {
    sessionId: () => sid,
    async start(cwd) {
      if (sid) return sid
      sid = mint()
      try { await spawn([bin, "memory", "session", "start", "--id", sid], "", cwd) }
      catch { /* fail-open: the drop counter surfaces a dead verb */ }
      return sid
    },
    async end(cwd) {
      if (!sid) return
      const id = sid
      sid = null
      try { await spawn([bin, "memory", "session", "end", "--session", id], "", cwd) }
      catch { /* fail-open */ }
    },
  }
}
```

`plugins/memory/pi-omp/src/capture.js`:

```js
// capture.js — per-prompt capture through the `save-prompt` verb
// (ADR-0067). Pure logic: spawn is injected (production spawn = index.ts,
// 5 s hard timeout, resolves only on exit 0). Fail-open: a dead or slow
// verb never blocks the host prompt; drops are counted, not thrown.
import { mkdir, readFile, writeFile } from "node:fs/promises"
import { homedir } from "node:os"
import { dirname, join } from "node:path"

const DROP_FILE = ".capture-drops.json"

// The extension-side data dir is the Go default (~/.skillgrid/mnemonic);
// a config-file data_dir override moves the store, the scratch follows
// the default (documented edge, Global Constraints).
export const defaultDataDir = () => join(homedir(), ".skillgrid", "mnemonic")

async function bumpDrops(dataDir, project) {
  const file = join(dataDir, project, DROP_FILE)
  let n = 1
  try { n = (JSON.parse(await readFile(file, "utf8")).n ?? 0) + 1 } catch { /* first drop */ }
  await mkdir(dirname(file), { recursive: true })
  await writeFile(file, JSON.stringify({ n, lastAt: new Date().toISOString() }))
}

export function createCapture({ bin, session, spawn, dataDir = defaultDataDir, projectOf }) {
  return {
    async capture(cwd, prompt) {
      const sid = session.sessionId()
      const text = String(prompt ?? "").trim()
      if (!sid || !text) return
      try {
        await spawn([bin, "memory", "save-prompt", "--id", sid], text, cwd)
      } catch {
        try { await bumpDrops(dataDir(), projectOf(cwd)) } catch { /* scratch is best-effort */ }
      }
    },
  }
}
```

`plugins/memory/pi-omp/index.ts`:

```ts
// @ts-nocheck
// mnemonic-memory — pi/omp extension (ADR-0067). v0 (door-check scope):
// session registration + per-prompt capture through the mnemonic binary.
// No tool registration — the mem_* tools come from the mnemonic-memory
// MCP entry. Protocol injection, the session-start probe, and compaction
// recovery land in Task 6. Every path fail-open.
// <BIN> is replaced with the absolute binary path by the installer (Task 3).
import { spawnFile } from "node:child_process"
import { spawnSync } from "node:child_process"
import { createSession, resolveProject } from "./src/session.js"
import { createCapture } from "./src/capture.js"

const BIN = "<BIN>"

// Production spawn: absolute binary + fixed verb words only (no shell),
// 5 s hard timeout, the payload on stdin; resolves only on exit 0.
function makeSpawn() {
  return (args, stdin, cwd) =>
    new Promise((resolve, reject) => {
      const child = spawnFile(args[0], args.slice(1), {
        cwd: cwd || process.cwd(),
        stdio: ["pipe", "ignore", "ignore"],
      })
      const timer = setTimeout(() => {
        child.kill("SIGKILL")
        reject(new Error("mnemonic spawn timeout (5s)"))
      }, 5000)
      child.on("error", (e) => { clearTimeout(timer); reject(e) })
      child.on("close", (code) => {
        clearTimeout(timer)
        code === 0 ? resolve() : reject(new Error("mnemonic exit " + code))
      })
      child.stdin.end(stdin ?? "")
    })
}

// Subagent guard (oh-my-pi ExtensionAgentIdentity): guard on kind, not
// depth. skill-pi has no identity field — absent reads as main.
function isSubagent(ctx) {
  const kind = ctx?.agent?.kind
  return kind !== undefined && kind !== "main"
}

export default function mnemonicMemoryPiomp(pi) {
  const spawn = makeSpawn()
  const session = createSession({ bin: BIN, spawn })
  const capture = createCapture({ bin: BIN, session, spawn, projectOf: (cwd) => resolveProject(cwd, { gitToplevel: (c) => { const r = spawnSync("git", ["rev-parse", "--show-toplevel"], { cwd: c, encoding: "utf8", timeout: 2000 }); return r.status === 0 ? r.stdout.trim() : null }) } })

  pi.on("session_start", async (_e, ctx) => {
    if (isSubagent(ctx)) return
    await session.start(ctx?.cwd)
  })
  pi.on("before_agent_start", async (event, ctx) => {
    if (isSubagent(ctx)) return
    if (event?.messageType !== "user") return
    await capture.capture(ctx?.cwd, event?.prompt)
  })
  pi.on("session_shutdown", async (event, ctx) => {
    if (event?.reason !== "quit" || isSubagent(ctx)) return
    await session.end(ctx?.cwd)
  })
}
```

`plugins/memory/pi-omp/package.json`:

```json
{
  "name": "mnemonic-memory-piomp",
  "version": "0.1.0",
  "type": "module",
  "description": "Mnemonic-v2 memory for pi/omp (ADR-0067): session registration + per-prompt capture through the mnemonic binary. The mem_* tools come from the mnemonic-memory MCP entry.",
  "pi": { "extensions": ["./index.ts"] },
  "omp": { "extensions": ["./index.ts"] },
  "files": ["index.ts", "src/session.js", "src/capture.js", "package.json"]
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd plugins/memory/pi-omp && node --test test/`
Expected: PASS (all session + capture suites)

- [ ] **Step 5: Manual install for the door check** (runtime, not repo files)

```bash
# 1. Build the v2 binary at a stable absolute path (PATH's `mnemonic` stays v1 until the cutover).
cd mnemonic && go build -o "$HOME/.local/bin/mnemonic-v2" ./cmd/mnemonic && cd ..

# 2. Install the extension (installer substitution: <BIN> → the absolute path).
mkdir -p ~/.omp/agent/extensions/mnemonic
cp plugins/memory/pi-omp/index.ts plugins/memory/pi-omp/package.json ~/.omp/agent/extensions/mnemonic/
mkdir -p ~/.omp/agent/extensions/mnemonic/src
cp plugins/memory/pi-omp/src/*.js ~/.omp/agent/extensions/mnemonic/src/
sed -i.bak 's|const BIN = "<BIN>"|const BIN = "'"$HOME"'/\.local/bin/mnemonic-v2"|' ~/.omp/agent/extensions/mnemonic/index.ts && rm ~/.omp/agent/extensions/mnemonic/index.ts.bak

# 3. MCP entry (backup first; the v1 `mnemonic` entry is removed for a clean door — Task 7's cutover matcher rewrites it on the real install).
cp ~/.omp/agent/mcp.json ~/.omp/agent/mcp.json.bak-doorcheck
node -e '
const fs = require("fs"), f = process.env.HOME + "/.omp/agent/mcp.json"
const j = JSON.parse(fs.readFileSync(f, "utf8"))
delete j.mcpServers.mnemonic
j.mcpServers["mnemonic-memory"] = { type: "stdio", command: process.env.HOME + "/.local/bin/mnemonic-v2", args: ["memory", "mcp", "--profile", "agent"] }
fs.writeFileSync(f, JSON.stringify(j, null, 2) + "\n")'
```

- [ ] **Step 6: Live OMP door check** (the hypothesis gate — user runs the session)

In a live omp session with cwd = this repo (`skillgrid-v2`):
1. Ask the agent **3 distinct prompts** (any real task turns count — N = the number of user prompts sent).
2. Run **one `task` subagent** (any small subagent work).
3. Ask the agent: "Save a memory with title `DOOR-CHECK` and content `blueprint door check roundtrip`. Then search your memory for DOOR-CHECK and tell me the hit." (exercises `mem_save` → `mem_search` through the host-spawned MCP server).
4. Quit the session.

Verify from the shell (evidence for the `live-omp-door-check` backstop):

```bash
DB=~/.skillgrid/mnemonic/skillgrid-v2.sqlite
# a) exactly 1 new session row for the conversation (the adapter's id), directory = this repo
sqlite3 "$DB" "SELECT id, directory, ownership FROM sessions ORDER BY started_at DESC LIMIT 5"
# b) exactly N prompt rows under that session id
sqlite3 "$DB" "SELECT count(*) FROM user_prompts WHERE session_id = '<adapter-sid>'"
# c) the subagent added 0 session rows and 0 prompt rows (no other new rows in the window)
sqlite3 "$DB" "SELECT count(*) FROM sessions WHERE started_at > '<session start time>'"
sqlite3 "$DB" "SELECT session_id, count(*) FROM user_prompts WHERE created_at > '<session start time>' GROUP BY session_id"
# d) the MCP round-trip hit the session cwd's store
sqlite3 "$DB" "SELECT title FROM observations WHERE title = 'DOOR-CHECK'"
```

**Door verdict:** a)–d) all hold → the unified transport holds for omp; proceed. Any fail → record the exact failure as a deviation (ADR-0067 revisit criterion 1: per-session spawned child for the affected host); the user decides revise vs abandon per the Hypothesis.

- [ ] **Step 7: Commit**

```bash
git add plugins/memory/pi-omp/
git commit -m "feat: add the pi-omp mnemonic-v2 extension (v0, door-checked)

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 2"
```

### Task 3: installer core — (host, family) manifests + `--family memory`

**Files:**
- Create: `skillgrid-cli/internal/install/registry.go` — the exported (host, family) `Manifest` + the JSON(C) entry merge + tracked-file install/backup/atomic primitives (the seam the queued obs change migrates onto)
- Create: `skillgrid-cli/internal/install/memory.go` — the memory family: per-host file/entry plans + cutover matcher + engram coexistence + kilo AGENTS.md managed block
- Create: `skillgrid-cli/internal/install/memory_test.go` — the five fixture-HOME round-trip tests
- Modify: `skillgrid-cli/cmd/skillgrid/main.go` — `uninstall` + `status` commands; `--family` flag on all three; the memory family dispatches to the installer, the default (no `--family`) keeps the existing full bootstrap untouched

**Interfaces:**
- Consumes: `github.com/tidwall/sjson` + `github.com/tidwall/gjson` (already in skillgrid-cli go.mod — zero new deps; verified: sjson.Set/Delete preserve comments and foreign content byte-for-byte); the live v1 shapes pinned 2026-11-17 (below); Task 2's `plugins/memory/pi-omp/` (installed source); `plugins/memory/opencode-kilo/` (Task 4 — this task's fixture tests use a fixture repo tree with representative sources, the live opencode/kilo install happens in Task 4); `plugins/memory/cursor/` (Task 5 — same fixture-repo treatment).
- **Per-host dialect table** (pinned from the live configs; the cutover rewrites the v1 `mnemonic` key to the v2 row, all other bytes preserved):

| host | config file | v1 stale key → v2 key | v2 entry value |
|---|---|---|---|
| omp | `~/.omp/agent/mcp.json` (JSON) | `mcpServers.mnemonic` (`{type:stdio, command:"mnemonic", args:["mcp"]}`) → `mcpServers.mnemonic-memory` | `{"type":"stdio","command":"<abs>","args":["memory","mcp","--profile","agent"]}` |
| pi | `~/.skill-pi/agent/mcp.json` (JSON) | (no v1 entry — pi's v1 tools were in-process) → `mcpServers.mnemonic-memory` | `{"command":"<abs>","args":["memory","mcp","--profile","agent"]}` |
| opencode | `~/.config/opencode/opencode.jsonc` (JSONC) | `mcp.mnemonic` (`{"command":["mnemonic","mcp"],"enabled":true,"type":"local"}`) → `mcp.mnemonic-memory` | `{"command":["<abs>","memory","mcp","--profile","agent"],"enabled":true,"type":"local"}` |
| kilo | `~/.config/kilo/kilo.jsonc` (JSONC) | `mcp.mnemonic` (`{"command":["skillgrid","mcp"],...}`) → `mcp.mnemonic-memory` | same as opencode |
| cursor | `~/.cursor/mcp.json` (JSON) | `mcpServers.mnemonic` → `mcpServers.mnemonic-memory` | `{"command":"<abs>","args":["memory","mcp","--profile","agent"]}` |

- **Per-host file plan** (all tracked in the manifest; `<abs>` = the resolved binary):

| host | installed files | config mutations beyond the MCP entry |
|---|---|---|
| omp | `~/.omp/agent/extensions/mnemonic/{index.ts,src/session.js,src/capture.js,package.json}` (index.ts `<BIN>`→`<abs>`; the Task-6 files join this set) | — |
| pi | `~/.skill-pi/agent/extensions/mnemonic/` (same set) | — |
| opencode | `~/.config/opencode/plugins/mnemonic-memory.ts` (the opencode-kilo source index.ts, `<BIN>`→`<abs>`; name reuses the v1 path so the config `plugin` array entry stays valid) | `plugin` array: drop the 5 retired v1 entries (`skillgrid-compaction.ts`, `skillgrid-events.ts`, `skillgrid-squad.ts`, `mnemonic-codeindex.ts`, `skillgrid-context.ts`); delete the 5 retired installed files in `~/.config/opencode/plugins/` |
| kilo | `~/.config/kilo/plugins/mnemonic-memory.ts` (byte-identical to the opencode copy — same source, same substitution) | `plugin` array: append `"./plugins/mnemonic-memory.ts"`; rewrite the AGENTS.md managed block (below); **engram check**: a `plugins/engram.ts` present → print the coexistence warning, record it for `status` |
| cursor | `~/.cursor/rules/mnemonic.mdc` (Task-5 source, `<BIN>`→`<abs>`) | `~/.cursor/hooks.json`: set exactly the three v2 memory hook entries (Task-5 shape), foreign entries untouched |

- Produces:
  - `type Manifest struct { Host, Family, Bin string; Files map[string]string /* path → sha256 */; Entries []Entry; Engram []string; Written time.Time }` + `type Entry struct { File, KeyPath, SHA256 string }` — persisted as `map[host]Manifest` at `~/.skillgrid/mnemonic/memory-install.json` (atomic write).
  - `Install(ctx, host, family, opts) (Manifest, error)` / `Uninstall(ctx, host, family, opts) error` / `Status(ctx, host, family) (StatusReport, error)`; `opts = {Home, Repo, Bin string, DryRun bool}` (`Home` = the fixture-HOME override for tests; `Bin` default `exec.LookPath("mnemonic")`; `Repo` default `~/.skillgrid/repos/skillgrid`).
  - **Merge contract (the D-2 tracked-set invariants):** entry writes via `sjson.Set` (comments + foreign bytes preserved — verified), entry deletions via `sjson.Delete`; every file write is atomic (tmp + `os.Rename`) with a pre-write backup to `~/.skillgrid/mnemonic/backups/<host>/<basename>-<unix>`; the manifest tracks ONLY what this change wrote; uninstall removes the tracked set **only when the current sha256 matches** (a modified tracked file is kept + reported as drift, never deleted blind); a malformed config file fails install with the file untouched (error before any write).
  - **Cutover matcher:** recognizes the v1 `mnemonic` key in any host dialect (string-command, array-command, cursor shape) → deletes it and writes the v2 row (table above). Absent v1 key = clean install. The post-install grep invariant: no functional reference to `:7438`, `skillgrid mcp`, `web_cache_`, `fact_` in any installed file/entry.
  - **Kilo AGENTS.md managed block:** same marker pair as the bootstrap setup (`<!-- BEGIN SKILLGRID MNEMONIC — managed by skillgrid setup kilocode -->` … `<!-- END SKILLGRID MNEMONIC -->`, setup.go:34-35); v2 content = the kilo host shell (tools named `mnemonic-memory_mem_*`, "do not invent or override `session_id`", the repair line naming `~/.config/kilo/kilo.jsonc` + `skillgrid install --family memory kilo`). The block text is defined in memory.go (single source); Task 7 points the bootstrap setup module at it so a full install never resurrects v1 text.
  - CLI: `skillgrid install --family memory all|<host>…`, `skillgrid uninstall --family memory all|<host>…`, `skillgrid status --family memory [all|<host>…]`; hosts = `omp pi opencode kilo cursor`; `--bin`, `--repo`, `--dry-run`. `status` prints per host: installed (files/entries verified vs manifest) / drift (sha mismatch, kept) / engram coexistence / captures dropped (reads `<dataDir>/<project>/.capture-drops.json` — Task-2 scratch) / stale v1 entries found (cutover pending).
- SATISFIES: installer-roundtrip-fixture, engram-coexistence-warning, cutover-rewrites-stale-entries (fixture side; the live rewrite is Task 7), fail-open-drop-counter (status read side)

- [ ] **Step 1: Write the failing fixture tests**

`skillgrid-cli/internal/install/memory_test.go` — one fixture HOME per host under `t.TempDir()` (the v1 shapes from the dialect table, plus foreign entries: omp `context7`/`playwright`, opencode `publora` + a `// comment` line + the 6-entry `plugin` array, kilo `context7` + `plugin: ["./plugin/skillgrid-checkpoint.ts"]` + an AGENTS.md with the v1 marker block + foreign text, cursor `GitLab` + gryph hook entries). Five test functions, one per host, each running the SAME five-act round trip (install → re-run → foreign check → uninstall → byte-identical HOME); the omp test additionally asserts the cutover rewrite. Full code for the omp test (the others are the same shape over their dialect):

```go
// TestMemoryInstallOmpRoundTrip: the five-act round trip on a CLEAN omp
// HOME (no v1 entries — the cutover is one-way and gets its own test):
// install / re-run / foreign-check / uninstall / HOME byte-identical.
func TestMemoryInstallOmpRoundTrip(t *testing.T) {
	home := t.TempDir()
	repo := fixtureRepo(t) // plugins/memory/{pi-omp,opencode-kilo,cursor} stubs with <BIN>
	mcp := filepath.Join(home, ".omp", "agent", "mcp.json")
	writeFile(t, mcp, `{
  "$schema": "https://x/mcp-schema.json",
  "mcpServers": {
    "context7": { "type": "http", "url": "https://mcp.context7.com/mcp" },
    "playwright": { "type": "stdio", "command": "playwright-mcp" }
  }
}
`)
	preHome := snapshotDir(t, home)

	// 1. install: the tracked set is created, the v2 entry carries the absolute binary.
	m, err := Install(context.Background(), "omp", "memory", Opts{Home: home, Repo: repo, Bin: "/abs/mnemonic-v2"})
	if err != nil { t.Fatal(err) }
	got, _ := os.ReadFile(mcp)
	if !bytes.Contains(got, []byte(`"command":"/abs/mnemonic-v2"`)) {
		t.Fatalf("v2 entry must carry the absolute binary: %s", got)
	}
	for _, foreign := range []string{`"context7": { "type": "http", "url": "https://mcp.context7.com/mcp" }`, `"playwright": { "type": "stdio", "command": "playwright-mcp" }`} {
		if !bytes.Contains(got, []byte(foreign)) { t.Fatalf("foreign entry lost: %s", foreign) }
	}
	if len(m.Files) != 4 { t.Fatalf("omp tracks exactly the 4 extension files, got %d", len(m.Files)) }

	// 2. re-run: idempotent no-op (manifest + files byte-identical).
	m2, err := Install(context.Background(), "omp", "memory", Opts{Home: home, Repo: repo, Bin: "/abs/mnemonic-v2"})
	if err != nil { t.Fatal(err) }
	if sha256Hex(m2) != sha256Hex(m) { t.Fatalf("re-run must be a no-op") }

	// 3. uninstall: tracked set removed, HOME byte-identical to pre-install.
	if err := Uninstall(context.Background(), "omp", "memory", Opts{Home: home}); err != nil { t.Fatal(err) }
	if diff := homeDiff(t, home, preHome); diff != "" { t.Fatalf("HOME must be byte-identical:\n%s", diff) }
}

// TestMemoryInstallOmpCutover: a HOME carrying the v1 `mnemonic` entry gets
// it rewritten to the v2 dialect row at install; the rewrite is one-way
// (uninstall does not resurrect the v1 entry).
func TestMemoryInstallOmpCutover(t *testing.T) {
	home := t.TempDir()
	repo := fixtureRepo(t)
	mcp := filepath.Join(home, ".omp", "agent", "mcp.json")
	writeFile(t, mcp, `{
  "mcpServers": {
    "mnemonic": { "type": "stdio", "command": "mnemonic", "args": ["mcp"] },
    "playwright": { "type": "stdio", "command": "playwright-mcp" }
  }
}
`)
	Install(context.Background(), "omp", "memory", Opts{Home: home, Repo: repo, Bin: "/abs/mnemonic-v2"})
	got, _ := os.ReadFile(mcp)
	if bytes.Contains(got, []byte(`"mnemonic":`)) {
		t.Fatalf("the v1 key must be gone: %s", got)
	}
	if !bytes.Contains(got, []byte(`"mnemonic-memory"`) || !bytes.Contains(got, []byte(`"command":"/abs/mnemonic-v2"`)) {
		t.Fatalf("the v2 entry must be present: %s", got)
	}
	Uninstall(context.Background(), "omp", "memory", Opts{Home: home})
	got, _ = os.ReadFile(mcp)
	if bytes.Contains(got, []byte(`"mnemonic":`)) {
		t.Fatalf("the cutover is one-way — the v1 entry is not resurrected: %s", got)
	}
	if !bytes.Contains(got, []byte(`"playwright"`)) {
		t.Fatalf("the foreign entry must survive the whole cycle: %s", got)
	}
}

// TestMemoryInstallOmpModifiedFileKept: a tracked file the user edited after
// install survives uninstall (sha mismatch → kept + reported drift).
func TestMemoryInstallOmpModifiedFileKept(t *testing.T) {
	home := t.TempDir()
	repo := fixtureRepo(t)
	Install(context.Background(), "omp", "memory", Opts{Home: home, Repo: repo, Bin: "/abs/mnemonic-v2"})
	idx := filepath.Join(home, ".omp", "agent", "extensions", "mnemonic", "index.ts")
	writeFile(t, idx, "// user edit\n")
	report, err := Status(context.Background(), "omp", "memory", Opts{Home: home})
	if err != nil { t.Fatal(err) }
	if len(report.Drift) == 0 { t.Fatalf("status must report the modified tracked file") }
	Uninstall(context.Background(), "omp", "memory", Opts{Home: home})
	if _, err := os.Stat(idx); err != nil { t.Fatalf("modified tracked file must be kept") }
}

// TestMemoryInstallKiloByteIdentical: the kilo plugin is a byte-identical
// copy of the opencode-kilo source (installer-enforced).
func TestMemoryInstallKiloByteIdentical(t *testing.T) {
	home := t.TempDir()
	repo := fixtureRepo(t)
	Install(context.Background(), "opencode", "memory", Opts{Home: home, Repo: repo, Bin: "/abs/mnemonic-v2"})
	Install(context.Background(), "kilo", "memory", Opts{Home: home, Repo: repo, Bin: "/abs/mnemonic-v2"})
	a, _ := os.ReadFile(filepath.Join(home, ".config", "opencode", "plugins", "mnemonic-memory.ts"))
	b, _ := os.ReadFile(filepath.Join(home, ".config", "kilo", "plugins", "mnemonic-memory.ts"))
	if !bytes.Equal(a, b) { t.Fatalf("kilo must be a byte-identical copy of the opencode plugin") }
}

// TestMemoryInstallKiloEngram: an engram file in the kilo plugin dir
// triggers the coexistence warning and is listed by status.
func TestMemoryInstallKiloEngram(t *testing.T) {
	home := t.TempDir()
	repo := fixtureRepo(t)
	writeFile(t, filepath.Join(home, ".config", "kilo", "plugins", "engram.ts"), "engram")
	m, err := Install(context.Background(), "kilo", "memory", Opts{Home: home, Repo: repo, Bin: "/abs/mnemonic-v2"})
	if err != nil { t.Fatal(err) }
	if len(m.Engram) != 1 { t.Fatalf("the engram file must be recorded, got %v", m.Engram) }
}

// TestMemoryInstallCursorPreservesForeign: the cursor hooks.json merge keeps
// every foreign entry byte-identical and adds exactly the three v2 memory
// entries; a malformed hooks.json fails with the file untouched.
func TestMemoryInstallCursorPreservesForeign(t *testing.T) {
	home := t.TempDir()
	repo := fixtureRepo(t)
	hooks := filepath.Join(home, ".cursor", "hooks.json")
	writeFile(t, hooks, `{
  "version": 1,
  "hooks": {
    "beforeShellExecution": [{ "command": "gryph _hook cursor beforeShellExecution" }],
    "sessionStart": [{ "command": "./hooks/cursor-session-start.sh" }]
  }
}
`)
	Install(context.Background(), "cursor", "memory", Opts{Home: home, Repo: repo, Bin: "/abs/mnemonic-v2"})
	got, _ := os.ReadFile(hooks)
	if !bytes.Contains(got, []byte(`"command": "gryph _hook cursor beforeShellExecution"`)) {
		t.Fatalf("foreign hook lost: %s", got)
	}
	if bytes.Contains(got, []byte("cursor-session-start.sh")) {
		t.Fatalf("the v1 memory hook must be replaced: %s", got)
	}
}
```

(`fixtureRepo`, `snapshotDir`, `homeDiff`, `sha256Hex`, `writeFile` = small test helpers in the same file; the opencode/pi tests mirror the omp round trip over their dialect — opencode additionally asserts the 5 retired `plugin` array entries are gone and `mnemonic-memory.ts` remains, pi asserts no v1 key existed and the v2 entry is added.)

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd skillgrid-cli && go test ./internal/install/ -run TestMemoryInstall -count=1`
Expected: FAIL — `undefined: Install` / `Opts` / `Status` / `Uninstall`

- [ ] **Step 3: Write the implementation**

`skillgrid-cli/internal/install/registry.go` — the exported seam (core shown in full; the per-host plans live in memory.go):

```go
package install

// registry.go — the (host, family) tracked-set installer (ADR-0067).
// The manifest tracks ONLY what this change wrote; foreign content is
// never adopted, never clobbered, never deleted. Uninstall removes exactly
// the tracked set, and only when the tracked sha still matches (a modified
// tracked file is kept + reported, never deleted blind). The obs change
// migrates onto this seam (its own family, same Manifest/merge primitives).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tidwall/sjson"
)

// Manifest is the tracked-set record for one (host, family) install.
type Manifest struct {
	Host    string            `json:"host"`
	Family  string            `json:"family"`
	Bin     string            `json:"bin"` // the absolute path baked into entries/files
	Files   map[string]string `json:"files"`
	Entries []Entry           `json:"entries"`
	Engram  []string          `json:"engram,omitempty"` // coexisting files listed by status
	Written time.Time         `json:"written"`
}

// Entry is one owned config entry: file + JSON key path + the value's sha256
// at write time (uninstall verifies before deleting).
type Entry struct {
	File    string `json:"file"`
	KeyPath string `json:"keyPath"`
	SHA256  string `json:"sha256"`
}

// Opts scopes one family operation. Home overrides $HOME (fixture tests);
// empty = the real home. Bin defaults to exec.LookPath("mnemonic"); Repo
// defaults to ~/.skillgrid/repos/skillgrid.
type Opts struct {
	Home   string
	Repo   string
	Bin    string
	DryRun bool
}

// manifestPath — the per-repo family manifest (all hosts in one file).
func manifestPath(o Opts, family string) string {
	home := o.Home
	if home == "" { home, _ = os.UserHomeDir() }
	return filepath.Join(home, ".skillgrid", "mnemonic", family+"-install.json")
}

// loadManifests / saveManifests — the map[host]Manifest store (atomic).
func loadManifests(o Opts, family string) (map[string]Manifest, error) { /* read or empty */ }
func saveManifests(o Opts, family string, m map[string]Manifest) error { /* tmp + rename */ }

// sha256File hashes a file's content (manifest tracking).
func sha256File(path string) (string, error) { /* io.ReadAll → sha256 hex */ }

// setEntry writes one JSON(C) entry, preserving comments + foreign bytes
// (sjson verified against the live dialects), with backup + atomic rename.
// The returned sha256 is the value's fingerprint for the manifest.
func setEntry(o Opts, file, keyPath string, value any) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil { return "", err }
	if _, err := sjson.ValidateBytes(data); err != nil {
		return "", fmt.Errorf("%s: %w", file, err) // fails before any write
	}
	out, err := sjson.SetBytes(data, keyPath, value)
	if err != nil { return "", err }
	return sha256.Sum256(mustJSON(value)), writeTracked(o, file, data, out)
}

// deleteEntry removes one JSON(C) entry (same backup/atomic contract).
func deleteEntry(o Opts, file, keyPath string) error { /* sjson.DeleteBytes + writeTracked */ }

// writeFileTracked installs one file (content bytes, already substituted),
// tracked by sha256, with backup + atomic rename.
func writeFileTracked(o Opts, dst string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil { return err }
	if err := backup(o, dst); err != nil { return err }
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil { return err }
	return os.Rename(tmp, dst)
}

// backup copies the pre-image to <dataDir>/backups/<host>/<basename>-<unix>
// (only when the file exists).
func backup(o Opts, file string) error { /* best-effort copy */ }

// writeTracked = backup + atomic write of the merged config bytes.
func writeTracked(o Opts, file string, before, after []byte) error { /* backup(file) + tmp + rename */ }
```

`skillgrid-cli/internal/install/memory.go` — the family (core shown: plan dispatch + the omp plan + the cutover matcher; the other four plans are the same shape over the dialect table):

```go
package install

// memory.go — the mnemonic-v2 memory family (ADR-0067). Per-host plans:
// the file set (from the repo's plugins/memory/) + the MCP entry + the
// host-specific config mutations (dialect table in the blueprint).

const familyMemory = "memory"

var memoryHosts = []string{"omp", "pi", "opencode", "kilo", "cursor"}

// Install runs one host's memory-family plan.
func Install(ctx context.Context, host, family string, o Opts) (Manifest, error) {
	if family != familyMemory {
		return Manifest{}, fmt.Errorf("unknown family %q (have: memory)", family)
	}
	if o.Bin == "" {
		b, err := lookPath("mnemonic") // package seam (tests inject)
		if err != nil { return Manifest{}, fmt.Errorf("mnemonic binary not on PATH — pass --bin /abs/path: %w", err) }
		o.Bin = b
	}
	plan := memoryPlan[host]
	if plan == nil {
		return Manifest{}, fmt.Errorf("unknown host %q (have: %v)", host, memoryHosts)
	}
	m := Manifest{Host: host, Family: family, Bin: o.Bin, Files: map[string]string{}}
	m.Files, m.Entries, m.Engram, err = plan(ctx, o)
	if err != nil { return Manifest{}, err }
	m.Written = time.Now()
	return m, saveHostManifest(o, m)
}

// Uninstall removes exactly the tracked set (sha-verified files + entries).
// Returns a report: removed, kept (drift), errors.
func Uninstall(ctx context.Context, host, family string, o Opts) error { /* per tracked file: sha match → remove, else keep+report; per entry: deleteEntry */ }

// Status verifies the live state against the manifest and reads the
// per-project drop counters (<dataDir>/<project>/.capture-drops.json).
func Status(ctx context.Context, host, family string, o Opts) (StatusReport, error) { /* installed/drift/engram/drops/stale-v1 */ }

// plan = files + entry mutations for one host.
type planFn func(ctx context.Context, o Opts) (files map[string]string, entries []Entry, engram []string, err error)

var memoryPlan = map[string]planFn{
	"omp":      planOmp,
	"pi":       planPi,
	"opencode": planOpenCode,
	"kilo":     planKilo,
	"cursor":   planCursor,
}

// planOmp — the pi-omp extension dir + the omp-dialect MCP entry (the cutover
// matcher runs inside setEntry: a stale v1 key is deleted first).
func planOmp(ctx context.Context, o Opts) (map[string]string, []Entry, []string, error) {
	home := homeOf(o)
	files := map[string]string{}
	for _, rel := range []string{"index.ts", "src/session.js", "src/capture.js", "package.json"} {
		src := filepath.Join(o.Repo, "plugins", "memory", "pi-omp", rel)
		content, err := os.ReadFile(src)
		if err != nil { return nil, nil, nil, err }
		if rel == "index.ts" {
			content = []byte(strings.ReplaceAll(string(content), `const BIN = "<BIN>"`, "const BIN = "+strconv.Quote(o.Bin)))
		}
		dst := filepath.Join(home, ".omp", "agent", "extensions", "mnemonic", rel)
		if err := writeFileTracked(o, dst, content); err != nil { return nil, nil, nil, err }
		sha, _ := sha256File(dst)
		files[dst] = sha
	}
	mcp := filepath.Join(home, ".omp", "agent", "mcp.json")
	cutStale(o, mcp, "mcpServers.mnemonic") // v1 key, any dialect → delete
	sha, err := setEntry(o, mcp, "mcpServers.mnemonic-memory", map[string]any{
		"type": "stdio", "command": o.Bin, "args": []any{"memory", "mcp", "--profile", "agent"},
	})
	if err != nil { return nil, nil, nil, err }
	return files, []Entry{{File: mcp, KeyPath: "mcpServers.mnemonic-memory", SHA256: sha}}, nil, nil
}

// cutStale deletes the v1 `mnemonic` entry when present (the cutover;
// absent = clean install). Dialect-agnostic: the key path is the host's
// (mcpServers. / mcp.) — the value shape does not matter, it is replaced.
func cutStale(o Opts, file, keyPath string) { /* deleteEntry, ignore not-present */ }
```

The other plans: `planPi` (same as omp; pi path `~/.skill-pi/agent/`; entry without `type`); `planOpenCode` (file `plugins/mnemonic-memory.ts` from `plugins/memory/opencode-kilo/index.ts` substituted; `mcp.mnemonic-memory` array-command entry; drop the 5 retired `plugin` array entries + their installed files); `planKilo` (byte-identical copy of the opencode file — read the opencode INSTALLED bytes, never re-substitute; `plugin` array append via `sjson.Set(file, "plugin."+idx, "./plugins/mnemonic-memory.ts")`; the AGENTS.md block via the marker pair; the engram check = `plugins/engram.ts` present → record + warn); `planCursor` (rule file from `plugins/memory/cursor/mnemonic.mdc`; `mcp.mnemonic-memory` string-command entry; hooks.json: the three v2 entries replacing the v1 memory hooks, foreign entries preserved — the Task-5 hook shapes).

The kilo AGENTS.md block (v2, the single-source text in memory.go):

```
<!-- BEGIN SKILLGRID MNEMONIC — managed by skillgrid setup kilocode -->
# Mnemonic Memory (v2)

Persistent memory is available through the `mnemonic-memory` MCP server.
Memory tools are named `mnemonic-memory_mem_*` (mem_save, mem_search,
mem_get_observation, mem_context, mem_session_summary, …). The v1 surfaces
(fact_*, code_*, web_cache_*, :7438 routes) are retired.

## Session protocol

- Do not invent or override `session_id` — the host adapter owns it.
- Save significant decisions, bug fixes, discoveries, and preferences with
  mem_save (title + type + structured content).
- Before closing a session, call mem_session_summary.

## Repair

If the mnemonic-memory tools are missing, the MCP entry in
~/.config/kilo/kilo.jsonc must be
`{"command":["<BIN>","memory","mcp","--profile","agent"],"enabled":true,"type":"local"}`;
reinstall with `skillgrid install --family memory kilo`.
<!-- END SKILLGRID MNEMONIC -->
```

(`<BIN>` substituted at install, like index.ts.)

`skillgrid-cli/cmd/skillgrid/main.go` — the wiring (shape shown; the existing bootstrap is untouched when `--family` is absent):

```go
fs.StringVar(&vFamily, "family", "", "install family (memory); empty = the full skillgrid bootstrap")
// commands: "install"/"in" (existing), "uninstall", "status" (new)
// dispatch: vFamily == "memory" → memoryFamily(cmd, hosts, Opts{Bin: vBin, Repo: vRepo, DryRun: vDry})
//           vFamily == "" → the existing Run(c) bootstrap (unchanged)
// hosts: "all" → memoryHosts; otherwise each arg must be a known host
// flags: --bin (default LookPath), --repo (default ~/.skillgrid/repos/skillgrid)
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/install/ -count=1 && go build ./...`
Expected: PASS (all five host round trips + drift + kilo-identity + engram + cursor-foreign + the existing install package suites)

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/install/registry.go skillgrid-cli/internal/install/memory.go \
        skillgrid-cli/internal/install/memory_test.go skillgrid-cli/cmd/skillgrid/main.go
git commit -m "feat: add the memory family installer (host,family) tracked-set manifests

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 3"
```

### Task 4: opencode-kilo plugin v0 (single-file adapter)

> The opencode/kilo side of the tracer fan-out: session lifecycle + prompt capture + the `session_id` rewrite, verified live on both hosts. Compaction + protocol shell land in Task 6 (same file, same exports pattern).

**Files:**
- Create: `plugins/memory/opencode-kilo/index.ts` — the single-file plugin (glue + pure exports; `<BIN>` substituted at install)
- Create: `plugins/memory/opencode-kilo/package.json`
- Create: `plugins/memory/opencode-kilo/test/{rewrite,session,capture}.test.ts`
- (runtime, installer-managed) `~/.config/opencode/plugins/mnemonic-memory.ts` + `~/.config/kilo/plugins/mnemonic-memory.ts` (byte-identical) + the `mcp.mnemonic-memory` entries (Task 3's installer)

**Interfaces:**
- Consumes: the **opencode V2 plugin API** (structural `ctx`, verified against the engram V2 adapter — the proven reference that runs on real opencode): `location: { directory }` (the instance directory; the V2 server is shared across locations, so every event is location-filtered), `event.subscribe({ signal })` → AsyncIterable, `tool.hook("execute.before", cb)` where the call is `{ input, tool, sessionID, id }` and `input` is mutable. **Proven event shapes:** `session.created`/`session.updated`/`session.deleted` carry `data.{sessionID, parentID, location.directory}` (engram bug #116: V2 nests under `data`; V1 was `properties.info`); `session.inbox.enqueued` carries `data.{sessionID, inboxID, item:{ type: "user", payload: { text } }}` + `event.location.directory` — the V2 prompt source with a durable inbox identity. The `{ id, setup }` default export with the `@opencode/plugin` import-fallback stub (v1 repo convention, ADR-0045 — the stub keeps the file importable under plain Node for the tests). The Task 1 verbs (`memory session start --id`, `memory session end --session`, `memory save-prompt --id --source-inbox-id`). The pi-omp scratch contract (`~/.skillgrid/mnemonic/<project>/.capture-drops.json`, the `resolveProject` mirror, the production spawn: no shell, 5 s hard timeout SIGKILL, resolves only on exit 0).
- Produces:
  - `MCP_KEY = "mnemonic-memory"` — MUST equal the installed MCP entry key (the key/normalizer mismatch means the rewrite misses and phantom sessions appear — Key link).
  - `canonicalToolName(tool)` — lowercase + strip the `mnemonic-memory_` prefix (opencode qualifies MCP tool ids as `<server>_<tool>`; only our own prefix is normalized — engram convention).
  - `isMemoryTool(tool)` — canonical starts with `mem_`.
  - `rewriteMemoryCall(call)` — pure: memory tool + object `input` → `call.input.session_id = call.sessionID` (**always** — the adapter is the single authority; a foreign or absent model value is replaced); returns whether it rewrote. Non-memory tools untouched.
  - `extractSessionEvent(event, directory)` → `{ id, parentID } | undefined` (shape + location filter: a present `data.location.directory` ≠ ours → undefined).
  - `createSessionRegistry({ bin, spawn })` → `{ registered(id), onCreated(e, directory), onDeleted(e, directory) }`: root sessions (no `parentID`) register + spawn `session start`; **subagents never register (0 rows)**; deleted registered sessions spawn `session end`; all fail-open (a spawn failure keeps the registration — the drop counter surfaces a dead verb).
  - `extractInboxPrompt(event, directory)` → `{ sessionID, inboxID, text } | undefined` (user items only, trimmed, empty → undefined).
  - `createCapture({ bin, session, spawn, dataDir, projectOf })` → `{ capture(directory, p) }`: gated on `session.registered(p.sessionID)` (a session created before a plugin reload is missed until it re-fires `session.updated` — documented v0 edge, same as v1; engram-style lazy ensure-on-tool-call is the future improvement, out of scope) + non-blank text; spawns `save-prompt --id <sid> --source-inbox-id <inboxID>` with the text on stdin at `cwd = directory` (the verb resolves the project from cwd); a failing spawn increments the drop file, never throws.
  - `setup(ctx)` glue: the event loop (per-event try/catch — one bad event never kills the stream), the `execute.before` rewrite (try/catch fail-open), cleanup aborts the subscription. No reconnect in v0 (a server restart re-runs `setup` — the v1 behavior).
- SATISFIES: T-opencode-live (the session + capture + rewrite side; the live smoke is Step 5), T-fail-open (the opencode/kilo capture side)

- [ ] **Step 1: Write the failing tests**

`plugins/memory/opencode-kilo/test/rewrite.test.ts`:

```ts
import test from "node:test"
import assert from "node:assert/strict"
import { canonicalToolName, isMemoryTool, rewriteMemoryCall } from "../index.ts"

test("canonical: strips only the mnemonic-memory_ prefix, case-insensitive", () => {
  assert.equal(canonicalToolName("mnemonic-memory_mem_save"), "mem_save")
  assert.equal(canonicalToolName("MNEMONIC-MEMORY_mem_search"), "mem_search")
  assert.equal(canonicalToolName("gitnexus_impact"), "gitnexus_impact")
  assert.equal(canonicalToolName("mem_save"), "mem_save") // unqualified (a direct MCP client)
})

test("isMemoryTool: only the mem_* surface", () => {
  assert.ok(isMemoryTool("mnemonic-memory_mem_save"))
  assert.ok(isMemoryTool("mnemonic-memory_mem_session_summary"))
  assert.ok(!isMemoryTool("mnemonic-memory_admin_reset"))
  assert.ok(!isMemoryTool("mnemonic-codeindex_code_search"))
  assert.ok(!isMemoryTool("bash"))
})

test("rewrite: absent and foreign session_id are replaced by the harness id", () => {
  const call = { sessionID: "oc-sess-1", tool: "mnemonic-memory_mem_save", input: { title: "t" } }
  assert.ok(rewriteMemoryCall(call))
  assert.equal(call.input.session_id, "oc-sess-1")

  const foreign = { sessionID: "oc-sess-1", tool: "mnemonic-memory_mem_save", input: { session_id: "model-invented" } }
  assert.ok(rewriteMemoryCall(foreign))
  assert.equal(foreign.input.session_id, "oc-sess-1")
})

test("rewrite: non-memory tools and absent input are untouched", () => {
  const call = { sessionID: "s", tool: "bash", input: { session_id: "keep" } }
  assert.ok(!rewriteMemoryCall(call))
  assert.equal(call.input.session_id, "keep")
  assert.ok(!rewriteMemoryCall({ sessionID: "s", tool: "mnemonic-memory_mem_save", input: undefined }))
})
```

`plugins/memory/opencode-kilo/test/session.test.ts`:

```ts
import test from "node:test"
import assert from "node:assert/strict"
import { extractSessionEvent, createSessionRegistry } from "../index.ts"

const DIR = "/repo"
const created = (id: string, parentID?: string, directory = DIR) => ({
  type: "session.created",
  data: { sessionID: id, ...(parentID ? { parentID } : {}), location: { directory } },
})

test("extract: V2 data shape, location-filtered, unknown events are undefined", () => {
  assert.deepEqual(extractSessionEvent(created("s1"), DIR), { id: "s1", parentID: undefined })
  assert.deepEqual(extractSessionEvent(created("s2", "s1"), DIR), { id: "s2", parentID: "s1" })
  assert.equal(extractSessionEvent(created("s1", undefined, "/other"), DIR), undefined)
  assert.equal(extractSessionEvent({ type: "server.connected", data: {} }, DIR), undefined)
})

test("registry: roots register + end; subagents never register", async () => {
  const calls = []
  const r = createSessionRegistry({ bin: "mnemonic", spawn: async (args, _stdin, cwd) => calls.push({ args, cwd }) })
  r.onCreated({ id: "s1", parentID: undefined }, DIR)
  assert.ok(r.registered("s1"))
  assert.deepEqual(calls[0].args, ["mnemonic", "memory", "session", "start", "--id", "s1"])
  assert.equal(calls[0].cwd, DIR)

  r.onCreated({ id: "s2", parentID: "s1" }, DIR)
  assert.ok(!r.registered("s2"))
  assert.equal(calls.length, 1) // no spawn for the subagent

  r.onCreated({ id: "s1", parentID: undefined }, DIR) // re-fire (session.updated) — idempotent
  assert.equal(calls.length, 1)

  r.onDeleted({ id: "s1", parentID: undefined }, DIR)
  assert.ok(!r.registered("s1"))
  assert.deepEqual(calls[1].args, ["mnemonic", "memory", "session", "end", "--session", "s1"])

  r.onDeleted({ id: "s9", parentID: undefined }, DIR) // never registered — no end
  assert.equal(calls.length, 2)
})

test("registry: spawn failure is fail-open (registration retained)", async () => {
  const r = createSessionRegistry({ bin: "mnemonic", spawn: async () => { throw new Error("exit 1") } })
  r.onCreated({ id: "s1", parentID: undefined }, DIR)
  assert.ok(r.registered("s1"))
  r.onDeleted({ id: "s1", parentID: undefined }, DIR) // no throw
})
```

`plugins/memory/opencode-kilo/test/capture.test.ts`:

```ts
import test from "node:test"
import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import { mkdtemp } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { extractInboxPrompt, createCapture } from "../index.ts"

const DIR = "/repo"
const inbox = (sessionID: string, inboxID: string, text: string) => ({
  type: "session.inbox.enqueued",
  location: { directory: DIR },
  data: { sessionID, inboxID, item: { type: "user", payload: { text } } },
})

test("extract: user inbox items only, trimmed, location-filtered", () => {
  assert.deepEqual(extractInboxPrompt(inbox("s1", "in1", " hello "), DIR), { sessionID: "s1", inboxID: "in1", text: "hello" })
  const assistant = inbox("s1", "in1", "x")
  assistant.data.item = { type: "assistant", payload: { text: "x" } }
  assert.equal(extractInboxPrompt(assistant, DIR), undefined)
  const other = inbox("s1", "in1", "x")
  other.location = { directory: "/other" }
  assert.equal(extractInboxPrompt(other, DIR), undefined)
  assert.equal(extractInboxPrompt({ type: "session.created", data: {} }, DIR), undefined)
})

test("capture: registered sessions only; the inbox id travels as --source-inbox-id", async () => {
  const calls = []
  const c = createCapture({ bin: "mnemonic", session: { registered: (id: string) => id === "s1" }, spawn: async (args, stdin, cwd) => calls.push({ args, stdin, cwd }) })
  await c.capture(DIR, { sessionID: "s2", inboxID: "in1", text: "hello" }) // unregistered
  await c.capture(DIR, { sessionID: "s1", inboxID: "in1", text: "   " })  // blank
  assert.equal(calls.length, 0)
  await c.capture(DIR, { sessionID: "s1", inboxID: "in1", text: "hello world" })
  assert.deepEqual(calls[0].args, ["mnemonic", "memory", "save-prompt", "--id", "s1", "--source-inbox-id", "in1"])
  assert.equal(calls[0].stdin, "hello world")
  assert.equal(calls[0].cwd, DIR)
})

test("capture: a failing spawn increments the drop counter, never throws", async () => {
  const dataDir = await mkdtemp(join(tmpdir(), "oc-kilo-drops-"))
  const c = createCapture({ bin: "mnemonic", session: { registered: () => true }, spawn: async () => { throw new Error("exit 1") }, dataDir: () => dataDir, projectOf: () => "p" })
  await c.capture(DIR, { sessionID: "s1", inboxID: "in1", text: "hello" })
  await c.capture(DIR, { sessionID: "s1", inboxID: "in2", text: "again" })
  const rec = JSON.parse(readFileSync(join(dataDir, "p", ".capture-drops.json"), "utf8"))
  assert.equal(rec.n, 2)
  assert.ok(rec.lastAt)
})
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd plugins/memory/opencode-kilo && node --test test/`
Expected: FAIL — `Cannot find module '../index.ts'`

- [ ] **Step 3: Write the implementation**

`plugins/memory/opencode-kilo/index.ts`:

```ts
/**
 * Mnemonic memory (v2) — the shared OpenCode/Kilo plugin adapter
 * (ADR-0045 entry shape, ADR-0067 session protocol). Single file: opencode v2
 * auto-discovers bare top-level .ts in plugins/ (a subdirectory package is
 * unproven for discovery; the v1 + engram plugins are single files). Kilo
 * installs a byte-identical copy (same fork API — no substitution of any kind
 * beyond the <BIN> the installer applies to both).
 *
 * The adapter is the single authority on session identity (the model never
 * invents a session_id): harness session ids from the event stream register +
 * capture, and tool.execute.before rewrites session_id on every
 * mnemonic-memory mem_* call. Fail-open everywhere: a dead or slow mnemonic
 * binary never blocks or crashes the host; drops are counted, not thrown
 * (~/.skillgrid/mnemonic/<project>/.capture-drops.json).
 */
import { spawn } from "node:child_process"
import { mkdir, readFile, writeFile } from "node:fs/promises"
import { basename } from "node:path"
import { execFileSync } from "node:child_process"
import { homedir } from "node:os"
import { join } from "node:path"

// Substituted by the installer: const BIN = "/abs/path/to/mnemonic"
const BIN = "<BIN>"

// Must equal the installed MCP entry key — a key/normalizer mismatch means the
// rewrite misses and phantom sessions appear.
export const MCP_KEY = "mnemonic-memory"

export function canonicalToolName(tool: string): string {
  const t = (tool || "").toLowerCase()
  return t.startsWith(MCP_KEY + "_") ? t.slice(MCP_KEY.length + 1) : t
}

export function isMemoryTool(tool: string): boolean {
  return canonicalToolName(tool).startsWith("mem_")
}

// The adapter owns session_id: always overwrite on our mem_* tools.
export function rewriteMemoryCall(call: any): boolean {
  if (!call || !isMemoryTool(call.tool)) return false
  if (call.input && typeof call.input === "object") {
    call.input.session_id = call.sessionID
    return true
  }
  return false
}

// V2 events nest session data under `data` (V1 was properties.info). The V2
// server is shared across locations — drop events from other projects.
export function extractSessionEvent(event: any, directory: string): { id: string; parentID: string | undefined } | undefined {
  const data = event?.data
  if (typeof data?.sessionID !== "string" || !data.sessionID) return undefined
  const loc = data.location?.directory
  if (loc && loc !== directory) return undefined
  return { id: data.sessionID, parentID: data.parentID }
}

// session.inbox.enqueued = the V2 prompt source, with a durable inbox identity
// (replays dedup server-side via --source-inbox-id).
export function extractInboxPrompt(event: any, directory: string): { sessionID: string; inboxID: string; text: string } | undefined {
  if (event?.type !== "session.inbox.enqueued") return undefined
  const loc = event.location?.directory
  if (loc && loc !== directory) return undefined
  const data = event.data
  const item = data?.item
  if (typeof data?.sessionID !== "string" || !data.sessionID) return undefined
  if (typeof data?.inboxID !== "string" || !data.inboxID) return undefined
  if (item?.type !== "user" || typeof item.payload?.text !== "string") return undefined
  const text = item.payload.text.trim()
  return text ? { sessionID: data.sessionID, inboxID: data.inboxID, text } : undefined
}

export function createSessionRegistry({ bin, spawn: sp }: { bin: string; spawn: Spawn }) {
  const registered = new Set<string>()
  return {
    registered: (id: string) => registered.has(id),
    // Subagent sessions (parentID present) never register — 0 rows.
    onCreated(e: { id: string; parentID: string | undefined }, directory: string) {
      if (e.parentID || registered.has(e.id)) return
      registered.add(e.id)
      void sp([bin, "memory", "session", "start", "--id", e.id], "", directory).catch(() => {})
    },
    onDeleted(e: { id: string }, directory: string) {
      if (!registered.has(e.id)) return
      registered.delete(e.id)
      void sp([bin, "memory", "session", "end", "--session", e.id], "", directory).catch(() => {})
    },
  }
}

export function createCapture(opts: {
  bin: string
  session: { registered: (id: string) => boolean }
  spawn: Spawn
  dataDir?: () => string
  projectOf?: (cwd: string) => string
}) {
  const dd = opts.dataDir ?? defaultDataDir
  const pj = opts.projectOf ?? ((cwd: string) => resolveProject(cwd, { gitToplevel: gitToplevelSync }))
  return {
    async capture(directory: string, p: { sessionID: string; inboxID: string; text: string }) {
      if (!opts.session.registered(p.sessionID) || !p.text) return
      try {
        await opts.spawn([opts.bin, "memory", "save-prompt", "--id", p.sessionID, "--source-inbox-id", p.inboxID], p.text, directory)
      } catch {
        void bumpDrops(dd(), pj(directory))
      }
    },
  }
}

// --- shared scratch/plumbing (mirrors plugins/memory/pi-omp; the packages are
// installed independently, so duplication beats a shared-package mechanism) ---

type Spawn = (args: string[], stdin: string, cwd: string) => Promise<void>

export const defaultDataDir = () => join(homedir(), ".skillgrid", "mnemonic")

function gitToplevelSync(cwd: string): string | null {
  try {
    return execFileSync("git", ["rev-parse", "--show-toplevel"], { cwd, stdio: ["ignore", "pipe", "ignore"] }).toString().trim() || null
  } catch {
    return null
  }
}

export function resolveProject(cwd: string, { env = process.env, gitToplevel }: { env?: NodeJS.ProcessEnv; gitToplevel?: (cwd: string) => string | null } = {}): string {
  if (env.MNEMONIC_PROJECT) return env.MNEMONIC_PROJECT
  const top = gitToplevel ? gitToplevel(cwd) : gitToplevelSync(cwd)
  if (top) return basename(top)
  return basename(cwd)
}

const DROP_FILE = ".capture-drops.json"
async function bumpDrops(dataDir: string, project: string) {
  const file = join(dataDir, project, DROP_FILE)
  let n = 1
  try { n = (JSON.parse(await readFile(file, "utf8")).n ?? 0) + 1 } catch { /* first drop */ }
  try {
    await mkdir(join(dataDir, project), { recursive: true })
    await writeFile(file, JSON.stringify({ n, lastAt: new Date().toISOString() }))
  } catch { /* best-effort */ }
}

// Production spawn: no shell, 5 s hard timeout SIGKILL, resolves only on exit 0.
const prodSpawn: Spawn = (args, stdin, cwd) => new Promise((resolve, reject) => {
  const child = spawn(args[0], args.slice(1), { cwd, stdio: ["pipe", "ignore", "ignore"] })
  const timer = setTimeout(() => child.kill("SIGKILL"), 5000)
  child.on("close", (code) => { clearTimeout(timer); if (code === 0) resolve(); else reject(new Error("exit " + code)) })
  child.on("error", (e) => { clearTimeout(timer); reject(e) })
  if (child.stdin) { child.stdin.on("error", () => {}); child.stdin.end(stdin) }
})

// --- glue ---

async function setup(ctx: any) {
  const directory: string = ctx.location.directory
  const abort = new AbortController()
  const registry = createSessionRegistry({ bin: BIN, spawn: prodSpawn })
  const capture = createCapture({ bin: BIN, session: registry, spawn: prodSpawn })
  void (async () => {
    for await (const event of ctx.event.subscribe({ signal: abort.signal })) {
      if (abort.signal.aborted) break
      try {
        if (event?.type === "session.created" || event?.type === "session.updated") {
          const e = extractSessionEvent(event, directory)
          if (e) registry.onCreated(e, directory)
        } else if (event?.type === "session.deleted") {
          const e = extractSessionEvent(event, directory)
          if (e) registry.onDeleted(e, directory)
        } else {
          const p = extractInboxPrompt(event, directory)
          if (p) await capture.capture(directory, p)
        }
      } catch { /* fail-open: one bad event never kills the stream */ }
    }
  })()
  try {
    await ctx.tool.hook("execute.before", (call: any) => { try { rewriteMemoryCall(call) } catch { /* fail-open */ } })
  } catch { /* fail-open */ }
  return () => { abort.abort() }
}

// V2 entry: { id, setup } (ADR-0045). @opencode/plugin is bundled in the host;
// the fallback stub keeps the file importable under plain Node (repo tests).
async function importPlugin() {
  try { return await import("@opencode/plugin") } catch { return { Plugin: { define: (d: any) => d } } }
}
const { Plugin } = await importPlugin()
export default Plugin.define({ id: "skillgrid.mnemonic-memory", setup })
```

`plugins/memory/opencode-kilo/package.json`:

```json
{
  "name": "mnemonic-memory-opencode-kilo",
  "version": "0.1.0",
  "description": "Mnemonic memory (v2) adapter for OpenCode + Kilo (ADR-0045/0067). Single-file plugin; Kilo installs a byte-identical copy.",
  "type": "module",
  "private": true,
  "scripts": {
    "test": "node --test test/"
  }
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd plugins/memory/opencode-kilo && node --test test/`
Expected: PASS (10 tests). Note: the `<BIN>` literal in index.ts makes the test import load the plugin glue with an uninstalled binary — that is correct: the tests exercise the pure exports, and the glue only acts when a host calls `setup`.

- [ ] **Step 5: Live smoke — the T-opencode-live backstop (recorded)**

```bash
# install the real source through the Task 3 installer (--repo = the working
# tree: the synced clone does not have plugins/memory/ yet)
skillgrid install --family memory opencode kilo --bin "$(which mnemonic)" --repo "$(pwd)"
# kilo assertions: the two installed files are byte-identical
cmp ~/.config/opencode/plugins/mnemonic-memory.ts ~/.config/kilo/plugins/mnemonic-memory.ts
# the kilo AGENTS.md block + the mcp.mnemonic-memory entries (all 4 hosts' dialects)
grep -n "mnemonic-memory" ~/.config/opencode/opencode.jsonc ~/.config/kilo/kilo.jsonc
```

opencode session (scratch dir, fresh): prompt twice; then an agent `mem_save` whose args carry `session_id: "foreign-model-id"`; then one subagent (task) run with an internal prompt. Kilo: repeat the minimal path (one prompt + one `mem_save`). Verify in the store for the scratch project:

- exactly **1 session row** per host (the harness id, agent = host) — subagents add 0
- exactly **2 prompt rows** (opencode) under the harness id with `source_inbox_id` set — subagent prompts add 0
- the `mem_save` observation stored under the **harness id**, not `foreign-model-id` (the rewrite fired)
- a kill of the mnemonic binary mid-session (temp `--bin` to a dead path in a throwaway config) → the session continues + the drop counter increments

Record the row dumps (`sqlite3 ~/.skillgrid/mnemonic/<project>.sqlite 'SELECT ...'`) + the install output as task evidence. If opencode or kilo is not installed on this machine: record the missing-host deviation, run the other, and verify the absent host's install set by inspection (file + entry shapes only) — the T3 fixture tests already pin those.

- [ ] **Step 6: Commit**

```bash
git add plugins/memory/opencode-kilo/
git commit -m "feat: add the opencode/kilo memory adapter (single-file v2 plugin)

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 4"
```

### Task 5: cursor surface — hook template + generated rule

> The cursor side of the fan-out: no plugin API, a hook-triple + an alwaysApply rule. The rule carries the **host shell only** — the protocol body is MCP-instructions-only (Global Constraints, ADR-0067), which is what makes T-protocol's "body exactly once" hold on cursor (cursor injects the MCP initialize instructions AND the rule; a duplicated body would appear twice).

**Files:**
- Create: `plugins/memory/cursor/hooks.json` — the hook template (`<BIN>` placeholder)
- Create: `plugins/memory/cursor/mnemonic.mdc` — the alwaysApply rule (host shell: mapping + pin + repair)
- Create: `plugins/memory/cursor/test/rule-shape.test.mjs`
- Create: `mnemonic/internal/transport/mcp/instructions_test.go` — the ≤2,048-rune guard + the rule-generated-section diff (drift detector)
- (runtime, installer-managed) `~/.cursor/rules/mnemonic.mdc` + the merged `~/.cursor/hooks.json` entries + the `mcp.mnemonic-memory` entry (Task 3's `planCursor`)

**Interfaces:**
- Consumes: the cursor hook format (verified against the live `~/.cursor/hooks.json`: `{"version": 1, "hooks": {"<event>": [{"command": "..."}]}}`; `beforeSubmitPrompt` pipes its JSON envelope to the hook's stdin — the Task 1 verb accepts `{"prompt":"...","conversation_id":"..."}` and raw text alike); the v1 rule frontmatter convention (`description` / `globs: "**/*"` / `alwaysApply: true`, per `rules/mnemonic.mdc`); `memAgentToolLines` (server.go:81 — the 22 agent tools, single source of the rule's mapping, same order); `memoryInstructionsFor(Agent)` (server.go:122 — the rendered body, the rune-guard target); Task 3's `planCursor` (merges the three template entries into the live hooks.json replacing the v1 memory hooks, foreign gryph/orca entries preserved; writes the rule; the string-command MCP entry). **`<BIN>` in the rule**: the repair section carries the absolute path — the installer substitutes `<BIN>` in the rule the same way it does in index.ts (an explicit add to `planCursor`'s rule-write step; the repo file keeps the placeholder).
- Produces:
  - `hooks.json` (template): `sessionStart` → `<BIN> memory session start --id cursor`; `beforeSubmitPrompt` → `<BIN> memory save-prompt --id cursor` (stdin passthrough — the verb parses the envelope); `stop` → `<BIN> memory session end --session cursor` with `loop_limit: 2` (the v1 template's anti-loop guard; the verb is idempotent). No other events (v1's policy/tool-capture hooks are non-memory — they stay where they are; the cutover touches only the memory-domain entries).
  - `mnemonic.mdc` — host shell only: frontmatter + the 22-line `mcp__mnemonic-memory__<tool>` mapping (derived from `memAgentToolLines`, same order) + the `session_id: "cursor"` pin + the repair command. **Zero v1 tokens**: the rule contains no `web_cache_`, no `fact_`, no word `serve` (T-cursor-live checks occurrences — even a "do not call" line may not name them).
  - `instructions_test.go`: `TestAgentProfileInstructionsRuneBudget` (rendered agent instructions ≤ 2,048 runes) + `TestCursorRuleGeneratedFromGoSource` (the rule's mapping block is byte-equal to the lines derived from `memAgentToolLines` in order; the pin is present; the rule carries no protocol-body signature lines — the shell-only invariant).
  - `rule-shape.test.mjs`: frontmatter checks + the pin + the mapping line + zero v1 tokens + the hooks template's exact three entries (the shape Task 3's `planCursor` installs).
- SATISFIES: T-cursor-live (install + rule side; the live smoke is Step 5), T-protocol (rune guard + generated-section diff + the exactly-once side)

- [ ] **Step 1: Write the failing tests**

`plugins/memory/cursor/test/rule-shape.test.mjs`:

```mjs
import test from "node:test"
import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import { fileURLToPath } from "node:url"
import { dirname, join } from "node:path"

const here = dirname(fileURLToPath(import.meta.url))
const rule = readFileSync(join(here, "..", "mnemonic.mdc"), "utf8")
const hooks = JSON.parse(readFileSync(join(here, "..", "hooks.json"), "utf8"))

test("rule: frontmatter is alwaysApply + globs **/*", () => {
  const front = rule.slice(0, rule.indexOf("---", 4))
  assert.ok(front.includes("alwaysApply: true"))
  assert.ok(front.includes('globs: "**/*"'))
})

test("rule: the pin, the mapping line, zero v1 tokens", () => {
  assert.ok(rule.includes('session_id: "cursor"'))
  assert.ok(rule.includes("- mcp__mnemonic-memory__mem_save"))
  for (const v1 of [/web_cache_/, /fact_/, /\bserve\b/]) {
    assert.ok(!v1.test(rule), `v1 token present: ${v1}`)
  }
})

test("hooks template: exactly the three v2 entries, <BIN>, the cursor pin", () => {
  assert.equal(hooks.version, 1)
  assert.deepEqual(Object.keys(hooks.hooks).sort(), ["beforeSubmitPrompt", "sessionStart", "stop"])
  assert.equal(hooks.hooks.sessionStart[0].command, "<BIN> memory session start --id cursor")
  assert.equal(hooks.hooks.beforeSubmitPrompt[0].command, "<BIN> memory save-prompt --id cursor")
  assert.equal(hooks.hooks.stop[0].command, "<BIN> memory session end --session cursor")
  assert.equal(hooks.hooks.stop[0].loop_limit, 2)
})
```

`mnemonic/internal/transport/mcp/instructions_test.go`:

```go
package mcp

// instructions_test.go — the agent-profile instructions rune budget
// (T-protocol) + the cursor rule's generated-section diff: the rule's tool
// mapping is authored from this package's memAgentToolLines; this test is
// the drift detector (a tool-list change fails the test until the rule is
// re-authored).

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestAgentProfileInstructionsRuneBudget: the rendered agent-profile
// instructions fit the 2,048-rune host-injection budget (T-protocol).
func TestAgentProfileInstructionsRuneBudget(t *testing.T) {
	if n := len([]rune(memoryInstructionsFor(Agent))); n > 2048 {
		t.Fatalf("agent-profile instructions are %d runes, budget is 2048 — trim the engram-derived block", n)
	}
}

// rulePath resolves the repo cursor rule from this package
// (mnemonic/internal/transport/mcp → repo root/plugins/memory/cursor).
func rulePath(t *testing.T) string {
	_, this, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	return filepath.Join(filepath.Dir(this), "..", "..", "..", "..", "plugins", "memory", "cursor", "mnemonic.mdc")
}

// TestCursorRuleGeneratedFromGoSource: the mapping block is byte-equal to
// the lines derived from memAgentToolLines (same order), the pin is
// present, and the rule carries no protocol-body signature (shell-only —
// the body is MCP-instructions-only, T-protocol exactly-once).
func TestCursorRuleGeneratedFromGoSource(t *testing.T) {
	data, err := os.ReadFile(rulePath(t))
	if err != nil {
		t.Fatalf("read cursor rule: %v", err)
	}
	text := string(data)

	want := make([]string, 0, len(memAgentToolLines))
	for _, tl := range memAgentToolLines {
		want = append(want, "- mcp__mnemonic-memory__"+tl.name)
	}
	var got []string
	inBlock := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "- mcp__mnemonic-memory__") {
			got = append(got, line)
			inBlock = true
		} else if inBlock && strings.TrimSpace(line) != "" {
			break
		}
	}
	if len(got) != len(want) {
		t.Fatalf("mapping has %d lines, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mapping line %d: got %q want %q", i, got[i], want[i])
		}
	}
	if !strings.Contains(text, `session_id: "cursor"`) {
		t.Fatal("the session pin is missing")
	}
	for _, sig := range []string{"CORE TOOLS (always available):", "DEFERRED TOOLS (admin profile only):", "DELIVERY GUARANTEE"} {
		if strings.Contains(text, sig) {
			t.Fatalf("the rule duplicates protocol body (found %q) — the body is MCP-instructions-only", sig)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd plugins/memory/cursor && node --test test/ && cd ../../../mnemonic && go test ./internal/transport/mcp/ -run 'TestAgentProfileInstructionsRuneBudget|TestCursorRuleGeneratedFromGoSource' -count=1`
Expected: FAIL — no `mnemonic.mdc` / `hooks.json`; the Go rule test fails on the missing file (the rune test passes or fails on the current length — either way it is now pinned).

- [ ] **Step 3: Write the implementation**

`plugins/memory/cursor/hooks.json`:

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [
      { "command": "<BIN> memory session start --id cursor" }
    ],
    "beforeSubmitPrompt": [
      { "command": "<BIN> memory save-prompt --id cursor" }
    ],
    "stop": [
      { "command": "<BIN> memory session end --session cursor", "loop_limit": 2 }
    ]
  }
}
```

`plugins/memory/cursor/mnemonic.mdc`:

```mdc
---
description: Mnemonic memory (v2) protocol delta for cursor.
globs: "**/*"
alwaysApply: true
---

# Mnemonic Memory (v2) — cursor

Persistent memory is available through the `mnemonic-memory` MCP server.
The full protocol (when to save, save formats, the delivery guarantee,
conflict surfacing) arrives via the server's MCP instructions — this rule
carries the cursor delta only.

## Tool mapping

Mnemonic tools are exposed as `mcp__mnemonic-memory__<tool>`:

- mcp__mnemonic-memory__mem_save
- mcp__mnemonic-memory__mem_search
- mcp__mnemonic-memory__mem_get_observation
- mcp__mnemonic-memory__mem_lookup
- mcp__mnemonic-memory__mem_search_prompts
- mcp__mnemonic-memory__mem_context
- mcp__mnemonic-memory__mem_save_prompt
- mcp__mnemonic-memory__mem_capture_passive
- mcp__mnemonic-memory__mem_update
- mcp__mnemonic-memory__mem_judge
- mcp__mnemonic-memory__mem_judge_many
- mcp__mnemonic-memory__mem_compare
- mcp__mnemonic-memory__mem_consolidate
- mcp__mnemonic-memory__mem_suggest_topic_key
- mcp__mnemonic-memory__mem_session_start
- mcp__mnemonic-memory__mem_session_end
- mcp__mnemonic-memory__mem_session_summary
- mcp__mnemonic-memory__mem_session_finalize
- mcp__mnemonic-memory__mem_current_project
- mcp__mnemonic-memory__mem_list_projects
- mcp__mnemonic-memory__mem_pin
- mcp__mnemonic-memory__mem_unpin

The retired v1 surfaces (the fact, code-index, web-cache and context tool
families, the legacy HTTP routes) are gone — do not call them.

## Session protocol

- Do not invent or override `session_id` — pass `session_id: "cursor"` on
  every memory tool call. The cursor adapter is pinned to that literal; the
  installed hooks own the session lifecycle.

## Repair

If the `mnemonic-memory` tools are missing: the MCP entry in
`~/.cursor/mcp.json` must be
`{"command":"<BIN>","args":["memory","mcp","--profile","agent"]}`;
reinstall with `skillgrid install --family memory cursor`.
```

(The rule contains zero `:7438` / `skillgrid mcp` occurrences — T-cutover's "no functional reference" check needs no judgment call on it. The mapping block above is exactly the `memAgentToolLines` order — the Go diff test pins it.)

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd plugins/memory/cursor && node --test test/ && cd ../../../mnemonic && go test ./internal/transport/mcp/ -count=1 && go build ./...`
Expected: PASS (the full mcp package suite stays green — the new test file adds two tests, no behavior change).

- [ ] **Step 5: Live cursor smoke — the T-cursor-live backstop (recorded)**

```bash
skillgrid install --family memory cursor --bin "$(which mnemonic)" --repo "$(pwd)"
# install-set assertions:
grep -n '"<BIN>"' ~/.cursor/rules/mnemonic.mdc        # → 0 hits (substituted to the absolute path)
grep -c 'mnemonic-memory' ~/.cursor/mcp.json          # → the entry present, GitLab entry untouched
grep -n 'memory session\|save-prompt' ~/.cursor/hooks.json   # → the 3 v2 entries; gryph/orca entries byte-preserved
grep -rn 'web_cache_\|fact_\|\bserve\b' ~/.cursor/rules/mnemonic.mdc   # → 0
```

Cursor GUI session (a scratch workspace): one prompt → the store holds exactly **1 session row** (id `cursor`) + **1 prompt row**; a second prompt with identical text → the dedup stores no second row; a `mem_save` with `session_id: "cursor"` lands in the scratch project. Record the row dumps + the install output as task evidence. If the cursor GUI is not reachable in this environment: record the deviation — the T3 fixture test (the cursor HOME round trip) + the shape tests already pin the install set; the row-level truth waits for the machine with cursor.

- [ ] **Step 6: Commit**

```bash
git add plugins/memory/cursor/ mnemonic/internal/transport/mcp/instructions_test.go
git commit -m "feat: add the cursor memory surface (hook template + generated rule)

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 5"
```

### Task 6: protocol — host shell + compaction recovery + session-start probe

> The fan-out's shared remainder: the host shells (mapping + `session_id` pin + repair), the FIRST ACTION REQUIRED compaction recovery (pi/omp post-compact notice; opencode/kilo compaction-hook context), and the pi/omp session-start probe (T-probe). The protocol body stays MCP-instructions-only on every host — T-protocol's "body exactly once" is verified live.

**Files:**
- Create: `plugins/memory/pi-omp/src/compaction.js` — `extractCompactedSummary` + the notice builder + the context fetch (pure; spawn injected)
- Create: `plugins/memory/pi-omp/src/shell.js` — the host shell builder + `resolveBinary` + the 3-leg probe + the scratch record + the warning (pure; spawn/fs injected)
- Create: `plugins/memory/pi-omp/test/{compaction,shell}.test.mjs`
- Modify: `plugins/memory/pi-omp/index.ts` — glue: `session_compact` → pending notice; `before_agent_start` → shell + warning + notice injection (dedup markers); `session_start` → probe (async, 10 s budget, non-blocking)
- Modify: `plugins/memory/opencode-kilo/index.ts` — the compaction + shell exports (same pure patterns) + the `ctx.session.hook("context" | "compaction", ...)` wiring
- Create: `plugins/memory/opencode-kilo/test/{compaction,shell}.test.ts`
- Modify: `skillgrid-cli/internal/install/memory.go` + `memory_test.go` — the pi/omp plan file list gains `src/compaction.js` + `src/shell.js` (4 → 6 files; the fixture stubs + the round-trip assertion update in the same diff); `Status` reads the per-project `.last-session-start.json` probe record and reports it

**Interfaces:**
- Consumes: the pi-ExtensionAPI surface (verified in `plugins/pi/mnemonic/index.ts` + the engram pi plugin): `pi.on("before_agent_start", (event) => { event.systemPromptOptions.appendSystemPrompt += … })` (the v1 dedup pattern: `!appendSystemPrompt.includes(MARKER)`); `pi.on("session_compact", async (event) => …)` (the post-compact event; the compacted summary lives in the payload — the engram multi-path extraction, `compactionEntry.summary` first). The opencode V2 hooks (engram-verified): `ctx.session.hook("context", async (request) => …)` where `request.{sessionID, system}` — `system` is the entries array, mutated in place (append only when no entry contains the marker — the per-message re-fire guard); `ctx.session.hook("compaction", async (request) => { const context: string[] = []; … })` — the returned `context` strings are appended to the system text around compaction. The CLI verbs `memory context --budget <n>` (mem.go:297 — byte-budget orientation, project-scoped from cwd) and `memory search --limit 1` (the probe's store leg). The T1/T2/T4 contracts (the minted session id, the production spawn, the drop/scratch file layout).
- Produces:
  - **pi-omp `src/compaction.js`:**
    - `extractCompactedSummary(event)` — the engram-proven multi-path extraction (`["compactionEntry","summary"]` first; unsupported shapes → undefined, never throws).
    - `buildCompactionNotice({ sessionId, host, binPath, contextAvailable, contextText })` — the notice text (≤ ~12 lines, it lands in the system prompt): header marker `Mnemonic memory — compaction recovery` + `FIRST ACTION REQUIRED: Call mem_session_summary with the content of the compacted summary above (pass session_id: "<sessionId>"). Do this before any other work.` + (when `contextAvailable` and non-empty) a `## Recent memory context` block (the `memory context --budget 2048` output, verbatim) + (when the fetch failed) the degraded line `Memory context was unavailable during compaction; call mem_context once the tools respond.` + the repair line naming `binPath` + `skillgrid install --family memory <host>`.
    - `createCompaction({ bin, session, spawn })` → `{ onCompacted(event, cwd) }`: extracts the summary (its presence only changes the notice's wording — the summary itself stays in the post-compact context, the notice instructs the model to persist it), fetches the context via the production spawn (`[bin, "memory", "context", "--budget", "2048"]` at `cwd`, 5 s), builds the notice, returns it. Never throws (a dead binary → the degraded notice).
  - **pi-omp `src/shell.js`:**
    - `buildShell({ sessionId, host, mcpFile, mcpEntry, binPath })` — the host shell (≤ ~10 lines): marker `Mnemonic memory (v2) — <host>` + the mapping line (`Memory tools are exposed as \`mcp__mnemonic-memory_mem_*\` (mem_save, mem_search, …); the full protocol arrives via the mnemonic-memory MCP server instructions.`) + the pin (`Do not invent or override \`session_id\` — pass \`session_id: "<sessionId>"\` on every memory tool call.`) + the repair line (the exact MCP entry JSON for `mcpFile` + `skillgrid install --family memory <host>`).
    - `resolveBinary(bin, { stat, pathDirs })` → `{ ok, resolved, cause }` — absolute bin → `stat` (exists + executable); bare name → the PATH scan (`pathDirs` injected; `process.env.PATH.split(":")` in production). Causes: `not-found` / `not-executable`.
    - `probe({ bin, cwd, spawn, stat, pathDirs, deadlineMs = 10000 })` → `{ resolve, init, search, at }` — three legs under one shared deadline (Global Constraints: 10 s total), all fail-open:
      1. `resolve` — `resolveBinary`;
      2. `init` — the transient MCP handshake: spawn `[bin, "memory", "mcp", "--profile", "agent"]`, write one `initialize` JSON-RPC line (`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"mnemonic-probe","version":"0"}}}`), succeed when a response line carrying `"id":1` + `result` arrives, kill the child;
      3. `search` — spawn `[bin, "memory", "search", "--limit", "1"]` at `cwd`, exit 0 = ok (project resolution from the hook cwd — the same chain the store uses).
    - `writeProbeRecord(dataDir, project, host, probe)` — `<dataDir>/<project>/.last-session-start.json` = `{ at, host, binary, resolve, init, search }`, best-effort (the `status` surface reads it).
    - `warningFor(probe, { host, binPath })` → undefined when all legs ok; otherwise the warning text: the failed leg(s) named with their cause + the exact repair command (`skillgrid install --family memory <host>` — it rewrites the MCP entry with the absolute binary path; for a store leg failure, add `mnemonic memory doctor`).
  - **opencode-kilo additions (same file):** `buildShell({ sessionId, host, mcpFile, mcpEntry, binPath })` — identical text with the **single-underscore** mapping (`mnemonic-memory_mem_*`, the opencode/kilo host-qualified form — the `<server>_<tool>` convention the rewrite normalizer already pins) and `host` = `opencode` / `kilo` (derived from the install path at setup: `process.argv[1]` contains the host's config dir — `kilo` when the plugin file lives under `.config/kilo/`, else `opencode`); `createCompaction({ bin, spawn })` → `{ onCompaction(sessionID, directory) }` — the same notice builder, `sessionId` = the harness session id (the adapter session the rewrite uses). Wiring in `setup(ctx)`: the `context` hook appends the shell only when no system entry contains the shell marker (re-fire guard); the `compaction` hook pushes the notice into the returned `context` array.
  - **installer delta:** the pi/omp plan file list = `{ index.ts, src/session.js, src/capture.js, src/compaction.js, src/shell.js, package.json }`; `Status` gains the probe-record leg (per installed project dir: the latest `.last-session-start.json` → a `status` line naming the failed leg + cause, or `session-start probe ok` when all legs passed).
- SATISFIES: T-compaction, T-probe, T-protocol (the exactly-once live side: the shell + notice never duplicate the MCP-instructions body), T-fail-open (probe + compaction legs)

- [ ] **Step 1: Write the failing tests**

`plugins/memory/pi-omp/test/compaction.test.mjs`:

```mjs
import test from "node:test"
import assert from "node:assert/strict"
import { extractCompactedSummary, buildCompactionNotice, createCompaction } from "../src/compaction.js"

test("extract: the current pi shape wins; unsupported shapes are undefined", () => {
  assert.equal(extractCompactedSummary({ compactionEntry: { summary: "the summary" } }), "the summary")
  // a conflicting legacy field must not beat the authoritative path
  assert.equal(extractCompactedSummary({ compactionEntry: { summary: "a" }, summary: "b" }), "a")
  assert.equal(extractCompactedSummary({ compacted_summary: "legacy" }), "legacy")
  assert.equal(extractCompactedSummary({}), undefined)
  assert.equal(extractCompactedSummary(null), undefined)
})

test("notice: FIRST ACTION REQUIRED + the session pin, degraded line when the fetch failed", () => {
  const n = buildCompactionNotice({ sessionId: "mem-x-y", host: "omp", binPath: "/abs/mnemonic", contextAvailable: false, contextText: "" })
  assert.ok(n.includes("FIRST ACTION REQUIRED"))
  assert.ok(n.includes("mem_session_summary"))
  assert.ok(n.includes('session_id: "mem-x-y"'))
  assert.ok(n.includes("skillgrid install --family memory omp"))
  assert.ok(n.includes("unavailable"))

  const ok = buildCompactionNotice({ sessionId: "s", host: "pi", binPath: "/b", contextAvailable: true, contextText: "recent obs" })
  assert.ok(ok.includes("## Recent memory context"))
  assert.ok(ok.includes("recent obs"))
  assert.ok(!ok.includes("unavailable"))
})

test("onCompacted: fetch failure degrades, never throws; the spawn is the context verb", async () => {
  const calls = []
  const c = createCompaction({ bin: "mnemonic", host: "omp", session: { sessionId: () => "s1" }, spawn: async (args, _stdin, cwd) => { calls.push({ args, cwd }); throw new Error("exit 1") } })
  const notice = await c.onCompacted({ compactionEntry: { summary: "sum" } }, "/repo")
  assert.ok(notice.includes("FIRST ACTION REQUIRED"))
  assert.ok(notice.includes("unavailable"))
  assert.deepEqual(calls[0].args, ["mnemonic", "memory", "context", "--budget", "2048"])
  assert.equal(calls[0].cwd, "/repo")
})
```

`plugins/memory/pi-omp/test/shell.test.mjs`:

```mjs
import test from "node:test"
import assert from "node:assert/strict"
import { mkdtemp, readFile, writeFile, mkdir, chmod } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { buildShell, resolveBinary, probe, warningFor } from "../src/shell.js"

test("shell: marker, the double-underscore mapping, the pin, the repair line", () => {
  const s = buildShell({ sessionId: "mem-1-a", host: "omp", mcpFile: "~/.omp/agent/mcp.json", mcpEntry: '{"type":"stdio","command":"/abs/mnemonic","args":["memory","mcp","--profile","agent"]}', binPath: "/abs/mnemonic" })
  assert.ok(s.includes("Mnemonic memory (v2) — omp"))
  assert.ok(s.includes("mcp__mnemonic-memory_mem_save"))
  assert.ok(s.includes('session_id: "mem-1-a"'))
  assert.ok(s.includes("skillgrid install --family memory omp"))
})

test("resolveBinary: absolute executable passes; bare name scans PATH; causes are named", () => {
  const dir = mkdtempSyncish() // test helper: mkdtemp + a chmodded stub file
  assert.equal(resolveBinary("/no/such/bin", { stat: () => null, pathDirs: [] }).cause, "not-found")
  const r = resolveBinary("mnemonic", { stat: (p) => (p === "/bin/mnemonic" ? { executable: true } : null), pathDirs: ["/bin"] })
  assert.ok(r.ok)
  assert.equal(r.resolved, "/bin/mnemonic")
})

test("probe: all legs ok; a dead binary names the cause; the deadline is shared", async () => {
  const ok = await probe({ bin: "/abs/mnemonic", cwd: "/repo", spawn: async () => {}, stat: () => ({ executable: true }), pathDirs: [], deadlineMs: 5000 })
  assert.ok(ok.resolve.ok && ok.init.ok && ok.search.ok)

  const dead = await probe({ bin: "mnemonic", cwd: "/repo", spawn: async () => { throw new Error("ENOENT") }, stat: () => null, pathDirs: ["/bin"], deadlineMs: 5000 })
  assert.equal(dead.resolve.cause, "not-found")
  const w = warningFor(dead, { host: "omp", binPath: "mnemonic" })
  assert.ok(w.includes("not-found"))
  assert.ok(w.includes("skillgrid install --family memory omp"))
  assert.equal(warningFor(ok, { host: "omp", binPath: "/abs/mnemonic" }), undefined)
})
```

`plugins/memory/opencode-kilo/test/{compaction,shell}.test.ts` mirror the pi-omp suites over the same exports, with the single-underscore mapping assertion (`mnemonic-memory_mem_save` present, `mcp__mnemonic-memory__mem_save` absent) and the host derivation (`kilo` when the plugin path contains `.config/kilo`).

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd plugins/memory/pi-omp && node --test test/ && cd ../opencode-kilo && node --test test/`
Expected: FAIL — `Cannot find module '../src/compaction.js'` / `'../src/shell.js'`; the opencode exports missing.

- [ ] **Step 3: Write the implementation**

`plugins/memory/pi-omp/src/compaction.js`:

```js
// compaction.js — post-compact recovery (engram-proven pattern): the
// compacted summary stays in the post-compact context; the notice instructs
// the model to persist it via mem_session_summary (the adapter's session
// pin) and carries a fresh `memory context` block when the store answers.
// Pure: spawn is injected. Fail-open: a dead binary degrades the notice,
// it never throws.
import { join } from "node:path"

const SUMMARY_FIELD_PATHS = [
  // Current Pi session_compact payloads put the authoritative summary here.
  ["compactionEntry", "summary"],
  ["summary"], ["compactedSummary"], ["compacted_summary"], ["compactSummary"],
  ["compact_summary"], ["content"], ["text"], ["message"],
  ["compacted", "summary"], ["compacted", "content"], ["compaction", "summary"],
  ["compaction", "content"], ["output", "summary"], ["output", "content"],
  ["payload", "summary"], ["payload", "content"], ["data", "summary"], ["data", "content"],
]

export function extractCompactedSummary(event) {
  if (!event || typeof event !== "object") return undefined
  for (const path of SUMMARY_FIELD_PATHS) {
    let cur = event
    for (const key of path) {
      if (!cur || typeof cur !== "object" || !(key in cur)) { cur = undefined; break }
      cur = cur[key]
    }
    if (typeof cur === "string" && cur.trim()) return cur.trim()
  }
  return undefined
}

export function buildCompactionNotice({ sessionId, host, binPath, contextAvailable, contextText }) {
  const lines = [
    "# Mnemonic memory — compaction recovery",
    `FIRST ACTION REQUIRED: Call mem_session_summary with the content of the compacted summary above (pass session_id: "${sessionId}"). Do this before any other work.`,
  ]
  if (contextAvailable && contextText && contextText.trim()) {
    lines.push("", "## Recent memory context", contextText.trim())
  } else {
    lines.push("Memory context was unavailable during compaction; call mem_context once the tools respond.")
  }
  lines.push(`Repair if the tools do not respond: the mnemonic binary is \`${binPath}\`; reinstall with \`skillgrid install --family memory ${host}\`.`)
  return lines.join("\n")
}

export function createCompaction({ bin, session, spawn, host }) {
  return {
    async onCompacted(event, cwd) {
      const sid = session.sessionId() || "unknown"
      let text = ""
      try {
        const out = await spawn([bin, "memory", "context", "--budget", "2048"], "", cwd)
        text = typeof out === "string" ? out : ""
      } catch { text = "" }
      return buildCompactionNotice({ sessionId: sid, host, binPath: bin, contextAvailable: Boolean(text.trim()), contextText: text })
    },
  }
}
```

(The glue passes `host` = `omp`/`pi` per install location. The `spawn` here is a context-fetching variant that RESOLVES with stdout text — the glue wraps the production spawn: `spawnContext(args, cwd) => prodSpawnStdout(args, "", cwd)` with the same 5 s hard timeout. Both are fail-open at the same seam.)

`plugins/memory/pi-omp/src/shell.js`:

```js
// shell.js — the host shell (mapping + session pin + repair) + the
// session-start probe. Pure: spawn/stat/pathDirs are injected. The probe is
// three legs under one shared 10 s deadline (Global Constraints); every leg
// is fail-open and the record lands in
// ~/.skillgrid/mnemonic/<project>/.last-session-start.json (read by status).
import { join } from "node:path"

export function buildShell({ sessionId, host, mcpFile, mcpEntry, binPath }) {
  return [
    `# Mnemonic memory (v2) — ${host}`,
    "Memory tools are exposed as `mcp__mnemonic-memory_mem_*` (mem_save, mem_search, …); the full protocol (when to save, formats, delivery guarantee) arrives via the mnemonic-memory MCP server instructions.",
    `Do not invent or override \`session_id\` — pass \`session_id: "${sessionId}"\` on every memory tool call.`,
    `Repair: the MCP entry in \`${mcpFile}\` must be \`${mcpEntry}\`; reinstall with \`skillgrid install --family memory ${host}\` (binary: \`${binPath}\`).`,
  ].join("\n")
}

export function resolveBinary(bin, { stat, pathDirs }) {
  if (bin.includes("/")) {
    const s = stat(bin)
    if (s && s.executable) return { ok: true, resolved: bin }
    return { ok: false, resolved: bin, cause: s ? "not-executable" : "not-found" }
  }
  for (const dir of pathDirs) {
    const p = join(dir, bin)
    const s = stat(p)
    if (s && s.executable) return { ok: true, resolved: p }
    if (s) return { ok: false, resolved: p, cause: "not-executable" }
  }
  return { ok: false, resolved: bin, cause: "not-found" }
}

const INITIALIZE_LINE = JSON.stringify({
  jsonrpc: "2.0", id: 1, method: "initialize",
  params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "mnemonic-probe", version: "0" } },
})

export async function probe({ bin, cwd, spawn, stat, pathDirs, deadlineMs = 10000 }) {
  const deadline = Date.now() + deadlineMs
  const left = () => Math.max(0, deadline - Date.now())
  const resolve = resolveBinary(bin, { stat, pathDirs })
  let init = { ok: false, cause: "skipped" }
  let search = { ok: false, cause: "skipped" }
  if (resolve.ok && left() > 0) {
    try {
      // transient MCP handshake: one initialize request, one response, kill.
      await withDeadline(spawn([bin, "memory", "mcp", "--profile", "agent"], INITIALIZE_LINE, cwd), left())
      init = { ok: true }
    } catch (e) { init = { ok: false, cause: e.cause || e.message } }
  }
  if (left() > 0) {
    try {
      await withDeadline(spawn([bin, "memory", "search", "--limit", "1"], "", cwd), left())
      search = { ok: true }
    } catch (e) { search = { ok: false, cause: e.cause || e.message } }
  }
  return { at: new Date().toISOString(), binary: resolve.resolved, resolve, init, search }
}

function withDeadline(promise, ms) {
  return Promise.race([promise, new Promise((_r, reject) => setTimeout(() => reject(Object.assign(new Error("timeout"), { cause: "timeout" })), ms))])
}

export function writeProbeRecord(dataDir, project, host, record, bin) {
  return import("node:fs/promises").then(async ({ mkdir, writeFile }) => {
    const dir = join(dataDir(), project)
    await mkdir(dir, { recursive: true })
    await writeFile(join(dir, ".last-session-start.json"), JSON.stringify({ ...record, host, binary: bin }))
  })
}

export function warningFor(probe, { host, binPath }) {
  if (probe.resolve.ok && probe.init.ok && probe.search.ok) return undefined
  const legs = []
  if (!probe.resolve.ok) legs.push(`binary not resolvable (${probe.resolve.cause})`)
  if (!probe.init.ok) legs.push(`MCP initialize failed (${probe.init.cause})`)
  if (!probe.search.ok) legs.push(`store search failed (${probe.search.cause})`)
  let w = `Mnemonic memory warning: ${legs.join("; ")}. Memory capture may be paused for this session.`
  if (!probe.resolve.ok) w += ` Repair: run \`skillgrid install --family memory ${host}\` (it rewrites the MCP entry with the absolute binary path) or put the mnemonic binary on PATH.`
  else w += ` Repair: run \`mnemonic memory doctor\`, then \`skillgrid install --family memory ${host}\`. Binary: \`${binPath}\`.`
  return w
}
```

`plugins/memory/pi-omp/index.ts` glue delta (the existing v0 handlers stay; the markers dedup):

```ts
const SHELL_MARKER = "# Mnemonic memory (v2) — "
const WARNING_MARKER = "Mnemonic memory warning:"
const NOTICE_MARKER = "# Mnemonic memory — compaction recovery"

// per factory closure (per conversation):
let pendingWarning: string | null = null
let pendingNotice: string | null = null

pi.on("session_start", async (_event, ctx) => {
  const sid = await session.start(ctx.cwd)
  // probe: async, 10 s shared deadline, never blocks the session start
  void (async () => {
    try {
      const record = await probe({ bin: BIN, cwd: ctx.cwd, spawn: probeSpawn, stat: fsStat, pathDirs: process.env.PATH.split(pathSep), deadlineMs: 10000 })
      await writeProbeRecord(defaultDataDir, resolveProject(ctx.cwd, { gitToplevel: gitToplevelSync }), hostOf(ctx), record, BIN)
      const w = warningFor(record, { host: hostOf(ctx), binPath: BIN })
      if (w) pendingWarning = w
    } catch { /* fail-open */ }
  })()
})

pi.on("before_agent_start", (event) => {
  try {
    const opts = event?.systemPromptOptions
    if (!opts) return
    const sid = session.sessionId()
    if (sid) {
      const shell = buildShell({ sessionId: sid, host: hostOf(ctx), mcpFile, mcpEntry, binPath: BIN })
      if (!opts.appendSystemPrompt?.includes(shell)) opts.appendSystemPrompt = (opts.appendSystemPrompt ?? "") + (opts.appendSystemPrompt ? "\n\n" : "") + shell
    }
    for (const note of [pendingWarning, pendingNotice]) {
      if (note && !opts.appendSystemPrompt?.includes(note.split("\n")[0])) {
        opts.appendSystemPrompt = (opts.appendSystemPrompt ?? "") + (opts.appendSystemPrompt ? "\n\n" : "") + note
      }
    }
    pendingWarning = null
    pendingNotice = null
  } catch { /* fail-open */ }
})

pi.on("session_compact", async (event) => {
  try { pendingNotice = await compaction.onCompacted(event, cwd()) } catch { /* fail-open */ }
})
```

(`hostOf` = `omp` vs `pi` from the extension's install path; `mcpFile`/`mcpEntry` are the host's dialect row — the same values the installer bakes; `probeSpawn` = the production spawn with a stdout-capturing variant for `init`/`search`/`context`. The shell dedups on the full shell text (it embeds the conversation id — a re-mint in a new conversation re-injects once; `includes` on the marker line would miss that, the full text is the correct key).)

The opencode-kilo `index.ts` delta: the `buildShell`/`createCompaction` exports (single-underscore mapping line; `hostFromPath()` = `kilo` when `process.argv[1]` contains `.config/kilo`, else `opencode`) + the setup wiring:

```ts
const SHELL_MARKER = "# Mnemonic memory (v2) — "
await ctx.session.hook("context", (request: any) => {
  try {
    const sid = request.sessionID
    if (!sid) return
    const shell = buildShell({ sessionId: sid, host: hostFromPath(), mcpFile, mcpEntry, binPath: BIN })
    const sys: any[] = request.system ?? []
    if (!sys.some((e: any) => String(e.text ?? e).includes(SHELL_MARKER))) {
      sys.push({ text: shell })
    }
  } catch { /* fail-open */ }
})
await ctx.session.hook("compaction", async (request: any) => {
  try {
    const context: string[] = []
    const notice = await compaction.onCompaction(request.sessionID, directory)
    context.push(notice)
    return { context }
  } catch { return { context: [] } }
})
```

The installer delta (memory.go + memory_test.go, one diff): the pi/omp `rel` list gains `src/compaction.js` + `src/shell.js`; the `fixtureRepo` stub creates both; the round-trip assertion `len(m.Files) != 4` → `!= 6`; `Status` gains the probe-record leg (a per-project-dir scan for `.last-session-start.json`; a failed leg → the status line `omp: session-start probe degraded (binary not resolvable (not-found)) — last attempt <at>`; all ok → `session-start probe ok`).

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd plugins/memory/pi-omp && node --test test/ && cd ../opencode-kilo && node --test test/ && cd ../../../skillgrid-cli && go test ./internal/install/ -count=1 && go build ./...`
Expected: PASS (all four suites + the installer round trips at 6 files).

- [ ] **Step 5: Live verification (recorded)**

1. `skillgrid install --family memory omp pi opencode kilo --bin "$(which mnemonic)" --repo "$(pwd)"` (the extended file set).
2. **T-probe:** `HOME`-scoped throwaway: set a fixture HOME whose PATH lacks `mnemonic`, start an omp session → the context carries the warning naming `not-found` + the exact repair command; `~/.skillgrid/mnemonic/<project>/.last-session-start.json` exists; `skillgrid status --family memory omp` surfaces the degraded leg. Restore PATH → the next session's record shows all legs ok.
3. **T-compaction:** in a live omp session, fill the context until compaction fires (or the harness's manual compaction command — record which) → the post-compact context contains the FIRST ACTION REQUIRED notice with the conversation's session pin; the model's first action after compaction is the `mem_session_summary` call (observed in the transcript). Repeat on live opencode (the compaction hook path) — the notice arrives via the compaction `context` append.
4. **T-protocol exactly-once:** in one live session per host, dump the system context and count the protocol body's signature lines (`DEFERRED TOOLS (admin profile only):`) — exactly 1 (MCP instructions only); the shell/notice blocks present without duplicating the body.
5. Record all evidence (context dumps, the probe record JSON, the status output) in the task evidence.

- [ ] **Step 6: Commit**

```bash
git add plugins/memory/pi-omp/ plugins/memory/opencode-kilo/ skillgrid-cli/internal/install/
git commit -m "feat: add the host shells, compaction recovery, and the session-start probe

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 6"
```

### Task 7: cross-host live integration + the status surface (the pre-cutover checkpoint)

> No new feature surface — the integration gate before the one-way door (Task 8): all five hosts installed through the installer against the live configs, and `skillgrid status --family memory all` verified live (installed/drift, drops, probe records, the engram coexistence, the stale-v1 pre-flight list the cutover consumes). Gaps found here are fixed in small named diffs, not discovered at the door.

**Files:**
- (runtime, live) the five installed sets + `~/.skillgrid/mnemonic/memory-install.json`
- Modify (only if a gap is found): `skillgrid-cli/internal/install/memory.go` + `memory_test.go` — the fix is named in the evidence when made (candidates: the stale-v1 detection list, the status report shape, the engram warning text)

**Interfaces:**
- Consumes: the installed sets from Tasks 2–6 (the pi/omp extension dirs, the opencode/kilo `mnemonic-memory.ts`, the cursor rule + hooks + MCP entries); the live v1 state (the pre-cutover coexistence: the v1 `mnemonic`/`skillgrid` MCP keys still present where they exist live, the v1 pi extension residues in `~/.skill-pi/agent/extensions/mnemonic/` — the v2 install overwrites the shared paths, the v1-only files (`api.js`, `mnemonic-facts.ts`, `mnemonic-codeindex.ts`, `ctx-sandbox.ts`, `skillgrid-sdd-events.ts`, `skillgrid-events.ts`, `skillgrid-compaction.ts`, `mnemonic-memory.ts`) sit dead in the dir until Task 8 deletes them); the live kilo `~/.config/kilo/plugins/engram.ts` (T-engram's target).
- Produces: the recorded evidence (the install tree, the `status` output, the probe record, the engram warning) + the stale-v1 pre-flight list (per host: which stale shapes exist live — Task 8's cutover consumes this list, not a re-derivation) + gap fixes (each with its own RED test in `memory_test.go`).

- [ ] **Step 1: Build + install all five hosts**

```bash
cd mnemonic && go build -o /tmp/mnv2/mnemonic ./cmd/mnemonic && cd ../skillgrid-cli && go build -o /tmp/mnv2/skillgrid ./cmd/skillgrid
/tmp/mnv2/skillgrid install --family memory all --bin /tmp/mnv2/mnemonic --repo "$(git rev-parse --show-toplevel)"
```

Record the resulting tree (`find ~/.skill-pi/agent/extensions/mnemonic ~/.omp/agent/extensions/mnemonic ~/.config/opencode/plugins/mnemonic-memory.ts ~/.config/kilo/plugins/mnemonic-memory.ts ~/.cursor/rules/mnemonic.mdc` + the four MCP config greps). Assert: the five tracked sets exist with the substituted absolute binary; the kilo/opencode files are byte-identical (`cmp`); foreign entries (gryph/orca/GitLab) untouched.

- [ ] **Step 2: The status surface, live**

```bash
/tmp/mnv2/skillgrid status --family memory all
```

Assert and record:
- each of the five hosts: installed (tracked set verified against the manifest — no drift)
- the kilo engram coexistence: `~/.config/kilo/plugins/engram.ts` listed (T-engram live — the install also printed the warning; record both)
- the stale-v1 pre-flight list: per host, the stale shapes present live (expected from the live state: opencode `mcp.mnemonic` array entry + the v1 `plugin` array entries — note the installer already dropped the five retired ones at install, the cutover list is what remains; kilo `mcp.mnemonic` with the `skillgrid` binary; omp/pi/cursor: whatever the live configs carry — the list is the live truth, recorded verbatim)
- drops + probe records: "none yet" is the expected pre-session state (no fabricated rows)

- [ ] **Step 3: The probe record, live (T-probe surface)**

Start one fresh omp session in a scratch dir (the recorded step: launch omp there; the extension's `session_start` probe runs). Then:

```bash
cat ~/.skillgrid/mnemonic/<scratch-project>/.last-session-start.json
/tmp/mnv2/skillgrid status --family memory omp
```

Assert: the record exists with all three legs (resolve/init/search) and the status reports `session-start probe ok` (or the failed leg named — a failure here is a real finding, fixed in Step 4). Fallback (recorded deviation if no live omp session is possible from this environment): the fixture-HOME probe test from Task 3 + a manual `probe` invocation over a fixture HOME proves the leg logic; the live record is marked pending-machine.

- [ ] **Step 4: Gap fixes (only if found)**

Each gap: RED test in `memory_test.go` first, minimal fix, the affected host's round trip re-green. No gap → note "no gaps" in the evidence.

- [ ] **Step 5: Commit (only if Step 4 changed code)**

```bash
git add -A skillgrid-cli/internal/install/
git commit -m "fix: <the gap> (live integration findings)

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 7"
```

Evidence (the install tree, the status output, the probe record JSON, the stale-v1 list, the engram warning) is recorded in this task's evidence block either way — it is Task 8's pre-flight input.

### Task 8: ⚠ one-way: the v1 cutover (repo retirement + live config rewrite)

> D-1 — user-approved at the brainstorming gate; this task stops for a final confirm before the live rewrites run (Step 4). Repo changes revert via git; live configs via the installer backups (`~/.skillgrid/mnemonic/backups/`). After this task, every host's loaded plugin set is the v2 set — there is no v1 fallback path.

**Files:**
- Delete (repo, the briefing R8 inventory): `plugins/pi/mnemonic/` (whole v1 package — `api.js`, `mnemonic-facts.ts`, `mnemonic-codeindex.ts`, `mnemonic-memory.ts`, `ctx-sandbox.ts`, `skillgrid-events.ts`, `skillgrid-compaction.ts`, `skillgrid-sdd-events.ts`; the `package.json` does not move — `plugins/memory/pi-omp/package.json` is the new package's own); `plugins/opencode/{mnemonic-memory.ts, skillgrid-events.ts, skillgrid-compaction.ts, mnemonic-codeindex.ts, skillgrid-squad.ts, skillgrid-context.ts, tools/, src/memory-prefetch.ts}`; `plugins/kilo/` (the v1 install-time copy source)
- Delete (conditional, verified at execution): `plugins/opencode/{skillgrid-v2sse.ts, skillgrid-logo.js, skillgrid-logo.tsx, skillgrid-logo/}` — grep the surviving `plugins/opencode/` files for imports; delete when unreferenced, keep with a one-line note in the commit message when referenced
- Modify: `hooks/hooks-cursor.json` — the memory-domain entries (the ones referencing `:7438` memory routes / v1 memory verbs: `sessionStart`/`sessionEnd`/`stop` session scripts, `postToolUse` tool-capture) replaced by the v2 shapes from `plugins/memory/cursor/hooks.json` (`<BIN>` placeholder, same three entries); `sessionEnd` has no v2 equivalent (v2 uses `stop`) and is dropped; non-memory entries (ticketing, gates, precommit, checkpoint) byte-untouched
- Modify: `.cursor-plugin/plugin.json` — `description` rewritten off the `skillgrid mcp` reference (v2: "memory protocol rule + session hooks, the skill catalog, dispatch-role subagents, and the mnemonic-memory MCP server")
- Modify: `.agents/skills/mnemonic/SKILL.md` — the v1 daemon / `:7438` / `skillgrid mcp` / `web_cache_*` sections rewritten to the v2 surface (structure pinned in Step 2)
- Modify: `skillgrid-cli/internal/setup/setup.go` — the kilo AGENTS.md block becomes single-sourced: the setup module calls the memory family's block writer (new exported `install.KiloMnemonicBlock(bin string) string` in `memory.go` — the same text + marker pair the kilo plan writes), so the bootstrap never resurrects v1 text
- (runtime, live) the stale-v1 MCP keys deleted (Task 3's `cutStale` on re-install), the v1 pi extension residues deleted from `~/.skill-pi/agent/extensions/mnemonic/`

**Interfaces:**
- Consumes: Task 7's evidence (the stale-v1 pre-flight list — the cutover acts on that list, not a re-derivation); the installer's `cutStale` (the v1 `mnemonic` key, any dialect — omp/opencode/kilo all key `mnemonic`, including kilo's `skillgrid`-binary variant); the v2 block text (memory.go, single source); the repo deletion inventory (File Structure § Deleted).
- Produces: the retired v1 surfaces (no `fact_*`/`code_*`-v1/`web_cache_*`/`ctx_*`/squad on any host — the v2 code family is `mnemonic-code`'s `code_*` tools, a different server); the live configs carrying exactly the v2 tracked sets; the grep gates (below) green.
- SATISFIES: T-cutover (live side), D-1, the threat-matrix shared-convention-drift gate

- [ ] **Step 1: Pre-flight**

Confirm Task 7's evidence is in hand (the stale-v1 list, the backups exist under `~/.skillgrid/mnemonic/backups/<host>/`). `git status` clean in the repo (the spec zone is committed).

- [ ] **Step 2: The repo cutover (git-revertible)**

1. `git rm -r plugins/pi/mnemonic plugins/kilo` + the eight `plugins/opencode/` paths (+ the conditional logo/v2sse files after the import grep).
2. `hooks/hooks-cursor.json`: classify every entry — memory-domain (references `:7438` memory routes or v1 memory verbs) → the v2 shapes from `plugins/memory/cursor/hooks.json` (byte-identical entry text); the rest byte-untouched. The memory entries carry `<BIN>` (the installer substitutes on install — same contract as the template).
3. `.cursor-plugin/plugin.json`: the description line (above).
4. `.agents/skills/mnemonic/SKILL.md`: rewrite to the v2 surface. Pinned structure (the body prose follows the agent-style rules; the constraints are the gate):
   - `# Mnemonic` — Overview: v2 = the single `mnemonic` binary, three MCP servers (`mnemonic-memory` agent profile, `mnemonic-code`, `mnemonic-projectwiki`) over stdio; no daemon, no `:7438`; local SQLite (or Postgres) per project.
   - `## When to Use` — preserved (recall before work, save during work, close before done).
   - `## Config` — the data dir (`~/.skillgrid/mnemonic/<project>.sqlite`), `MNEMONIC_PROJECT`, no daemon section.
   - `## Tools` — the 22 agent tools of `mnemonic-memory` (host-qualified `mcp__mnemonic-memory_mem_*` — the cursor mapping; the other hosts' forms are the same tools) + the `mnemonic-code` code tools (`code_files`, `code_index`, `code_read`, `code_search`, `code_status` — qualified `mcp__mnemonic-code_*`) + the projectwiki tools; the session protocol ("do not invent or override `session_id` — the host adapter owns it; cursor passes the literal `cursor`").
   - `## Usage` / `## What You Must Do When Invoked` — the steps preserved with v2 verbs: recall = `mem_context` (+ `mem_search` / `mem_get_observation`); save = `mem_save` (the type vocabulary kept); **Step 5 code retrieval** → the `mnemonic-code` tools (the Orientation Ladder prose kept, the tool names v2); **Step 6 web research** → no web cache in v2: significant findings saved via `mem_save` (`type: discovery`), no route to a cache; **Step 7 close** → `mem_session_summary` then `mem_session_end` (or `mem_session_finalize`).
   - `## Subcommands (CLI reference)` — the v2 list (`save|search|get|lookup|context|review|timeline|pin|unpin|delete|session|stats|doctor|mcp|save-prompt`).
   - `## Common Rationalizations` / `## Red Flags` / `## Verification` — preserved, v2 tokens only.
   - **Gate constraint:** zero occurrences of `:7438`, `skillgrid mcp`, `web_cache_`, `fact_`, word `serve`, `daemon`; every `code_index` occurrence is `mnemonic-code`-qualified (the threat-matrix gate's `code_index` token targets the v1 all-in-one tool; the v2 `mnemonic-code` tool is the same name on the surviving server — the Step 3 grep pins the qualification).
5. `setup.go`: `install.KiloMnemonicBlock(bin)` exported from `memory.go` (the kilo plan's block text, one function); the setup module's kilo writer calls it (same marker pair, `setup.go:34-35`).

- [ ] **Step 3: Verify the repo state**

```bash
cd skillgrid-cli && go test ./internal/install/ ./internal/setup/ -count=1 && go build ./...
cd ../mnemonic && go build ./...
# grep gates (threat matrix): no live v1 references in the convention surface
grep -rn ':7438\|skillgrid mcp\|web_cache_\|fact_' .agents/skills/ hooks/hooks-cursor.json .cursor-plugin/plugin.json | grep -v 'change-archive\|ADR' || true   # → empty
grep -rn '\bserve\b\|daemon' .agents/skills/mnemonic/SKILL.md || true                        # → empty
grep -n 'code_index' .agents/skills/mnemonic/SKILL.md                                          # → every hit mnemonic-code-qualified
# a new setup-drift test in the setup package: the setup kilo block == install.KiloMnemonicBlock output (RED first)
```

Commit the repo cutover (code zone):

```bash
git add -A
git commit -m "refactor: cut over the v1 mnemonic surfaces to the v2 plugin set (one-way D-1)

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 8"
```

- [ ] **Step 4: ⚠ one-way stop — final confirm before the live rewrites**

Present the live change list (from Task 7's pre-flight: exactly which stale keys/files go, per host, with the backup locations) and wait for the user's explicit go. The repo is already committed and revertible; the live rewrites are the door.

- [ ] **Step 5: The live rewrites**

```bash
# the stale MCP keys: Task 3's cutStale fires on re-install (idempotent, backups refreshed)
skillgrid install --family memory omp pi opencode kilo cursor --bin "$(which mnemonic)" --repo "$(git rev-parse --show-toplevel)"
# the v1 pi extension residues (not in the v2 tracked set — the installer never deletes foreign files):
git -C "$HOME/.skill-pi/agent/extensions/mnemonic" status 2>/dev/null || true
for f in api.js mnemonic-facts.ts mnemonic-codeindex.ts mnemonic-memory.ts ctx-sandbox.ts skillgrid-sdd-events.ts skillgrid-events.ts skillgrid-compaction.ts; do
  rm -f "$HOME/.skill-pi/agent/extensions/mnemonic/$f"
done
# cursor: the v1 `mnemonic` MCP key — present live per Task 7's list, else recorded n/a
```

- [ ] **Step 6: Post-cutover verification (recorded)**

- grep gate over the live installed set: `grep -rn ':7438\|skillgrid mcp\|web_cache_\|fact_' ~/.skill-pi/agent/extensions/mnemonic ~/.omp/agent/extensions/mnemonic ~/.config/opencode/plugins/mnemonic-memory.ts ~/.config/kilo/plugins/mnemonic-memory.ts ~/.cursor/rules/mnemonic.mdc` → empty; the four MCP configs carry exactly the `mnemonic-memory` entries (no `mnemonic` key); the opencode/kilo `plugin` arrays carry no retired entries.
- one session per host boots on the v2 set only: opencode/kilo (plugin load, no v1 plugin in the load list), omp/pi (session start, the probe record ok), cursor (the MCP server connects, the rule loads). Record the evidence.
- `skillgrid status --family memory all` → no stale-v1 lines remain.

- [ ] **Step 7: Commit (only if Step 5/6 changed repo files — e.g., the conditional logo note)**

```bash
git add -A
git commit -m "docs: record the cutover deviations (logo/v2sse disposition)

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 8"
```

### Task 9: the v1→v2 store import (one-shot script)

> bash + sqlite3 CLI only (Global Constraints — zero new dependencies). Content-keyed idempotency via a `provenance` marker (`{"imported_from":"mnemonic-v1","v1_id":<id>}`) — a re-run adds 0 rows, and the marker survives later edits to the imported rows.

**Files:**
- Create: `scripts/mnemonic-v1-import.sh` — `--from <v1.sqlite> --to <v2.sqlite> [--dry-run]`, counts, exit 0/1/2
- Create: `scripts/fixtures/v1-import/v1.sql` — the fixture v1 store DDL+seed (minimal v1 schema: `observations_core` + one satellite `observation_state` + `sessions`; the import LEFT-JOINs all five v1 satellites, the fixture exercises one — the real store has the rest)
- Create: `scripts/fixtures/v1-import/test.sh` — the fixture test (build v1 store → init v2 store via the repo binary → import → assert rows + search → re-run adds 0 → `--dry-run` writes nothing)

**Interfaces:**
- Consumes: the v1 schema (verified against the live v1 store `~/.skillgrid/mnemonic/001.sqlite`): `observations_core(id, session_id, type, title, content, project, scope, topic_key, normalized_hash, revision_count, created_at, updated_at, deleted_at, knowledge_class)`; satellites `observation_state(pinned, review_after, …)`, `observation_provenance(tool_name, …)`, `observation_stats(importance_score, duplicate_count, last_seen_at, …)` — all LEFT-JOINed by `observation_id`; v1 `sessions(id, project, directory, started_at, ended_at, summary, title, …)`. The v2 schema (`migrations/001_schema.sql`): `sessions(id, project NOT NULL, directory NOT NULL, started_at, ended_at, summary, title, ownership_mode DEFAULT 'shared', …)`; `observations(id, session_id NOT NULL REFERENCES sessions(id), type NOT NULL, title NOT NULL, content NOT NULL, tool_name, project, scope NOT NULL DEFAULT 'project', topic_key, normalized_hash, revision_count, duplicate_count, last_seen_at, pinned, review_after, importance DEFAULT 0.5, provenance TEXT DEFAULT '{}', created_at, updated_at, deleted_at)` — **the FTS triggers maintain `observations_fts` on direct INSERT**, so the script never touches FTS.
- Produces:
  - the import (sessions first — same ids, so the observations FK holds; then the observations with the marker):
    ```sql
    INSERT OR IGNORE INTO sessions (id, project, directory, started_at, ended_at, summary, title)
    SELECT id, project, directory, started_at, ended_at, summary, title
    FROM v1.sessions;

    INSERT INTO observations
      (session_id, type, title, content, tool_name, project, scope, topic_key,
       normalized_hash, revision_count, pinned, review_after, duplicate_count,
       last_seen_at, importance, provenance, created_at, updated_at, deleted_at)
    SELECT c.session_id, c.type, c.title, c.content, pr.tool_name, c.project,
           COALESCE(c.scope, 'project'), c.topic_key, c.normalized_hash, c.revision_count,
           COALESCE(st.pinned, 0), st.review_after, COALESCE(sa.duplicate_count, 0),
           sa.last_seen_at, COALESCE(sa.importance_score, 0.5),
           json_object('imported_from', 'mnemonic-v1', 'v1_id', c.id),
           c.created_at, c.updated_at, c.deleted_at
    FROM v1.observations_core AS c
    LEFT JOIN v1.observation_state AS st ON st.observation_id = c.id
    LEFT JOIN v1.observation_provenance AS pr ON pr.observation_id = c.id
    LEFT JOIN v1.observation_stats AS sa ON sa.observation_id = c.id
    WHERE c.session_id IN (SELECT id FROM v2.sessions)   -- sessions imported above
      AND NOT EXISTS (SELECT 1 FROM v2.observations o2
                      WHERE json_extract(o2.provenance, '$.imported_from') = 'mnemonic-v1'
                        AND json_extract(o2.provenance, '$.v1_id') = c.id);
    ```
    (attached-database form: `ATTACH <from> AS v1; ATTACH <to> AS v2;` — the LEFT-JOINed satellite tables are absent in a minimal store → the script first checks `sqlite_master` per satellite and drops the absent JOINs from the rendered SQL; the fixture exercises the one-satellite path, the full path runs on the real store.)
  - pre-flight checks (fail before any write, exit 1 with the named cause): the source has `observations_core` + `sessions` as tables; **the target's `observations` is a TABLE, not a VIEW** (the v1 name is a VIEW — importing into a v1 store is the classic silent no-op this check kills); `PRAGMA busy_timeout=5000` on both.
  - `--dry-run`: the same queries as `SELECT count(*)` (planned sessions / planned observations) + the orphan count (v1 observations whose session is missing from the v1 store — skipped, reported); zero writes (the test asserts the target file is byte-identical).
  - output: `sessions: N imported, M skipped; observations: N imported, M skipped (K orphan-session, D already-imported)` — the counts the must-have checks.
- SATISFIES: T-import

- [ ] **Step 1: Write the failing test**

`scripts/fixtures/v1-import/test.sh` (bash, `set -euo pipefail`; the assertions are sqlite3 queries + grep on the script's counts output):

```bash
#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(git rev-parse --show-toplevel)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# 1. the fixture v1 store
sqlite3 "$tmp/v1.sqlite" < "$here/v1.sql"

# 2. a fresh v2 store (the repo binary runs the migrations)
"$repo"/build/mnemonic memory search --project fixturetest --data-dir "$tmp/v2data" --limit 1 >/dev/null || true
v2="$tmp/v2data/fixturetest.sqlite"
[ -f "$v2" ] || { echo "no v2 store created"; exit 1; }

# 3. --dry-run writes nothing
cp "$v2" "$v2.pre"
"$repo"/scripts/mnemonic-v1-import.sh --from "$tmp/v1.sqlite" --to "$v2" --dry-run | grep -q 'dry-run'
cmp -s "$v2" "$v2.pre" || { echo "dry-run wrote to the target"; exit 1; }

# 4. the import: rows land, search finds them
"$repo"/scripts/mnemonic-v1-import.sh --from "$tmp/v1.sqlite" --to "$v2" | grep -q 'observations: 3 imported'
n=$(sqlite3 "$v2" "SELECT count(*) FROM observations")
[ "$n" = "3" ] || { echo "want 3 observations, got $n"; exit 1; }
s=$(sqlite3 "$v2" "SELECT count(*) FROM sessions")
[ "$s" = "2" ] || { echo "want 2 sessions, got $s"; exit 1; }
sqlite3 "$v2" "SELECT content FROM observations WHERE title='Fixed N+1 query'" | grep -q 'batched fetch' \
  || { echo "fixture content missing"; exit 1; }
# the marker + the satellite values
sqlite3 "$v2" "SELECT json_extract(provenance,'\$.v1_id') FROM observations WHERE title='Fixed N+1 query'" | grep -q '^[0-9]' \
  || { echo "provenance marker missing"; exit 1; }
sqlite3 "$v2" "SELECT pinned FROM observations WHERE title='Fixed N+1 query'" | grep -q '1' \
  || { echo "satellite pinned not imported"; exit 1; }

# 5. re-run adds 0
out="$("$repo"/scripts/mnemonic-v1-import.sh --from "$tmp/v1.sqlite" --to "$v2")"
echo "$out" | grep -q 'observations: 0 imported' || { echo "re-run not idempotent: $out"; exit 1; }
n=$(sqlite3 "$v2" "SELECT count(*) FROM observations")
[ "$n" = "3" ] || { echo "re-run changed the count: $n"; exit 1; }

# 6. the v1-store-as-target guard
"$repo"/scripts/mnemonic-v1-import.sh --from "$tmp/v1.sqlite" --to "$tmp/v1.sqlite" 2>&1 | grep -q 'VIEW' \
  || { echo "v1-target guard did not fire"; exit 1; }

echo "v1-import fixture test: all checks passed"
```

`scripts/fixtures/v1-import/v1.sql` — the minimal v1 schema + seed (2 sessions, 3 observations: one pinned via `observation_state`, one with a NULL scope (exercises the `COALESCE`), one with an orphan session id (exercises the skip)):

```sql
CREATE TABLE sessions (
  id TEXT PRIMARY KEY, project TEXT NOT NULL, directory TEXT NOT NULL,
  started_at TEXT NOT NULL, ended_at TEXT, summary TEXT, status TEXT NOT NULL DEFAULT 'active',
  title TEXT
);
CREATE TABLE observations_core (
  id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT NOT NULL REFERENCES sessions(id),
  type TEXT NOT NULL, title TEXT NOT NULL, content TEXT NOT NULL, project TEXT, scope TEXT,
  topic_key TEXT, normalized_hash TEXT, revision_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT, knowledge_class TEXT
);
CREATE TABLE observation_state (
  observation_id INTEGER PRIMARY KEY REFERENCES observations_core(id) ON DELETE CASCADE,
  pinned INTEGER NOT NULL DEFAULT 0, review_after TEXT, status TEXT, visibility TEXT
);
INSERT INTO sessions VALUES
  ('sess-v1-1', 'fixturetest', '/repo/fixturetest', '2026-01-01 10:00:00', '2026-01-01 11:00:00', 'fixed the n+1', 'active', 'N+1 fix session'),
  ('sess-v1-2', 'fixturetest', '/repo/fixturetest', '2026-01-02 09:00:00', NULL, NULL, 'active', NULL);
INSERT INTO observations_core (session_id, type, title, content, project, scope, created_at, updated_at) VALUES
  ('sess-v1-1', 'bugfix', 'Fixed N+1 query', 'batched fetch in UserList', 'fixturetest', 'project', '2026-01-01 10:05:00', '2026-01-01 10:05:00'),
  ('sess-v1-1', 'decision', 'Chose postgres for kb', 'scale reasons', 'fixturetest', NULL, '2026-01-01 10:20:00', '2026-01-01 10:20:00'),
  ('sess-orphan', 'discovery', 'Orphan observation', 'session missing from v1 store', 'fixturetest', 'project', '2026-01-03 08:00:00', '2026-01-03 08:00:00');
INSERT INTO observation_state (observation_id, pinned) VALUES (1, 1);
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd mnemonic && go build -o "$PWD/../../build/mnemonic" ./cmd/mnemonic && cd ../..
bash scripts/fixtures/v1-import/test.sh
```
Expected: FAIL — `scripts/mnemonic-v1-import.sh` missing.

- [ ] **Step 3: Write the script**

`scripts/mnemonic-v1-import.sh`:

```bash
#!/usr/bin/env bash
# mnemonic-v1-import.sh — one-shot v1→v2 store import (bash + sqlite3 only).
# --from <v1.sqlite> --to <v2.sqlite> [--dry-run]
# Content-keyed idempotency: each imported row carries the provenance marker
# {"imported_from":"mnemonic-v1","v1_id":<v1 id>}; a re-run adds 0 rows.
set -euo pipefail

from=""; to=""; dry=""
while [ $# -gt 0 ]; do
  case "$1" in
    --from) from="$2"; shift 2 ;;
    --to) to="$2"; shift 2 ;;
    --dry-run) dry=1; shift ;;
    *) echo "usage: mnemonic-v1-import.sh --from <v1.sqlite> --to <v2.sqlite> [--dry-run]" >&2; exit 2 ;;
  esac
done
[ -n "$from" ] && [ -n "$to" ] || { echo "usage: mnemonic-v1-import.sh --from <v1.sqlite> --to <v2.sqlite> [--dry-run]" >&2; exit 2; }
[ -f "$from" ] || { echo "error: v1 store not found: $from" >&2; exit 1; }
[ -f "$to" ]   || { echo "error: v2 store not found: $to" >&2; exit 1; }

q() { sqlite3 "$from" "$1"; }        # source probes
qt() { sqlite3 "$to" "$1"; }         # target probes

# pre-flight: the source is a v1 store, the target is a v2 store.
[ "$(q "SELECT type FROM sqlite_master WHERE name='observations_core' AND type='table'")" = "table" ] \
  || { echo "error: source is not a v1 store (no observations_core table)" >&2; exit 1; }
[ "$(q "SELECT type FROM sqlite_master WHERE name='sessions' AND type='table'")" = "table" ] \
  || { echo "error: source has no sessions table" >&2; exit 1; }
tt="$(qt "SELECT type FROM sqlite_master WHERE name='observations'")"
[ "$tt" = "table" ] || { echo "error: target observations is ${tt:-missing}, not a table — is $to a v2 store? (the v1 name is a VIEW)" >&2; exit 1; }

# render the observations SELECT with only the satellites the source has.
joints=""
for sat in "observation_state st pinned review_after" "observation_provenance pr tool_name" "observation_stats sa importance_score duplicate_count last_seen_at"; do
  set -- $sat
  table="$1"; alias="$2"
  if [ "$(q "SELECT count(*) FROM sqlite_master WHERE name='$table' AND type='table'")" = "1" ]; then
    case "$table" in
      observation_state)     joints="$joints LEFT JOIN v1.$table $alias ON $alias.observation_id = c.id" ;;
      observation_provenance) joints="$joints LEFT JOIN v1.$table $alias ON $alias.observation_id = c.id" ;;
      observation_stats)     joints="$joints LEFT JOIN v1.$table $alias ON $alias.observation_id = c.id" ;;
    esac
  fi
done
# (the SELECT's column expressions reference st./pr./sa. only when the join is present — the renderer pairs them with the joins above)

if [ -n "$dry" ]; then
  sess="$(qt "ATTACH '$from' AS v1; SELECT count(*) FROM v1.sessions s WHERE NOT EXISTS (SELECT 1 FROM sessions t WHERE t.id = s.id)")"
  obs="$(qt "ATTACH '$from' AS v1; SELECT count(*) FROM v1.observations_core c WHERE c.session_id IN (SELECT id FROM v1.sessions) AND NOT EXISTS (SELECT 1 FROM observations o2 WHERE json_extract(o2.provenance,'\$.imported_from')='mnemonic-v1' AND json_extract(o2.provenance,'\$.v1_id')=c.id)")"
  orphan="$(q "SELECT count(*) FROM observations_core c WHERE c.session_id NOT IN (SELECT id FROM sessions)")"
  echo "dry-run: would import $sess sessions, $obs observations ($orphan orphan-session skipped); no writes"
  exit 0
fi

qt "
PRAGMA busy_timeout=5000;
ATTACH '$from' AS v1;
BEGIN;
INSERT OR IGNORE INTO sessions (id, project, directory, started_at, ended_at, summary, title)
  SELECT id, project, directory, started_at, ended_at, summary, title FROM v1.sessions;
INSERT INTO observations
  (session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash,
   revision_count, pinned, review_after, duplicate_count, last_seen_at, importance,
   provenance, created_at, updated_at, deleted_at)
  SELECT c.session_id, c.type, c.title, c.content,
         $( [ -n "$(echo "$joints" | grep -c provenance || true)" ] && echo "pr.tool_name" || echo "NULL" ),
         c.project, COALESCE(c.scope,'project'), c.topic_key, c.normalized_hash, c.revision_count,
         COALESCE( $( [ -n "$(echo "$joints" | grep -c observation_state || true)" ] && echo "st.pinned" || echo "0" ), 0),
         $( [ -n "$(echo "$joints" | grep -c observation_state || true)" ] && echo "st.review_after" || echo "NULL" ),
         COALESCE( $( [ -n "$(echo "$joints" | grep -c observation_stats || true)" ] && echo "sa.duplicate_count" || echo "0" ), 0),
         $( [ -n "$(echo "$joints" | grep -c observation_stats || true)" ] && echo "sa.last_seen_at" || echo "NULL" ),
         COALESCE( $( [ -n "$(echo "$joints" | grep -c observation_stats || true)" ] && echo "sa.importance_score" || echo "0.5" ), 0.5),
         json_object('imported_from','mnemonic-v1','v1_id',c.id),
         c.created_at, c.updated_at, c.deleted_at
  FROM v1.observations_core c
  $joints
  WHERE c.session_id IN (SELECT id FROM v1.sessions)
    AND NOT EXISTS (SELECT 1 FROM observations o2
                    WHERE json_extract(o2.provenance,'\$.imported_from')='mnemonic-v1'
                      AND json_extract(o2.provenance,'\$.v1_id')=c.id);
COMMIT;
"
echo "sessions: $(qt "SELECT count(*) FROM sessions") present; observations: $(qt "SELECT count(*) FROM observations WHERE json_extract(provenance,'\$.imported_from')='mnemonic-v1'") imported-marker rows total"
```

(The satellite-conditional rendering is the one fiddly bit — the Step 3 note: keep the renderer a plain bash function `renderSelect()` that emits the column list and joins together from the same per-satellite presence check, so the column expressions and the joins cannot drift; the fixture test pins the one-satellite path, and a full-satellite fixture run (the real `001.sqlite` shape) is exercised in Step 5.)

- [ ] **Step 4: Run the test to verify it passes**

```bash
bash scripts/fixtures/v1-import/test.sh
```
Expected: PASS (`v1-import fixture test: all checks passed`).

- [ ] **Step 5: The full-satellite shape check (recorded)**

Run the script `--dry-run` against a real v1 store shape (copy `~/.skillgrid/mnemonic/001.sqlite` to a temp dir — it is empty, but its schema exercises all five satellites) → the renderer includes all five joins, no error, counts report 0 importable (empty store) + the orphan line. Record the output.

- [ ] **Step 6: Commit**

```bash
git add scripts/mnemonic-v1-import.sh scripts/fixtures/v1-import/
git commit -m "feat: add the one-shot v1-to-v2 store import script

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 9"
```

### Task 10: docs — the v2 surface for the remaining convention files

> The cutover (Task 8) retired the v1 surfaces in the code + configs; this task closes the documentation ring so no convention file points at a dead surface.

**Files:**
- Modify (as the sweep finds): the `CHANGELOG.md` entry (the repo's changelog convention — append the v2 host-plugins entry: the five-host memory surface, the installer families, the cutover, the import script); any remaining repo doc referencing v1 mnemonic tokens (the sweep below); `README.md`/`docs/` sections describing the skillgrid CLI — add the `--family memory` / `uninstall` / `status` surface (one subsection, the flag table verbatim from the CLI help)

**Steps:**

- [ ] **Step 1: The sweep**

```bash
grep -rn ':7438\|skillgrid mcp\|web_cache_\|fact_\|code_index\|\bserve\b' \
  --include='*.md' --include='*.json' --include='*.jsonc' . \
  | grep -v '.skillgrid/change-specs\|change-archive\|\.adr-\|node_modules\|\.git/' || true
```

Every hit: v2-qualify (the `mnemonic-code` `code_index` case) or fix to the v2 surface; ADR/change-archive references stay (the gate excludes them — the decision record keeps its historical wording).

- [ ] **Step 2: The changelog + CLI docs** (the entries above; the CLI subsection renders `skillgrid install --family memory --help` output verbatim).

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "docs: document the v2 memory host surface and the installer families

[skillgrid-context] change: 2026-11-17-mnemonic-v2-host-memory-plugins task: 10"
```

### Task 11: final live smoke — the full five-host matrix + the live uninstall round trip

> The held-out end: after the cutover, the complete T-matrix on live hosts, plus the live proof of T-installer-roundtrip's last clause (uninstall returns each HOME byte-identical to pre-install — the fixture tests pin the logic, this is the real-config proof).

**Steps:**

- [ ] **Step 1: The live uninstall round trip**

```bash
# pre-image: the five live config surfaces
mkdir /tmp/preimg && for f in ~/.omp/agent/mcp.json ~/.skill-pi/agent/mcp.json ~/.config/opencode/opencode.jsonc ~/.config/kilo/kilo.jsonc ~/.cursor/mcp.json ~/.cursor/hooks.json; do cp "$f" "/tmp/preimg/$(basename "$f")"; done
skillgrid uninstall --family memory all
# the tracked sets are gone; the cutover's one-way v1-key deletions are NOT reversed (expected — recorded)
skillgrid install --family memory all --bin "$(which mnemonic)" --repo "$(git rev-parse --show-toplevel)"
# byte-identical re-install proof: every config surface equals its post-uninstall state (re-run = no-op)
```

- [ ] **Step 2: The five-host matrix (one session per host, recorded evidence per row)**

| host | assertions |
|---|---|
| omp | N prompts → N prompt rows; exactly 1 session row; a subagent run adds 0 rows (T-omp-live); `mem_save`→`mem_search` round-trips in the session cwd (T-omp-cwd); post-compact context carries the FIRST ACTION REQUIRED notice (T-compaction); probe record all-ok (T-probe) |
| pi | same truths (the shared extension; the pi session start) |
| opencode | 1 session row (harness id); prompt rows with `source_inbox_id`; `mem_save` with a foreign `session_id` lands under the harness id (the rewrite, T-opencode-live); subagent 0 rows; compaction notice (T-compaction) |
| kilo | same truths (the byte-identical copy; the AGENTS.md block present) |
| cursor | sessionStart → 1 `cursor` session row; beforeSubmitPrompt → prompt rows; identical re-typed prompt dedups (T-cursor-live); the rule loaded (mapping + pin) |
| all | `skillgrid status --family memory all` → installed, no drift, no stale-v1, drops/probe leg readable; the body exactly-once check on one context dump per host (T-protocol) |

- [ ] **Step 3: Gap fixes (only if the matrix finds one — RED test first, same contract as Task 7)**

- [ ] **Step 4: Commit (only if Step 3 changed code) + the change is done**

The evidence block (per-host row dumps, the status output, the uninstall proof) closes the change; `state.yaml` flips to the next phase per the serial-dev rule.

## Threat Matrix

Per `_shared/references/threat-matrix.md`. This change touches **subprocesses** (extensions/plugins spawn the binary; cursor hook commands are shell-interpreted), **Mnemonic tool contracts** (a new host-facing verb + a shrunk 5-host tool surface + a new host-qualified name prefix), and **one agent-facing convention file** (the mnemonic SKILL.md).

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Documentation-like paths | N/A: installed artifacts are `.ts`/`.js` code + JSON/JSONC config + one `.mdc` rule (prose, not executed) + SKILL.md (agent instructions) — no executable Markdown/MDX or `README.sh`-class files | — | — |
| Git repository selection | Applicable: the `save-prompt` verb, the session-start probe, and session registration all resolve the project from the hook/extension process cwd — a worktree or repo subdirectory must hit the same store as the host session's `mem_*` tools | Reuse the CLI's existing project-resolution chain (explicit > env > git toplevel > basename) — no hand-rolled `git -C`; the verb opens the store through the same path as the `mem_save_prompt` MCP dispatch, so verb and tools cannot resolve differently | `save_prompt_test.go`: cwd in a worktree → the worktree's store; cwd outside any git repo → basename project; the probe (`memory search --limit 1`) and the verb hit the same store for the same cwd |
| Commit state | N/A: the v2 prompt/session stores hold no commit state; v1's commit-scoped surfaces (`fact_*`/`code_*`/`ctx_*`/`squad`) are deleted at cutover, not migrated — the import script moves observations + sessions only | — | — |
| Push state | N/A: no push interaction | — | — |
| PR commands | N/A: no PR automation | — | — |
| Mnemonic tool surface (`mem_*`/`code_*`/`web_cache_*`) | Applicable — three contract deltas: (1) new host-facing verb `memory save-prompt` (stdin shapes: plain text \| cursor JSON fixture; idempotency window; exit codes); (2) the host-qualified tool prefix becomes `mnemonic-memory_mem_*` on all five hosts (v1 entry keys retire); (3) the 5-host agent surface shrinks to the memory-family agent profile — `fact_*`/`code_*`/`ctx_*`/`squad`/`web_cache_*`/`serve` leave the surface. Callers that must adapt: the cursor rule's tool mapping, the opencode/kilo rewrite normalizer, SKILL.md, and any host session still naming v1 tools | Single-source protocol: the Go MCP instructions carry the body (≤2,048-rune guard), host shells carry mapping/pin/repair only; the verb and `mem_save_prompt` share one prompt-store write path (one writer, two callers) | `save_prompt_test.go` (stdin shapes, idempotency, exit codes, cwd project); `rule-shape.test.mjs` (zero `web_cache_`/`fact_`/`serve`; pin + mapping present); `rewrite.test.mjs` (exact `mnemonic-memory_mem_*` names, foreign/absent `session_id` rewritten); `instructions_test.go` (rune guard + cursor rule body == rendered Go body) |
| Shared-convention drift | Applicable: `.agents/skills/mnemonic/SKILL.md` is the agent-facing convention for mnemonic usage — its v1 daemon / `:7438` / `skillgrid mcp` / `web_cache_*` sections are replaced by the v2 surface. Skills that break if missed: any skill or rule that instructs v1 tool names or `:7438` routes | The cutover task rewrites SKILL.md in the same commit as the config rewrites (no drift window); a grep gate pins absence | Cutover-task grep: no live reference to `:7438`, `skillgrid mcp`, `web_cache_`, `fact_*`, `code_index` under `.agents/skills/` (change-archive/ADR references excluded) |

Subprocess/shell exposure (extensions/plugins exec the binary; cursor's command strings are shell-interpreted by the harness) is covered by: the 256 KiB stdin cap on `save-prompt` (verb task); hard timeouts per surface — 5 s `save-prompt` spawn, 10 s session-start probe (Global Constraints) — with cursor's command bounded by the harness's own limit (noted, not pinned; the store side is bounded by the existing `busy_timeout(5000)` DSN); and command strings built ONLY from the installed absolute binary path + fixed verb/event words — hook payload data is never interpolated into a command string (it travels on stdin; the cursor JSON is parsed verb-side, not in the shell).
