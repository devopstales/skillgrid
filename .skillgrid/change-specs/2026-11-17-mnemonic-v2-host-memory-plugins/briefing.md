# Mnemonic-v2 5-Host Memory Plugins — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-17) — shootdown gate passed (6/6 lenses, 2026-11-17), all Blockers user-dispositioned (2026-11-17).

Supersedes `2026-11-04-mnemonic-v2-memory-plugins` (2 hosts × 3 tool families; that spec is in `.skillgrid/change-archive/` with a SUPERSEDED header). This change is **memory-only** (the `mem_*` family) across **5 hosts**: pi, omp, opencode, kilo, cursor.

## Problem / Intent

The mnemonic-v2 engine (repo module `mnemonic/`) exposes its 22 agent-profile memory tools over MCP stdio (`mnemonic memory mcp --profile agent`), but none of the five coding-agent hosts has a working v2 memory surface: the v1 plugins speak a removed HTTP protocol (`:7438` no longer exists), and the stale v1 MCP config entries (`mnemonic mcp` / `skillgrid mcp`) no longer resolve to a v2 command. Agents therefore have no functioning memory on any host. This change gives all five hosts the v2 memory surface — tools, prompt capture, session registration, compaction recovery — porting the proven engram per-host plugin architecture, plus a clean v1 cutover.

## Purpose & Success Criteria

- **Purpose:** any agent session on pi, omp, opencode, kilo, or cursor can save, search, and recall project memory through the v2 engine, with prompts and compaction summaries captured automatically, and one installer command that sets up or tears down the whole surface.
- **Success criteria:**
  - On all 5 hosts after install: the 22 agent-profile `mem_*` tools are present and callable; a `mem_save` lands in the correct project store; `mem_search` recalls it.
  - Prompt capture: every user prompt is stored (fail-open, never blocks the host).
  - Session registration: one session row per real conversation (subagent sessions do not inflate the count).
  - Compaction recovery: on compaction the agent is told to re-inject memory context (FIRST ACTION REQUIRED notice).
  - `skillgrid install --family memory` (all hosts) then `skillgrid uninstall --family memory` leaves no tracked files/entries behind and no foreign entries touched (fixture HOMEs).
  - v1 cutover: no v1-HTTP wiring remains registered on any host; no stale `mnemonic mcp`/`skillgrid mcp` entries remain.
  - Data continuity: the one-shot import script recovers the v1 store's observations into the v2 store, verified by `mem_search`.
  - Zero Go changes in the `mnemonic/` module except the new `memory save-prompt` verb (and any §-scoped text/test edits recorded in the shootdown table).
- **Out of scope:** code-index (`code_*`) and project-wiki (`projectwiki_*`) families (deferred to the ship-pending code-index change and its successor); the observation layer (`obs_events`, queued `2026-11-17-mnemonic-v2-observation-layer`); the router-bootstrap/skillgrid auto-load plugins (queued `2026-11-17-skillgrid-host-plugins`); `web_*` (no v2 equivalent — gap recorded); `ctx_*` (gap recorded, ADR-0044 overflow receipts lost); team/squad v1 surface (gap recorded); passive capture (descoped, recorded); team-shared PG sync; any new third-party dependency.

## Context

