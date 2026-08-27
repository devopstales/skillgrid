# AISkillGrid

A **configuration hub** for opinionated AI-assisted development: reusable **skills**, **slash commands**, merged **MCP server** definitions, and an **install CLI** that copies normalized settings into your application repo. The workflow uses **OpenSpec**-style change management (specs under `openspec/`, `openspec` CLI) and spec-driven development skills together with production-oriented practices (testing, security, documentation).

---

## skillgrid (CLI)

The `skillgit` CLI installs the hub into this machine. It creates
`~/.skillgrid/`, copies the repo into `~/.skillgrid/repos/skillgrid`,
verifies Node is available, lets you pick which agent (Kilo, OpenCode,
Cursor) to install, runs `npm install -g` for the chosen agents and the
shared tools (`skills`, `openspec`), and copies the hub's `.agents/`
into `~/.agents/`.

```
skillgit --version                 print version
skillgit --help                    show help
skillgit                           install (default)
skillgit --dry-run                 print planned changes, don't write
skillgit --verbose                 print detailed changes
skillgit --yes                     use defaults (opencode + kilo)
skillgit --skip-clone              skip the git clone step
skillgit --agents opencode,kilo    preset agents (skip prompt)
skillgit --skip-tools              skip global npm tool install
skillgit --skip-agents             skip the ~/.agents override step
skillgit --sync-repo PATH          sync a local repo path into the hub instead of cloning
```

### Build

Requires [Task](https://taskfile.dev).

```bash
cd skillgrid-cli
task                 # local build          → dist/skillgit
task all             # linux amd64+386 + darwin amd64+arm64
task test            # go vet + go test
task clean           # remove dist/
task install         # copy dist/skillgit-linux-amd64 → ~/.skillgrid/bin/skillgit
```

Override the version string baked in with `-ldflags`:

```bash
SKILLGRID_VERSION=v1.0.0 task all
```

### Install

Once built (and `dist/skillgit-linux-amd64` is present), run:

```bash
task install
# or, anywhere on PATH:
skillgit install
```

`install` reuses the same code path: `skillgit install` will
clone-or-sync the hub repo, check Node, install the selected agents
(`opencode-ai`, `@kilocode/cli`, etc.) and the shared tools, then
copy `.agents/` into `~/.agents/`.

### Layout

```
~/.skillgit/
├── bin/                      # skillgit binary (put on PATH)
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
