# Replace opencode + Kilo shell hooks with native TS plugins — Implementation Blueprint (steps, files, verification)

## Goal

Retire the shell-era hook system for opencode and Kilo, replaced by 5 native TS plugins:

| Plugin | Replaces |
|--------|----------|
| `skillgrid-compaction.ts` | `hooks/opencode-session-start.sh`, `hooks/opencode-session-end.sh`, retired `skillgrid-checkpoint.ts` (folded in) |
| `skillgrid-events.ts` | `hooks/opencode-tool-capture.sh` + `hooks/opencode-policy.sh` (policy gate) + `hooks/tool-call-capture.js` logic |
| `skillgrid-squad.ts` | `hooks/opencode-squad.sh` (if present; else new) |
| `mnemonic-memory.ts` | Memory fact routes + mem_save/search/get observation tools |
| `mnemonic-codeindex.ts` | Code index status/index/search/read/files tools |

Kilo gets the same 5 plugins via a copy from `plugins/opencode/` in the Go installer. Cursor keeps its own shell hooks (`cursor-tool-capture.sh` → `tool-call-capture.js`); git-hook Stop gates (`stop-tests.js`, `gate-stop.js`) are untouched.

## Decisions

- Single source of truth: `plugins/opencode/<name>.ts` — Kilo's installer copies each file to `~/.config/kilo/plugin/<name>.ts`. No physical `plugins/kilo/` mirror dir in the repo.
- In-repo plugin convention: `// @ts-nocheck`, inlined types, no `@opencode-ai/plugin` import.
- Base URL: `SKILLGRID_CHECKPOINT_URL || SKILLGRID_MNEMONIC_HTTP_URL || http://127.0.0.1:7438`.
- Agent label: `SKILLGRID_AGENT || "opencode"` (kilo set at runtime).
- Retire `skillgrid-checkpoint.ts` (both `plugins/opencode/` and `plugins/kilo/`) — logic folds into `skillgrid-compaction.ts`.
- Start injection → `client.session.prompt` event (session-started).
- `tool.execute.before` can block a tool by `throw new Error(...)`; fail-closed on policy errors.
- `@opencode-ai/plugin` is resolved at install time via `~/.config/opencode/plugin/package.json` (dep `@opencode-ai/plugin: 1.14.41`). `@ts-nocheck` handles the type.
- `tool` + `tool.schema` come from `@opencode-ai/plugin`. `tool.schema` is zod.
- Facts routes: `POST /facts`, `GET /facts/search`, `POST /facts/{id}/forget`, `POST /facts/{id}/decay`, `POST /facts/decay-all`. Wired via `facts.New(db, project)`.

## Shared-worker blast radius

- `hooks/tool-call-capture.js` is also copied by Cursor (`cursor.go:107`) → **kept**.
- `hooks/stop-tests.js`, `hooks/gate-stop.js` are git-hook Stop gates → **kept**.
- `hooks/opencode-session-start.sh`, `hooks/opencode-session-end.sh`, `hooks/opencode-policy.sh`, `hooks/opencode-tool-capture.sh` → **deleted** (ported into native plugins).
- `plugins/opencode/hooks.yaml`, `plugins/kilo/hooks.yaml` → **deleted** (no more shell hooks for opencode/Kilo).
- `plugins/opencode/skillgrid-checkpoint.ts`, `plugins/kilo/skillgrid-checkpoint.ts` → **deleted** (folded into compaction).

## Files to write

### New plugins under `plugins/opencode/`

1. `plugins/opencode/skillgrid-compaction.ts`
   - `event` handler for `session.created` / `session.idle` / `session.compacted` / `experimental.session.compacting`.
   - On `session.idle` → POST `/memory/context` to fetch prime text, then `client.session.prompt` (start-injection).
   - On `session.compacted` / `experimental.session.compacting` → build distilled summary from session events, POST to `/memory/context`.
   - Re-uses `primeText` logic from `loop_cmd.go` and `DistillSummary` / `EstimateTokens` from `session_inject/summary.go`.

2. `plugins/opencode/skillgrid-events.ts`
   - `tool.execute.before(input, output)` → policy check: GET `/policy/check?tool=...&path=...` (or equivalent); if `deny` → `throw new Error("skillgrid policy: ...")` to block.
   - `tool.execute.after(input, output)` → capture tool call: POST `/memory/session-events` with `agent`, `sessionID`, `tool`, `type`, `preview`, `contentHash`, `usage` (sum of `output.tokens.input/output` if present).
   - Port `mapToolType`, `stripPrivateTags`, `sensitivePath`, `contentHash`, `sumOpenCodeUsage` from `hooks/tool-call-capture.js`.

3. `plugins/opencode/skillgrid-squad.ts`
   - 6 custom tools: `squad_list_tasks`, `squad_read_task`, `squad_spawn_task`, `squad_pull_next_task`, `squad_submit_output`, `squad_mark_done`.
   - Each tool → POST/GET `/teams/tasks*` with `{ agent, sessionId, ...args }`.

4. `plugins/opencode/mnemonic-memory.ts`
   - Custom tools: `mem_save`, `mem_search`, `mem_get_observation`, `fact_add`, `fact_search`, `fact_forget`, `fact_decay`.
   - Each → POST/GET to `/memory/...` or `/facts*` routes.
   - `mem_save` → POST `/memory/save` with `{ title, type, content, session_id, topic_key?, scope? }`.
   - `mem_search` → POST `/memory/search` with `{ query, limit? }`.
   - `mem_get_observation` → GET `/memory/observations/{id}`.
   - `fact_add` → POST `/facts` with `{ content }`.
   - `fact_search` → POST `/facts/search` with `{ query }`.
   - `fact_forget` → POST `/facts/{id}/forget`.
   - `fact_decay` → POST `/facts/{id}/decay`.

