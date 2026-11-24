# Mnemonic Pi / omp Code Index Plugin — Design

Status: approved design (brainstorming). Next step: implementation plan via `superpowers:writing-plans`.
Related: `2026-11-24-mnemonic-pi-omp-plugin-design.md` (sibling memory plugin), `2026-11-24-mnemonic-phase2-code-indexing-design.md` (code subsystem), plans `2026-11-24-mnemonic-phase2a-core-code-graph.md` et al.
Reference only (do not copy): `tintinweb/pi-gitnexus` (architecture template) and `skillgrid-v2/plugins/pi/mnemonic` (v0 `@devopstales/pi-mnemonic`, HTTP, old `code_*` surface — superseded).

## 1. Goals and decisions

Give Pi and omp (a Pi fork) agents the Phase 2 `mnemonic code` subsystem as **native tools + automatic result augmentation**, modeled on pi-gitnexus: when the agent greps or reads, graph context (callers, callees, tests, contracts, linked memories) is appended to the tool result; deeper analysis is available through registered `code_*` tools, `/codeindex` commands, and bundled workflow skills.

- **Native tools, not MCP**: `pi.registerTool` with `exposure` tiers, same decision as the memory plugin. The `mnemonic code mcp` server remains for other agents (Claude/Cursor/etc.); the plugin does not mount it.
- **Transport**: spawn `mnemonic code call <tool>` (JSON on stdin, envelope on stdout), fail-open, 5 s default / 120 s for `code_index`, at most 4 spawns in flight. Identical contract shape to `mnemonic memory call` so both plugins share one transport design.
- **Capability-gated registration**: `mnemonic code call --schema` dumps the tool schemas present in the running binary. The plugin registers only what exists — 2A-only binaries get 9 tools, 2D-complete binaries get all 16. No plugin release is needed when slices 2B–2D land.
- **Augment verb**: `mnemonic code augment <pattern>` — a new small Go verb (added to Plan A, Task 10) that composes `code_neighbors` + `code_affected` + contracts + `observation_code_links` into one compact (≤ 1.5 KiB) block. Rationale: `code_explore`'s 24 KiB budget is correct for a tool call but far too heavy to inline after every grep; gitnexus solves the same problem with its dedicated `gitnexus augment` verb.
- **Packaging**: `@devopstales/mnemonic-code-pi`, a **sibling** of `@devopstales/mnemonic-memory-pi`. Same internal architecture (`host.ts`, `transport.ts`, `controller.ts`) duplicated, not shared: the memory spec is approved with its own package name and schedule, code-only users don't need memory capture, and a code-index bug must never take down memory. A shared `@devopstales/mnemonic-pi-shared` core is a v0.2 extraction if drift hurts.
- **v0.1 behaviours** (all selected): session lifecycle (binary probe, index-state cache), capability-gated native tools, system-prompt protocol injection, auto-augment hook, `/codeindex` commands + status line + doctor, 4 bundled skills.
- **No index-dir detection**: pi-gitnexus walks up looking for `.gitnexus/`; we instead call `mnemonic code status --json` once per session and after every index run, caching the result. The index lives in `~/.skillgrid/mnemonic/<project>.db` with hash-suffixed names — file naming is not a stable contract, `status` is.

### Deliberate differences from pi-gitnexus

| pi-gitnexus | This plugin | Why |
|---|---|---|
| MCP stdio child (`gitnexus mcp`, idle-reaped) | `mnemonic code call` spawns (one per call) | House pattern from the memory plugin; no JSON-RPC framing in TS; `--schema` enables capability gating; calls are < 50 ms local SQLite, no need for a persistent process |
| `gitnexus augment` writes to **stderr**, 8 s, 8 KiB | `mnemonic code augment` on **stdout** (envelope), 5 s, 1.5 KiB | Consistent JSON contract; local index is fast; augment must stay small or it floods the agent's context on busy greps |
| `repo` parameter on every tool (multi-repo) | `project` / `cwd` parameters (memory-plugin convention) | One `<project>.db` per active root; no multi-repo axis in the code subsystem |
| 7 tools, all always-on | 16 tools across `direct` / `deferred` / admin exposures | Progressive disclosure matches the 2A→2D slice cadence; `tool_search` finds deferred tools on demand |
| No memory linkage | Augment and `code_explore`/`code_read` surface `[Memory: …]` annotations via `observation_code_links` | The differentiator: past decisions/bugfixes attached to the code the agent is about to touch |
| Static snapshot; `/gitnexus analyze` is the only freshness mechanism | Daemon FS watcher auto-syncs (2A); `code_index` remains user-initiated for full/index runs | Staleness is a normal, minor state, not the failure state |
| PolyForm noncommercial license note | None | Own binary |

