# add-mcp integration — design

> **STATUS: DRAFT (2026-08-26)**

**Plan:** [2026-08-26-add-mcp-integration.md](../plans/2026-08-26-add-mcp-integration.md)

**Related:** [04-mcp-servers.md](../../04-mcp-servers.md), [03-config-reference.md](../../03-config-reference.md), [2026-08-25-skillgrid-cli-design.md](2026-08-25-skillgrid-cli-design.md)

## Summary

Replace skillgrid's hand-rolled JSONC MCP merger with [**add-mcp**](https://github.com/neon-solutions/add-mcp) as the install-time writer. **`config.d/mcp.yaml` stays the source of truth**; a small Node sync script translates YAML entries into `upsertServer()` calls so each agent gets native config shapes (Kilo, OpenCode, Cursor, Codex, …).

## Problem

Today skillgrid merges MCP servers with custom Go code (`config.MergeMCP` + `tidwall/sjson`):

- Only **Kilo** and **OpenCode** are wired; other agents in `docs/NOTE.md` have no path.
- One JSON shape (`type` + `url` / `command`) — agents differ (TOML, separate MCP files, capability-gated fields).
- No registry search, sync, or remove — operators hand-edit YAML and re-run install.
- Type-conflict rename (`context7-remote`) is clever but agent-specific edge cases grow in Go.

