# Installation

## Install the CLI

### From a release

macOS / Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/aiskillgrid/aiskillgrid/main/scripts/install.sh | bash
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/aiskillgrid/aiskillgrid/main/scripts/install.ps1 | iex
```

Release assets: `aiskillgrid-<version>-<os>-<arch>`  
(e.g. `aiskillgrid-0.1.0-darwin-arm64`, `aiskillgrid-0.1.0-windows-amd64.exe`).

Optional env: `AISKILLGRID_REPO`, `AISKILLGRID_VERSION`, `AISKILLGRID_INSTALL_DIR`.

### From source

```bash
cd aiskillgrid-cli
go build -o ../bin/aiskillgrid .
```

### Planned: Homebrew

```bash
brew install aiskillgrid/tap/aiskillgrid   # formula/tap TBD
```

Ship a Homebrew formula (or tap) that installs the `aiskillgrid` binary from GitHub Releases. Not implemented yet — see [TODO.md](TODO.md).

### Planned: Nix flake

Ship a **Nix flake** in this repo so users can install/run without a separate installer:

```bash
nix run .#aiskillgrid -- version
nix profile install .#aiskillgrid
# from GitHub (once published):
nix run github:aiskillgrid/aiskillgrid#aiskillgrid -- version
nix profile install github:aiskillgrid/aiskillgrid#aiskillgrid
```

Flake output name TBD (`aiskillgrid` packages/apps). Not a classic nixpkgs-only package as the primary story — flake-first. See [TODO.md](TODO.md).

## Requirements

- `git` on PATH (for `sync`)
- Network for first sync / release download

## Managed home

Override with `AISKILLGRID_HOME`.

```text
~/.aiskillgrid/
  config.yaml
  tools/              # git checkout of this repo
  dependencies/
    bin/              # native binaries (e.g. skills from qntx/skill)
  npm/                # isolated npm prefix + cache for MCP / Node tools + npx
  state.json
  logs/
  sessions/
  memories/
```

`aiskillgrid install` creates `npm/` and ensures Skillgrid-managed `npx` is available there. Skill packs use the **qntx/skill** binary under `dependencies/bin/`, not `npx skills`.

## After binary install

```bash
aiskillgrid sync
aiskillgrid install
aiskillgrid status
```

See [02-usage.md](02-usage.md).
