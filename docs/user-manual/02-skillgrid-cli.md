# skillgrid CLI

The `skillgrid` CLI is a Go binary (module `github.com/devopstales/skillgrid/skillgrid-cli`,
source in `skillgrid-cli/`) that installs the AI-assisted development hub onto a machine
and powers the Mnemonic persistent-memory engine.

## Commands

```
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
> Only the `mcp`, `serve`, `index`, and `setup` subcommands own private flag sets.

## Install pipeline

`install` (or bare `skillgrid`) executes these steps in order, writing progress to
stderr so stdout stays clean for scripts.

1. **Create structure** — `mkdir -p ~/.skillgrid` and `~/.skillgrid/repos/skillgrid`.
2. **Sync repo** — `git clone --branch <branch> <repo-url>` into
   `~/.skillgrid/repos/skillgrid`, or `git pull --ff-only` if it already exists.
3. **Check node + npm** — both must be on `PATH`. If missing, the run stops and
   prints the path to `scripts/install_node.sh` from the cloned repo.
4. **Select agents** — interactive multi-select (opencode + kilo pre-checked),
   or preset via `--agents`/`--yes`.
5. **Install agents** — `npm install -g <package>` for each agent that ships an
   npm package.
6. **Install MCP packages** — `npm install -g` for each package listed in
   `config.d/tools.yaml` (`mcp:` section).
7. **Configure MCP** — merge all servers from `config.d/mcp.yaml` into each
   selected agent's config file.
8. **Install global tools** — `npm install -g skills` and `npm install -g @cucumber/cucumber`.
9. **Copy `.agents/`** — the repo's `.agents/` directory is copied to `~/.agents/`.

### Available agents

| Key | Name | npm package | Notes |
|---|---|---|---|
| `opencode` | OpenCode | `opencode-ai` | |
| `kilo` | Kilo | `@kilocode/cli` | |
| `cursor` | Cursor | — | app-side only; no npm install |

### Global tools

| Name | npm package |
|---|---|
| `skills` | `skills` |
| `cucumber` | `@cucumber/cucumber` |

## Flags

### Install flags

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
| `--dry-run` | | off | Print planned changes without writing; each action is prefixed `[dry-run]` |

### Subcommand flags

| Subcommand | Flags |
|---|---|
| `mcp` | `--debug` |
| `serve` | `--port` (default `7438`, env `SKILLGRID_MNEMONIC_PORT`), `--bind` (default `127.0.0.1`), `--dir` (data dir, env `SKILLGRID_MNEMONIC_DATA_DIR`) |
| `index` | `--dir` (default `.`), `--project` (env `SKILLGRID_MNEMONIC_PROJECT`) |
| `setup` | `--agent`, `--repo-root`, `--dry-run` |

### Agent-selection modes

| Mode | Behavior |
|---|---|
| Interactive (no flags, TTY) | Multi-select prompt; opencode + kilo pre-checked |
| `--yes` / `-y` | Selects all three (opencode, kilo, cursor) |
| `--agents opencode,kilo` | Preset list, no prompt |
| Non-interactive TTY | Defaults to opencode + kilo |

## sync-repo

`sync-repo` copies an existing local checkout of the skillgrid repo into
`~/.skillgrid/repos/skillgrid` and then copies that repo's `.agents/` to
`~/.agents/`. The destination is removed first (overwrite semantics).

```
skillgrid sync-repo /path/to/skillgrid       # subcommand form
skillgrid --sync-repo /path/to/skillgrid      # flag form (also works in install)
skillgrid --sync-repo /path --dry-run         # preview without writing
```

## Build & install

Requires [Task](https://taskfile.dev). All commands run from the repo root.

```
task              # build local binary → dist/skillgrid
task all          # cross-build: linux amd64+386, darwin amd64+arm64
task test         # go vet && go test ./...  (in skillgrid-cli/)
task fmt          # gofmt -w .
task clean        # remove dist/
task install      # cp dist/skillgrid-linux-amd64 → ~/.skillgrid/bin/skillgrid
task version      # show the version string that would be baked in
```

The version string is derived from `git describe --tags --always --dirty` and can be
overridden with `SKILLGRID_VERSION=vX.Y.Z task all`. It is injected at build time via
`-ldflags "-X main.version=..."`.

## Layout

```
~/.skillgrid/
├── bin/                         # skillgrid binary (put on PATH)
├── repos/
│   └── skillgrid/               # cloned or synced hub
│   │   ├── .agents/            # source copied into ~/.agents/
│   │   ├── scripts/
│   │   │   └── install_node.sh  # used when Node is missing
│   │   └── plugins/             # Mnemonic agent plugins
│   │       ├── opencode/       # OpenCode plugin
│   │       ├── kilo/           # Kilo plugin
│   │       └── cursor/         # Cursor rule template
│   └── ...
└── mnemonic/                    # Mnemonic data: per-project SQLite stores
    ├── <project>.sqlite
    └── ...
```

### Config files

| Path | Purpose |
|---|---|
| `config.d/mcp.yaml` | MCP servers to merge into each agent's config |
| `config.d/tools.yaml` | npm packages to install for agents and MCP |
| `config.d/indexing.yaml` | Code indexing include/exclude patterns |

| Path | Purpose |
|---|---|
| `~/.skillgrid/bin/skillgrid` | The CLI binary |
| `~/.skillgrid/repos/skillgrid` | Cloned hub (source of `.agents/`, plugins, scripts) |
| `~/.agents/` | Agent override instructions (AGENTS.md, skills) injected into `~/.agents/` from the hub |
| `~/.skillgrid/mnemonic/` | Mnemonic data directory — one SQLite file per resolved project |

## Environment variables

| Variable | Scope | Description |
|---|---|---|
| `SKILLGRID_MNEMONIC_DATA_DIR` | Mnemonic | Override the data directory (default `~/.skillgrid/mnemonic`) |
| `SKILLGRID_MNEMONIC_PORT` | `serve` | Override the HTTP listen port (default `7438`) |
| `SKILLGRID_HTTP_TOKEN` | `serve` | Bearer token required for write routes (POST); read routes are unauthenticated |
| `SKILLGRID_MNEMONIC_PROJECT` | `index` | Fixed project identity |
