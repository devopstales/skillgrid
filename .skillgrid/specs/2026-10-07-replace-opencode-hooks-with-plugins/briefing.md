# Replace opencode + Kilo shell hooks with native TS plugins — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-10-08)

**Topic:** 2026-10-07-replace-opencode-hooks-with-plugins
**Date:** 2026-10-07 (blueprint) / 2026-10-08 (briefing backfill)
**Classification:** medium (installer refactor + 4 file deletions + 1 new HTTP package) — T2
**Build shape:** Single change (one wiring path, one deletion set, one new route)
**Reference logic:** `plugins/opencode/*.ts` (5 plugins already written), `hooks/opencode-*.sh` (4 retired shell hooks), `mnemonic/internal/setup/{setup,opencode,kilocode}.go` (current installer)

## Problem / Intent

The shell-era hook system for opencode and Kilo is a 4-layer indirection: `hooks.yaml` → shell script → `node tool-call-capture.js` → HTTP POST to the mnemonic server. Five native TS plugins already exist in `plugins/opencode/` that replace all four shell hooks and the retired `skillgrid-checkpoint.ts` plugin. The installer (`mnemonic/internal/setup/`) still wires the old path: it copies `hooks.yaml`, the 4 shell scripts, and `skillgrid-checkpoint.ts`, and registers `opencode-yaml-hooks` in the harness config. This change completes the switchover: the installer copies the 5 TS plugins, registers them, and drops the retired shell-hook path. The 4 shell scripts, 2 `hooks.yaml` files, and 2 `skillgrid-checkpoint.ts` copies are deleted from the repo.

## Purpose & Success Criteria

- **Purpose:** Replace the shell-era hook system for opencode and Kilo with 5 native TS plugins, making the installer's wiring single-path (no dual-path), and retire the retired files from the repo.
- **Success criteria (verifiable):**
    1. `skillgrid setup opencode` → `~/.config/opencode/plugin/` contains exactly 5 `.ts` files (`skillgrid-compaction.ts`, `skillgrid-events.ts`, `skillgrid-squad.ts`, `mnemonic-memory.ts`, `mnemonic-codeindex.ts`); config lists all 5 as `./plugin/<name>.ts`.
    2. `skillgrid setup kilocode` → `~/.config/kilo/plugin/` contains the same 5 `.ts` files (copied from `plugins/opencode/`); config lists all 5.
    3. `~/.config/opencode/hook/hooks.yaml` and `~/.config/kilo/hook/hooks.yaml` are absent after setup.
    4. `~/.config/opencode/plugin/skillgrid-checkpoint.ts` and `~/.config/kilo/plugin/skillgrid-checkpoint.ts` are absent after setup.
    5. `~/.skillgrid/hooks/` still contains `tool-call-capture.js`, `stop-tests.js`, `gate-stop.js` (shared workers, kept for Cursor).
    6. `~/.skillgrid/hooks/` does NOT contain `opencode-session-start.sh`, `opencode-session-end.sh`, `opencode-policy.sh`, `opencode-tool-capture.sh` after setup.
    7. The config `plugin` array does NOT contain `opencode-yaml-hooks` after setup.
    8. `go test ./mnemonic/internal/http/... ./mnemonic/internal/setup/...` passes.
    9. `pnpm test` / `pnpm lint` / `pnpm typecheck` pass.
    10. `POST /facts` returns `200 { id }`; `POST /facts/search` returns `200 { facts }`; `POST /facts/{id}/forget` returns `204`; `POST /facts/{id}/decay` returns `200 { score }`; `POST /facts/decay-all` returns `200 { decayed, purged }`.
- **Out of scope:** Cursor hooks (kept — `cursor-*.sh` → `tool-call-capture.js`); git-hook Stop gates (`stop-tests.js`, `gate-stop.js`); `tool-call-capture.js` shared worker (kept); MCP config (unchanged); a physical `plugins/kilo/` mirror dir (Kilo copies from `plugins/opencode/` at install time).