5. `plugins/opencode/mnemonic-codeindex.ts`
   - Custom tools: `code_status`, `code_index`, `code_search`, `code_read`, `code_files`.
   - Each → POST/GET to `/code/...` routes.

### Go: HTTP routes

- `mnemonic/internal/http/facts.go` (new)
  - `registerFactsRoutes(s *Server, mux *http.ServeMux)`:
    - `POST /facts` → `h.store().Add(content)` → 200 `{ id }`.
    - `POST /facts/search` → `h.store().SearchWith(query, limit)` → 200 `{ facts }`.
    - `POST /facts/{id}/forget` → `h.store().Forget(id)` → 204.
    - `POST /facts/{id}/decay` → `h.store().Decay(id)` → 200 `{ score }`.
    - `POST /facts/decay-all` → `h.store().DecayAll(threshold)` → 200 `{ decayed, purged }`.
  - `h.store()` constructs `facts.New(h.Store().DB, h.project)` lazily.
- `mnemonic/internal/http/facts_test.go` (new) — cover 5 routes.
- `mnemonic/internal/http/server.go` — add `registerFactsRoutes(s, mux)` call in `registerRoutes()`.

### Go: installer

- `mnemonic/internal/setup/setup.go`
  - Add 5 new constants:
    ```go
    opencodeCompactionPluginRel = "plugins/opencode/skillgrid-compaction.ts"
    opencodeEventsPluginRel     = "plugins/opencode/skillgrid-events.ts"
    opencodeSquadPluginRel      = "plugins/opencode/skillgrid-squad.ts"
    opencodeMemoryPluginRel     = "plugins/opencode/mnemonic-memory.ts"
    opencodeCodeindexPluginRel  = "plugins/opencode/mnemonic-codeindex.ts"
    ```
  - Trim `openCodeHookScripts` to `["tool-call-capture.js", "stop-tests.js", "gate-stop.js"]` (drop the 4 `opencode-*.sh`).
  - Update `FindRepoRoot` to look for the 5 new plugin rels instead of `opencodePluginRel` / `kiloPluginRel` (retired `hooks.yaml`).
  - Keep `opencode-yaml-hooks` / `hook/hooks.yaml` in `retiredPluginEntry`.

- `mnemonic/internal/setup/opencode.go`
  - `SetupOpenCode`: remove `upsertPluginKey("opencode-yaml-hooks")`, remove `hooks.yaml` copy, remove checkpoint copy + upsert.
  - Keep `installOpenCodeHookScripts` (still copies shared workers: `tool-call-capture.js`, `stop-tests.js`, `gate-stop.js`).
  - Add: copy 5 plugins from `plugins/opencode/<name>.ts` → `~/.config/opencode/plugin/<name>.ts` + `upsertPluginKey("./plugin/<name>.ts")`.

- `mnemonic/internal/setup/kilocode.go`
  - `SetupKiloCode`: mirror opencode — copy 5 plugins from `plugins/opencode/<name>.ts` → `~/.config/kilo/plugin/<name>.ts` + `upsertPluginKey("./plugin/<name>.ts")`.
  - Remove `upsertPluginKey("opencode-yaml-hooks")`, remove `hooks.yaml` copy, remove checkpoint copy + upsert.

- `mnemonic/internal/setup/opencode_test.go` + `kilocode_test.go`
  - `..._InstallsPlugins`: assert 5 `.ts` files under `~/.config/opencode/plugin/` (or `kilo`), config lists 5 `./plugin/<name>.ts` keys; old `hook/hooks.yaml` + `skillgrid-checkpoint.ts` absent.
  - `..._InstallsHookScripts`: keep shared workers present; 4 `opencode-*.sh` + `hook/hooks.yaml` absent.
  - `DropsRetiredPlugins`: keep-list updated to include `skillgrid-checkpoint.ts` and `opencode-yaml-hooks`.

### Deletions

- `plugins/opencode/skillgrid-checkpoint.ts`
- `plugins/kilo/skillgrid-checkpoint.ts`
- `plugins/opencode/hooks.yaml`
- `plugins/kilo/hooks.yaml`
- `hooks/opencode-session-start.sh`
- `hooks/opencode-session-end.sh`
- `hooks/opencode-policy.sh`
- `hooks/opencode-tool-capture.sh`

### Docs / scripts

- `scripts/test-hooks.mjs` — drop `opencode-*.sh` sections; keep stop-tests + shared-worker checks.
- `docs/user-guide/04-hooks.md` — opencode/Kilo now load 5 native plugins; shell era retired for both.

## Verification

- `go test ./mnemonic/internal/http/... ./mnemonic/internal/setup/...`
- `pnpm test`
- `pnpm lint`
- `pnpm typecheck`
- Manual: `skillgrid setup opencode` → `~/.config/opencode/plugin/` has 5 `.ts` + `package.json` (existing), config lists 5, no `hook/hooks.yaml`, no `skillgrid-checkpoint.ts`; `~/.skillgrid/hooks/` still has `tool-call-capture.js`.
- Manual: `skillgrid setup kilocode` → `~/.config/kilo/plugin/` has 5 `.ts` (copied from `plugins/opencode/`), config lists 5, no `hook/hooks.yaml`, no `skillgrid-checkpoint.ts`.

## Out of scope

- Cursor hooks (kept).
- Git-hook Stop gates (kept).
- `tool-call-capture.js` shared worker (kept).
- MCP config (unchanged).
- Any new `plugins/kilo/` physical mirror dir in the repo.
