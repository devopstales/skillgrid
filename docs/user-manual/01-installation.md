# Installation

Install the `skillgrid` binary, provision the hub and agents, then initialize SDD in each project.

## Requirements

- `git` on `PATH`
- `node` and `npm` on `PATH` (`install` hard-fails without them)
- Go 1.22+ and [Task](https://taskfile.dev/) to build from source

Skillgrid does **not** install Node or third-party API keys.

## Install the CLI

### From source

```bash
task build
task install   # copies to ~/.skillgrid/bin/skillgrid
```

`task all` cross-compiles linux/macOS amd64+arm64 into `dist/`.

Put the binary on your `PATH`:

```bash
export PATH="$HOME/.skillgrid/bin:$PATH"
```

### First machine setup

```bash
skillgrid install --agents opencode,kilo,cursor
# or omit --agents for interactive multi-select
```

This creates `~/.skillgrid/`, syncs the hub repo, installs npm tools (`skills`, `@cucumber/cucumber`), installs selected agent CLIs, merges MCP config, and copies hub `.agents/` → `~/.agents/`.

| Flag | Meaning |
|------|---------|
| `--yes` / `-y` | Defaults: opencode, kilo, cursor |
| `--skip-clone` | Hub already at `~/.skillgrid/repos/skillgrid` |
| `--sync-repo PATH` | Copy a local checkout instead of cloning |
| `--repo-url` / `--branch` | Override hub remote (default branch `release/2`) |
| `--dry-run` | Print planned actions only |
| `--skip-tools` / `--skip-agents` | Skip npm tools or `~/.agents/` copy |

```bash
skillgrid sync-repo /path/to/skillgrid   # local hub → ~/.skillgrid/repos/skillgrid
```

### Managed home

```text
~/.skillgrid/
├── bin/skillgrid
├── repos/skillgrid/     # hub (.agents/, plugins/, config.d/, scripts/)
└── mnemonic/            # per-project SQLite stores
```

Override data dir with `SKILLGRID_MNEMONIC_DATA_DIR`.

## Initialize a project (SDD)

Machine install ≠ project init. In each application repo, ask the agent:

> Run Skillgrid onboard / `use-skillgrid` — initialize SDD.

That routes to **`sdd-onboard` → `sdd-init`**, which:

1. Detects project name, stack, testing, and issue tracker
2. Confirms facts with you (blocking)
3. Writes `docs/skillgrid/` skeleton + `AGENTS.md` Skillgrid block

**Initialized when** `docs/skillgrid/config.yaml` exists (prefer also the `<!-- skillgrid-sdd:start -->` … `<!-- skillgrid-sdd:end -->` sentinel in `AGENTS.md`).

Default tracker: **Backlog.md** unless remote/config says GitHub, GitLab, or Jira.

Skeleton written:

```text
docs/skillgrid/
├── config.yaml
├── agents/          # issue-tracker, triage-labels (+ optional skill-registry)
├── glossary/        # business.md, technical.md
├── changes/
└── archive/
```

## Agents

Shipped targets: **OpenCode**, **Kilo**, **Cursor** (global config).

| | OpenCode | Kilo | Cursor |
|--|----------|------|--------|
| `--agents` key | `opencode` | `kilo` | `cursor` |
| npm package | `opencode-ai` | `@kilocode/cli` | — (app only) |
| Config | `~/.config/opencode/opencode.jsonc` | `~/.config/kilo/kilo.jsonc` | `~/.cursor/mcp.json` |
| Rules / plugins | `~/.config/opencode/` | `~/.config/kilo/` | `~/.cursor/rules/` |

MCP `command` values use absolute paths — agents do not need `~/.skillgrid/bin` on `PATH` to start servers.

After install, wire Mnemonic for one agent:

```bash
skillgrid setup opencode    # or kilocode / cursor (kilo → kilocode)
```

See [Plugins](09-plugins.md).

## Checklist

- [ ] `skillgrid --version` works
- [ ] `skillgrid install` completed for your agents
- [ ] Project has `docs/skillgrid/config.yaml` and AGENTS sentinel
- [ ] Agent can call `skillgrid mcp` (Mnemonic tools)

## Next step

[Workflow usage](02-workflow-usage.md) — run your first change.