## 2. Package layout

Location: `plugins/pi/pi-mnemonic-code/` (same repo/placement as the memory plugin).

```text
plugins/pi/pi-mnemonic-code/
├── package.json                 # pi.extensions ["./src/index.ts"], skills ["./skills"];
│                                #   zero runtime deps; peerDeps pi-coding-agent, pi-tui, typebox
├── src/
│   ├── index.ts                 # register(pi): controller + tools + hooks + commands (<200 LOC)
│   ├── host.ts                  # pi/omp event normalisation (mirror of memory plugin)
│   ├── controller.ts            # probe, index-state cache, dedup caches, drop counters; no module globals
│   ├── transport.ts             # spawn `mnemonic code call`, envelope, timeouts, ≤4 concurrency
│   ├── augment.ts               # pattern extraction + runAugment (adapted from pi-gitnexus, proven corpus)
│   ├── status.ts                # status-line text + /codeindex doctor payload
│   └── tools/{schemas,register}.ts
├── skills/
│   ├── codeindex-exploring/SKILL.md
│   ├── codeindex-debugging/SKILL.md
│   ├── codeindex-pr-review/SKILL.md
│   └── codeindex-refactoring/SKILL.md
└── test/                        # node:test, injected fake Transport/host; pattern-extraction corpus
```

- `host.ts` normalises the known deltas (memory spec §1): prompt event → `{kind:"user"|"synthetic"|"revive"}`; real shutdown vs reload; `isSubagent(ctx)`; and for this plugin specifically **which result-mutating event the host fires** (`tool_result` with a content-modifying result, falling back to `tool_execution_end` — see §6 and §10).
- `typebox` / `pi-tui` imported lazily in `tools/register.ts` / command menus only. If missing: plain JSON-schema parameters, notify-based menus. The plugin must never fail to load.

## 3. Go-side contract (additions to Plan A, Task 10)

```text
mnemonic code call <tool_name> [--timeout 5s]
  stdin : JSON object of tool arguments (same schema as the MCP tool)
  stdout: {"ok":true,"result":{...}}
  failure: {"ok":false,"error":{"code":"...","message":"...","retryable":bool}} + exit code 2-11
mnemonic code call --schema      # dumps all registered code tool schemas (JSON)
mnemonic code augment <pattern> [--max-bytes 1536]
  stdout: compact context block (see §6.2); exit 3 when no index (hook skips silently)
```

- One Go tool registry for the code subsystem (name, JSON schema, handler, tier) feeding MCP (existing), `code call`, and `augment` composition — mirrors the memory plugin's `internal/tools` arrangement.
- `augment` composes, in priority order, all within the byte budget: top-5 callers (name + `file:line` + confidence), top-5 callees, related test symbols, contract line if 2C is active, top-3 linked memories (`[Memory: <type>] <title> (uid)`), weakest-hop confidence footer. Pattern resolution reuses the `code_search` lexical legs (name + segments) and takes the top symbol; ties → top-2 symbols, clearly labeled.
- Binary resolution: `MNEMONIC_BIN` → `mnemonic` on `PATH` → `~/.skillgrid/mnemonic/bin/mnemonic`. Missing binary = offline mode (tools return a clear error, hooks no-op).
- Concurrency: at most 4 spawns in flight; hook calls fire-and-forget with a bounded queue of 8 (overflow dropped + counted for doctor); tool calls awaited.
- `augment` and `call` are read-only against the DB; they never trigger writes (no index auto-start), so the hook cannot race an in-flight `code_index` writer beyond SQLite's normal locking (fail-open treats `SQLITE_BUSY`/`INDEX_BUSY` as "skip").

## 4. Tool surface (capability-gated)

Registered from the `--schema` dump; expected tiers per slice (the plugin never hardcodes the list — `--schema` is the source of truth; the table is what v0.1 *should* see):

