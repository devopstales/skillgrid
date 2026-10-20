---
type: TechnicalDecision
title: "skill-pi Mnemonic memory extension: Pi-native folder `plugins/pi/mnemonic/` (index.ts entrypoint + mem_*/fact_*/code_* tool files over existing HTTP routes, prompt capture on before_agent_start, Memory Protocol injection, compaction context), `mnemonic setup pi` copies the folder into the agent-dir extensions"
status: "accepted"
supersedes: none
date: 2026-10-16
---

# ADR-0043: skill-pi Mnemonic memory extension (Pi-native tools + prompt capture)
## Context

skillgrid-mnemonic already ships a native plugin layer for the OpenCode and Kilo
harnesses (ADR-0032: five TS plugins from `plugins/opencode/`) and an MCP +
hook adapter for Cursor. skill-pi is a first-class harness in this workspace
(ADR-0038 ticketing, ADR-0039 permission layer, ADR-0042 bundled pi runtime),
but it had no Mnemonic integration: a Pi session had neither `mem_*` tools,
prompt capture, nor compaction context.

The upstream reference is Engram's `gentle-engram` Pi plugin
(`Gentleman-Programming/engram` `plugin/pi/`): a single extension that
registers compact Pi-native `mem_*` tools over the memory HTTP server, captures
user prompts on `before_agent_start`, injects a Memory Protocol into the system
prompt, and survives compaction.

## Decision

**Add one Pi extension folder, `plugins/pi/mnemonic/`, plus a `pi` agent
to `mnemonic setup` (ADR-0032 pattern).**

- **Folder format** (`plugins/pi/mnemonic/`), mirroring the opencode/kilo
  per-concern file split and the skill-pi `extensions/<name>/` layout. The
  skill-pi launcher enumerates `<agent-dir>/extensions` subdirectories and
  resolves each one's entry point with pi's own convention (package.json
  `"pi.extensions"` manifest, else `index.ts`), so a folder — not a single
  file — is the install unit: `index.ts` (entrypoint: tool registration +
  event wiring) + `api.js` (shared HTTP/redaction/TypeBox helpers) +
  `mnemonic-memory.ts` / `mnemonic-facts.ts` / `mnemonic-codeindex.ts`
  (tool groups) + `skillgrid-events.ts` (session/prompt capture) +
  `skillgrid-compaction.ts` (compaction recovery). Source of truth stays the
  Go server; the extension is a thin adapter (same shape as Engram's
  `gentle-engram` `index.ts`).

- **Tools over the existing HTTP routes** — no new Go routes. `mem_save`
  (`POST /observations`), `mem_search` (`GET /search`),
  `mem_get_observation` (`GET /observations/{id}`), `mem_update`
  (`PATCH /memory/observations/{id}`), `mem_session_summary`
  (session-registered typed observation), `mem_context` (`GET /context`),
  `mem_current_project` (`GET /project/current`), `mem_doctor`
  (`GET /memory/doctor`), `mem_capture_passive` (`POST /observations/passive`),
  `fact_add`/`fact_search`/`fact_forget` (`POST /facts*`),
  `code_status`/`code_index`/`code_search`/`code_read` (`GET|POST /code*`).
  TypeBox schemas come from `typebox`, which the skill-pi runtime already
  bundles (ADR-0042), so the extension adds **no new npm dependency**.

- **Prompt capture** on `before_agent_start` (user-message runs only):
  `POST /prompts` with `<private>` blocks redacted, trimmed prompts
  ≤10 chars skipped, content capped at 2000 chars — the same policy the Go
  server enforces for MCP `mem_save` prompt links (`minPromptLength`/
  `MaxPromptLength`).

- **Memory Protocol injection**: one `appendSystemPrompt` section per
  extension instance, idempotent (guarded by a section header match),
  instructing the recovery ladder (`mem_context` → `mem_search` →
  `mem_get_observation`), the save triggers, topic-key upsert discipline, and
  the six-section session-summary shape from the shared mnemonic-memory
  convention.

- **Compaction survival**: `session_before_compact` fetches
  `GET /context/compaction?session_id=` so the session row's title/summary +
  newest observations are fresh before Pi summarizes. Fail-open.

- **Session lifecycle**: `session_start` idempotently registers the host
  session id (`POST /sessions`, server is idempotent on client id);
  `session_shutdown` with reason `quit` ends it (`POST /sessions/{id}/end`).
  `reload`/`new`/`resume`/`fork` replace the session without ending it.

- **Fail-open everywhere**: every capture path (prompt, registration,
  compaction, shutdown, per-tool session ensure) swallows errors; a dead
  `mnemonic serve` never blocks a prompt, tool call, or shutdown. Tool
  execute errors return an error JSON string to the model rather than
  throwing.

- **Setup**: `mnemonic setup pi` (alias `skill-pi`) copies the extension
  folder to `<agent-dir>/extensions/mnemonic/` (all source files + manifest;
  entry point verified against the launcher's resolution convention). The
  skill-pi launcher (`bin/skill-pi.js`) explicitly enumerates
  `<agent-dir>/extensions` subdirectories under `--no-extensions` isolation,
  so no settings.json or MCP registration is needed. The agent dir honors
  `SKILL_PI_AGENT_DIR`, default `~/.skill-pi/agent`.

## Consequences

- **Positive:** Pi sessions get the same memory contract as OpenCode/Kilo:
  cross-session recall, prompt-captured saves, compaction survival. No new
  Go or npm dependencies (TypeBox is already a bundled skill-pi dep; the Go
  server reuses existing routes). The extension is a 1:1 transport adapter,
  so server-side fixes (redaction, limits, auth) apply to Pi for free.
- **Negative:** Pi has no tool-capture hook yet (opencode's
  `skillgrid-events.ts` policy gate + tool-call capture are not ported); the
  `pi-hooks` extension covers event hooks but the tool-call-capture worker is
  not wired — acceptable, memory is the gap being closed, and a follow-up can
  add capture via `pi-hooks` without touching this extension. The
  `mem_session_summary` tool persists the summary as a typed observation
  rather than the session row's `summary` column (the end route owns that and
  ends the session); the row summary is filled when the session ends.
- **Environment:** `SKILLGRID_MNEMONIC_HTTP_URL` (default
  `http://127.0.0.1:7438`), `SKILLGRID_HTTP_TOKEN` (bearer for write routes
  when the server sets one), `SKILLGRID_AGENT` (default `pi`).
- **One-way gate:** the folder is already the split unit — a new concern
  (e.g. tool-call capture) is a new file under `plugins/pi/mnemonic/` wired
  into `index.ts`, added to `piExtensionFiles` in the setup installer and to
  the `files` array in `package.json`.