- **Engine:** module `mnemonic/` (renamed from `mnemonic-v2` by the 2026-11-08 mcp-split). MCP: `mnemonic memory mcp --profile agent|all` — 22 agent-profile tools / 28 total; server identity name `mnemonic-memory`, version `0.2.0` (`mnemonic/internal/transport/mcp/server.go`). `mem_save`'s `session_id` is Required, and saving to an unknown session **auto-registers** it (`mnemonic/internal/memory/trails_test.go`). The server sends profile-aware `instructions` at initialize (`memoryInstructionsFor`, 2,596 runes rendered). There is **no v2 HTTP server** — root `mcp`/`serve` commands were removed with did-you-mean errors.
- **CLI:** `mnemonic memory {save|search|get|lookup|context|review|timeline|pin|unpin|delete|session|stats|doctor|mcp}`; `session {start|end|summary|finalize}`; `start` takes `--id`, `end|finalize` take `--session`. `memory save` captures the current prompt by default (`--no-capture` opt-out). **`save-prompt` does not exist yet — it is the one new Go surface of this change.**
- **v1 plugin surface (to be cut over):** repo `plugins/pi/mnemonic/` (index.ts wiring memory+fact_+code_+ctx_+SDD-events; api.js v1 HTTP client; ctx-sandbox.ts; mnemonic-{memory,facts,codeindex}.ts; skillgrid-{compaction,events,sdd-events}.ts), repo `plugins/opencode/` (same family + squad + v2sse + logo + tools/ v1 tool defs), repo `plugins/kilo/` (install-time copy of the opencode plugin), repo `plugins/cursor/mcp.json` (key `mnemonic`), repo `hooks/hooks-cursor.json` + `rules/mnemonic.mdc` + `.cursor-plugin/plugin.json`.
- **Live stale configs (verified 2026-11-17, all dead post-rename):** `~/.omp/agent/mcp.json` `mnemonic = {type:"stdio", command:"mnemonic", args:["mcp"]}` (string command); `~/.config/opencode/opencode.jsonc` `mnemonic = ["mnemonic","mcp"]` (JSONC); `~/.config/kilo/kilo.jsonc` `mnemonic = ["skillgrid","mcp"]` (JSONC); `~/.cursor/` MCP entry key `mnemonic`.
- **Live data fork (verified 2026-11-17):** `~/.skillgrid/mnemonic/aiskillgrid.sqlite` = v1 store for this repo (189 observations, 422 sessions); v2 resolves the repo to `skillgrid-v2.sqlite` (1 obs, 2 sessions). v1 `observations` is a VIEW over `observations_core` + satellites; v2 is a TABLE with different columns — v2 opening a v1 file "opens clean, fails on first use".
- **Live coexisting memory plugin:** `~/.config/kilo/plugins/engram.ts` (engram 1.11.0, own bare `mem_*` tools + protocol). Kilo auto-discovers plugin files in `~/.config/kilo/plugins/` (engram's live, working location) and also `~/.config/kilo/plugin/` + the `kilo.jsonc` `plugin` array (skillgrid-checkpoint.ts is in both).
- **Existing installer:** `skillgrid-cli/internal/install/` already resolves all five hosts' config paths, merges MCP/LSP entries into JSON(C) preserving foreign keys, and installs pi/kilo plugins (`--dry-run`/`--verbose`/`--sync-repo`). Separate Go module from `mnemonic/` (not importable from it).
- **Reference architecture:** `~/git/ai-test/engram` — `plugin/pi/index.ts` (3,382 lines: per-session spawned child, registerTool, self-heal, adapter-owns `session_id` injection, 278-line generated contract projection), `plugin/opencode/engram.ts` (V1 `server()`/V2 `setup()` lifecycle, `capturePrompt` = `truncate(stripPrivateTags(content), 2000)`, `tool.execute.before` `session_id` rewrite, `parentID` subagent guard — engram issue #116 "170 sessions for 1 real conversation").
- **Operational rules:** serial development (one change at a time); spec-zone commits before code-zone; no new dependencies without an ADR; conventional commits with `[skillgrid-context]` block naming the Backlog task.

## Approaches Considered

- **Chosen:** **Unified host-MCP transport** — all five hosts get a `mnemonic-memory` MCP config entry (`mnemonic memory mcp --profile agent`, absolute binary path) as the single tool-dispatch mechanism; pi/omp and opencode/kilo additionally get a **thin hooks-only extension/plugin** (shared TS source per pair) doing session registration, prompt capture, compaction recovery, protocol host-shell injection, and session-start liveness probe; cursor gets hooks + an alwaysApply rule. One shared installer (the extended skillgrid-cli installer) owns all five hosts' entries. Adapter-owns the session ID (engram pattern). One-shot v1 data import script (no Go verb).
  Rationale: one transport for five hosts (no second MCP wire-protocol implementation, no per-session child lifecycle, no hand-mirrored schemas); keeps the engram architecture's substance (per-host capture, compaction recovery, session ownership) while dropping its heaviest machinery (spawned child, self-heal, contract projection) — all of which existed to serve an independently-released HTTP server this local-binary design does not have.
- **Rejected:** **Engram literal port for pi/omp** (per-session spawned `mnemonic memory mcp` child behind `pi.registerTool` + 22 static typebox schemas + hand-rolled ~150-line TS MCP client + generated contract projection). Proven in engram, but: omp already runs a host-spawned MCP child (`~/.omp/agent/mcp.json`), pi has a built-in `mcp.json` mechanism, and the projection/self-heal machinery targets a risk profile (independently-released HTTP server) this design lacks. Cost: ~1.5k more TS, a second MCP wire impl in the repo, and a schema-duplication maintenance seam. User dispositioned host-MCP on 2026-11-17.
- **Rejected:** **Second persistent child for opencode/kilo capture** (engram's resident-HTTP model). v2 has no HTTP server; a plugin-owned MCP child costs ~30–80 MB RSS per session for calls a one-shot CLI verb serves in ~30–80 ms fail-open. One-shot CLI verbs chosen (user-confirmed transport in shootdown).
- **Rejected:** **`mnemonic memory migrate` Go verb for v1 data** — user chose the one-shot script to preserve the zero-Go-changes lock (2026-11-17).

## Requirements

1. **R1 — Five-host MCP config entries:** every host's MCP config carries a `mnemonic-memory` entry with command `[<abs-bin>, "memory", "mcp", "--profile", "agent"]` (absolute resolved binary path), installed/removed by the installer, shape-preserving per host dialect (omp string-`command`+`args`; opencode/kilo JSONC array form + `enabled`/`type`; cursor mcp.json).
   - **Current:** stale v1 entries on all 5 hosts (`mnemonic mcp` / `skillgrid mcp`), all non-resolving under the v2 binary; `plugins/cursor/mcp.json` uses key `mnemonic` ≠ server identity `mnemonic-memory`.
   - **Target:** cutover rewrites exactly those entries to the v2 shape (key normalized to `mnemonic-memory`), preserving every foreign entry, comment, and field; fresh hosts get the entry added.
   - **Acceptance:** per-host fixture tests (real-shape config in → config out) where the diff is exactly the mnemonic entry; a `status` run reports each host's entry present + correct.
   - **Acceptance scenario:** `happy path five-host-install` → `acceptance.feature`

2. **R2 — pi/omp thin extension (one shared TS source):** a pi-ExtensionAPI extension installed for both pi (`~/.skill-pi/agent/extensions/`) and omp (`~/.omp/agent/extensions/`) providing: (a) session registration via `memory session start --id <adapter-id>` with a **subagent guard** (child sessions skip registration + capture); (b) per-prompt capture via `memory save-prompt` (fail-open, fire-and-forget); (c) compaction recovery (context injection + FIRST ACTION REQUIRED notice); (d) protocol host-shell system injection (see R10); (e) **session-start liveness probe** (see R11) whose failure injects an actionable warning (cause + exact repair command) instead of failing the session.
   - **Current:** `plugins/pi/mnemonic/index.ts` is the v1 wiring (memory+fact_+code_+ctx_+SDD-events over HTTP `:7438`); omp loads the same package as an extension.
   - **Target:** the v1 `index.ts` is replaced by the thin extension; v1 modules deleted (R8); the extension makes **zero** direct engine calls beyond CLI verbs (no MCP client, no HTTP).
   - **Acceptance:** unit tests for the subagent guard, fail-open capture (verb failure → host unaffected), compaction notice text; a pi and an omp smoke session each produce exactly one session row and N prompt rows for N prompts.
   - **Acceptance scenario:** `happy path piomp-capture` → `acceptance.feature`

3. **R3 — opencode/kilo shared plugin (one shared TS source; kilo byte-identical copy):** opencode plugin API plugin providing: (a) session registration with the `parentID` subagent guard (engram issue-#116 fix); (b) per-prompt capture via `memory save-prompt`; (c) compaction recovery (context injection + FIRST ACTION REQUIRED); (d) **`tool.execute.before` `session_id` rewrite** on the host-qualified `mnemonic-memory_mem_*` tool names (with prefix normalizer, engram `canonicalEngramToolName` shape); (e) protocol host-shell injection. Kilo receives a **byte-identical install copy** (no `__AGENT__` substitution — no v2 consumer).
   - **Current:** `plugins/opencode/{mnemonic-memory,skillgrid-events,skillgrid-compaction}.ts` do the equivalent over v1 HTTP; `plugins/kilo/` is an install-time copy.
   - **Target:** one new shared TS source; the three v1 files are retired/replaced (R8); `session_id` on every stored write matches the adapter-registered session (no phantom sessions).
   - **Acceptance:** unit test: a `mem_save` call with a foreign/absent `session_id` is rewritten to the adapter session; subagent (parentID set) does not create a session row; opencode + kilo smoke sessions each produce one session row.
   - **Acceptance scenario:** `happy path opencode-kilo-capture` → `acceptance.feature`

4. **R4 — Adapter-owns session identity:** the adapter (extension/plugin/hook) is the single authority on `session_id`; protocol text on every host says "do not invent or override `session_id`". Cursor (hooks cannot rewrite MCP args) uses a **stable per-host ID** `cursor` pinned in the rule text; per-project SQLite stores keep projects apart (known simplification: concurrent cursor sessions in one project share the session row — recorded).
   - **Current:** `mem_save.session_id` Required + unknown-ID auto-register (`trails_test.go`) — a wrong guess silently forks the session; no host conveys the ID today.
   - **Target:** pi/omp inject the ID into forwarded context; opencode/kilo rewrite it in `tool.execute.before`; cursor rule pins `session_id: "cursor"`; a typo/foreign ID on opencode/kilo cannot create a phantom session.
   - **Acceptance:** unit tests per host (injection/rewrite/pinned value); the auto-register behavior is unchanged in the engine (documented as defense-in-depth, not the primary mechanism).
   - **Acceptance scenario:** `happy path session-identity` → `acceptance.feature`

5. **R5 — Cursor surface:** `~/.cursor/hooks.json` entries (absolute binary path, fail-open): sessionStart → `memory session start --id cursor`; beforeSubmitPrompt → `memory save-prompt` (stdin = cursor's raw JSON payload); sessionEnd/stop → `memory session end --session cursor` (plus finalize where the host event fires). `rules/mnemonic.mdc` rewritten to the v2 surface: protocol host-shell, `session_id: "cursor"` pin, tool-name mapping line (`mcp__mnemonic-memory__mem_*`), and the v1-only sections (Code Index Ladder, Web Cache, Passive Capture) **deleted**.
   - **Current:** `hooks/hooks-cursor.json` (v1 hooks), `rules/mnemonic.mdc` (v1 protocol incl. dead `web_cache_*`/`fact_*`), `.cursor-plugin/plugin.json` (description names the removed `skillgrid mcp`).
   - **Target:** v2 hooks + v2 rule; installer owns both files (manifest-tracked, restore-on-uninstall); `.cursor-plugin/plugin.json` description updated.
   - **Acceptance:** a cursor smoke session registers one session and captures prompts; the rule text contains no `web_cache_`/`fact_`/`serve` references (grep check in test).
   - **Acceptance scenario:** `happy path cursor-hooks` → `acceptance.feature`

6. **R6 — `memory save-prompt` CLI verb (the one new Go surface):** `mnemonic memory save-prompt` reads stdin (plain text or a host hook JSON payload — the cursor shape is pinned by a fixture, obs-blueprint Task-3 pattern), resolves project from cwd, attaches the prompt to the pinned session (per-host ID convention, R4), stores it via the same prompt-store path as the `mem_save_prompt` MCP dispatch, is **idempotent under re-fire** (duplicate identical content + session within the turn window → no second row), exits 0 quietly on success, and never blocks > ~200 ms.
   - **Current:** does not exist; per-turn prompt capture has no CLI seam (MCP `mem_save_prompt` exists; `memory save` captures the *current* prompt as a side effect of saving an observation).
   - **Target:** the verb exists, is exercised by all five hosts' capture paths (pi/omp/opencode/kilo/cursor), and is the single per-turn prompt-capture seam.
   - **Acceptance:** unit tests: plain-text stdin, cursor-JSON stdin, re-fire idempotency, project resolution from cwd, fail-open exit code; `go test ./mnemonic/...` green.
   - **Acceptance scenario:** `happy path save-prompt-verb` → `acceptance.feature`

7. **R7 — Shared installer (extends skillgrid-cli):** `skillgrid-cli/internal/install/` gains the memory family: `skillgrid install --family memory [agent|all]`, `uninstall`, `status` — host detection for the 5 hosts; a manifest at `~/.skillgrid/mnemonic/memory-install.json` keyed **(host, family)** carrying a sha256 file map + hook/MCP/AGENTS.md-marker entry list; idempotent re-run (no-op on matching state); **no-clobber** (foreign entries/files are never adopted, never modified, never deleted — uninstall removes exactly the tracked set); atomic write + pre-write backup; cutover rewrite of stale v1 entries (R1) inside the same run; `status` reports per host: entry state, binary resolvability, a liveness probe result, stale v1 leftovers, capture drop counts (R11), and an engram-coexistence warning on kilo (R12). The merge/manifest machinery is an exported package boundary so the queued obs change can migrate onto it (its recorded follow-up).
   - **Current:** skillgrid-cli installs router/LSP/pi/kilo artifacts with no memory family and no manifest; the v1 setup code (`skillgrid-cli/internal/setup/*`) left v1 markers (e.g. kilo `AGENTS.md` marker block) with no removal path.
   - **Target:** one installer owns the entire memory surface per host; `uninstall` returns a fixture HOME to exactly its pre-install state (byte-diff).
   - **Acceptance:** five fixture-HOME tests (one per host, real-shape configs incl. the three JSON dialects): install → assert; re-run → byte-identical; foreign-entry clobber check; uninstall → byte-identical to pre-install; cutover rewrite diff = exactly the mnemonic entry.
   - **Acceptance scenario:** `happy path installer-roundtrip` → `acceptance.feature`

8. **R8 — v1 cutover (per-host retirement inventory):**
   - **opencode:** `mnemonic-memory.ts` → replaced by the new plugin (R3); `skillgrid-events.ts`, `skillgrid-compaction.ts` → retired (capture/compaction move into the new plugin); `mnemonic-codeindex.ts`, `skillgrid-squad.ts`, `skillgrid-context.ts` → retired (v1-HTTP, dead at runtime under v2; `code_*`/`squad`/`ctx_*` losses recorded as known gaps until their owning changes); `tools/*` v1 tool defs, `src/memory-prefetch.ts`, `skillgrid-v2sse.ts`, logo → retired with the package rewrite (v2sse/logo re-added only if still consumed by the new package — verify at execution); opencode.jsonc plugin array + kilo.jsonc array updated to the new plugin path.
   - **pi/omp:** `plugins/pi/mnemonic/` → `index.ts` rewritten as the thin extension (R2); `api.js`, `mnemonic-facts.ts`, `mnemonic-codeindex.ts`, `ctx-sandbox.ts`, `skillgrid-sdd-events.ts`, `skillgrid-events.ts`, `skillgrid-compaction.ts` → deleted (SDD-event capture gap owned by the queued obs change; `ctx_*` ADR-0044 overflow-receipt loss recorded).
   - **cursor:** `hooks-cursor.json` v1 entries → replaced (R5); `rules/mnemonic.mdc` → rewritten (R5); `.cursor-plugin/plugin.json` description → updated.
   - **kilo:** v1 AGENTS.md marker block (v1 setup `upsertMarkerBlock`) → updated-or-removed at install, restored at uninstall (manifest-tracked).
   - **docs:** `.agents/skills/mnemonic/SKILL.md` (daemon/`:7438`/`skillgrid mcp` sections) and any AGENTS.md preamble memory blocks → updated to the v2 surface in the same change.
   - **Current:** v1 wiring registered on all 5 hosts against a removed HTTP server; agents with `code_*`/`ctx_*`/`squad` tools that fail open on every call (registered-but-broken).
   - **Target:** no v1-HTTP wiring remains registered anywhere; `grep -r 7438 plugins/` (post-cutover) returns nothing functional.
   - **Acceptance:** post-cutover fixture: each host's registered plugin/hook/rule set matches the v2 inventory exactly; no file in the installed set references `:7438`, `skillgrid mcp`, `web_cache_`, or `fact_`.
   - **Acceptance scenario:** `happy path v1-cutover` → `acceptance.feature`

9. **R9 — One-shot v1 data import script:** a repo script (`scripts/mnemonic-v1-import.sh`, sqlite3-CLI based) takes `--from <v1.sqlite> --to <v2.sqlite>`, maps v1 observations (from the `observations` view/core + satellites) and session rows into v2 rows (column mapping derived from both schemas at execution), is **idempotent** (content-keyed, re-run = no-op), supports `--dry-run`, and prints counts. Run manually per project; documented in the wiki + this change's tasks.
   - **Current:** no bridge between `aiskillgrid.sqlite` (v1, 189 obs/422 sessions for this repo) and `skillgrid-v2.sqlite` (v2, 1 obs/2 sessions).
   - **Target:** after one run per project, v1-era observations are recallable via `mem_search`/`memory search` in the v2 store; a second run adds zero rows.
   - **Acceptance:** test with a fixture v1 store: import → `memory search` finds v1 content; re-run → counts unchanged; dry-run → no writes.
   - **Acceptance scenario:** `happy path v1-import` → `acceptance.feature`

10. **R10 — Protocol text single source:** the Go MCP initialize `instructions` block is the **only authored protocol source**. Every host surface carries at most a host-shell (tool-name mapping line, `session_id` pin, repair command, FIRST ACTION REQUIRED): pi/omp + opencode/kilo obtain the protocol body at session start by a transient `mnemonic memory mcp` spawn that reads the initialize response and exits (cached per session, fail-open — on failure inject the host-shell only); if a host is verified to surface MCP server instructions natively, the injection degrades to host-shell only (verify per host at execution). The cursor rule is generated from / **diff-tested against** the Go source; the installed copy is manifest-tracked. A Go test pins the rendered instructions to ≤ 2,048 runes (engram's production truncation ceiling); if it fails, the text is trimmed and that Go text edit is recorded here.
    - **Current:** the protocol exists in up to 4 diverging copies (Go instructions 2,596 runes; v1 pi/opencode `MEMORY_INSTRUCTIONS`; cursor rule; kilo AGENTS.md marker).
    - **Target:** one authored source; zero drift (diff test); no host shows the full protocol twice in one context.
    - **Acceptance:** diff test (cursor rule ↔ Go source; each host-shell ↔ its pinned template) green; a pi + opencode smoke shows the protocol exactly once.
    - **Acceptance scenario:** `happy path protocol-single-source` → `acceptance.feature`

11. **R11 — Fail-open observability:** capture hooks and the liveness probe are fail-open (host never blocked/crashed by memory). The session-start probe verifies (a) binary resolves, (b) MCP `initialize` succeeds (transient spawn), (c) one store read succeeds (`memory search --limit 1`); on failure the extension injects the actionable warning (exact cause + repair command) and records the failure to a per-project scratch file (`~/.skillgrid/mnemonic/<project>/.last-session-start.json`) that `status` surfaces. Hooks that drop a capture (store busy, verb timeout) increment a per-project drop counter surfaced by `status` ("N captures dropped (store busy)").
    - **Current:** no surface answers "is memory working in this host right now, and why did it die" — `memory doctor` checks engine capabilities only; the GUI-PATH case (bare `mnemonic` on PATH, host launched by a GUI) is green-everywhere-but-dead.
    - **Target:** one command (`status`) answers per host: entry state, liveness, last failure + cause, drop counts.
    - **Acceptance:** unit tests: probe failure → warning text contains cause + repair command, scratch file written, host session continues; drop counter increments on forced verb timeout.
    - **Acceptance scenario:** `happy path fail-open-observability` → `acceptance.feature`

12. **R12 — Engram coexistence detection:** on kilo (and any host) the installer detects a competing memory plugin (engram files/MCP entries in the host's plugin dirs or config) and prints an explicit warning ("engram coexists: bare `mem_*` names resolve to engram; uninstall engram or accept dual protocols"); `status` re-reports it.
    - **Current:** live `~/.config/kilo/plugins/engram.ts` + engram's own MCP entry coexist invisibly with the incoming mnemonic surface.
    - **Target:** the conflict is surfaced at install and at `status`, with both repair paths named.
    - **Acceptance:** fixture HOME with an engram file present → install prints the warning; `status` lists it.
    - **Acceptance scenario:** `happy path engram-coexistence` → `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:**
  - New: `plugins/memory/{pi-omp,opencode-kilo}/` (one shared TS source per pair + per-host manifests; kilo copy byte-identical; pi+omp one source), `plugins/cursor/` (v2 hook template + generated rule), `scripts/mnemonic-v1-import.sh`, `mnemonic/cmd/mnemonic` `save-prompt` wiring (one verb + its handler on the existing prompt-store path).
  - Modified: `skillgrid-cli/internal/install/` (memory family + exported manifest/merge package boundary), `plugins/opencode/` + `plugins/pi/mnemonic/` (rewritten/retired per R8), `rules/mnemonic.mdc` (generated), `.agents/skills/mnemonic/SKILL.md`, `.cursor-plugin/plugin.json`.
  - Unchanged: everything else in the `mnemonic/` engine (zero-Go lock); the queued obs + host-plugins changes (they adopt this change's conventions — see Open Questions).
- **Interfaces:**
  - MCP entry (all 5 hosts): key `mnemonic-memory`, command `[<abs-bin>,"memory","mcp","--profile","agent"]` (omp dialect: `command` string + `args`).
  - `mnemonic memory save-prompt` — stdin: plain text | host hook JSON (cursor shape fixture-pinned); stdout: quiet; exit 0; project from cwd; session from the per-host convention (R4).
  - Manifest: `{ host, family: "memory", files: {path: sha256}, entries: [{kind: mcp|hooks-json|agents-md-marker|rule, file, key}], written_at }` at `~/.skillgrid/mnemonic/memory-install.json`.
  - Scratch state: `~/.skillgrid/mnemonic/<project>/.last-session-start.json` (last probe failure + ts), drop counter file — both read by `status`.
  - Installer package boundary (skillgrid-cli): exported `Install/Uninstall/Status(ctx, host, family, Manifest)` + `MergeJSONC` — the seam the obs change migrates onto.
- **Data flow:** prompt → host hook/extension → `save-prompt` CLI → prompt store (per-project SQLite). Observation → agent `mem_*` MCP tool call → host MCP client → `mnemonic memory mcp` child (host-owned) → engine. Compaction → host compaction event → extension injects memory re-injection instruction (context from `memory context` CLI if needed) + FIRST ACTION REQUIRED notice.
- **Error handling:** every capture path is fail-open (verb error/timeout → drop + count, host unaffected). The session-start probe is fail-open with an injected actionable warning. Store-busy exhaustion (WAL + `busy_timeout(5000)` + `withWriteRetry` backoffs in the engine) → counted drop, never a host-visible error. The only fail-closed behavior is "MCP server absent → tools absent" (host-MCP host behavior, surfaced by the probe warning).
- **Dependencies:** none new. pi/omp extension + opencode/kilo plugin: node builtins only (CLI spawn, fs). Import script: bash + sqlite3 CLI.
- **Host verification assumptions (checked at execution, design degrades gracefully):** (1) pi and omp spawn `mcp.json`-configured servers at the session cwd (project resolution depends on it — if not, the affected host falls back to a per-session spawned child for that host only, recorded as a deviation); (2) which hosts surface MCP server `instructions` natively (R10 injection degrades); (3) kilo's auto-discovered plugin dir is `~/.config/kilo/plugins/` (engram's live location — installer writes there; if discovery fails, a config-array entry is the fallback).

## Shootdown Findings

**Anchored hypothesis (as confirmed by the user at the readiness gate, 2026-11-17):** Port engram's per-host memory-plugin architecture onto mnemonic-v2's MCP-stdio memory engine for five hosts: pi+omp share one pi-ExtensionAPI TS source (native `mem_*` tools backed by a per-session spawned `mnemonic memory mcp --profile agent` child, spawn-time contract check, fail-closed on mismatch); opencode+kilo share one hooks-only TS plugin (session registration with subagent guard, prompt capture, `MEMORY_INSTRUCTIONS` transform, compaction recovery; tools via host MCP config); cursor uses its existing MCP config + `hooks.json` + alwaysApply rule. The memory plugin owns capture; one shared Go host-registry installer (`install|uninstall|status`); one new CLI verb `memory save-prompt`; clean v1 cutover (v1-HTTP wiring replaced, `fact_*` dropped, stale MCP entries rewritten); zero Go engine changes otherwise.

Six fresh-context lenses ran against this hypothesis (2026-11-17, ~48 findings). **Verdict: survived-with-notes.** Two Blockers were user-dispositioned before this write (B1 → one-shot import script; B2 → adapter-owns session ID). Three further findings changed the anchored design via user disposition during the same gate: pi/omp transport → host-MCP (removes the spawned child + registerTool + wire client + projection), installer home → skillgrid-cli, and the protocol/contract consequences listed below. Full reports: `agent://Lens{Sre,Cost,Yagni,Ux,Debt,Mig}General`.

| Severity | Lens | Description | Disposition | Note |
|----------|------|-------------|-------------|------|
| 🔴 Blocker | UX | Cursor has no channel to convey the session ID; unknown IDs auto-register ⇒ silent session fork | **resolved** | Adapter-owns (user): cursor uses stable per-host ID `cursor` pinned in the rule; opencode/kilo rewrite; pi/omp inject (R4, R5) |
| 🔴 Blocker | Mig | v1 memory orphaned by project-ID fork (`aiskillgrid.sqlite` 189 obs/422 sessions vs `skillgrid-v2.sqlite`); any fix costs the zero-Go lock | **resolved** | One-shot import script, no Go verb (user) (R9) |
| 🟡 Major | YAGNI | pi/omp transport = a third mechanism (registerTool + spawned child + hand-rolled client + 22 static schemas) | **resolved** | Host-MCP config entry for pi/omp too (user); thin hooks-only extension kept (R1, R2) |
| 🟡 Major | YAGNI | Contract check (278-line generated projection) sized for an independently-released server this design lacks | **resolved** | Projection dropped; session-start liveness probe (R11) + host-MCP fail-open (tools absent) replace it |
| 🟡 Major | YAGNI | Second installer: skillgrid-cli already resolves all 5 hosts | **resolved** | Extend skillgrid-cli (user); exported package boundary for the obs migration (R7) |
| 🟡 Major | YAGNI | Cutover drops `code_*`/`web_*` though `mnemonic code mcp` already exists | **accepted-with-note** | Memory-only scope held (Q1 lock); `code_*`/`web_*` loss recorded as known gaps; the ship-pending code-index change adds the code entry |
| 🟡 Major | YAGNI | "v1-HTTP memory wiring replaced" has undefined blast radius (pi index.ts wires memory+fact+code+ctx+SDD; opencode array registers 6 files) | **resolved** | Per-host retirement inventory pinned (R8) |
| 🟡 Major | YAGNI | pi/omp subagent process model unhandled (eager spawn ⇒ child per subagent) | **resolved** | Subagent guard on all hosts (R2, R3); moot for child count under host-MCP |
| 🟡 Major | UX | `session_id` provenance unspecified on all 5 hosts; engram adapter-owns pattern missing from hook list | **resolved** | Adapter-owns everywhere the host allows; cursor static (R4) |
| 🟡 Major | UX | Fail-closed "one warning" surface + tool behavior during the disabled window unspecified | **resolved** | Fail-open host behavior + injected actionable warning (cause + repair) + scratch record (R11) |
| 🟡 Major | UX | Protocol from two sources on MCP hosts; v1 AGENTS.md marker blocks not in cutover | **resolved** | Single Go source (R10); kilo AGENTS.md marker in the manifest (R8) |
| 🟡 Major | UX | Engram coexistence on kilo: two memory backends, identically-named bare `mem_*`; 3 documents disagree on kilo's plugin path | **resolved** | Installer detects + warns (R12); kilo dir pinned to `~/.config/kilo/plugins/` (engram's live location, verify at execution) |
| 🟡 Major | UX | Tool-name shape differs per host; MCP server key unpinned, existing entry key ≠ server identity | **resolved** | Key `mnemonic-memory` on all hosts (R1); mapping line in protocol host-shell (R10) |
| 🟡 Major | SRE | Fail-closed mismatch leaves no queryable record; no surface reports it | **resolved** | Scratch record + `status` liveness (R11) |
| 🟡 Major | SRE | Agent-visible fail state unspecified (22 tools absent vs erroring) | **resolved** | Host-MCP: tools absent when server down; warning names the cause (R11) |
| 🟡 Major | SRE | Mid-session child death: no respawn/self-heal named | **resolved** | N/A under host-MCP — host owns the server lifecycle; recorded |
| 🟡 Major | SRE | Startup failure taxonomy unpinned (ENOENT vs spawn vs pre-handshake exit vs contract) | **resolved** | Probe reports the cause class; warning text carries cause + repair (R11) |
| 🟡 Major | SRE | Shutdown 2 s drain: no confirmed-delivery record for final writes | **resolved** | N/A under host-MCP (host-managed drain); session `finalize` sent by the extension's end hook before the host tears down |
| 🟡 Major | SRE | opencode/kilo capture transport unspecified; busy-exhaustion drops uncounted | **resolved** | One-shot CLI verbs, fail-open (R3); drop counts surfaced by `status` (R11) |
| 🟡 Major | SRE | `install status`/`doctor` can't answer "working right now"; GUI-PATH case green-but-dead | **resolved** | Absolute binary path in every entry (R1); `status` liveness probe (R11) |
| 🟡 Major | Cost | Per-session context overhead ~1.8–2.8k tokens, ~0.3–0.5k duplicated protocol on 3 hosts | **resolved** | Single protocol source (R10) removes the duplication; per-host budget recorded in blueprint |
| 🟡 Major | Cost | Instructions render at 2,596 runes — over engram's 2,048 production truncation ceiling | **resolved** | Go guard test ≤ 2,048 runes; text trimmed if it fails (recorded Go text edit) (R10) |
| 🟡 Major | Cost | opencode/kilo lifecycle transport: second persistent child vs one-shot CLI | **resolved** | One-shot CLI verbs (R3, R6) |
| 🟡 Major | Cost | Subagent spawn unbounded on pi/omp (K children × 30–80 MB) | **resolved** | Guard (R2); no per-session child under host-MCP |
| 🟡 Major | Debt | Shared behaviors (redaction/truncation/protocol) have no single owner; engram's per-host copies drifted | **resolved** | Go owns normalization + protocol; TS passes raw, host-shell only (R10) |
| 🟡 Major | Debt | pi/omp transport pinned to the memory family; deferred families rework exactly those hosts | **resolved** | Host-MCP is family-agnostic by construction (one entry per family); the extension is memory-scoped (R2) |
| 🟡 Major | Debt | Host-registry installer package home/API unpinned | **resolved** | skillgrid-cli exported boundary (R7) |
| 🟡 Major | Debt | Cutover leaves a second (v1-HTTP) transport convention dead at runtime | **resolved** | Full retirement inventory (R8) |
| 🟡 Major | Debt | Protocol text has up to four homes | **resolved** | Single Go source + diff tests (R10) |
| 🟡 Major | Debt | `__AGENT__` substitution has no v2 consumer | **resolved** | Byte-identical kilo copy; no substitution (R3) |
| 🟡 Major | Debt | Contract-generator sync mechanics unpinned; strict equality = skew ⇒ session-wide outage | **resolved** | Moot (projection dropped, R11 probe) |
| 🟡 Major | Debt | pi/omp transport = second MCP wire-protocol impl in the repo; child-death state unnamed | **resolved** | Moot under host-MCP — the client is gone (R2) |
| 🟡 Major | Debt | pi/omp child-session handling absent | **resolved** | Guard (R2) |
| 🟡 Major | Mig | v2 opening a v1-schema store "opens clean, fails on first use" — contract check can't catch it | **resolved** | Probe's store-read step (R11) catches it at session start |
| 🟡 Major | Mig | Contract couples installed plugin to binary; engine-only upgrade fail-closes until reinstall | **resolved** | Moot under host-MCP (no bundled projection); server identity drift = host-visible broken server, caught by probe |
| 🟡 Major | Mig | Mid-session child death has no named behavior | **resolved** | N/A under host-MCP |
| 🟡 Major | Mig | v1-retirement list incomplete (capture wiring outside the named modules) | **resolved** | Inventory pinned incl. events/compaction/sdd-events/v2sse (R8) |
| 🟡 Major | Mig | Three installer families share paths; manifest adoption/no-clobber semantics unpinned | **resolved** | Tracked-set-only manifests; foreign files never adopted (R7) |
| 🟡 Major | Mig | Stale-MCP rewrites span three JSON dialects (string vs array command, JSONC) | **resolved** | Per-host real-shape fixture tests (R1, R7) |
| 🟡 Major | Mig | v1-era guidance docs outlive the cutover (SKILL.md, .mdc, plugin.json) | **resolved** | Added to the cutover list (R8) |
| 🟢 Nit | YAGNI | Two capture paths for the same data (`save-prompt` per-turn vs `memory save`'s default prompt capture) | **recorded** | Per-turn capture is canonical for the prompt store; save-time capture is a separate observation side effect; no corruption either way |
| 🟢 Nit | YAGNI | The 22-tool list includes the `mem_capture_passive` no-op + long-tail tools | **recorded** | K5 profile is locked (22); description text left as-is (zero-Go lock); deferral seam noted for a later budget change |
| 🟢 Nit | YAGNI | `MEMORY_INSTRUCTIONS` exists twice for opencode/kilo | **resolved** | Single source (R10) |
| 🟢 Nit | UX | Stale v1 MCP entries: make `status` report leftovers the rewrite skipped | **resolved** | `status` lists stale shapes per host (R7) |
| 🟢 Nit | UX | All-22-direct registration trades away engram's deferred-tool pattern | **recorded** | Host-MCP carries all 22 natively; token budget recorded in blueprint; deferral is a later budget change |
| 🟢 Nit | UX | `mem_capture_passive` ships a model-visible "honest no-op" description | **recorded** | Zero-Go lock; no-op is harmless; revisit with the passive-capture wiring |
| 🟢 Nit | UX | Pin the `save-prompt` stdin contract now (raw cursor payload, idempotency) | **resolved** | Pinned (R6), obs Task-3 fixture pattern |
| 🟢 Nit | SRE | No child-process visibility; leak path = session destroyed without shutdown | **resolved** | N/A under host-MCP (host reaps its own server on disconnect) |
| 🟢 Nit | SRE | Contract enforcement asymmetric across the five hosts | **accepted-with-note** | v1-acceptable: probe covers pi/omp; MCP hosts fail open and surface via `status` |
| 🟢 Nit | SRE | Project binding fixed at spawn cwd; cross-project sessions are an explainability risk | **recorded** | Session rows carry directory provenance in the store; satellite sessions only if a host's cwd moves mid-session (none verified) |
| 🟢 Nit | Cost | Out-of-band binary upgrade fail-closes pi/omp until reinstall (engram UX) | **resolved** | Moot (no bundled projection); upgrade = server identity change, caught by probe |
| 🟢 Nit | Cost | Shared registry manifest must own entries per-family (obs writes the same hooks.json) | **resolved** | (host, family) keying (R7) |
| 🟢 Nit | Cost | Mid-session child death + `ctx_*` fate unpriced | **resolved/recorded** | Child death N/A; `ctx_*` dropped, ADR-0044 receipt loss recorded (R8) |
| 🟢 Nit | Debt | Cross-change coupling with the queued router-bootstrap change (separate omp source, pi seam assumption, third installer) | **resolved** | Sequenced: router change runs after this and adopts single-source + skillgrid-cli conventions; coupling recorded in both specs |
| 🟢 Nit | Mig | Task-output passive capture silently descoped from the engram baseline | **recorded** | Descope stated in R8/out-of-scope; rule section deleted (R5) |
| 🟢 Nit | Mig | pi+omp bullet enumerated tools but not the capture hooks | **resolved** | pi/omp hook list enumerated (R2) |
| 🟢 Nit | Mig | Supersession left two R6 folds (pi subagent read-guard; launcher exit-code 78) in limbo | **resolved** | Read-guard: deferred, owner = host-plugins change (pi extension seam). Exit-code 78: N/A (no owned launcher under host-MCP) |

**Overall verdict:** survived-with-notes → **survived** (all Blockers dispositioned; all Majors pinned or recorded above).

Gate: all 🔴 Blocker findings dispositioned before the spec is written — an undispositioned blocker blocks the write. **Passed** (2026-11-17, user dispositions recorded in the table).

## Testing Decisions

- **What makes a good test:** external behavior only — installed file/entry state, stored rows, host-visible text; never TS/Go internals. Fixture-HOME round-trips for the installer; real-shape config fixtures for every dialect.
- **Modules to test:** installer (5 fixture HOMEs: install/re-run/clobber/uninstall + cutover rewrites), pi-omp extension (guard, fail-open, probe warning, compaction text), opencode-kilo plugin (session_id rewrite, parentID guard, byte-identity of the kilo copy), `save-prompt` verb (stdin shapes, idempotency, cwd project, exit codes), import script (fixture v1 store, idempotency, dry-run), protocol diff tests (cursor rule ↔ Go source; host-shell templates).
- **Prior art:** skillgrid-cli installer tests (existing merge/fixture patterns); engram plugin tests (`test/*.test.mjs` — guard, redaction, lifecycle); obs blueprint Task 3 (per-host stdin fixtures) and Task 6 (installer fixture HOMEs) — reuse the shapes.
- **Edge cases:** JSONC with comments + string-vs-array command; foreign MCP entries present; re-install over a hand-edited entry (no-clobber skip + `status` report); store busy (drop count); missing binary (GUI-PATH); engram co-located; subagent session (no row); compaction without prior session (notice still correct); import re-run; kilo plugin-dir discovery failure (array-entry fallback).

## Impact on Global Docs

- `.skillgrid/projectwiki/reference/00-prd.md`: None (agent-facing memory is in-PRD-scope; no metric change).
- `.skillgrid/projectwiki/ASSUMPTIONS.md`: add ADR-0067 index row; add one VERIFIED fact (the v1/v2 store fork: `aiskillgrid.sqlite` 189 obs/422 sessions vs `skillgrid-v2.sqlite`, v1 VIEW vs v2 TABLE, verified 2026-11-17) — load-bearing for R9.
- `.skillgrid/projectwiki/ARCHITECTURE.md`: §12 (Plugin Architecture) — add the 5-host memory plugin family + the extended skillgrid-cli installer; §4 (Command Surface) — `memory save-prompt` verb; §10.1 — note the `mnemonic-memory` MCP entry shape all hosts share.

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.80  | 0.75 | ✅     | "v2 memory on 5 hosts, engram-based" locked from round 1 |
| Boundary Clarity    | 0.75  | 0.70 | ✅     | memory-only scope, cutover boundaries, queued-change seams all explicit |
| Constraint Clarity  | 0.70  | 0.65 | ✅     | zero-Go lock, no-new-deps, serial dev, spec-before-code zone order |
| Acceptance Criteria | 0.75  | 0.70 | ✅     | per-host smoke + fixture round-trips named as the bar |
| **Clarity**         | **0.13** | ≤0.20 | ✅ | all dimensions ≥ minimums; shootdown re-verified (2026-11-17) |

**Interview log:** rounds 1–11 locked Q1–Q11 (see Open Questions for the full list); rounds 12–15 are the shootdown dispositions.

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | Scope: memory-only or all 3 families? | memory-only (`mem_*`); code-index/project-wiki deferred (Q1) |
| 2     | Architecture: engram port or fresh? | engram literal split per host (Q2) — later refined at round 13 to host-MCP transport |
| 3     | v1 handling: coexist or cut over? | clean cutover (Q3/Q5) |
| 4     | Spawn timing: eager or lazy? | eager at session start (Q4) — replaced by host-MCP + session-start probe at round 13 |
| 5     | Installer ownership | shared host-registry installer owned by this change (Q6) — home pinned to skillgrid-cli at round 15 |
| 6     | Child process model | per-session MCP child (Q7) — replaced at round 13 |
| 7     | Tool surface | all 22 agent-profile tools (Q8) |
| 8     | Protocol injection | `MEMORY_INSTRUCTIONS` system transform (Q9) — refined to host-shell + single Go source |
| 9     | Compaction handling | summary → `mem_save(type=learning)` + FIRST ACTION REQUIRED (Q10) |
| 10    | Prompt capture seam | one new CLI verb `memory save-prompt` (Q11) |
| 11    | Confirmation of the 11-point anchored hypothesis | user accept |
| 12    | B1: v1 data fork — verb vs fork vs script | one-shot import script, no Go verb (keeps zero-Go lock) |
| 13    | B2/M1: session-ID channel + pi/omp transport | adapter-owns session ID (engram); host-MCP config entry for pi/omp too |
| 14    | (folded into 13 — transport consequences: contract projection, wire client, child lifecycle dropped) | liveness probe replaces contract gate; host owns server lifecycle |
| 15    | Installer home: mnemonic pkg vs skillgrid-cli vs cmd-local | extend skillgrid-cli's installer |

## Open Questions & Assumptions

- **Assumption (verified at execution, degrades gracefully):** pi and omp spawn `mcp.json` servers at the session cwd. If not: the affected host falls back to a per-session spawned child for that host only, recorded as a deviation in the blueprint.
- **Assumption (verified at execution):** which hosts surface MCP server `instructions` natively — R10 injection degrades to host-shell-only where it does.
- **Assumption (verified at execution):** kilo auto-discovers `~/.config/kilo/plugins/` (engram's live, working location); fallback = config-array entry.
- **Question:** the queued `2026-11-17-skillgrid-host-plugins` change plans its pi module against the v1 `index.ts` seam this change rewrites, ships a separate omp source, and extends the same skillgrid-cli installer — sequence after this change and adopt the single-source + registry conventions (coupling recorded in both specs).
- **Question:** the queued obs change's installer migrates onto the skillgrid-cli boundary this change exports (its recorded follow-up) — the boundary shape (R7) must stay stable across the two executions.
- **Known gaps (recorded, not defects):** `code_*`/`web_*`/`squad`/`ctx_*` tools absent post-cutover until their owning changes; SDD-event capture gap until the obs change lands; concurrent cursor sessions in one project share the `cursor` session row; `mem_capture_passive` is a registered no-op.
- **Locked decisions this change rests on (Q1–Q11):** memory-only scope; engram-architecture per-host split (transport refined round 13); clean v1 cutover; eager session-start behavior (now: host-MCP + probe); shared installer owned here (home: skillgrid-cli); all 22 agent-profile tools; protocol injection (host-shell + single Go source); compaction → `mem_save(type=learning)` + FIRST ACTION REQUIRED; prompts via one new CLI verb.

## Decisions (ADR)

- `.skillgrid/projectwiki/decisions/04-adr-0067-mnemonic-v2-host-memory-plugins.md` — the 5-host memory plugin architecture: unified host-MCP transport (not engram's spawned-child + registerTool), adapter-owns session ID, skillgrid-cli installer home, one-shot v1 data import (zero-Go lock held).

## Terms

- Pointer: `.skillgrid/projectwiki/glossary/01-business-terms.md` / `02-technical-terms.md` — canonical terms live there.
- Introduced/relied by this change (glossary update due at ADR authoring): **host-shell** (the per-host protocol delta a host surface carries, distinct from the Go-authored protocol body), **adapter-owns session identity** (the adapter, not the model, is the authority on `session_id`), **host-registry installer** (the shared skillgrid-cli installer keyed (host, family) with tracked-set-only manifests).