## Context

- **Existing flows:** `SetupOpenCode` / `SetupKiloCode` in `mnemonic/internal/setup/` copy `hooks.yaml` → `~/.config/<agent>/hook/hooks.yaml`, copy 4 shell scripts → `~/.skillgrid/hooks/`, copy `skillgrid-checkpoint.ts` → `~/.config/<agent>/plugin/`, and register `opencode-yaml-hooks` + `./plugin/skillgrid-checkpoint.ts` in the harness JSONC config.
- **Locked artifacts this extends (not duplicates):**
    - The 5 TS plugins in `plugins/opencode/` are already written and tested via `pnpm test`; this change does not modify their logic, only the installer wiring.
    - `tool-call-capture.js` is shared with Cursor (`cursor-tool-capture.sh` calls it); it is kept in `~/.skillgrid/hooks/`.
- **Constraint:** The compaction-v2 spec (`.skillgrid/specs/2026-10-07-mnemonic-compaction-v2/briefing.md`) assumes this plan lands first — all its plugin wiring targets the 5 new TS plugins, NOT the retired shell hooks.
- **Files:** `mnemonic/internal/setup/{setup.go, opencode.go, kilocode.go, opencode_test.go, kilocode_test.go}`, `mnemonic/internal/http/{facts.go (new), facts_test.go (new), server.go}`, `plugins/opencode/*.ts` (5 new, already present), `plugins/kilo/skillgrid-checkpoint.ts` (deleted), `plugins/{opencode,kilo}/hooks.yaml` (deleted), `hooks/opencode-*.sh` (4 deleted), `scripts/test-hooks.mjs`, `docs/user-guide/04-hooks.md`.

## Approaches Considered

- **Chosen:** Single-source `plugins/opencode/<name>.ts`; Kilo's installer copies each file to `~/.config/kilo/plugin/<name>.ts`. No physical `plugins/kilo/` mirror dir in the repo. The installer's `openCodeHookScripts` list is trimmed to the 3 shared workers. `FindRepoRoot` is updated to look for the 5 new plugin rels.
- **Rejected:** Physical `plugins/kilo/` mirror dir — duplicates the 5 files, requires keeping them in sync; a copy at install time is simpler and the installer already has `copyFromRepo`.
- **Rejected:** Dual-path (new TS plugins + retired shell hooks coexisting) — the compaction-v2 spec assumes single-path; dual-path would mean two systems handling the same events, causing double-firing and debugging confusion.
- **Rejected:** Keeping `skillgrid-checkpoint.ts` alongside `skillgrid-compaction.ts` — the checkpoint logic is folded into `skillgrid-compaction.ts` (the `session.compacted` handler does what `skillgrid-checkpoint.ts` did).

## Requirements

1. **Go: HTTP facts routes** — `mnemonic/internal/http/facts.go` (NEW).
    - **Current:** No `/facts*` HTTP routes; the `facts` package (`facts.New(db, project)`) exists but is not wired to HTTP.
    - **Target:** `registerFactsRoutes(s *Server, mux *http.ServeMux)`:
        - `POST /facts` → `h.store().Add(content)` → 200 `{ id }`.
        - `POST /facts/search` → `h.store().SearchWith(query, limit)` → 200 `{ facts }`.
        - `POST /facts/{id}/forget` → `h.store().Forget(id)` → 204.
        - `POST /facts/{id}/decay` → `h.store().Decay(id)` → 200 `{ score }`.
        - `POST /facts/decay-all` → `h.store().DecayAll(threshold)` → 200 `{ decayed, purged }`.
    - `h.store()` constructs `facts.New(h.Store().DB, h.project)` lazily.
    - Wire into `registerRoutes()` in `server.go`.
    - **Acceptance:** All 5 routes return the documented status + body; a missing store returns 500.
    - **Acceptance scenario:** `happy path facts routes` → `acceptance.feature`.