| Tool | Exposure | Slice | Description contract (prompt guidance) |
|---|---|---|---|
| `code_explore` | direct | 2A | "PRIMARY tool — call FIRST before reading files in unknown areas; budgeted RRF + PageRank source with call trails, staleness and linked memories" |
| `code_read` | direct | 2A | "Fetch indexed source after search/explore narrows the location; `mode` slice/skeleton/symbol" |
| `code_impact` | direct | 2A | "Blast radius before editing: `will_break` (hop 1) vs `likely_affected`, confidence-tiered" |
| `code_affected` | direct | 2A | "Map git diff (working/staged/base_ref) to changed symbols, impacted callers, affected tests, flows — the PR-review entry point" |
| `code_status` | direct | 2A | "Freshness (not_indexed/fresh/stale/corrupt), counts, capabilities, watch state — check before searching" |
| `code_index` | direct | 2A | "Incremental or full sync; user-initiated only (never auto-run); 120 s tool timeout" |
| `code_search` | deferred | 2A | "Compact ranked hits: hybrid/lexical/semantic/structural — for narrow questions where explore's budget is overkill" |
| `code_neighbors` | deferred | 2A | "1-hop structural queries: callers/callees/implementors/hierarchy/tests" |
| `code_rename` | deferred | 2A | "TWO-STEP: dry_run (default) → review diff → apply with returned `plan_token`; `PLAN_STALE` on drift" |
| `code_topology` | deferred | 2B | "overview/communities/god_nodes/flows/cycles/path — architecture questions start here" |
| `code_contracts` | deferred | 2C | "routes/queries/events/tools/docs/configs — cross-boundary questions (which handler owns this URL?)" |
| `code_pdg` | deferred | 2D | "CFG/CDG/reaching-defs slice for one symbol — control-flow bugs" |
| `code_taint` | deferred | 2D | "Source→sink traces across call chains — security and injection bugs" |
| `code_watch` | deferred (admin) | 2A | "Inspect/manage the daemon FS watcher" |
| `code_backfill_embeddings` | deferred (admin) | 2A | "Force vector backfill" |
| `code_lsp_sync` | deferred (admin) | 2D | "Targeted LSP edge promotion" |