[add-mcp](https://github.com/neon-solutions/add-mcp) already solves multi-agent MCP install, transport mapping, capability-gated fields, `find`/`list`/`remove`/`sync`, and documents Kilo (`kilo-code`) and OpenCode agent IDs.

## Goal

After `skillgrid install`:

1. `config.d/mcp.yaml` still declares which MCP servers to install.
2. **add-mcp** (via programmatic API) writes global agent configs for every **selected** agent.
3. Backups, dry-run, and warn-and-continue semantics match existing skillgrid conventions.
4. Operators can use the managed **`add-mcp` CLI** (on PATH after install) for ad-hoc discovery; skillgrid-managed entries remain YAML-driven.

## Non-goals

- **Runtime `npx`** for add-mcp or any skillgrid-managed tool — install via `npm --prefix ~/.skillgrid/npm`, invoke from `~/.skillgrid/npm/node_modules/.bin/` (same pattern as `skills`, playwright)
- Replacing `mcp.yaml` with add-mcp's registry as source of truth
- Project-scoped MCP install by default (skillgrid targets **global** `~/.config/` agent configs)
- Wrapping every add-mcp subcommand in skillgrid CLI v1 (`find`, `sync` documented in usage only)
- Auto-approve (`--auto-approve`) by default — opt-in per server in YAML
- Removing user-managed MCP entries that are not in `mcp.yaml` (add-mcp upsert only; no mass delete)

## Decisions (proposed)

| Topic | Decision |
|-------|----------|
| Source of truth | `config.d/mcp.yaml` (unchanged role) |
| Writer | add-mcp `upsertServer()` via Node sync script |
| Package install | `add-mcp` in `config.d/tools.yaml` → `npm install --prefix ~/.skillgrid/npm` |
| Runtime invoke | **`~/.skillgrid/npm/node_modules/.bin/add-mcp`** or `add-mcp` after PATH — **never `npx add-mcp`** |
| Sync script | `node scripts/sync-mcp-from-yaml.mjs` — imports `add-mcp` from skillgrid npm prefix via `createRequire` |
| Scope | **Global** (`local: false` / `-g`) — matches current `~/.config/kilo` install |
| Agent IDs | Map skillgrid selector → add-mcp: `kilo` → `kilo-code`, `opencode` → `opencode` |
| Go MergeMCP | **Deprecate** after parity validation; remove in same PR series |
| Backups | Go step backs up each agent config file **before** invoking sync script |
| Dry-run | Sync script prints planned upserts; Go skips writes and add-mcp calls |
| Missing local binary | Keep `PrecheckDependencies`; warn, continue (same as today) |
| Extended YAML | Optional `headers`, `env`, `transport`, `timeout`, `auto_approve` per server |

## Architecture

```
config.d/mcp.yaml
       ↓
node scripts/sync-mcp-from-yaml.mjs   (Node runtime; imports add-mcp from ~/.skillgrid/npm)
       ↓
add-mcp package (~/.skillgrid/npm/node_modules/add-mcp)
       ↓
~/.config/kilo/kilo.jsonc
~/.config/opencode/opencode.jsonc
~/.cursor/mcp.json          (when cursor agent selected — future)
```

**Install flow change** (step 7):

```
load mcp.yaml → PrecheckDependencies (warn) → backup agent configs → node sync-mcp-from-yaml.mjs → log results
```

Go invokes **`node`** (from PATH after ensureNode), not `npx`. Replace direct `config.MergeMCP` loop in `cmd/install.go`.

## YAML schema extension

Backward-compatible with current `mcp.yaml`:

```yaml
# config.d/mcp.yaml
agents:                    # optional override; default from install selector
  - kilo-code
  - opencode

servers:
  context7:
    type: remote
    url: https://mcp.context7.com/mcp
    transport: http        # optional: http | sse

  engram:
    type: local
    command:
      - engram
      - mcp
    env:                   # optional, stdio only
      - NAME=value

  exa:
    type: remote
    url: https://mcp.exa.ai/mcp
    headers:               # optional
      - "Authorization: Bearer ${EXA_TOKEN}"

  playwright:
    type: local
    command:
      - playwright-mcp
    auto_approve: false    # optional; maps to --auto-approve when true
```

Translation rules:

| mcp.yaml | add-mcp config |
|----------|----------------|
| `type: remote` + `url` | `{ type: "http", url }` or SSE when `transport: sse` |
| `type: local` + `command[]` | `{ command, args? }` — first elem command, rest args |
| `command: [gitnexus, mcp]` | Prefer binary on PATH (install via `tools.yaml`); not `npx -y …` |
| `headers` | passed to upsert; capability-gated per agent |
| `env` | stdio env vars |

## Agent mapping

| skillgrid selector | add-mcp `--agent` | Global config path |
|--------------------|-------------------|---------------------|
| `kilo` | `kilo-code` | `~/.config/kilo/kilo.jsonc` |
| `opencode` | `opencode` | `~/.config/opencode/opencode.jsonc` |
| `cursor` (future) | `cursor` | `~/.cursor/mcp.json` |
| `codex` (future) | `codex` | `~/.codex/config.toml` |
| `claude` (future) | `claude-code` | `~/.claude.json` |

## Sync script contract

**Path:** `scripts/sync-mcp-from-yaml.mjs`

**Inputs (env / flags):**

| Input | Purpose |
|-------|---------|
| `SKILLGRID_NPM_PREFIX` | `~/.skillgrid/npm` — resolve `add-mcp` package for `createRequire` |
| `SKILLGRID_MCP_YAML` | Path to mcp.yaml (default `~/.skillgrid/config.d/mcp.yaml`) |
| `--agents kilo-code,opencode` | Subset from install selector |
| `--dry-run` | Print upserts only |
| `--global` | Always true for skillgrid install |

**Behavior:**

1. Parse YAML `servers` map.
2. For each selected agent, for each server: `upsertServer(agent, name, config, { global: true })`.
3. Log `{ success, path, error? }` per upsert; exit non-zero only on parse errors (not per-server failure — match warn-and-continue).
4. Import add-mcp via `createRequire(path.join(SKILLGRID_NPM_PREFIX, 'node_modules/add-mcp/package.json'))` — **no `npx`**.

**Managed CLI path** (printed at end of install, same as other tools):

```bash
export PATH="$HOME/.skillgrid/npm/node_modules/.bin:$PATH"
add-mcp list -g -a kilo-code
```

**Example (remote):**

```javascript
upsertServer("kilo-code", "context7", {
  type: "http",
  url: "https://mcp.context7.com/mcp",
}, { global: true });
```

**Example (stdio):**

```javascript
upsertServer("opencode", "engram", {
  command: "engram",
  args: ["mcp"],
}, { global: true });
```

## Preserved skillgrid behaviors

| Behavior | How |
|----------|-----|
| Config-as-source-of-truth | Edit `mcp.yaml`, re-run install |
| Backups | Go backs up before sync (keep last 10) |
| Dry-run | `--dry-run install` → script logs only |
| Idempotent re-install | upsert overwrites same server name |
| Missing binary warn | Go Precheck before sync |
| PATH hint | Unchanged end-of-install output; includes `~/.skillgrid/npm/node_modules/.bin` |

## Behaviors not ported from MergeMCP

| Old MergeMCP | add-mcp approach |
|--------------|------------------|
| Rename on type conflict (`name-oldtype`) | upsert replaces; document manual backup if user relied on rename |
| JSONC comment preservation | add-mcp owns write path — verify comment retention on kilo.jsonc; if lost, document tradeoff |

**Spike required:** confirm add-mcp preserves JSONC comments on Kilo/OpenCode configs or accept rewrite.

## Operator workflows (post-integration)

All commands use the **managed binary** on PATH after install — not `npx`:

| Task | Command |
|------|---------|
| Managed install (canonical) | Edit `mcp.yaml` → `./bin/skillgrid install` |
| Discover new server | `add-mcp find vercel` (uses [add-mcp registry](https://add-mcp.com)) |
| List installed | `add-mcp list -g -a kilo-code` |
| Sync names across agents | `add-mcp sync -g -y` |
| Remove ad-hoc server | `add-mcp remove <name> -g -y` |

Full path before PATH setup: `~/.skillgrid/npm/node_modules/.bin/add-mcp`.

Document in `docs/04-mcp-servers.md`.

## Testing

| Area | Bar |
|------|-----|
| Sync script unit | Fixture mcp.yaml → mock upsertServer calls |
| Go install step | dry-run logs; fake add-mcp path |
| Integration | install → kilo.jsonc contains all mcp.yaml servers with correct shape |
| Regression | Existing `merger_test.go` retired or adapted |

## Success criteria

1. All servers in `config.d/mcp.yaml` appear in selected agent configs after install.
2. No hardcoded MCP URLs in Go — only YAML + sync script.
3. add-mcp installed from `tools.yaml` under `~/.skillgrid/npm`.
4. `docs/04-mcp-servers.md` documents YAML + add-mcp operator commands.
5. Adding a new agent = extend agent map + selector, not new merge code.

## Risks

| Risk | Mitigation |
|------|------------|
| JSONC comments stripped | Spike on kilo.jsonc; backup before write |
| add-mcp API drift | Pin version in tools.yaml |
| Runtime npx drift | Forbidden — install step uses `npm --prefix`; docs and Go never call `npx` |
| Env placeholders `${VAR}` | Document in mcp.yaml; interactive add-mcp behavior for manual installs only |
| Duplicate servers from old + new path | Single writer in install; remove MergeMCP in same release |

## References

- [neon-solutions/add-mcp](https://github.com/neon-solutions/add-mcp) — CLI + programmatic API
- [add-mcp.com](https://add-mcp.com) — registry and docs
- skillgrid [04-mcp-servers.md](../../04-mcp-servers.md) — current merge semantics
- [2026-08-26-gryph-integration-design.md](2026-08-26-gryph-integration-design.md) — pattern for npm tool + install step