2. **Go: installer — opencode** — `mnemonic/internal/setup/opencode.go`.
    - **Current:** Copies `hooks.yaml` → `~/.config/opencode/hook/hooks.yaml`; copies 4 `opencode-*.sh` + 3 shared workers → `~/.skillgrid/hooks/`; copies `skillgrid-checkpoint.ts` → `~/.config/opencode/plugin/`; registers `opencode-yaml-hooks` + `./plugin/skillgrid-checkpoint.ts` in config.
    - **Target:** Copy 5 plugins from `plugins/opencode/<name>.ts` → `~/.config/opencode/plugin/<name>.ts` + `upsertPluginKey("./plugin/<name>.ts")` for each. Remove `upsertPluginKey("opencode-yaml-hooks")`, remove `hooks.yaml` copy, remove checkpoint copy + upsert. Keep `installOpenCodeHookScripts` (still copies shared workers: `tool-call-capture.js`, `stop-tests.js`, `gate-stop.js`).
    - **Acceptance:** After setup, 5 `.ts` files exist under `~/.config/opencode/plugin/`; config lists 5; `hook/hooks.yaml` absent; `skillgrid-checkpoint.ts` absent; `opencode-yaml-hooks` absent from config.
    - **Acceptance scenario:** `happy path opencode installer wires 5 plugins` → `acceptance.feature`.

3. **Go: installer — kilocode** — `mnemonic/internal/setup/kilocode.go`.
    - **Current:** Mirrors opencode (hooks.yaml, shell scripts, checkpoint plugin, `opencode-yaml-hooks`).
    - **Target:** Mirror opencode — copy 5 plugins from `plugins/opencode/<name>.ts` → `~/.config/kilo/plugin/<name>.ts` + `upsertPluginKey("./plugin/<name>.ts")`. Remove `upsertPluginKey("opencode-yaml-hooks")`, remove `hooks.yaml` copy, remove checkpoint copy + upsert.
    - **Acceptance:** After setup, 5 `.ts` files exist under `~/.config/kilo/plugin/`; config lists 5; `hook/hooks.yaml` absent; `skillgrid-checkpoint.ts` absent.
    - **Acceptance scenario:** `happy path kilocode installer wires 5 plugins` → `acceptance.feature`.

4. **Go: installer — setup.go constants + shared worker list + FindRepoRoot** — `mnemonic/internal/setup/setup.go`.
    - **Current:** `opencodePluginRel = "plugins/opencode/hooks.yaml"`, `opencodeCheckpointPluginRel = "plugins/opencode/skillgrid-checkpoint.ts"`, `kiloPluginRel = "plugins/kilo/hooks.yaml"`, `kiloCheckpointPluginRel = "plugins/kilo/skillgrid-checkpoint.ts"`. `openCodeHookScripts` includes 4 `opencode-*.sh` + 3 shared workers. `FindRepoRoot` looks for `opencodePluginRel` / `kiloPluginRel`.
    - **Target:** Add 5 new constants for the 5 plugin rels. Trim `openCodeHookScripts` to `["tool-call-capture.js", "stop-tests.js", "gate-stop.js"]`. Update `FindRepoRoot` to look for the 5 new plugin rels instead of the retired `hooks.yaml` rels. Keep `opencode-yaml-hooks` / `hook/hooks.yaml` in `retiredPluginEntry` (so existing configs get cleaned on next setup).
    - **Acceptance:** `FindRepoRoot` returns the repo root when the 5 plugin files are present; `openCodeHookScripts` has exactly 3 entries.
    - **Acceptance scenario:** `happy path FindRepoRoot uses new plugin rels` → `acceptance.feature`.