- `direct` = always in the prompt (7 tools, comparable to gitnexus' 7 always-on). `deferred` = registered but prompt-hidden, discoverable via `tool_search` (verified API: `exposure?: ToolExposure`, values `direct`/`codemode`/`deferred` in the installed pi-coding-agent build). Admin tools additionally `defaultActive: false`.
- Tool names are the bare subsystem names (`code_explore`, …), matching the MCP server — no `mnemonic_` prefix (memory-plugin convention: `mem_*`).
- Supersedes the v0 `@devopstales/pi-mnemonic` `code_*` tools: when the new package ships, the four legacy code tools are removed from the v0 package (clean cutover, no coexisting duplicate names).

## 5. Commands, status & doctor

| Command | Description |
|---|---|
| `/codeindex` | Main menu (status, Index, Settings, Help) — pi-tui component, notify fallback |
| `/codeindex status` | Freshness, counts, capabilities, watch state, session augment stats |
| `/codeindex index` / `index full` | Run `mnemonic code index [--full]` (command path: unbounded, progress notify; tool path: 120 s) |
| `/codeindex explore <q>` | Manual `code_explore` |
| `/codeindex search <q>` / `impact <sym>` / `affected` / `topology <mode>` | Manual lookups |
| `/codeindex on` / `/codeindex off` | Enable/disable auto-augment (tools unaffected); resets to on on session switch |
| `/codeindex settings` | Toggle, augment timeout, max patterns, max bytes, binary command |
| `/codeindex help` | Command reference |

- Status line (when a UI exists): `ctx.ui.setStatus("codeindex", "code ✓ 1.2kF 8.4kS | stale | offline")` — counts from the cached `status`; freshness word; `offline` when binary missing.
- `/codeindex doctor`: binary path/version, schema tool count vs registered count, freshness + `unresolved_count`, watch state, last 5 errors, augment drop counter. Falls back to a `code_doctor` tool if the host has no command API.

## 6. Auto-augment hook (the core value)

### 6.1 Triggers

Registered on the host's result-mutating event (`tool_result`; `tool_execution_end` fallback — §10). Mirrors pi-gitnexus' extraction logic (port the ~150 LOC extractor + its test corpus; it is proven against real agent traffic):

| Tool | Pattern used |
|---|---|
| `grep` | Longest identifier literal extracted from the search pattern |
| `bash` with grep/rg | First non-flag arg after `grep`/`rg` (quote/pipe-aware tokenizer) |
| `bash` with cat/head/tail | Target filename (code extensions only) |
| `bash` with find | Value of `-name`/`-iname` |
| `glob`/`find` | Glob basename |
| `read` | File being read (code extensions only) |
| Any grep/bash result | Filenames extracted from `path/file.ext:line:` result lines (≤2 secondary) |

Limits: up to 3 fresh patterns per result in parallel; identifiers ≥3 chars; ≤1.5 KiB per pattern; combined block capped at 8 KiB. Appended as `\n\n---\n[CodeIndex: <pattern>]\n<body>\n---`.

### 6.2 Augment payload shape (≤1.5 KiB)

```
callers (3/5):
  HandleUpload   internal/handler/upload.go:88   (EXTRACTED)
  HandleBackup   internal/handler/backup.go:41   (INFERRED)
callees (5):
  parseQuery     internal/net/url.go:210         (EXTRACTED)
  ...
tests: TestParseHTTPRequestURL (internal/handler/upload_test.go:45)
contract: POST /v1/upload (SYNTHESIZED)
memories: [decision] upload auth via signed URLs (uid a1b2c3)
weakest hop: EXTRACTED
```

### 6.3 Guard rules

- **Session dedup**: `augmentedCache` (had results) + `emptyCache` (returned empty) per pattern, case-insensitive; both cleared on session start and after a successful `/codeindex index`.
- **Pause conditions**: toggle off; `status` = `not_indexed`/`corrupt`; a `code_index` run is in flight (analyze-guard, same as gitnexus); subagent session (main agent only, halves spawn pressure).
- **Stale**: still augment (the daemon watcher keeps the index mostly fresh), but the first augment of a session in `stale` state appends one notice line: `index stale — run /codeindex index for a full refresh`.
- **Fail-open**: spawn error, timeout, malformed envelope, `INDEX_BUSY` → pattern goes to `emptyCache`, hook returns the original result unmodified; nothing ever throws into the host; errors counted for doctor.

## 7. System-prompt protocol injection

On `before_agent_start` (user prompts only), when an index is present and `MNEMONIC_CODE_PI_INJECT != 0`:

```
[CodeIndex active] Graph context is appended to grep/read results.
- Unknown area? Call code_explore FIRST — it returns budgeted source + call trails.
- Before editing: code_impact (or code_affected for git changes); before merging: code_affected.
- code_rename is two-step: dry_run → plan_token → apply.
- Confidence tiers: trust LSP_RESOLVED/EXTRACTED; question INFERRED/AMBIGUOUS.
- Stale index: run /codeindex index.
```

## 8. Bundled skills (on-demand; only the description is in context)

| Skill | Workflow (summary) |
|---|---|
| `codeindex-exploring` | `code_topology overview` → communities/god_nodes → `code_explore <concept>` → `code_read`; check `[Memory: …]` annotations for prior decisions before deep-diving |
| `codeindex-debugging` | `code_explore "<error text>"` → suspect symbol → `code_taint` (2D) for injection/flow bugs, `code_pdg` for control-flow bugs → `code_affected --staged` for tests to run → linked memories (`bugfix_in`) for past fixes of the same area |
| `codeindex-pr-review` | `code_affected` on the diff → `code_impact` on changed hop-1 symbols → `code_topology cycles` for new cycles → verdict with confidence tiers |
| `codeindex-refactoring` | `code_impact` first → `code_rename` dry_run → review unified diff + `unverified_candidates` → apply with `plan_token` → verify via `code_affected` + build |

No separate impact-analysis skill: `code_impact` is a direct tool, so the workflow is one line inside pr-review/refactoring (gitnexus needed 5 skills partly because its tools were all behind a single `context` verb).

## 9. Session lifecycle

| Hook | Behaviour |
|---|---|
| `session_start` | Skip subagents. Resolve shell PATH (login-shell probe, gitnexus' `resolveShellPath` pattern — nvm/fnm). Probe binary. `code status` → index-state cache. Reset dedup caches, counters, `stale-notice-sent`. Notify: index absent → "run /codeindex index"; stale → informational; offline → warning. |
| `session_shutdown` | Real exit only (host adapter decides). Bounded 2 s drain of in-flight spawns. No daemon lifecycle management — the daemon is independent, watcher state is the user's choice. |
| `before_agent_start` | Protocol injection (§7), user prompts only. |
| `tool_result` | Auto-augment (§6). |
| `session_compact` | No-op in v0.1 (the memory plugin owns compaction recovery; revisit in v0.2 whether augment caches should persist across compaction — they are per-session by design, so no-op is correct for now). |

## 10. omp compatibility

Shares the memory plugin's Task 0 spike (same `host.ts` shape, same packaging mechanics, same `~/.omp/plugins` install path). Code-specific spike items, recorded in the shared `docs/.../omp-compat.md`:

1. `tool_result` fires on omp with a **content-modifying result** (augment needs the result form, not just the fire-and-forget `tool_execution_end`). If omp only offers `tool_execution_end`, fall back to it and append via the same mechanism the memory plugin uses for hidden messages — decision recorded at spike time.
2. `exposure: "deferred"` tools are discoverable via `tool_search` on omp.
3. pi-tui menu availability (fallback: notify-line menu).

Outcome rule (same as memory spec): `host.ts`-absorbable differences → no separate omp package.

## 11. Testing

- Unit (`node --test`, type stripping): pattern extraction (port gitnexus' corpus — regex literals, bash tokenizer, read_many-style batches), transport (stub script: canned JSON / exit-3 / hang / malformed), controller state machine (dedup, pause, stale-notice, drop counter), capability-gated registration against two fixture `--schema` dumps (2A-only vs 2D-complete).
- Contract: TypeBox schemas vs `mnemonic code call --schema` snapshot against the real Go binary in CI (skipped if absent) — same rule as the memory plugin.
- Fail-open: missing binary, timeout, malformed JSON, no index, `INDEX_BUSY`, corrupt DB → hook never rejects, tools return readable errors.
- E2E smoke (manual, documented): small polyglot fixture repo → `/codeindex index` → grep a symbol in pi/omp, assert `[CodeIndex: …]` block appears with correct callers → `code_explore` → `code_rename` dry_run → apply → re-augment reflects the new name.
- Banned: source splicing and `new Function` sandboxes (same rule as the memory plugin).

## 12. Rollout order

1. Go (Plan A, Task 10 additions): `code call` + `--schema` + `code augment` (~300 LOC; reuses resolver + CSR + `observation_code_links`).
2. Plugin core: transport, host, controller, pattern extraction.
3. Tools (capability-gated registration).
4. Auto-augment hook.
5. Commands + status line + doctor.
6. Skills.
7. Packaging (`@devopstales/mnemonic-code-pi`) + docs; v0 `pi-mnemonic` code tools removed (clean cutover); `mnemonic setup pi|omp` (Plan 5) installs both packages.

## 13. Risks / open items

- `tool_result` mutability on omp is unverified until the spike (§10.1).
- Augment noise: prose greps yield identifier false-positives; mitigated by the ≥3-char identifier rule, `emptyCache`, per-result caps, and the `/codeindex off` toggle.
- `code_explore`'s 24 KiB default budget is large for a tool result; the plugin passes through the server default and the `budget` parameter — no plugin-side truncation (server owns the format).
- Both plugins depend on the `* call` verbs shipping (memory: Plan 2; code: Plan A). Until then only stubbed transports can be tested; both plugins' rollouts are gated on the same Go work, which is one small shared task.
- Duplicated `host.ts`/`transport.ts` across the two packages (~300 LOC twice): accepted for v0.1 independence; extract a shared package at v0.2 if the two diverge in incompatible directions.
- Alternative considered and rejected: MCP-stdio transport (pi-gitnexus style). It would let v0.1 work against the installed 0.2.0 binary today, but costs a JSON-RPC client in TS, loses `--schema` capability gating, and duplicates the transport the memory plugin already defines. Keep as a v0.2 fallback option if `code call` slips.
