# AISkillGrid

A **configuration hub** for opinionated AI-assisted development: reusable **skills**, **slash commands**, merged **MCP server** definitions, and an **install CLI** that copies normalized settings into your application repo. The workflow uses **OpenSpec**-style change management (specs under `openspec/`) and spec-driven development skills together with production-oriented practices (testing, security, documentation).

---

## skillgrid (CLI)

The `skillgrid` CLI installs the hub into this machine. It creates
`~/.skillgrid/`, copies the repo into `~/.skillgrid/repos/skillgrid`,
verifies Node is available, lets you pick which agent (Kilo, OpenCode,
Cursor) to install, runs `npm install -g` for the chosen agents and the
shared tools (`skills`, `cucumber`), and copies the hub's `.agents/`
into `~/.agents/`.

```
skillgrid --version                 print version
skillgrid --help                    show help
skillgrid                           install (default)
skillgrid --dry-run                 print planned changes, don't write
skillgrid --verbose                 print detailed changes
skillgrid --yes                     use defaults (opencode + kilo)
skillgrid --skip-clone              skip the git clone step
skillgrid --agents opencode,kilo    preset agents (skip prompt)
skillgrid --skip-tools              skip global npm tool install
skillgrid --skip-agents             skip the ~/.agents override step
skillgrid --sync-repo PATH          sync a local repo path into the hub instead of cloning
```

### Build

Requires [Task](https://taskfile.dev). Run from the repo root — the
`Taskfile.yml` at the root builds the Go module in `skillgrid-cli/`.

```bash
task                 # local build          → dist/skillgrid
task all             # linux amd64+386 + darwin amd64+arm64
task test            # go vet + go test (in skillgrid-cli/)
task clean           # remove dist/
task install         # copy dist/skillgrid-linux-amd64 → ~/.skillgrid/bin/skillgrid
```

Override the version string baked in with `-ldflags`:

```bash
SKILLGRID_VERSION=v1.0.0 task all
```

### Install

Once built (and `dist/skillgrid-linux-amd64` is present), run:

```bash
task install
# or, anywhere on PATH:
skillgrid install
```

`install` reuses the same code path: `skillgrid install` will
clone-or-sync the hub repo, check Node, install the selected agents
(`opencode-ai`, `@kilocode/cli`, etc.) and the shared tools, then
copy `.agents/` into `~/.agents/`.

### Layout

```
~/.skillgrid/
├── bin/                      # skillgrid binary (put on PATH)
└── repos/
    └── skillgrid/            # cloned or synced hub
        ├── .agents/          # source that is copied into ~/.agents/
        ├── scripts/
        │   └── install_node.sh   # used when Node is missing
        └── ...
```

---

## License

[MIT](LICENSE)