5. **Deletions** — retired files removed from the repo.
    - **Current:** 8 files exist in the repo.
    - **Target:** Delete: `plugins/opencode/skillgrid-checkpoint.ts`, `plugins/kilo/skillgrid-checkpoint.ts`, `plugins/opencode/hooks.yaml`, `plugins/kilo/hooks.yaml`, `hooks/opencode-session-start.sh`, `hooks/opencode-session-end.sh`, `hooks/opencode-policy.sh`, `hooks/opencode-tool-capture.sh`.
    - **Acceptance:** The 8 files are absent from the repo; `git status` shows 8 deletions.
    - **Acceptance scenario:** `happy path retired files deleted` → `acceptance.feature`.

6. **Go: installer tests** — `mnemonic/internal/setup/opencode_test.go` + `kilocode_test.go`.
    - **Current:** Tests assert the old behavior (checkpoint plugin present, hooks.yaml present, `opencode-yaml-hooks` in config, 4 shell scripts present).
    - **Target:** `..._InstallsPlugins`: assert 5 `.ts` files under `~/.config/opencode/plugin/` (or `kilo`), config lists 5 `./plugin/<name>.ts` keys; old `hook/hooks.yaml` + `skillgrid-checkpoint.ts` absent; `opencode-yaml-hooks` absent. `..._InstallsHookScripts`: keep shared workers present; 4 `opencode-*.sh` absent. `DropsRetiredPlugins`: keep-list updated to include `skillgrid-checkpoint.ts` and `opencode-yaml-hooks` as retired.
    - **Acceptance:** `go test ./mnemonic/internal/setup/...` passes with the new assertions.
    - **Acceptance scenario:** `happy path installer tests assert new wiring` → `acceptance.feature`.

7. **Docs / scripts** — `scripts/test-hooks.mjs` + `docs/user-guide/04-hooks.md`.
    - **Current:** `test-hooks.mjs` has `opencode-*.sh` sections; `04-hooks.md` documents the shell-hook era.
    - **Target:** `test-hooks.mjs` — drop `opencode-*.sh` sections; keep stop-tests + shared-worker checks. `04-hooks.md` — opencode/Kilo now load 5 native plugins; shell era retired for both.
    - **Acceptance:** `test-hooks.mjs` runs clean; `04-hooks.md` references the 5 plugins, not the 4 shell scripts.
    - **Acceptance scenario:** `happy path docs updated` → `acceptance.feature`.

## Implementation Decisions

- **Modules to build/modify:**
    - `mnemonic/internal/http/facts.go` (NEW): `registerFactsRoutes`.
    - `mnemonic/internal/http/facts_test.go` (NEW): 5 route tests.
    - `mnemonic/internal/http/server.go`: add `registerFactsRoutes(s, mux)` call in `registerRoutes()`.
    - `mnemonic/internal/setup/setup.go`: 5 new constants; trim `openCodeHookScripts`; update `FindRepoRoot`; keep retired entries.
    - `mnemonic/internal/setup/opencode.go`: copy 5 plugins; remove hooks.yaml + checkpoint + `opencode-yaml-hooks`.
    - `mnemonic/internal/setup/kilocode.go`: mirror opencode.
    - `mnemonic/internal/setup/opencode_test.go`: rewrite assertions.
    - `mnemonic/internal/setup/kilocode_test.go`: rewrite assertions.
    - `scripts/test-hooks.mjs`: drop `opencode-*.sh` sections.
    - `docs/user-guide/04-hooks.md`: update.
- **Interfaces:**
    - `func registerFactsRoutes(s *Server, mux *http.ServeMux)`
    - 5 plugin constants: `opencodeCompactionPluginRel`, `opencodeEventsPluginRel`, `opencodeSquadPluginRel`, `opencodeMemoryPluginRel`, `opencodeCodeindexPluginRel`.
