# Plugins

skillgrid installs **plugins** per agent in step 5 of the install flow. Plugins are richer than single MCP endpoints or skills — they bundle agent behaviors, hooks, and (for engram) a persistent memory backend.

Two plugins are installed by default:

| Plugin | Source | How it lands |
|--------|--------|--------------|
| superpowers | `obra/superpowers` (git) | npm `--prefix` install per agent + `plugin` key registration |
| engram | `Gentleman-Programming/engram` (binary) | `engram setup opencode` + `engram.ts` copied to kilo |

This doc focuses on the plugin mechanics. For skills (behavior definitions) see [05-skills](05-skills.md); for MCP servers see [04-mcp-servers](04-mcp-servers.md).

## What a plugin is here

- **superpowers** — a zero-dependency plugin from `obra/superpowers` that auto-loads a skill bootstrap at session start so skills trigger at the right moments. It is installed from the git ref `superpowers@git+https://github.com/obra/superpowers.git` (pinned in code as `superpowersRef`), not from the npm registry.
- **engram** — a prebuilt binary (installed in step 3) that provides the persistent-memory MCP server and a per-agent plugin file (`engram.ts`).

## superpowers

For each selected agent (kilo, opencode) the CLI does:

1. Installs the git ref into the agent's config dir:

```bash
npm install superpowers@git+https://github.com/obra/superpowers.git --prefix "$HOME/.config/kilo"

ln -sf ~/.config/kilo/node_modules/superpowers/.kilo/plugins/superpowers.js \
  ~/.config/kilo/plugins/superpowers.js
```

2. Registers the resolved path under the top-level `plugin` key of that agent's config (idempotent append):

```json
{
  "skills": {
    "paths": ["~/.config/kilo/node_modules/superpowers/skills"]
  }
}
```

```mermaid
flowchart LR
  Ref[obra/superpowers git ref] --> NPM[npm install --prefix .config/agent]
  NPM --> Reg["plugin: [ node_modules/superpowers ]"]
  Reg --> Kilo[kilo.jsonc]
  Reg --> Open[opencode.jsonc]
```

Notes:

- The `plugin` array append is idempotent — re-running does not duplicate the entry.
- The path is stored with `~` in place of the home directory, so the config stays portable across machines.
- A backup of the agent config is taken before the `plugin` key is written (`~/.skillgrid/backups/`).
- Plugin install failure **warns and continues** (per the error-handling contract) — the MCP and rules steps still run.

## engram

engram spans two install steps:

- **Step 3** installs the prebuilt binary to `~/.skillgrid/bin/engram`.
- **Step 5 (plugin)** wires it into the agents that need it:

```mermaid
flowchart LR
  Bin[engine binary in ~/.skillgrid/bin] --> Setup[engram setup opencode]
  Setup --> OpFile[.config/opencode/plugins/engram.ts]
  OpFile -->|copy if missing| KiFile[.config/kilo/plugins/engram.ts]
```

Concretely (`installPlugins`, `cmd/steps.go`):

1. If `opencode` is among the selected agents, run `engram setup opencode` (using the installed binary at `~/.skillgrid/bin/engram`, falling back to `engram` on PATH).
2. If kilo is selected and `~/.config/kilo/plugins/engram.ts` is missing, copy it from `~/.config/opencode/plugins/engram.ts`. This gives kilo the same engram plugin without running a second `engram setup`.

The engram memory backend is also exposed to agents as an MCP server (see [04-mcp-servers](04-mcp-servers.md) — the `engram` local entry `engram mcp`).

```bash
~/.aiskillgrid/bin/engram setup opencode
~/.aiskillgrid/bin/engram setup codex
~/.aiskillgrid/bin/engram setup cursor

cp ~/.config/opencode/plugins/engram.ts ~/.config/kilo/plugin/
cp ~/.config/opencode/tui.json ~/.config/kilo/tui.json

```

## Context Mode

After Superpowers, `install` wires [context-mode](https://github.com/mksglu/context-mode) for the selected agents. It is a **plugin**, not an MCP server.

```bash
git clone https://github.com/mksglu/context-mode.git ~/.cursor/plugins/local/context-mode

npm install context-mode --prefix "$HOME/.config/opencode"
# then
"plugin": ["context-mode"]
cp ~/.config/opencode/node_modules/context-mode/configs/opencode/AGENTS.md ~/.config/opencode/AGENTS.md

npm install context-mode --prefix "$HOME/.config/kilo"
# then
"plugin": ["context-mode"]
cp ~/.config/kilo/node_modules/context-mode/configs/opencode/AGENTS.md ~/.config/kilo/AGENTS.md

codex plugin marketplace add mksglu/context-mode
# then in ~/.codex/config.toml
[features]
plugin_hooks = true
hooks = true
```

## Why two mechanisms

- **superpowers** is source-first (skills bootstrap), so it ships as an npm/git plugin and is *registered* in the config.
- **engram** is binary-first (a runtime service + plugin file), so it is *installed* as a binary and its plugin file is *materialized* into each agent's `plugins/` dir.

Both end up as "the agent knows about this capability at session start", but the delivery path differs by what the tool actually is.

## Reconcile / re-run

`install` is idempotent for plugins:

- superpowers npm install is re-run (npm dedupes), and the `plugin` registration is an append-no-dup.
- engram `setup opencode` is re-run only when opencode is selected; the `engram.ts` copy is skipped if the target already exists.

To force a fresh superpowers build, remove the prefix first:

```bash
rm -rf ~/.config/kilo/node_modules/superpowers ~/.config/opencode/node_modules/superpowers
./bin/skillgrid install
```

## Reference

- superpowers: https://github.com/obra/superpowers
- engram: https://github.com/Gentleman-Programming/engram
