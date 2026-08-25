# Usage

Once built, the CLI is intentionally small: two commands, a handful of flags. The value is in what it does for you — reproducing the environment, not re-remembering it.

## Commands

| Command | Alias | Purpose |
|---------|-------|---------|
| `install` | `in` | Run the full install flow |
| `sync-repo` | — | Copy a local checkout into `~/.aiskillgrid/repos/aiskillgrid` (plus `config.d`) without running the rest |
| `help` | — | Print usage |

Flags parse the same before or after the command name: `install --dry-run` and `--dry-run install` both work.

## Flags

| Flag | Applies to | Effect |
|------|-----------|--------|
| `--skip-clone` | install | Skip the clone step; use existing `~/.aiskillgrid` state |
| `--sync-repo <path>` | install, sync-repo | Sync a local repo into `~/.aiskillgrid/repos/aiskillgrid` |
| `--dry-run` | install | Print planned changes; no npm installs, no MCP/rules writes, no backups |
| `--verbose` | install | Print detailed MCP entries instead of one-line summaries |
| `--yes` | install | Skip the interactive agent selector (defaults to all agents) |

Environment: `AISKILLGRID_REPO_URL` overrides the clone source.

## Typical Day-Loops

Reconcile the environment on this machine:

```bash
./bin/aiskillgrid install
```

Change to `config.d/` and preview the diff before committing:

```bash
./bin/aiskillgrid install --dry-run
```

Develop against a local checkout (fastest iteration loop — no network clone needed):

```bash
./bin/aiskillgrid install --sync-repo $(pwd)
```

See exactly what would land in each agent config:

```bash
./bin/aiskillgrid install --sync-repo $(pwd) --verbose
```

## PATH Setup

After a successful install, the CLI prints the two lines to add to your shell rc:

```bash
export PATH="$HOME/.aiskillgrid/bin:$PATH"
export PATH="$HOME/.aiskillgrid/node_modules/.bin:$PATH"
```

- `~/.aiskillgrid/bin` — the `engram` binary
- `~/.aiskillgrid/node_modules/.bin` — all agent CLIs and tools (kilo, opencode, skills, playwright, …) once `npm --prefix` installed them

Once in your rc file, you can invoke `kilo`, `opencode`, `engram`, `skills`, etc. directly.

## Logs

Every run appends to `~/.aiskillgrid/logs/install.log` at INFO/WARN/ERROR. If the terminal is unclear about what failed, read the log:

```bash
tail -50 ~/.aiskillgrid/logs/install.log
```

## Safety Model

- Every edit to `~/.config/kilo/kilo.jsonc` or `~/.config/opencode/opencode.jsonc` is preceded by a timestamped backup under `~/.aiskillgrid/backups/`.
- Backups keep the last 10 per file; older ones are pruned.
- The merge is JSON-aware and idempotent — running install twice does not duplicate keys.
- The `--dry-run` flag guarantees zero writes (no npm, no config edits, no backups).
