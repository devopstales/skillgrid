# Mnemonic-v2 Observation Layer (gryph-style agent event capture) — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-17; brainstorming output — pending blueprint)
> **Queued behind:** `2026-11-04-mnemonic-v2-memory-plugins` (serial development; NOT current_change).

## Problem / Intent

mnemonic-v2 (the `mnemonic/` Go module) records what agents *remember* (observations,
prompts, trails) but not what agents *do*. There is no audit trail of tool calls,
commands run, files touched, or session lifecycle in v2 — the v1 module had one
(`session_events`, migration 040, on `skillgrid serve :7438`), but v2 dropped the
HTTP serve (`mnemonic serve` is a deprecated no-op; transport = MCP stdio only,
ADR-0004-transport-mcp) and did not port the event layer.

Reference design: **safedep/gryph** (Go, ~183★) — "security layer for AI coding
agents". Its observation plane (this change ports that plane, not the policy
plane): lightweight per-agent hooks exec the gryph binary with a JSON payload on
stdin (no daemon), events land in a local SQLite DB, and a CLI surface
(`logs` / `query` / `sessions` / `session <id>` / `stats` / `export` / `retention`
/ `install` / `status` / `doctor`) reads them. Privacy by construction:
sensitive-path detection (path logged, content never stored), `<private>`-tag
stripping, preview truncation, full-content SHA-256 hashing, 90-day retention.

User direction (2026-11-17): build this observation layer **for mnemonic-v2**,
similar to gryph — install plugins for **pi, omp, opencode, kilo, cursor**; store
events in **SQLite**; add **observation commands to the mnemonic CLI**.

## Purpose & Success Criteria

- **Purpose:** after any supported agent (pi, omp, opencode, kilo, cursor) does
  work in a project, `mnemonic obs logs` / `obs query` / `obs sessions` answer —
  from the v2 per-project SQLite store — what the agent read, wrote, ran, and in
  what order. Local-first, no daemon, fail-open.
- **Success criteria:**
  1. One command per host installs the capture hook/plugin; a real session on any
     of the 5 hosts produces rows in `<project>.sqlite` `obs_events`.
  2. `mnemonic obs logs|query|sessions|session|stats|export` read those rows with
     gryph-parity filters (agent, action, file, command, session, since).
  3. Privacy contract holds: no `<private>` content, no sensitive-file content,
     previews truncated, full content only as hash.
  4. A missing/broken `mnemonic` binary never blocks an agent (fail-open, exit 0).
