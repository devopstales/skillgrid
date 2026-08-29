# Installation

Install the `skillgrid` binary, then run `skillgrid install` to provision tools and wire agents.

## From source

```bash
task build
task install   # copies to ~/.skillgrid/bin/skillgrid
```

Requires Go 1.22+ and [Task](https://taskfile.dev/). `task all` cross-compiles linux/macOS amd64+arm64 into `dist/`.

Put `~/.skillgrid/bin` on your `PATH`:

```bash
export PATH="$HOME/.skillgrid/bin/:$PATH"
```

## First run

```bash
skillgrid install --agents opencode,kilo,cursor
# or omit --agents for interactive multi-select
```

Creates `~/.skillgrid/`, requires Node, syncs the hub, installs npm tools, installs selected agent CLIs + plugins, and copies `.agents/` to `~/.agents/`.

## Hub config

First use of `install` or `sync-repo` creates `~/.skillgrid/` if missing. The repo is cloned into `~/.skillgrid/repos/skillgrid`.

Override with `--repo-url` and `--branch`:

```bash
skillgrid install --repo-url https://github.com/fork/skillgrid --branch main
```

`--skip-clone` skips clone/update. The repo must already exist at `~/.skillgrid/repos/skillgrid`.

## Managed home

```text
~/.skillgrid/
├── bin/                         # skillgrid binary (put on PATH)
├── repos/
│   └── skillgrid/               # cloned or synced hub
│   │   ├── .agents/            # source copied into ~/.agents/
│   │   └── plugins/             # Mnemonic agent plugins
│   └── ...
└── mnemonic/                    # Mnemonic data: per-project SQLite stores
    ├── <project>.sqlite
    └── ...
```

## Uninstall

```bash
skillgrid install --help
```

There is no standalone uninstall command. To remove wiring, delete the agent config entries and `~/.agents/` overrides manually.

Removing the CLI binary is separate (`rm ~/.skillgrid/bin/skillgrid`).
