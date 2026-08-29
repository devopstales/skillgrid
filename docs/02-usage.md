# Usage

```bash
skillgrid <command> [flags]
```

| Command | Description |
|---|---|
| `install`, `in` | Run the full install pipeline (default) |
| `sync-repo PATH` | Sync a local git repo into `~/.skillgrid/repos/skillgrid` without cloning |
| `mcp` | Start the Mnemonic MCP stdio server (serves `mem_*`, `code_*`, `web_*` tools) |
| `serve` | Start the Mnemonic HTTP API (default `http://127.0.0.1:7438`) |
| `index` | Run incremental code indexing for a directory |
| `setup <agent>` | Install Mnemonic plugins for an agent (`opencode`, `kilocode`, `cursor`) |
| `help` | Show usage |

> Flags may appear before or after the subcommand — the parser reorders arguments
> so `skillgrid install --skip-clone` works the same as `skillgrid --skip-clone install`.

## Install flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--version` | `-v` | — | Print version and exit |
| `--skip-clone` | `-s` | off | Skip the git clone step (repo must already exist) |
| `--sync-repo` | `-n` | — | Sync a local repo PATH instead of cloning |
| `--verbose` | `-vv` | off | Print detailed output (node/npm versions, command output) |
| `--yes` | `-y` | off | Use defaults and skip prompts (`opencode, kilo, cursor`) |
| `--agents` | | — | Comma-separated agent keys, e.g. `opencode,kilo` |
| `--skip-tools` | | off | Skip global npm tool installation |
| `--skip-agents` | | off | Skip the `~/.agents/` copy step |
| `--repo-url` | | `https://github.com/devopstales/skillgrid` | Git URL to clone |
| `--branch` | | `release/2` | Branch to check out |
| `--dry-run` | | off | Print planned changes without writing |

## Agent-selection modes

| Mode | Behavior |
|---|---|
| Interactive (no flags, TTY) | Multi-select prompt; opencode + kilo pre-checked |
| `--yes` / `-y` | Selects all three (opencode, kilo, cursor) |
| `--agents opencode,kilo` | Preset list, no prompt |
| Non-interactive TTY | Defaults to opencode + kilo |

## Subcommand flags

| Subcommand | Flags |
|---|---|
| `mcp` | `--debug` |
| `serve` | `--port` (default `7438`, env `SKILLGRID_MNEMONIC_PORT`), `--bind` (default `127.0.0.1`), `--dir` (data dir, env `SKILLGRID_MNEMONIC_DATA_DIR`) |
| `index` | `--dir` (default `.`), `--project` (env `SKILLGRID_MNEMONIC_PROJECT`) |
| `setup` | `--agent`, `--repo-root`, `--dry-run` |

## sync-repo

```bash
skillgrid sync-repo /path/to/skillgrid       # subcommand form
skillgrid --sync-repo /path/to/skillgrid      # flag form (also works in install)
skillgrid --sync-repo /path --dry-run         # preview without writing
```

Copies an existing local checkout into `~/.skillgrid/repos/skillgrid` and then copies that repo's `.agents/` to `~/.agents/`. The destination is removed first (overwrite semantics).

## Troubleshooting

| Symptom | What to check |
|---|---|
| `missing from PATH: node, npm` | Install Node; `install` will not provision it |
| `repo root not found` | Run from a skillgrid checkout or use `--sync-repo` |
| `unknown agent` | Only `opencode`, `kilo`, `cursor` |