- **Data flow:**
    - `skillgrid setup opencode` → `SetupOpenCode` → copy 5 `plugins/opencode/<name>.ts` → `~/.config/opencode/plugin/<name>.ts` + `upsertPluginKey` for each.
    - `skillgrid setup kilocode` → `SetupKiloCode` → copy 5 `plugins/opencode/<name>.ts` → `~/.config/kilo/plugin/<name>.ts` + `upsertPluginKey` for each.
    - Harness loads the 5 plugins at session start; each plugin handles its events (session.created, session.idle, session.compacted, tool.execute.before, tool.execute.after) and POSTs to the mnemonic HTTP server.
    - `mnemonic-memory.ts` `fact_add` / `fact_search` / `fact_forget` / `fact_decay` → `POST /facts*` → `facts.New(db, project)` → SQLite.
- **Error handling:**
    - Plugin `fetch` calls are all fail-open (catch → return null, never throw to the agent) except `tool.execute.before` policy block, which throws to block the tool.
    - `registerFactsRoutes` on store error → 500; on invalid body → 400.
    - `SetupOpenCode` / `SetupKiloCode` on missing repo root → error (no plugins to copy).

## Testing Decisions

- **What makes a good test:** Only external behavior — the installed file set, the config `plugin` array, the HTTP route responses.
- **Modules to test:** `setup` (opencode + kilocode), `http` (facts routes).
- **Prior art:** `TestSetupOpenCode_CheckpointPlugin`, `TestSetupOpenCode_DropsRetiredPlugins`, `TestSetupOpenCode_InstallsHookScripts`, `facts_test.go` (existing for other routes).
- **Edge cases:** Dry-run must not write files; existing config with `opencode-yaml-hooks` must have it dropped; `FindRepoRoot` must not false-positive on a dir without the 5 plugin files; facts routes with no store → 500; facts search with empty query → 200 `{ facts: [] }`.

## Impact on Global Docs

- `.skillgrid/artifacts/04-adr-index.md`: add ADR-0030 (if a new ADR is created for this change).
- `.skillgrid/ARCHITECTURE.md`: note the 5-plugin architecture and the retired shell-hook era.
- `docs/user-guide/04-hooks.md`: update (in-scope).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.95  | 0.75 | OK     | Single switchover, clear boundary. |
| Boundary Clarity    | 0.90  | 0.70 | OK     | Installer + deletions + 1 new HTTP package. |
| Constraint Clarity  | 0.85  | 0.65 | OK     | Single-path; no new deps; shared workers kept. |
| Acceptance Criteria | 0.90  | 0.70 | OK     | Specific file/config/route assertions. |
| **Clarity**         | 0.10  | ≤0.20| OK     |                                    |

## Open Questions & Assumptions

- **Assumption:** The 5 TS plugins in `plugins/opencode/` are functionally complete for the events they handle (session.created, session.idle, session.compacted, tool.execute.before, tool.execute.after). The compaction-v2 spec will extend them with advisory/steering logic but does not change the wiring contract.
- **Assumption:** `@opencode-ai/plugin` is resolved at install time via `~/.config/opencode/plugin/package.json` (dep `@opencode-ai/plugin: 1.14.41`); `@ts-nocheck` handles the type at build time. No new Go dependency.

## Decisions (ADR)

- `04-adr-0030-native-ts-plugins-replace-shell-hooks.md` (Planned): Document the single-source `plugins/opencode/` convention, the Kilo-copy-at-install approach, and the retirement of the 4 shell hooks + 2 hooks.yaml + 2 checkpoint plugins.

## Terms

- **Shell-era hooks:** The 4 `opencode-*.sh` scripts + `hooks.yaml` + `opencode-yaml-hooks` plugin that handled opencode/Kilo events before this change.
- **Native TS plugins:** The 5 `plugins/opencode/*.ts` files that replace the shell-era hooks.
- **Shared workers:** `tool-call-capture.js`, `stop-tests.js`, `gate-stop.js` — kept in `~/.skillgrid/hooks/` for Cursor and git-hook Stop gates.
- **Single-path:** Only the 5 TS plugins handle opencode/Kilo events; no dual-path with retired shell hooks.