- **Out of scope (this change):**
  - **Policy plane** (block/warn/guide YAML rules, `obs policy test`) — gryph's
    headline feature, v1 ADR-0021's domain. Phase 2. The design keeps the door
    open: pre-tool hooks, exit-code-2 channel, `result_status` column.
  - Dashboard/HTTP surface (v2 dropped the dashboard; CLI is the surface).
  - MCP `obs_*` tools (CLI-only in v1 of this layer).
  - Cross-project central database (v2 architecture is per-project stores).
  - Token/cost tracking (v1's `/sessions/{id}/usage` + pricing.yaml; separate concern).
  - Sync/export of events (local-only audit, like `session_trails`).

## Context — what exists today (evidence, 2026-11-17)

### gryph's model (the reference)

- Hook → `spawn("gryph", ["_hook", "<agent>", "<hookType>"])`, JSON payload on
  stdin, fail-open, 30s timeout. Exit codes: 0 = allow, 1 = error (allow + log),
  2 = block (pre-tool policy only).
- Live evidence on this host (gryph already installed):
  - **omp**: `~/.omp/agent/extensions/gryph-hooks.ts` — a **pi-ExtensionAPI**
    extension (`pi.on("session_start" | "session_shutdown" | "tool_call" | "tool_result")`),
    `spawnSync` to the binary. omp (v18.16.0, pi-based, `PI_*` env) **auto-discovers
    `~/.omp/agent/extensions/*.ts`** (herdr's integration file sits there too:
    "add custom hooks/plugins beside this file instead of editing it").
  - **cursor**: `~/.cursor/hooks.json` entries `"command": "gryph _hook cursor <hookType>"`
    (afterAgentResponse, afterAgentThought, afterFileEdit, …).
  - **opencode**: `~/.config/opencode/plugins/gryph.js`.
- CLI surface (README): `install [--dry-run] [--agent]`, `uninstall [--purge]`,
  `status`, `doctor`, `logs [--today|--agent|--follow|--format json]`,
  `query --action|--file|--command|--session|--since|--count`, `sessions`,
  `session <id> [--show-diff]`, `diff <event-id>`, `export [--since|-o|--export-profile]`
  (JSONL, `$schema` per line, digest-only export profile), `stats [--since|--agent]`,
  `retention status|cleanup [--dry-run]`, `config show|get|set|reset`.
- Privacy: logging levels minimal/standard/full; sensitive files (`.env`, `*.pem`,
  `*.key`, `*secret*`, `.ssh/**`, `.aws/**`) — logged, content never stored;
  redaction of passwords/tokens; content labels (classes, size, pre-redaction
  SHA-256, flags); 90-day default retention.

### v1 observation layer (this repo, `skillgrid-cli` — the port source)

- `session_events` table (migration 040): per-project store, columns include
  `session_id, project, sequence, action_type (tool_call|session_start|
  session_end|checkpoint|…), tool_name, tool_type (shell|file_read|file_write|
  other), path, command, result_status (success|blocked|warned|guided), timestamp`;
  `sessions` extended with `from_commit/to_commit` + counters.
- 90-day retention pass (`harnessconfig.DefaultRetentionDays`).
- HTTP ingest on `skillgrid serve :7438`: `/sessions`, `/sessions/{id}/end`,
  `/prompts`, `/sessions/{id}/usage`, `/policy/evaluate`, `/events`, SSE frames.
- Host adapters: Cursor `~/.cursor/hooks.json` → staged shell hooks →
  `hooks/tool-call-capture.js` (shared node script; stdin JSON;
  `mapToolType`, `sensitivePath`, `stripPrivateTags`, MAX_PREVIEW=500,
  private-tools allowlist env); OpenCode/Kilo native TS plugins
  (`plugins/opencode/skillgrid-events.ts`; Kilo = copy of the same source);
  Pi extension `plugins/pi/mnemonic/skillgrid-events.ts` (session + prompt
  capture). All fail-open; only the policy gate may block (ADR-0021).
- CLI: `skillgrid logs`.

### v2 substrate (`mnemonic/` module — the target)

- **Store:** per-project SQLite at `~/.skillgrid/mnemonic/<project>.sqlite`
  (pure-Go modernc, embedded migrations `001..004, 100, 200..202` — next number
  for this change: **300**). Project binding chain: explicit > env > git toplevel
  > basename (`core/project.Detector`).
- **Tables relevant here:** `sessions` (id, project, directory, started_at,
  ended_at, title, summary — auto-registered by the save transaction),
  `session_trails` (migration 002: append-only per-session sequence log —
  memory *saves* only; v1 `session_events` pattern, ported), `user_prompts`
  (+ FTS; `mem_save_prompt` MCP tool), `observations` (**v2 "observation" =
  typed memory entity** — naming collision, see Decisions D1),
  `projectwiki_events` (migration 202 — the in-store event-table precedent).
- **CLI:** family-first tree (`memory | code | projectwiki | sync | trail |
  doctor | setup`), stdlib-flag dispatch in `mnemonic/cmd/mnemonic/main.go`,
  `--json` = the K6 envelope seam (CLI and MCP byte-identical). `setup <agent>`
  is a placeholder ("delegates to the agent-plugin change").
- **Transport:** MCP stdio only; **no HTTP server** — which makes gryph's
  exec-the-binary ingest the natural transport (Resurrecting a daemon is the
  larger departure; see Decisions D2).
- **In-flight predecessor:** `2026-11-04-mnemonic-v2-memory-plugins` (current
  change, execution) lands the pi + opencode plugin families (memory /
  code-index / knowledge) — this change's obs plugins are a **4th family** per
  host, queued behind it; it also owns the `mnemonic setup` installer
  machinery (manifest-tracked install/uninstall/status, never clobbers foreign
  entries) which this change reuses for `obs install`.

### Host hook mechanisms (install targets, per-host evidence)

| Host | Mechanism | Install target (this host) | Events available |
|---|---|---|---|
| **pi** | TS extension folder, `pi.extensions` manifest, `pi.on(...)`; spawn (async, fire-and-forget) for non-blocking hooks, spawnSync 30s cap for `tool_call` (gryph `gryph-hooks.ts` shape) | `<pi-agent-dir>/extensions/<name>/` (default `~/.skill-pi/agent/extensions/`; `SKILL_PI_AGENT_DIR` honored; launcher resolves manifest `pi.extensions`, else `index.ts`/`index.js` — no settings edit) | gryph parity: `session_start`, `tool_call` (pre, input, exit-2 blocking), `tool_result` (post, output/status), `session_shutdown` (quit = end) **+ extension:** `before_agent_start` user prompts (gryph's pi/omp hook has no prompt capture; v1 `/prompts` parity — see R1) |
| **omp** | pi-compatible ExtensionAPI; **single .ts file auto-discovered** | `~/.omp/agent/extensions/mnemonic-obs.ts` (gryph-hooks.ts + herdr precedent) | same as pi (same API) |
| **opencode** | native TS plugin, direct `.ts` discovery (OpenCode v2), V1+V2 dual entrypoint (`@opencode/plugin` bundled dep; V2 tool hooks pass one mutable `event`); `cwd` from the plugin's `directory` param; 5s exec cap, only `tool.execute.before` re-throws on exit 2 (gryph `gryph.js` shape) | `~/.config/opencode/plugins/mnemonic-obs.ts` | gryph parity: `tool.execute.before/after` (pre carries `args`, post carries `title/output/metadata`) + `session.created` / `session.idle` / `session.error` (no other session events; no prompt hook — see R1) |
| **kilo** | same TS plugin as opencode, **copied** + registered in kilo's own config file | `~/.config/kilo/plugin/mnemonic-obs.ts` + config registration | same as opencode |
| **cursor** | `~/.cursor/hooks.json` merge, **direct binary command** per entry — `mnemonic obs _ingest cursor <hookType>` (gryph's shape: no shell wrapper, no node script — v1's script existed only because its transport was HTTP fetch) | `~/.cursor/hooks.json` (merge, foreign entries preserved: gryph/orca/herdr stay) | **full gryph 20-hook set** (full copy, D6): `preToolUse`, `postToolUse`, `postToolUseFailure`, `beforeShellExecution`, `afterShellExecution`, `beforeMCPExecution`, `afterMCPExecution`, `beforeReadFile`, `beforeTabFileRead`, `afterTabFileEdit`, `afterFileEdit`, `beforeSubmitPrompt` (**prompt capture** — gryph's only prompt hook), `sessionStart`, `sessionEnd`, `stop`, `preCompact`, `afterAgentResponse`, `afterAgentThought`, `subagentStart`, `subagentStop` |

## Architecture

```
 pi / omp / opencode / kilo TS plugins          cursor hooks.json (20 entries)
        |  spawn(mnemonic, [obs,_ingest,<agent>,<event]), JSON stdin
        |  (fail-open, per-host cap, exit-2 channel reserved)     | direct: <bin> obs _ingest cursor <hookType>
        v                                                        v
              mnemonic obs _ingest <agent> <event>   (hidden verb, stdin JSON)
                       |
                       | normalize: tool_type map, <private> strip,
                       | truncation (500), sensitive-path gate, sha256(content)
                       v
        project store (~/.skillgrid/mnemonic/<project>.sqlite)
        project resolution: payload.project > git detector(event.cwd) > basename
        tx: auto-register sessions row (v2 SaveEvent pattern) + INSERT obs_events
                       ^
                       | read
              mnemonic obs <logs|query|sessions|session|stats|export|prune|status|install|uninstall>
```

- **New Go package** `mnemonic/internal/obs/` — event model, normalization
  (the single source of truth for redaction/truncation/mapping — the v1 logic
  lived in JS; v2 moves it to Go so CLI and ingest share it), store ops
  (ingest, query, sessions, stats, export, prune), installer.
- **Migration `300_obs_events.sql`** — additive, per-project store
  (precedent: `projectwiki_events` 202; local-only, never synced):

  ```sql
  CREATE TABLE obs_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    session_id TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    agent TEXT NOT NULL,
    action_type TEXT NOT NULL,        -- normalized: session_start|session_end|tool_call|tool_result|prompt|turn_end|compact|agent_response|agent_thought|subagent_start|subagent_stop
    tool_name TEXT,
    tool_type TEXT,                   -- shell|file_read|file_write|other
    path TEXT,
    command TEXT,                     -- truncated 500
    input_preview TEXT,               -- truncated 500, <private> stripped
    output_preview TEXT,
    result_status TEXT,               -- success|error|blocked (blocked: phase 2)
    error TEXT,
    duration_ms INTEGER,
    content_hash TEXT,                -- sha256 of full input/output, pre-truncation
    prompt TEXT,                      -- action_type=prompt; truncated 2000 (v1 MaxPromptLength)
    occurred_at TEXT NOT NULL DEFAULT (datetime('now'))
  );
  CREATE INDEX idx_obs_events_project_time ON obs_events(project, occurred_at);
  CREATE INDEX idx_obs_events_session ON obs_events(session_id, id);
  CREATE INDEX idx_obs_events_agent ON obs_events(agent, occurred_at);
  CREATE INDEX idx_obs_events_action ON obs_events(project, action_type, occurred_at);
  ```

- **CLI family `obs`** (new top-level, `mnemonic/cmd/mnemonic/obs.go`, dispatched
  from `main.go` switch; CLI-only in v1 — like `trail`/`doctor`, direct store
  access, no MCP registration):

  | Verb | Grep parity | Notes |
  |---|---|---|
  | `obs logs [--since 24h\|--today] [--agent] [--session] [--action] [--limit] [--format json]` | `gryph logs` | default: last 24h, human table |
  | `obs query --action\|--file\|--command\|--session\|--agent\|--since [--count]` | `gryph query` | `--file` glob, `--command` prefix/glob |
  | `obs sessions` | `gryph sessions` | joins `sessions` (title, started/ended) + event counts |
  | `obs session <id>` | `gryph session` | ordered event detail |
  | `obs stats [--since] [--agent]` | `gryph stats` | counters by agent/tool_type/action (no TUI in v1; table output) |
  | `obs export [--since] [--agent] -o file` | `gryph export` | JSONL, one event per line |
  | `obs prune --retention 90d [--dry-run]` | `gryph retention cleanup` | 90d default (v1 precedent) |
  | `obs status` | `gryph status` | per-agent install state + store event counts |
  | `obs install [pi\|omp\|opencode\|kilo\|cursor\|all] [--dry-run]` | `gryph install` | reuse memory-plugins setup machinery |
  | `obs uninstall [agent\|all] [--dry-run]` | `gryph uninstall` | removes exactly what install created |
  | `obs _ingest <agent> <event>` | `gryph _hook` | **hidden** hook-facing verb; stdin JSON; exit 0 allow / 1 error(allow) / 2 reserved(block, phase 2) |

- **Plugin sources** (repo-truth under `plugins/obs/`; 3 TS sources serve 5
  hosts):
  - `plugins/obs/pi/` — `mnemonic-obs.ts` + `package.json` (`pi.extensions`
    manifest). **pi** installs the folder (`<pi-agent-dir>/extensions/mnemonic-obs/`);
    **omp** installs the single `.ts` (auto-discovered dir). One source, two targets.
  - `plugins/obs/opencode/` — `mnemonic-obs.ts` (V1+V2 dual entrypoint, the
    `skillgrid-events.ts` `importPlugin()` pattern). **opencode** installs direct;
    **kilo** installs the copy + config registration.
  - `plugins/obs/cursor/` — `hooks-obs-cursor.json` only: one entry per hook
    type with `command: "<mnemonic-bin> obs _ingest cursor <hookType>"`
    (gryph's shape — the binary is the hook; no shell wrapper, no node script).
    `<mnemonic-bin>` = the install-time absolute path (falls back to bare
    `mnemonic` on PATH when installed from a checkout).
  - Every hook: fail-open (missing binary / timeout / parse error → exit 0,
    stderr note only), 30s cap on the pi/omp `tool_call` path, 5s cap on the
    opencode `tool.execute.before` path (gryph per-host caps), binary via
    `MNEMONIC_BIN` env else the installed path / PATH.
  - Per-host event wiring (requirements R1): the per-host sets in the table
    above — gryph parity everywhere, with prompt capture as the only
    extension (pi/omp `before_agent_start` user messages; cursor
    `beforeSubmitPrompt` — which is gryph's own cursor prompt hook).

## Requirements

- **R1 — Host capture (5 hosts), gryph parity + one extension.**
  `obs install` (re)installs the capture adapter for pi, omp, opencode, kilo,
  cursor individually or `all`. Event surface = **gryph's installed set
  per host** (the 2026-11-17 evidence table above), normalized into the
  `action_type` vocabulary:
  - **pi/omp:** `session_start`, `tool_call`/`tool_result` pairs per tool use
    (pre input / post status+duration), `session_end` (shutdown-quit),
    **+ extension** `prompt` on `before_agent_start` (user messages only).
  - **opencode/kilo:** `tool.execute.before`/`after` pairs, `session.created`
    → `session_start`, `session.error` (and `session.idle` → `turn_end`), no
    prompt hook (gryph parity — prompts on these hosts stay the memory
    plane's `mem_save_prompt` job).
  - **cursor:** the full 20-hook set, mapped: pre/before* → `tool_call`,
    post*/after*/postToolUseFailure → `tool_result` (failure = `error`
    status), beforeSubmitPrompt → `prompt`, sessionStart/End →
    `session_start`/`session_end`, stop → `turn_end`, preCompact → `compact`,
    afterAgentResponse/afterAgentThought → `agent_response`/`agent_thought`,
    subagentStart/Stop → `subagent_start`/`subagent_stop`.
- **R2 — SQLite storage.** Events persist to the v2 per-project store via
  migration `300_obs_events.sql` (schema above); ingest auto-registers the
  `sessions` row in the same transaction (FK always resolves — v2 SaveEvent
  pattern); project resolves from payload > git detector > basename.
- **R3 — No-daemon transport.** Ingest is the hidden CLI verb
  `mnemonic obs _ingest <agent> <event>` reading a JSON envelope from stdin
  (gryph `_hook` model). No HTTP server is introduced or required.
  Envelope: `{session_id, cwd, agent, event, tool_name, tool_call_id, input,
  output, status, error, duration_ms, prompt?, project?}`.
- **R4 — CLI observation surface.** The verb table above, gryph-parity
  filters, `--format json` on `logs`/`query`/`sessions`/`session`/`stats`.
- **R5 — Privacy.** `<private>…</private>` stripped from all stored text;
  previews truncated (tool 500 / prompt 2000 chars); sensitive paths
  (`(^|/)\.(git|ssh|aws|config)(/|$)`, `.env`, `*.pem`, `*.key`, `*secret*`)
  store the path and **no** content; `content_hash` = sha256 of the full
  pre-truncation value; private-tools allowlist env honored
  (`SKILLGRID_MNEMONIC_PRIVATE_TOOLS` — input/output previews suppressed for
  listed tools).
- **R6 — Install lifecycle.** `install`/`uninstall` per agent + `all`,
  `--dry-run`, manifest-tracked (reuse the memory-plugins setup machinery):
  merges into existing hook configs, never clobbers foreign entries
  (e.g. gryph's own entries, herdr, orca), backs up before modifying,
  uninstall removes exactly what it created, idempotent re-run is a no-op.
- **R7 — Retention.** Default 90-day window (v1 `DefaultRetentionDays`
  precedent); `obs prune [--dry-run]` deletes only `obs_events` rows older
  than the window (sessions rows and all other tables untouched).
- **R8 — Fail-open everywhere.** Missing/renamed `mnemonic` binary, store
  lock, or parse error in any hook → the agent's tool call proceeds
  (hook exit 0, stderr note only). A capture bug can never freeze an agent.
- **R9 — Observability of the installer.** `obs status` reports per-host:
  adapter present/missing/stale (content hash vs installed manifest), and
  store-side event counts.

## Approaches Considered

- **Exec-the-binary ingest (chosen)** — gryph's model; matches v2 no-serve
  reality; hook cost = one short-lived process + one SQLite write; no
  lifecycle/daemon ownership.
- **HTTP endpoint (rejected)** — resurrect a v2 serve (or reuse v1 :7438) for
  `POST /obs/events`. Fails against ADR-0004-transport-mcp (MCP-only transport)
  and v2's dropped-serve decision; adds a daemon the user must keep running for
  capture to work at all. The v1 HTTP layer stays v1's.
- **JSONL spool + lazy ingest (rejected)** — decouples capture from the
  binary but adds a spool to compact, a path to lose, and `obs logs` would
  miss events until ingest runs. SQLite-direct is what gryph and v1 both do.
- **MCP `obs_*` tools (deferred)** — would let agents query their own trail;
  v1 is CLI-only; a later change adds tools over the same store package.

## Decisions (to confirm — load-bearing)

- **D1 — Command family name: `mnemonic obs` (recommended).** The user's term
  ("observation layer", "observation commands"). **Collision:** v2's memory
  domain already uses "observation" for typed memory entities
  (`observations` table, `mem_save` "Save a curated observation"). Resolution:
  glossary entry disambiguates — *observation layer / obs events* = agent
  activity audit trail (`obs_events`); *observation* (memory) = typed memory
  entity. The two share no table, no command, no MCP tool.
  Alternative: `mnemonic logs` / `activity` — avoids the collision but the
  `logs` verb already *exists inside* the family (`obs logs`), so `mnemonic
  logs <family?>` fights the family-first tree.
- **D2 — Transport: exec-the-binary, no daemon (recommended).** Per
  "similar to gryph" + v2 no-serve. Fails if: a host hook cannot exec a
  process (none of the 5 can't).
- **D3 — Policy plane = phase 2 (recommended).** v1 of this change is
  capture + query only. The door stays open: pre-tool hooks (pi `tool_call` is
  pre, cursor `before*` are pre), exit-code-2 channel in `_ingest`,
  `result_status='blocked'` column. Alternative: policy in v1 — ~2× scope
  (YAML rules, CEL-ish matching, per-agent overrides) with no observation data
  to author rules against yet.
- **D4 — Storage: per-project store, migration 300 (recommended).** v2
  architecture (sessions/observations/projectwiki_events all per-project);
  gryph's single central DB assumes no project concept. Alternative: central
  `obs.sqlite` — simpler cross-project queries, breaks v2's per-project model
  and would need its own retention/ownership rules.
- **D5 — Normalization lives in Go (`internal/obs`), not in the JS hooks
  (recommended).** v1's mapToolType/sensitivePath/stripPrivateTags live in JS
  (tool-call-capture.js) — duplicated per host script. v2: hooks are dumb
  transports; the Go `_ingest` normalizes once; the `--private`/sensitive/
  truncation contract is unit-tested in one place.
- **D6 — Event surface = full gryph copy per host, plus prompt capture on
  pi/omp (recommended).** Confirmed 2026-11-17 against the live gryph install
  on this host (cursor hooks.json = direct-binary commands, 20 hook types;
  opencode `gryph.js` = 5s exec, before-only blocking, 3 session events;
  omp `gryph-hooks.ts` = 4 hooks, 30s cap). The only deliberate extension is
  `prompt` on pi/omp (`before_agent_start`) — gryph doesn't capture prompts
  there, but v1 did (`/prompts`) and the cost is one hook per host.
  Alternative: v1's 7-hook cursor subset — rejected: strictly less data and
  keeps the shell+node indirection gryph dropped.

## Acceptance (falsifiable)

1. `mnemonic obs install all --dry-run` prints a per-host plan; real install on
   this host merges into `~/.cursor/hooks.json` **without touching** the existing
   gryph/orca/herdr entries; omp gets `~/.omp/agent/extensions/mnemonic-obs.ts`;
   `obs status` reports all 5 installed.
2. A real omp session (tool calls included) in a git project produces
   `obs_events` rows: `session_start`, `tool_call`+`tool_result` pairs,
   `session_end`; `mnemonic obs logs` (default 24h) shows them. *(backstop —
   live host; held-out test can't settle it alone)*
3. `mnemonic obs query --action tool_call --command "git *"` returns exactly the
   matching shell events; `--file "src/**" --action file_write` returns writes.
4. Prompt with `<private>SECRET</private>` → stored `prompt` column lacks
   "SECRET"; input with `<private>` → `input_preview` stripped.
5. Tool on `.env` or `~/.ssh/id_ed25519` → row has `path`, empty
   `output_preview`/`input_preview`.
6. `PATH` without `mnemonic` → agent tool call proceeds; hook exits 0.
   *(backstop — live host)*
7. `obs prune --retention 90d --dry-run` counts only rows older than the window;
   real run deletes them and leaves `sessions` intact.
8. `obs export --since 1d` emits valid JSONL (one parseable object/line).
9. Re-run `obs install all` → no duplicate config entries (idempotent).
10. Full `mnemonic` suite green; migration 300 applies clean on fresh + existing
    stores (existing store: new table, no data touched).

## Open items for the blueprint (pinned during drafting, not TBD here)

- (Resolved) Opencode surface is pinned by gryph's own plugin (gryph.js):
  `tool.execute.before/after` + `session.created`/`session.idle`/
  `session.error`, 5s exec, before-only blocking. The repo's existing
  `skillgrid-events.ts` already pins the V2 mutable-event signature shape.
- Cursor stdin JSON shape for each of the 20 hook types (probe a live
  cursor session with a debug hook; v1's tool-call-capture.js already
  handles the `postToolUse`/`before*` shapes — port, don't re-derive; the
  rest — tab-file, subagent, thought/response, preCompact — pin from
  Cursor's hook docs at blueprint).
- Kilo config file name + registration key on this host (v1 installer precedent:
  `~/.config/kilo/` config; confirm at blueprint).
- `obs install` source-of-truth for plugin files: staged copy in the data dir
  (version-stamped, the v1 `stagedPluginPath` pattern) vs repo checkout —
  decided by what the memory-plugins setup machinery provides when it lands.

## Next step

Blueprint (`writing-blueprints`): file layout, per-task steps with code,
must-haves with backstop tags, threat matrix (subprocess + shell boundaries
apply), one-way doors (migration 300; installed-file ownership), build shape
(tracer thread: omp end-to-end first — the host I'm running on, fastest
feedback loop — then pi → opencode → kilo → cursor).
