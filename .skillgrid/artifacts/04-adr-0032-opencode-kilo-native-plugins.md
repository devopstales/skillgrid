# ADR-0032: opencode + Kilo native TS plugin architecture

---
status: "accepted"
supersedes: none
date: 2026-10-08
---

## Context

The opencode and Kilo harnesses ran on a shell-era hook system: 4 POSIX
shell scripts (`opencode-session-start.sh`, `opencode-session-end.sh`,
`opencode-policy.sh`, `opencode-tool-capture.sh`) orchestrated via 2
`hooks.yaml` files, plus a `skillgrid-checkpoint.ts` plugin that handled
session checkpointing. The shell hooks spawned the Go binary as a child
process for each event, parsing environment variables and piping JSON
through stdin. This was brittle (shell quoting, env var limits, no type
safety) and slow (process spawn per event).

opencode and Kilo both support native TypeScript plugins that run in
process: `tool.execute.before` / `tool.execute.after` hooks, `event`
handlers for session lifecycle, and custom tool registration via
`tool.schema` (zod). The `@opencode-ai/plugin` package is already resolved
at install time via `~/.config/opencode/plugin/package.json`.

The mnemonic HTTP server already exposes the endpoints the shell hooks
called. The `mnemonic/internal/facts` package already has the full
`Store` API (`Add`, `Search`, `Forget`, `Decay`, `DecayAll`). What was
missing: the facts HTTP routes and the installer wiring for the 5
replacement plugins.

## Decision

**Single source of truth: `plugins/opencode/<name>.ts`.** Kilo's installer
copies each file from `plugins/opencode/` to `~/.config/kilo/plugin/` at
install time. No physical `plugins/kilo/` mirror directory in the repo.

The 5 plugins:

| Plugin | Responsibility |
|--------|---------------|
| `skillgrid-compaction.ts` | Session lifecycle events: start injection (`session.created` / `session.idle` → `client.session.prompt`), compaction summary (`session.compacted` → POST `/memory/context`). Folds in the retired `skillgrid-checkpoint.ts`. |
| `skillgrid-events.ts` | `tool.execute.before` policy gate (fail-closed: `throw` on deny), `tool.execute.after` tool-call capture (POST `/memory/session-events`). Ports `mapToolType`, `stripPrivateTags`, `sensitivePath`, `contentHash`, `sumOpenCodeUsage` from the retired shell hook + `tool-call-capture.js`. |
| `skillgrid-squad.ts` | 6 custom tools (`squad_list_tasks`, `squad_read_task`, `squad_spawn_task`, `squad_pull_next_task`, `squad_submit_output`, `squad_mark_done`) → `/teams/tasks*` routes. |
| `mnemonic-memory.ts` | Custom tools: `mem_save`, `mem_search`, `mem_get_observation`, `fact_add`, `fact_search`, `fact_forget`, `fact_decay` → `/memory/...` and `/facts*` routes. |
| `mnemonic-codeindex.ts` | Custom tools: `code_status`, `code_index`, `code_search`, `code_read`, `code_files` → `/code/...` routes. |

**Base URL:** `SKILLGRID_CHECKPOINT_URL || SKILLGRID_MNEMONIC_HTTP_URL || http://127.0.0.1:7438`.
**Agent label:** `SKILLGRID_AGENT || "opencode"` (Kilo set at runtime).
**In-repo plugin convention:** `// @ts-nocheck`, inlined types, no
`@opencode-ai/plugin` import (resolved at install time).

**Facts HTTP routes** (new file `mnemonic/internal/http/facts.go`):
`POST /facts`, `POST /facts/search`, `POST /facts/{id}/forget`,
`POST /facts/{id}/decay`, `POST /facts/decay-all` — wrapping the existing
`mnemonic/internal/facts.Store`.

**Installer changes:**
- `SetupOpenCode`: copy 5 plugins → `~/.config/opencode/plugin/`, upsert
  5 config keys. Remove `hooks.yaml` copy, `skillgrid-checkpoint.ts` copy
  + upsert, `opencode-yaml-hooks` upsert.
- `SetupKiloCode`: mirror opencode, copy from `plugins/opencode/` →
  `~/.config/kilo/plugin/`.
- `openCodeHookScripts` trimmed to shared workers only:
  `["tool-call-capture.js", "stop-tests.js", "gate-stop.js"]`.
- `FindRepoRoot` looks for the 5 new plugin rels.
- `dropRetiredPlugins` keep-list updated.

**Deletions:** 4 shell scripts, 2 `hooks.yaml`, 2 `skillgrid-checkpoint.ts`.

**Preserved:** `tool-call-capture.js` (Cursor), `stop-tests.js` +
`gate-stop.js` (git-hook Stop gates), `cursor-tool-capture.sh` (Cursor).

## Consequences

- **Positive:** No process spawn per event (in-process plugins). Type
  safety via `tool.schema` (zod). Single source of truth for plugin
  code. Simpler installer (no `hooks.yaml` orchestration). Facts
  available over HTTP for the `mnemonic-memory.ts` plugin tools.
- **Negative:** Kilo depends on the opencode plugin source directory
  (coupling, but intentional — same runtime, same `@opencode-ai/plugin`
  API). The `// @ts-nocheck` convention means no compile-time type
  checking on plugin files (mitigated by `pnpm typecheck` on the repo
  and the `pnpm test` suite).
- **Migration:** Existing users running `skillgrid setup opencode` or
  `skillgrid setup kilocode` will get the new plugins on the next
  install. The old `opencode-yaml-hooks` config entry is dropped
  automatically by `dropRetiredPlugins`. The retired shell scripts are
  no longer copied, so they disappear from `~/.skillgrid/hooks/` on
  the next install.
- **Testing:** `go test ./mnemonic/internal/setup/...` and
  `./mnemonic/internal/http/...` cover the installer and facts routes.
  `pnpm test` covers the plugin files.
